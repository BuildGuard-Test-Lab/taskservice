package repository

import (
	"context"
	"fmt"
	"sync"

	"github.com/BuildGuard-Test-Lab/taskservice/internal/service"
)

type Memory struct {
	mu     sync.RWMutex
	tasks  map[string]*service.Task
	nextID int
}

func NewMemory() *Memory {
	return &Memory{
		tasks: make(map[string]*service.Task),
	}
}

func (m *Memory) List(ctx context.Context) ([]service.Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tasks := make([]service.Task, 0, len(m.tasks))
	for _, t := range m.tasks {
		tasks = append(tasks, *t)
	}
	return tasks, nil
}

func (m *Memory) Get(ctx context.Context, id string) (*service.Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	task, ok := m.tasks[id]
	if !ok {
		return nil, service.ErrNotFound
	}
	return task, nil
}

func (m *Memory) Create(ctx context.Context, task *service.Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.nextID++
	task.ID = fmt.Sprintf("%d", m.nextID)
	m.tasks[task.ID] = task
	return nil
}

func (m *Memory) Update(ctx context.Context, task *service.Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.tasks[task.ID]; !ok {
		return service.ErrNotFound
	}
	m.tasks[task.ID] = task
	return nil
}

func (m *Memory) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.tasks[id]; !ok {
		return service.ErrNotFound
	}
	delete(m.tasks, id)
	return nil
}
