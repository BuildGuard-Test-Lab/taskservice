package health

import (
	"context"
	"fmt"
	"sync"
)

type CheckFunc interface {
	Check(ctx context.Context) error
}

type Checker struct {
	mu     sync.RWMutex
	checks map[string]CheckFunc
}

func NewChecker() *Checker {
	return &Checker{
		checks: make(map[string]CheckFunc),
	}
}

func (c *Checker) AddCheck(name string, check CheckFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.checks[name] = check
}

func (c *Checker) Check(ctx context.Context) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for name, check := range c.checks {
		if err := check.Check(ctx); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	return nil
}
