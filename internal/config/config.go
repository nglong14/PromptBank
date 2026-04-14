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
	// LLM configuration — optional; LLM features are disabled when GeminiAPIKey is empty.
	GeminiAPIKey      string
	GeminiModel       string
	LLMMaxConcurrent  int
}

func FromEnv() Config {
	return Config{
		Port:             env("PORT", "8080"),
		DatabaseURL:      env("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/promptbank?sslmode=disable"),
		JWTSecret:        env("JWT_SECRET", "dev-only-secret-change-me"),
		JWTExpiresIn:     envDuration("JWT_EXPIRES_MINUTES", 120),
		GeminiAPIKey:     env("GEMINI_API_KEY", ""),
		GeminiModel:      env("GEMINI_MODEL", "gemini-2.0-flash"),
		LLMMaxConcurrent: envInt("LLM_MAX_CONCURRENT", 5),
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envDuration(key string, fallbackMinutes int) time.Duration {
	value := env(key, strconv.Itoa(fallbackMinutes))
	minutes, err := strconv.Atoi(value)
	if err != nil || minutes <= 0 {
		minutes = fallbackMinutes
	}
	return time.Duration(minutes) * time.Minute
}

func envInt(key string, fallback int) int {
	value := env(key, strconv.Itoa(fallback))
	n, err := strconv.Atoi(value)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}
