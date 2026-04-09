package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Eahtasham/live-pulse/apps/api/internal/config"
	"github.com/Eahtasham/live-pulse/apps/api/internal/db"
	"github.com/Eahtasham/live-pulse/apps/api/internal/router"
	"github.com/Eahtasham/live-pulse/apps/api/internal/service"
)

func main() {
	cfg := config.Load()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Connect to PostgreSQL via GORM
	gormDB, err := db.NewPostgres(cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}
	sqlDB, _ := gormDB.DB()
	defer sqlDB.Close()

	// Connect to Redis
	rdb, err := db.NewRedis(cfg.RedisURL)
	if err != nil {
		slog.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	defer rdb.Close()

	startTime := time.Now()
	svc := service.New(gormDB, cfg.JWTSecret, cfg.JWTExpiry)
	sessionSvc := service.NewSessionService(gormDB, rdb)
	r := router.New(startTime, svc, sessionSvc, cfg.JWTSecret)

	srv := &http.Server{
		Addr:         ":" + cfg.APIPort,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("api service starting", "port", cfg.APIPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-done
	slog.Info("api service shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	}

	slog.Info("api service stopped")
}
