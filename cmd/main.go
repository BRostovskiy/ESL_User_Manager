package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/BorisRostovskiy/ESL/internal/clients"
	"github.com/BorisRostovskiy/ESL/internal/handlers"
	grpcServer "github.com/BorisRostovskiy/ESL/internal/handlers/grpc"
	pb "github.com/BorisRostovskiy/ESL/internal/handlers/grpc/gen/user-manager"
	httpHandler "github.com/BorisRostovskiy/ESL/internal/handlers/http"
	pgStorage "github.com/BorisRostovskiy/ESL/internal/repository/pg"
	"github.com/BorisRostovskiy/ESL/internal/service"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"

	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	httpHealth "github.com/hellofresh/health-go/v5"
	grpcHealthCheck "github.com/hellofresh/health-go/v5/checks/grpc"
	"github.com/namsral/flag"
	"github.com/sirupsen/logrus"
	"github.com/soheilhy/cmux"
	"google.golang.org/grpc"
	grpcHealth "google.golang.org/grpc/health"
	grpcHealthv1 "google.golang.org/grpc/health/grpc_health_v1"
	"gopkg.in/yaml.v3"
)

const (
	grpcHealthService = "health-service-grpc"
	postgresStorage   = "postgres"
)

var version = "dev"

func setupLogger(lvl string) *logrus.Logger {
	logger := logrus.New()
	logger.SetFormatter(&nested.Formatter{
		HideKeys:        true,
		FieldsOrder:     []string{"proto", "method", "component", "uri", "status_code", "bytes"},
		NoFieldsColors:  true,
		TimestampFormat: "2006-01-02 15:04:05",
	})
	switch strings.ToLower(lvl) {
	case "debug":
		logger.SetLevel(logrus.DebugLevel)
	case "info":
		logger.SetLevel(logrus.InfoLevel)
	case "error":
		logger.SetLevel(logrus.ErrorLevel)
	default:
		logger.SetLevel(logrus.DebugLevel)
	}
	return logger
}

func setupGRPC(l *logrus.Logger, users handlers.UsersService) *grpc.Server {
	loggingOptions := []logging.Option{
		logging.WithLogOnEvents(logging.StartCall, logging.FinishCall),
		logging.WithDurationField(logging.DurationToDurationField),
	}
	grpcS := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			logging.UnaryServerInterceptor(interceptorLogger(l), loggingOptions...),
		),
		grpc.ChainStreamInterceptor(
			logging.StreamServerInterceptor(interceptorLogger(l), loggingOptions...),
		),
	)

	reflection.Register(grpcS)
	pb.RegisterUserManagerServer(grpcS, grpcServer.New(users, l))
	healthServer := grpcHealth.NewServer()
	healthServer.SetServingStatus(grpcHealthService, grpcHealthv1.HealthCheckResponse_SERVING)
	grpcHealthv1.RegisterHealthServer(grpcS, healthServer)
	return grpcS
}

func mustSetupHTTP(logger *logrus.Logger, users handlers.UsersService, cfg config) *http.Server {
	h, err := httpHealth.New(
		httpHealth.WithSystemInfo(),
		httpHealth.WithComponent(httpHealth.Component{
			Name:    "User's manager HTTP service",
			Version: version,
		}))
	if err != nil {
		logrus.Fatal(err)
	}
	if err = h.Register(httpHealth.Config{
		Name:      "repository",
		Timeout:   time.Second * 5,
		SkipOnErr: true,
		Check: func(ctx context.Context) error {
			return users.HealthCheck(context.Background())
		},
	}); err != nil {
		logrus.Fatal(err)
	}

	if cfg.GRPC {
		check := grpcHealthCheck.New(grpcHealthCheck.Config{
			Target:  cfg.Handler.Addr,
			Service: grpcHealthService,
			DialOptions: []grpc.DialOption{
				grpc.WithTransportCredentials(insecure.NewCredentials()),
			},
		})
		if err = h.Register(httpHealth.Config{
			Name:      "grpc",
			Timeout:   time.Second * 5,
			SkipOnErr: true,
			Check: func(ctx context.Context) error {
				return check(context.Background())
			},
		}); err != nil {
			logrus.Fatal(err)
		}
	}

	// Creating a normal HTTP handlers
	return &http.Server{
		Handler: httpHandler.New(logger, users, h),
	}
}

func mustSetupStorage(cfg config, log *logrus.Logger) service.UserRepo {
	switch cfg.Storage.Type {
	case postgresStorage:
		store, err := pgStorage.New(&cfg.Storage.Config, log)
		if err != nil {
			logrus.Fatalf("failed to create repository: %v", err)
		}
		return store
	default:
		logrus.Fatalf("unknown repository: %s", cfg.Storage.Type)
	}
	return nil
}

