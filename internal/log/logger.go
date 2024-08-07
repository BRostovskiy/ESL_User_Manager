package log

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/sirupsen/logrus"
)

func WithError(r *http.Request, err error) {
	if le := getLogEntry(r); le != nil {
		le.WithError(err)
	}
}

func WithErrorf(r *http.Request, format string, a ...interface{}) error {
	err := fmt.Errorf(format, a...)
	if le := getLogEntry(r); le != nil {
		le.WithError(err)
	}
	return err
}

type Logger interface {
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

type logEntry struct {
	log Logger
	err []error
}

func (le *logEntry) Write(status, bytes int, header http.Header, elapsed time.Duration, extra interface{}) {
	var printf func(format string, args ...interface{})
	switch {
	case status < 400:
		if len(le.err) > 0 {
			printf = le.log.Warnf
		} else {
			printf = le.log.Infof
		}
	case status < 500:
		printf = le.log.Warnf
	default:
		printf = le.log.Errorf
	}

	elapsed = elapsed.Round(time.Millisecond)
	text := http.StatusText(status)
	if len(le.err) > 0 {
		var message string
		for i := len(le.err) - 1; i >= 0; i-- {
			if len(message) > 0 {
				message += " | "
			}
			message += le.err[i].Error()
		}
		printf("%03d (%s) in %s - %s", status, text, elapsed, message)
	} else {
		printf("%03d (%s) in %s", status, text, elapsed)
	}
}

func (le *logEntry) Panic(v interface{}, stack []byte) {
	middleware.PrintPrettyStack(v)
}

func (le *logEntry) WithError(err error) *logEntry {
	if err != nil {
		le.err = append(le.err, err)
	}
	return le
}

func getLogEntry(r *http.Request) *logEntry {
	le, _ := middleware.GetLogEntry(r).(*logEntry)
	return le
}

// LoggerWithLevel returns a request logging middleware
func LoggerWithLevel(component string, logger logrus.FieldLogger, level logrus.Level) func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			reqID := middleware.GetReqID(r.Context())
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			t1 := time.Now()
			defer func() {
				remoteIP, _, err := net.SplitHostPort(r.RemoteAddr)
				if err != nil {
					remoteIP = r.RemoteAddr
				}

				fields := logrus.Fields{
					"status_code":      ww.Status(),
					"bytes":            ww.BytesWritten(),
					"duration_display": time.Since(t1).String(),
					"component":        component,
					"remote_ip":        remoteIP,
					"proto":            r.Proto,
					"method":           r.Method,
				}
				if len(reqID) > 0 {
					fields["request_id"] = reqID
				}

				logger.
					WithField("uri", fmt.Sprintf("%s%s", r.Host, r.RequestURI)).
					WithFields(fields).Log(level)
			}()

			h.ServeHTTP(ww, r)
		}

		return http.HandlerFunc(fn)
	}
}
