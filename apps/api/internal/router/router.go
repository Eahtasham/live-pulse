package router

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	_ "github.com/Eahtasham/live-pulse/apps/api/docs"
	"github.com/Eahtasham/live-pulse/apps/api/internal/handler"
	"github.com/Eahtasham/live-pulse/apps/api/internal/middleware"
	httpSwagger "github.com/swaggo/http-swagger"
)

func New(startTime time.Time, authSvc handler.AuthService, sessionSvc handler.SessionServiceInterface, pollSvc handler.PollServiceInterface, voteSvc handler.VoteServiceInterface, qaSvc handler.QAServiceInterface, qaVoteSvc handler.QAVoteServiceInterface, jwtSecret string, corsOrigins []string) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(middleware.Logger)
	r.Use(chimw.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   corsOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Client-ID", "X-Audience-UID"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health check
	r.Get("/healthz", handler.Health(startTime))

	// Swagger docs
	r.Get("/swagger", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/index.html", http.StatusMovedPermanently)
	})
	r.Get("/swagger/*", httpSwagger.WrapHandler)

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

		// Vote endpoints (public, requires audience UID)
		if voteSvc != nil {
			voteHandler := handler.NewVoteHandler(voteSvc)
			r.Post("/sessions/{code}/polls/{pollID}/vote", voteHandler.CastVote)
		}

		// Q&A endpoints
		if qaSvc != nil {
			qaHandler := handler.NewQAHandler(qaSvc)
			// Public routes
			r.Get("/sessions/{code}/qa", qaHandler.List)
			r.Post("/sessions/{code}/qa", qaHandler.Create)
			// Protected moderation routes
			r.Group(func(r chi.Router) {
				r.Use(middleware.JWTAuth(jwtSecret))
				r.Patch("/sessions/{code}/qa/{id}", qaHandler.Moderate)
			})
		}

		// Q&A Vote endpoints (public, requires audience UID)
		if qaVoteSvc != nil {
			qaVoteHandler := handler.NewQAVoteHandler(qaVoteSvc)
			r.Post("/sessions/{code}/qa/{id}/vote", qaVoteHandler.CastVote)
		}

		// Protected routes — JWT required
		r.Group(func(r chi.Router) {
			r.Use(middleware.JWTAuth(jwtSecret))
			r.Post("/sessions", sessionHandler.Create)
			r.Get("/sessions", sessionHandler.List)
			r.Patch("/sessions/{code}/close", sessionHandler.Close)
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
