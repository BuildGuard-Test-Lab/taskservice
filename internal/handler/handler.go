package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/BuildGuard-Test-Lab/taskservice/internal/service"
	"github.com/BuildGuard-Test-Lab/taskservice/pkg/health"
)

type Handler struct {
	taskService   *service.TaskService
	healthChecker *health.Checker
	version       string
}

func New(ts *service.TaskService, hc *health.Checker, version string) *Handler {
	return &Handler{
		taskService:   ts,
		healthChecker: hc,
		version:       version,
	}
}

func (h *Handler) Router() http.Handler {
	r := chi.NewRouter()

	// Middleware stack
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(structuredLogger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	// Health & metrics (no auth)
	r.Get("/healthz", h.handleLiveness)
	r.Get("/readyz", h.handleReadiness)
	r.Handle("/metrics", promhttp.Handler())

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/", h.handleRoot)

		r.Route("/tasks", func(r chi.Router) {
			r.Get("/", h.handleListTasks)
			r.Post("/", h.handleCreateTask)
			r.Get("/{id}", h.handleGetTask)
			r.Put("/{id}", h.handleUpdateTask)
			r.Delete("/{id}", h.handleDeleteTask)
		})
	})

	return r
}

func structuredLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		defer func() {
			slog.Info("request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"duration_ms", time.Since(start).Milliseconds(),
				"bytes", ww.BytesWritten(),
				"request_id", middleware.GetReqID(r.Context()),
			)
		}()

		next.ServeHTTP(ww, r)
	})
}

func (h *Handler) handleRoot(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{
		"service": "taskservice",
		"version": h.version,
	})
}

func (h *Handler) handleLiveness(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"status": "alive"})
}

func (h *Handler) handleReadiness(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if err := h.healthChecker.Check(ctx); err != nil {
		slog.Warn("readiness check failed", "error", err)
		respondJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "not ready",
			"error":  err.Error(),
		})
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to encode response", "error", err)
	}
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}
