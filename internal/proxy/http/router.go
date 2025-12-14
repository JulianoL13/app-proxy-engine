package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter creates the HTTP router with all routes configured
func NewRouter(h *Handler) http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	// Routes
	r.Get("/health", h.Health)
	r.Get("/proxies", h.GetProxies)
	r.Get("/proxies/random", h.GetRandomProxy)

	return r
}
