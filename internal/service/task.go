package service

import (
	"context"
	"errors"
	"time"
)

var ErrNotFound = errors.New("not found")

type Task struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Completed   bool      `json:"completed"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type TaskRepository interface {
	List(ctx context.Context) ([]Task, error)
	Get(ctx context.Context, id string) (*Task, error)
	Create(ctx context.Context, task *Task) error
	Update(ctx context.Context, task *Task) error
	Delete(ctx context.Context, id string) error
}

type TaskService struct {
	repo TaskRepository
}

func NewTaskService(repo TaskRepository) *TaskService {
	return &TaskService{repo: repo}
}

type UpdateTaskInput struct {
	Title       *string
	Description *string
	Completed   *bool
}

func (s *TaskService) List(ctx context.Context) ([]Task, error) {
	return s.repo.List(ctx)
}

func (s *TaskService) Get(ctx context.Context, id string) (*Task, error) {
	return s.repo.Get(ctx, id)
}

func (s *TaskService) Create(ctx context.Context, title, description string) (*Task, error) {
	task := &Task{
		Title:       title,
		Description: description,
		Completed:   false,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if err := s.repo.Create(ctx, task); err != nil {
		return nil, err
	}

	return task, nil
}

func (s *TaskService) Update(ctx context.Context, id string, input UpdateTaskInput) (*Task, error) {
	task, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.Title != nil {
		task.Title = *input.Title
	}
	if input.Description != nil {
		task.Description = *input.Description
	}
	if input.Completed != nil {
		task.Completed = *input.Completed
	}
	task.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, task); err != nil {
		return nil, err
	}

	return task, nil
}

func (s *TaskService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
