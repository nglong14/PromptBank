package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port         string
	DatabaseURL  string
	JWTSecret    string
	JWTExpiresIn time.Duration
}

func FromEnv() Config {
	return Config{
		Port:         env("PORT", "8080"),
		DatabaseURL:  env("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/promptbank?sslmode=disable"),
		JWTSecret:    env("JWT_SECRET", "dev-only-secret-change-me"),
		JWTExpiresIn: envDuration("JWT_EXPIRES_MINUTES", 120),
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// Convert environment variable to duration
func envDuration(key string, fallbackMinutes int) time.Duration {
	value := env(key, strconv.Itoa(fallbackMinutes))
	minutes, err := strconv.Atoi(value)
	if err != nil || minutes <= 0 {
		minutes = fallbackMinutes
	}
	return time.Duration(minutes) * time.Minute
}
