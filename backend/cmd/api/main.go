package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"career-consultations-backend/internal/ai"
	"career-consultations-backend/internal/auth"
	"career-consultations-backend/internal/config"
	"career-consultations-backend/internal/httpapi"
	"career-consultations-backend/internal/notifications"
	"career-consultations-backend/internal/repository"
	"career-consultations-backend/internal/storage"
	"career-consultations-backend/internal/workers"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		_, _ = os.Stderr.WriteString("configuration error: " + err.Error() + "\n")
		os.Exit(1)
	}
	logger, err := buildLogger(cfg.Env)
	if err != nil {
		_, _ = os.Stderr.WriteString("logger initialization failed: " + err.Error() + "\n")
		os.Exit(1)
	}
	defer func() { _ = logger.Sync() }()

	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	repo, err := repository.New(rootCtx, cfg.DatabaseURL)
	if err != nil {
		logger.Fatal("database initialization failed", zap.Error(err))
	}
	defer repo.Pool.Close()
	fileStorage, err := storage.NewLocal(cfg.StorageDir)
	if err != nil {
		logger.Fatal("file storage initialization failed", zap.Error(err))
	}

	var provider ai.Provider
	aiReady := cfg.AIProvider == "openai" || cfg.AIProvider == "openai-compatible"
	if aiReady {
		provider = ai.NewOpenAICompatible(cfg.AIBaseURL, cfg.AIAPIKey, cfg.AIModel, cfg.AITimeout)
		worker := workers.NewResumeWorker(repo, fileStorage, provider, logger)
		go worker.Run(rootCtx)
	} else if cfg.AIProvider != "" {
		logger.Fatal("unsupported AI_PROVIDER; use openai or openai-compatible", zap.String("provider", cfg.AIProvider))
	}
	if cfg.ReminderEnabled {
		sender := notifications.NewVKMessenger(cfg.NotificationAPIToken, cfg.VKAPIVersion)
		go workers.NewReminderWorker(repo, sender, logger).Run(rootCtx)
	} else {
		logger.Info("consultation reminders disabled; set REMINDER_ENABLED=true and configure a VK community token to enable delivery")
	}

	tokenService := auth.NewTokenService(cfg.JWTSecret, cfg.JWTAccessTTL)
	api := httpapi.New(cfg, repo, tokenService, fileStorage, logger, aiReady)
	server := &http.Server{
		Addr: cfg.HTTPAddr, Handler: api.Handler(),
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout, IdleTimeout: cfg.IdleTimeout,
	}
	go func() {
		logger.Info("HTTP server starting", zap.String("addr", cfg.HTTPAddr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("HTTP server failed", zap.Error(err))
		}
	}()
	<-rootCtx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", zap.Error(err))
		_ = server.Close()
	}
	logger.Info("server stopped")
}

func buildLogger(env string) (*zap.Logger, error) {
	if env == "production" {
		return zap.NewProduction()
	}
	return zap.NewDevelopment()
}
