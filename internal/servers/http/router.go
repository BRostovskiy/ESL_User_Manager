package http

import (
	"github.com/BorisRostovskiy/ESL/internal/log"
	"github.com/hellofresh/health-go/v5"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/sirupsen/logrus"
)

func router(h *handler, l *logrus.Logger, hh *health.Health) *chi.Mux {
	r := chi.NewRouter()

	r.Use(log.LoggerWithLevel("router", l, l.Level))
	r.Use(middleware.Recoverer)

	r.Route("/api/v1", func(r chi.Router) {
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
