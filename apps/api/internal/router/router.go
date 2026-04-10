package router

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/Eahtasham/live-pulse/apps/api/internal/handler"
	"github.com/Eahtasham/live-pulse/apps/api/internal/middleware"
)

func New(startTime time.Time, authSvc handler.AuthService, sessionSvc handler.SessionServiceInterface, pollSvc handler.PollServiceInterface, jwtSecret string) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(middleware.Logger)
	r.Use(chimw.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Client-ID"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health check
	r.Get("/healthz", handler.Health(startTime))

	authHandler := handler.NewAuthHandler(authSvc)
	sessionHandler := handler.NewSessionHandler(sessionSvc)

	r.Route("/v1", func(r chi.Router) {
		// Public routes — no JWT required
		r.Post("/auth/callback", authHandler.Callback)
		r.Post("/auth/register", authHandler.Register)
		r.Post("/auth/login", authHandler.Login)
		r.Get("/sessions/{code}", sessionHandler.GetByCode)
		r.Post("/sessions/{code}/join", sessionHandler.Join)

		// Poll public routes (optional auth to detect host)
		if pollSvc != nil {
			pollHandler := handler.NewPollHandler(pollSvc)
			r.Group(func(r chi.Router) {
				r.Use(middleware.OptionalJWTAuth(jwtSecret))
				r.Get("/sessions/{code}/polls", pollHandler.List)
				r.Get("/sessions/{code}/polls/{pollID}", pollHandler.Get)
			})

			// Protected poll routes
			r.Group(func(r chi.Router) {
				r.Use(middleware.JWTAuth(jwtSecret))
				r.Post("/sessions/{code}/polls", pollHandler.Create)
				r.Patch("/sessions/{code}/polls/{pollID}", pollHandler.Update)
			})
		}

		// TODO: vote endpoints, Q&A submission endpoints (public)

		// Protected routes — JWT required
		r.Group(func(r chi.Router) {
			r.Use(middleware.JWTAuth(jwtSecret))
			r.Post("/sessions", sessionHandler.Create)
			r.Get("/sessions", sessionHandler.List)
			r.Patch("/sessions/{id}", placeholderHandler)
			r.Delete("/sessions/{id}", placeholderHandler)
		})
	})

	return r
}

func placeholderHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"not_implemented"}`))
}
