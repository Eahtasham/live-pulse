package router

import (
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/Eahtasham/live-pulse/apps/api/internal/handler"
	"github.com/Eahtasham/live-pulse/apps/api/internal/middleware"
)

func New(startTime time.Time) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(middleware.Logger)
	r.Use(chimw.Recoverer)

	// Health check
	r.Get("/healthz", handler.Health(startTime))

	return r
}
