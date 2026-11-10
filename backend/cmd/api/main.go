package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"career-consultations-backend/internal/ai"
	"career-consultations-backend/internal/auth"
	"career-consultations-backend/internal/config"
	"career-consultations-backend/internal/httpapi"
	"career-consultations-backend/internal/notifications"
	"career-consultations-backend/internal/repository"
	"career-consultations-backend/internal/storage"
	"career-consultations-backend/internal/workers"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("configuration error", "error", err)
		os.Exit(1)
	}

	logger, err := buildLogger(cfg.Env)
	if err != nil {
		slog.Error("logger initialization failed", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := logger.Sync(); err != nil {
			slog.Warn("logger sync failed", "error", err)
		}
	}()

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
		if closeErr := server.Close(); closeErr != nil {
			logger.Error("HTTP server close failed", zap.Error(closeErr))
		}
	}
	logger.Info("server stopped")
}

func buildLogger(env string) (*zap.Logger, error) {
	if env == "production" {
		return zap.NewProduction()
	}
	return zap.NewDevelopment()
}
