package config

import (
	"os"
	"strings"
	"time"
)

type Config struct {
	APIPort      string
	DatabaseURL  string
	RedisURL     string
	JWTSecret    string
	JWTExpiry    time.Duration
	CORSOrigins  []string
}

func Load() *Config {
	return &Config{
		APIPort:      getEnv("API_PORT", "8080"),
		DatabaseURL:  getEnv("DATABASE_URL", "postgres://livepulse:livepulse_dev@localhost:5432/livepulse?sslmode=disable"),
		RedisURL:     getEnv("REDIS_URL", "redis://localhost:6379/0"),
		JWTSecret:    getEnv("JWT_SECRET", "dev-secret-change-me"),
		JWTExpiry:    parseDuration(getEnv("JWT_EXPIRY", "24h")),
		CORSOrigins:  parseCORSOrigins(getEnv("CORS_ORIGINS", "http://localhost:3000")),
	}
}

func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 24 * time.Hour
	}
	return d
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseCORSOrigins(s string) []string {
	var origins []string
	for _, o := range strings.Split(s, ",") {
		o = strings.TrimSpace(o)
		if o != "" {
			origins = append(origins, o)
		}
	}
	return origins
}
