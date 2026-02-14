package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/BuildGuard-Test-Lab/taskservice/internal/service"
)

type createTaskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type updateTaskRequest struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	Completed   *bool   `json:"completed,omitempty"`
}

func (h *Handler) handleListTasks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tasks, err := h.taskService.List(ctx)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list tasks")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"tasks": tasks,
		"count": len(tasks),
	})
}

func (h *Handler) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req createTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Title == "" {
		respondError(w, http.StatusBadRequest, "title is required")
		return
	}

	task, err := h.taskService.Create(ctx, req.Title, req.Description)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create task")
		return
	}

	respondJSON(w, http.StatusCreated, task)
}

func (h *Handler) handleGetTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	task, err := h.taskService.Get(ctx, id)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			respondError(w, http.StatusNotFound, "task not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to get task")
		return
	}

	respondJSON(w, http.StatusOK, task)
}

func (h *Handler) handleUpdateTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	var req updateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	task, err := h.taskService.Update(ctx, id, service.UpdateTaskInput{
		Title:       req.Title,
		Description: req.Description,
		Completed:   req.Completed,
	})
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			respondError(w, http.StatusNotFound, "task not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to update task")
		return
	}

	respondJSON(w, http.StatusOK, task)
}

func (h *Handler) handleDeleteTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	err := h.taskService.Delete(ctx, id)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			respondError(w, http.StatusNotFound, "task not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to delete task")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
