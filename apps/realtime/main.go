package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Eahtasham/live-pulse/apps/realtime/internal/config"
	"github.com/Eahtasham/live-pulse/apps/realtime/internal/handler"
	"github.com/Eahtasham/live-pulse/apps/realtime/internal/hub"
)

func main() {
	cfg := config.Load()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Start the WebSocket hub
	h := hub.NewHub()
	go h.Run()

	startTime := time.Now()

	r := chi.NewRouter()
	r.Get("/healthz", handler.Health(startTime))
	r.Get("/ws/{code}", handler.WebSocket(h, cfg.APIBaseURL))

	srv := &http.Server{
		Addr:        ":" + cfg.RealtimePort,
		Handler:     r,
		ReadTimeout: 15 * time.Second,
		IdleTimeout: 60 * time.Second,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("realtime service starting", "port", cfg.RealtimePort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-done
	slog.Info("realtime service shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	}

	slog.Info("realtime service stopped")
}
