package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(h *Handler, logger Logger) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(LoggerMiddleware(logger))
	r.Use(RequestLoggerMiddleware(logger))
	r.Use(middleware.Recoverer)

	r.Get("/health", h.Health)
	r.Get("/proxies", h.GetProxies)
	r.Get("/proxies/random", h.GetRandomProxy)

	return r
}
