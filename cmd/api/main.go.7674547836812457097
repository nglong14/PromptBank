package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/nglong14/PromptBank/internal/config"
	"github.com/nglong14/PromptBank/internal/db"
	apihttp "github.com/nglong14/PromptBank/internal/http"
	"github.com/nglong14/PromptBank/internal/llm"
	"github.com/nglong14/PromptBank/internal/repository"
	"github.com/nglong14/PromptBank/internal/security"
)

func main() {
	cfg := config.FromEnv()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer pool.Close()

	if err := db.ApplyMigrations(ctx, pool); err != nil {
		log.Fatalf("apply migrations: %v", err)
	}

	// Dependency injection: Initialize repositories and services
	userRepo := repository.NewUserRepository(pool)
	promptRepo := repository.NewPromptRepository(pool)
	jwtManager := security.NewJWTManager(cfg.JWTSecret, cfg.JWTExpiresIn)

	// Initialize the LLM client (optional — LLM endpoints return 503 when nil).
	var llmClient *llm.Client
	if cfg.GeminiAPIKey != "" {
		c, err := llm.NewClient(cfg.GeminiAPIKey, cfg.GeminiModel, cfg.LLMMaxConcurrent)
		if err != nil {
			log.Fatalf("llm client: %v", err)
		}
		defer c.Close()
		llmClient = c
		log.Printf("llm: gemini enabled (model=%s, maxConcurrent=%d)", cfg.GeminiModel, cfg.LLMMaxConcurrent)
	} else {
		log.Println("llm: GEMINI_API_KEY not set — LLM features disabled")
	}

	// Initialize HTTP router
	router := apihttp.NewRouter(apihttp.Dependencies{
		UserRepo:    userRepo,
		PromptRepo:  promptRepo,
		JWTManager:  jwtManager,
		TokenPrefix: "Bearer",
		LLMClient:   llmClient,
	})

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Start HTTP server in background
	go func() {
		<-ctx.Done() // Wait for shutdown signal
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil { // Shutdown server gracefully
			log.Printf("server shutdown error: %v", err)
		}
	}()

	log.Printf("api listening on :%s", cfg.Port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed { // Listen and serve HTTP requests
		log.Fatalf("listen: %v", err)
	}
}
