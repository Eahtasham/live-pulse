package config

import "os"

type Config struct {
	RealtimePort string
	RedisURL     string
	APIBaseURL   string
}

func Load() *Config {
	return &Config{
		RealtimePort: getEnv("REALTIME_PORT", "8081"),
		RedisURL:     getEnv("REDIS_URL", "redis://localhost:6379/0"),
		APIBaseURL:   getEnv("API_BASE_URL", "http://localhost:8080"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
