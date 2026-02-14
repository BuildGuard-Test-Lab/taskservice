package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/BuildGuard-Test-Lab/taskservice/internal/service"
)

type Postgres struct {
	pool *pgxpool.Pool
}

func NewPostgres(ctx context.Context, databaseURL string) (*Postgres, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parsing database URL: %w", err)
	}

	// Production-ready pool settings
	config.MaxConns = 25
	config.MinConns = 5

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("creating connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	return &Postgres{pool: pool}, nil
}

func (p *Postgres) Close() {
	p.pool.Close()
}

func (p *Postgres) Check(ctx context.Context) error {
	return p.pool.Ping(ctx)
}

func (p *Postgres) List(ctx context.Context) ([]service.Task, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT id, title, description, completed, created_at, updated_at
		FROM tasks
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("querying tasks: %w", err)
	}
	defer rows.Close()

	var tasks []service.Task
	for rows.Next() {
		var t service.Task
		if err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.Completed, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning task: %w", err)
		}
		tasks = append(tasks, t)
	}

	if tasks == nil {
		tasks = []service.Task{}
	}

	return tasks, rows.Err()
}

func (p *Postgres) Get(ctx context.Context, id string) (*service.Task, error) {
	var t service.Task
	err := p.pool.QueryRow(ctx, `
		SELECT id, title, description, completed, created_at, updated_at
		FROM tasks
		WHERE id = $1
	`, id).Scan(&t.ID, &t.Title, &t.Description, &t.Completed, &t.CreatedAt, &t.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, service.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("querying task: %w", err)
	}

	return &t, nil
}

func (p *Postgres) Create(ctx context.Context, task *service.Task) error {
	err := p.pool.QueryRow(ctx, `
		INSERT INTO tasks (title, description, completed, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, task.Title, task.Description, task.Completed, task.CreatedAt, task.UpdatedAt).Scan(&task.ID)

	if err != nil {
		return fmt.Errorf("inserting task: %w", err)
	}

	return nil
}

func (p *Postgres) Update(ctx context.Context, task *service.Task) error {
	result, err := p.pool.Exec(ctx, `
		UPDATE tasks
		SET title = $2, description = $3, completed = $4, updated_at = $5
		WHERE id = $1
	`, task.ID, task.Title, task.Description, task.Completed, task.UpdatedAt)

	if err != nil {
		return fmt.Errorf("updating task: %w", err)
	}

	if result.RowsAffected() == 0 {
		return service.ErrNotFound
	}

	return nil
}

func (p *Postgres) Delete(ctx context.Context, id string) error {
	result, err := p.pool.Exec(ctx, `DELETE FROM tasks WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deleting task: %w", err)
	}

	if result.RowsAffected() == 0 {
		return service.ErrNotFound
	}

	return nil
}
