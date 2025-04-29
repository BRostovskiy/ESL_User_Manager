package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/BorisRostovskiy/ESL/internal/handlers"
	"github.com/BorisRostovskiy/ESL/internal/log"
	health "github.com/hellofresh/health-go/v5"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/sirupsen/logrus"
)

const (
	HeaderContentType   = "Content-Type"
	HeaderContentLength = "Content-Length"
)

type response interface {
	WriteTo(w http.ResponseWriter) error
}

type handler struct {
	log *logrus.Logger
	api handlers.UsersService
}

func New(log *logrus.Logger, api handlers.UsersService, h *health.Health) http.Handler {
	return router(&handler{
		log: log,
		api: api,
	}, log, h)
}

func (h handler) handle(hf func(r *http.Request) response) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.respond(w, hf(r))
	}
}

func (h handler) respond(w http.ResponseWriter, r response) {
	w.Header().Set(HeaderContentType, "application/json; charset=utf-8")
	if err := r.WriteTo(w); err != nil {
		h.log.Errorf("failed to write response: %s", err)
	}
}

func router(h *handler, l *logrus.Logger, hh *health.Health) *chi.Mux {
	r := chi.NewRouter()

	r.Use(log.LoggerWithLevel("router", l, l.Level))
	r.Use(middleware.Recoverer)

	r.Route("/service/v1", func(r chi.Router) {
		r.Route("/users", func(r chi.Router) {
			r.Get("/", h.handle(h.listUsers))
			r.Post("/", h.handle(h.createUser))

			r.Route("/{uid}", func(r chi.Router) {
				r.Put("/", h.handle(h.updateUser))
				r.Delete("/", h.handle(h.deleteUser))
			})
		})
		r.Get("/health", hh.HandlerFunc)
	})

	return r
}

func responseObject(w http.ResponseWriter, status int, obj interface{}) error {
	data, err := json.Marshal(obj)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}
	w.Header().Set(HeaderContentLength, strconv.Itoa(len(data)))
	w.WriteHeader(status)
	_, err = w.Write(data)
	return err
}
