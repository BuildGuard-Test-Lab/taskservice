package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/BuildGuard-Test-Lab/taskservice/internal/config"
	"github.com/BuildGuard-Test-Lab/taskservice/internal/handler"
	"github.com/BuildGuard-Test-Lab/taskservice/internal/repository"
	"github.com/BuildGuard-Test-Lab/taskservice/internal/service"
	"github.com/BuildGuard-Test-Lab/taskservice/pkg/health"
)

func main() {
	if err := run(); err != nil {
		slog.Error("application error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Setup structured logging
	logLevel := slog.LevelInfo
	if cfg.Debug {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	slog.Info("starting service",
		"version", cfg.Version,
		"environment", cfg.Environment,
	)

	// Initialize health checker
	healthChecker := health.NewChecker()

	// Initialize database
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var taskRepo service.TaskRepository
	if cfg.DatabaseURL != "" {
		db, err := repository.NewPostgres(ctx, cfg.DatabaseURL)
		if err != nil {
			return fmt.Errorf("connecting to database: %w", err)
		}
		defer db.Close()

		taskRepo = db
		healthChecker.AddCheck("database", db)
		slog.Info("connected to database")
	} else {
		// Use in-memory store for development
		taskRepo = repository.NewMemory()
		slog.Warn("using in-memory store (no DATABASE_URL)")
	}

	// Initialize service layer
	taskService := service.NewTaskService(taskRepo)

	// Initialize HTTP handler
	h := handler.New(taskService, healthChecker, cfg.Version)

	// Create server
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      h.Router(),
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		slog.Info("server listening", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- err
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errChan:
		return fmt.Errorf("server error: %w", err)
	case sig := <-quit:
		slog.Info("received shutdown signal", "signal", sig)
	}

	// Graceful shutdown
	slog.Info("shutting down server", "timeout", cfg.ShutdownTimeout)
	ctx, cancel = context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}

	slog.Info("server stopped gracefully")
	return nil
}
