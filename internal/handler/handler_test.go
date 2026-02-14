package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/BuildGuard-Test-Lab/taskservice/internal/handler"
	"github.com/BuildGuard-Test-Lab/taskservice/internal/repository"
	"github.com/BuildGuard-Test-Lab/taskservice/internal/service"
	"github.com/BuildGuard-Test-Lab/taskservice/pkg/health"
)

func setupTestHandler() *handler.Handler {
	repo := repository.NewMemory()
	svc := service.NewTaskService(repo)
	hc := health.NewChecker()
	return handler.New(svc, hc, "test")
}

func TestHealthEndpoints(t *testing.T) {
	h := setupTestHandler()
	router := h.Router()

	tests := []struct {
		name     string
		path     string
		wantCode int
	}{
		{"liveness", "/healthz", http.StatusOK},
		{"readiness", "/readyz", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantCode {
				t.Errorf("got status %d, want %d", rec.Code, tt.wantCode)
			}
		})
	}
}

func TestTaskCRUD(t *testing.T) {
	h := setupTestHandler()
	router := h.Router()

	// Create task
	body := bytes.NewBufferString(`{"title":"Test Task","description":"A test task"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("create: got status %d, want %d", rec.Code, http.StatusCreated)
	}

	var created map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	taskID, ok := created["id"].(string)
	if !ok {
		t.Fatal("created task has no id")
	}

	// Get task
	req = httptest.NewRequest(http.MethodGet, "/api/v1/tasks/"+taskID, nil)
	rec = httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("get: got status %d, want %d", rec.Code, http.StatusOK)
	}

	// List tasks
	req = httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil)
	rec = httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("list: got status %d, want %d", rec.Code, http.StatusOK)
	}

	var listResp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&listResp); err != nil {
		t.Fatalf("decode list response: %v", err)
	}

	count, ok := listResp["count"].(float64)
	if !ok || count != 1 {
		t.Errorf("list: got count %v, want 1", listResp["count"])
	}

	// Update task
	body = bytes.NewBufferString(`{"completed":true}`)
	req = httptest.NewRequest(http.MethodPut, "/api/v1/tasks/"+taskID, body)
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("update: got status %d, want %d", rec.Code, http.StatusOK)
	}

	// Delete task
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/tasks/"+taskID, nil)
	rec = httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("delete: got status %d, want %d", rec.Code, http.StatusNoContent)
	}

	// Verify deleted
	req = httptest.NewRequest(http.MethodGet, "/api/v1/tasks/"+taskID, nil)
	rec = httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("get deleted: got status %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestCreateTaskValidation(t *testing.T) {
	h := setupTestHandler()
	router := h.Router()

	// Missing title
	body := bytes.NewBufferString(`{"description":"No title"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("got status %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
