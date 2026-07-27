package tasks

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RepositoryInterface interface {
	CreateTask(ctx context.Context, task Task) (Task, error)
	GetTaskByID(ctx context.Context, id string) (Task, error)
	GetTaskBySlug(ctx context.Context, slug string) (Task, error)
	ListTasks(ctx context.Context, limit, offset int) ([]Task, int, error)
	UpdateTask(ctx context.Context, id string, updates Task) (Task, error)
	DeleteTask(ctx context.Context, id string) error
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) CreateTask(ctx context.Context, task Task) (Task, error) {
	var created Task
	err := r.pool.QueryRow(ctx, `
		INSERT INTO tasks (slug, title, statement, difficulty, time_limit_ms, memory_limit_mb)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, slug, title, statement, difficulty, time_limit_ms, memory_limit_mb, created_at, updated_at
	`, task.Slug, task.Title, task.Statement, task.Difficulty, task.TimeLimitMs, task.MemoryLimitMb).Scan(
		&created.ID,
		&created.Slug,
		&created.Title,
		&created.Statement,
		&created.Difficulty,
		&created.TimeLimitMs,
		&created.MemoryLimitMb,
		&created.CreatedAt,
		&created.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err, "tasks_slug_key") {
			return Task{}, ErrSlugAlreadyExists
		}
		return Task{}, fmt.Errorf("create task: %w", err)
	}
	return created, nil
}

func (r *Repository) GetTaskByID(ctx context.Context, id string) (Task, error) {
	var task Task
	err := r.pool.QueryRow(ctx, `
		SELECT id, slug, title, statement, difficulty, time_limit_ms, memory_limit_mb, created_at, updated_at
		FROM tasks
		WHERE id = $1
	`, id).Scan(
		&task.ID,
		&task.Slug,
		&task.Title,
		&task.Statement,
		&task.Difficulty,
		&task.TimeLimitMs,
		&task.MemoryLimitMb,
		&task.CreatedAt,
		&task.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return Task{}, ErrTaskNotFound
		}
		return Task{}, fmt.Errorf("get task by id: %w", err)
	}
	return task, nil
}

func (r *Repository) GetTaskBySlug(ctx context.Context, slug string) (Task, error) {
	var task Task
	err := r.pool.QueryRow(ctx, `
		SELECT id, slug, title, statement, difficulty, time_limit_ms, memory_limit_mb, created_at, updated_at
		FROM tasks
		WHERE slug = $1
	`, slug).Scan(
		&task.ID,
		&task.Slug,
		&task.Title,
		&task.Statement,
		&task.Difficulty,
		&task.TimeLimitMs,
		&task.MemoryLimitMb,
		&task.CreatedAt,
		&task.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return Task{}, ErrTaskNotFound
		}
		return Task{}, fmt.Errorf("get task by slug: %w", err)
	}
	return task, nil
}

func (r *Repository) ListTasks(ctx context.Context, limit, offset int) ([]Task, int, error) {
	tasks := []Task{}
	rows, err := r.pool.Query(ctx, `
		SELECT id, slug, title, statement, difficulty, time_limit_ms, memory_limit_mb, created_at, updated_at
		FROM tasks
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var task Task
		if err := rows.Scan(
			&task.ID,
			&task.Slug,
			&task.Title,
			&task.Statement,
			&task.Difficulty,
			&task.TimeLimitMs,
			&task.MemoryLimitMb,
			&task.CreatedAt,
			&task.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan task: %w", err)
		}
		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows error: %w", err)
	}

	var total int
	err = r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM tasks`).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count tasks: %w", err)
	}

	return tasks, total, nil
}

func (r *Repository) UpdateTask(ctx context.Context, id string, updates Task) (Task, error) {
	var task Task
	err := r.pool.QueryRow(ctx, `
		UPDATE tasks
		SET
			slug = COALESCE(NULLIF($2, ''), slug),
			title = COALESCE(NULLIF($3, ''), title),
			statement = COALESCE(NULLIF($4, ''), statement),
			difficulty = COALESCE($5, difficulty),
			time_limit_ms = COALESCE($6, time_limit_ms),
			memory_limit_mb = COALESCE($7, memory_limit_mb),
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, slug, title, statement, difficulty, time_limit_ms, memory_limit_mb, created_at, updated_at
	`, id, updates.Slug, updates.Title, updates.Statement, updates.Difficulty, updates.TimeLimitMs, updates.MemoryLimitMb).Scan(
		&task.ID,
		&task.Slug,
		&task.Title,
		&task.Statement,
		&task.Difficulty,
		&task.TimeLimitMs,
		&task.MemoryLimitMb,
		&task.CreatedAt,
		&task.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return Task{}, ErrTaskNotFound
		}
		if isUniqueViolation(err, "tasks_slug_key") {
			return Task{}, ErrSlugAlreadyExists
		}
		return Task{}, fmt.Errorf("update task: %w", err)
	}
	return task, nil
}

func (r *Repository) DeleteTask(ctx context.Context, id string) error {
	cmdTag, err := r.pool.Exec(ctx, `DELETE FROM tasks WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrTaskNotFound
	}
	return nil
}

func isUniqueViolation(err error, constraint string) bool {
	if err == nil {
		return false
	}
	pgErr, ok := err.(*pgconn.PgError)
	if !ok {
		return false
	}
	return pgErr.Code == "23505" && pgErr.ConstraintName == constraint
}