func mustSetupConfig(configFile string) config {
	var cfg config
	if file, err := os.ReadFile(configFile); err != nil {
		logrus.Fatalf("failed to read configuration file: %s", err)
	} else if err = yaml.Unmarshal(file, &cfg); err != nil {
		logrus.Fatalf("failed to unmarshal configuration: %s", err)
	}
	return cfg
}

// interceptorLogger adapts logrus logger to interceptor logger
func interceptorLogger(logger logrus.FieldLogger) logging.Logger {
	return logging.LoggerFunc(func(_ context.Context, lvl logging.Level, msg string, fields ...any) {
		logrusFields := make(map[string]any, len(fields))
		iterator := logging.Fields(fields).Iterator()
		for iterator.Next() {
			fieldName, fieldValue := iterator.At()
			logrusFields[fieldName] = fieldValue
		}
		logger = logger.WithFields(logrusFields)

		switch lvl {
		case logging.LevelDebug:
			logger.Debug(msg)
		case logging.LevelInfo:
			logger.Info(msg)
		case logging.LevelWarn:
			logger.Warn(msg)
		case logging.LevelError:
			logger.Error(msg)
		default:
			panic(fmt.Sprintf("unknown level %v", lvl))
		}
	})
}

type canServe interface {
	Serve(net.Listener) error
}

func serve(s canServe, lis net.Listener) {
	go func() {
		if err := s.Serve(lis); err != nil &&
			!errors.Is(err, cmux.ErrServerClosed) {
			logrus.Fatal(err)
		}
	}()
}

type config struct {
	HTTP    bool `yaml:"HTTP"`
	GRPC    bool `yaml:"GRPC"`
	Handler struct {
		Addr string `yaml:"addr"`
	} `yaml:"handler"`

	Storage struct {
		Type   string           `yaml:"type"`
		Config pgStorage.Config `yaml:"config"`
	} `yaml:"storage"`
	Filters []string `yaml:"filters"`
}

func main() {
	var configFile string
	var logLevel string

	flags := flag.NewFlagSet("User Manager Service", flag.ContinueOnError)
	flags.StringVar(&logLevel, "log-level", "debug",
		"Log level. Available options: debug, info, warn, error")
	flags.StringVar(&configFile, "config_file", "/etc/um_config.yaml", "configuration file")
	flags.SetOutput(io.Discard)
	err := flags.Parse(os.Args[1:])

	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Printf("UserManager\n\n")
			fmt.Printf("USAGE\n\n  %s [OPTIONS]\n\n", os.Args[0])
			fmt.Print("OPTIONS\n\n")
			flags.SetOutput(os.Stdout)
			flags.PrintDefaults()
			os.Exit(0)
		} else {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	logger := setupLogger(logLevel)
	cfg := mustSetupConfig(configFile)
	storage := mustSetupStorage(cfg, logger)
	users := service.New(storage, logger, clients.NewChannelNotificationSvc(logger))

	// creating a listener for handlers
	l, err := net.Listen("tcp", cfg.Handler.Addr)
	if err != nil {
		logrus.Fatal(err)
	}

	m := cmux.New(l)

	var grpcSrv *grpc.Server
	if cfg.GRPC {
		grpcSrv = setupGRPC(logger, users)
		serve(grpcSrv, m.Match(cmux.HTTP2()))
	}

	var httpSrv *http.Server
	if cfg.HTTP {
		httpSrv = mustSetupHTTP(logger, users, cfg)
		serve(httpSrv, m.Match(cmux.HTTP1Fast()))
	}

	go func() {
		// actual listener
		if serveErr := m.Serve(); serveErr != nil &&
			(!strings.Contains(serveErr.Error(), "use of closed network connection")) &&
			!(errors.Is(serveErr, cmux.ErrServerClosed)) {
			logrus.Fatal(serveErr)
		}
	}()
	logger.Printf("======| listen on %s | server version: %s |======\n", cfg.Handler.Addr, version)

	gracefulStop := make(chan os.Signal, 2)
	signal.Notify(gracefulStop, syscall.SIGTERM, syscall.SIGINT)

	<-gracefulStop
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	if cfg.GRPC {
		grpcSrv.GracefulStop()
	}

	if cfg.HTTP {
		_ = httpSrv.Shutdown(ctx)
	}

	m.Close()
	logrus.Println("===DONE===")

}
