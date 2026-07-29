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
	AddTagsToTask(ctx context.Context, taskID string, tagNames []string) ([]string, error)
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
		INSERT INTO tasks (slug, title, statement, difficulty, time_limit_ms, memory_limit_mb, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, slug, title, statement, difficulty, time_limit_ms, memory_limit_mb, created_by, created_at, updated_at
	`, task.Slug, task.Title, task.Statement, task.Difficulty, task.TimeLimitMs, task.MemoryLimitMb, task.CreatedBy).Scan(
		&created.ID,
		&created.Slug,
		&created.Title,
		&created.Statement,
		&created.Difficulty,
		&created.TimeLimitMs,
		&created.MemoryLimitMb,
		&created.CreatedBy,
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
		SELECT id, slug, title, statement, difficulty, time_limit_ms, memory_limit_mb, created_by, created_at, updated_at
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
		&task.CreatedBy,
		&task.CreatedAt,
		&task.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return Task{}, ErrTaskNotFound
		}
		return Task{}, fmt.Errorf("get task by id: %w", err)
	}
	if err := r.attachTags(ctx, []*Task{&task}); err != nil {
		return Task{}, err
	}
	return task, nil
}

func (r *Repository) GetTaskBySlug(ctx context.Context, slug string) (Task, error) {
	var task Task
	err := r.pool.QueryRow(ctx, `
		SELECT id, slug, title, statement, difficulty, time_limit_ms, memory_limit_mb, created_by, created_at, updated_at
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
		&task.CreatedBy,
		&task.CreatedAt,
		&task.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return Task{}, ErrTaskNotFound
		}
		return Task{}, fmt.Errorf("get task by slug: %w", err)
	}
	if err := r.attachTags(ctx, []*Task{&task}); err != nil {
		return Task{}, err
	}
	return task, nil
}

func (r *Repository) ListTasks(ctx context.Context, limit, offset int) ([]Task, int, error) {
	tasks := []Task{}
	rows, err := r.pool.Query(ctx, `
		SELECT t.id, t.slug, t.title, t.statement, t.difficulty, t.time_limit_ms, t.memory_limit_mb,
		       t.created_by, u.username, t.created_at, t.updated_at
		FROM tasks t
		JOIN users u ON u.id = t.created_by
		ORDER BY t.created_at DESC
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
			&task.CreatedBy,
			&task.CreatedByName,
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

	ptrs := make([]*Task, len(tasks))
	for i := range tasks {
		ptrs[i] = &tasks[i]
	}
	if err := r.attachTags(ctx, ptrs); err != nil {
		return nil, 0, err
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
		RETURNING id, slug, title, statement, difficulty, time_limit_ms, memory_limit_mb, created_by, created_at, updated_at
	`, id, updates.Slug, updates.Title, updates.Statement, updates.Difficulty, updates.TimeLimitMs, updates.MemoryLimitMb).Scan(
		&task.ID,
		&task.Slug,
		&task.Title,
		&task.Statement,
		&task.Difficulty,
		&task.TimeLimitMs,
		&task.MemoryLimitMb,
		&task.CreatedBy,
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

// attachTags fetches tags for the given tasks in a single query and fills
// in each task's Tags field (in place, hence the pointers).
func (r *Repository) attachTags(ctx context.Context, tasks []*Task) error {
	if len(tasks) == 0 {
		return nil
	}

	ids := make([]string, len(tasks))
	byID := make(map[string]*Task, len(tasks))
	for i, t := range tasks {
		ids[i] = t.ID
		byID[t.ID] = t
		t.Tags = []string{}
	}

	rows, err := r.pool.Query(ctx, `
		SELECT tt.task_id, t.name
		FROM task_tags tt
		JOIN tags t ON t.id = tt.tag_id
		WHERE tt.task_id = ANY($1)
		ORDER BY t.name
	`, ids)
	if err != nil {
		return fmt.Errorf("attach tags: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var taskID, name string
		if err := rows.Scan(&taskID, &name); err != nil {
			return fmt.Errorf("scan task tag: %w", err)
		}
		if t, ok := byID[taskID]; ok {
			t.Tags = append(t.Tags, name)
		}
	}
	return rows.Err()
}

// AddTagsToTask creates any tags that don't already exist (by name) and
// attaches all of them to the task, then returns the task's full tag list.
func (r *Repository) AddTagsToTask(ctx context.Context, taskID string, tagNames []string) ([]string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, name := range tagNames {
		var tagID string
		// ON CONFLICT DO UPDATE (a no-op change) so RETURNING still yields the
		// existing row's id when the tag already exists.
		err := tx.QueryRow(ctx, `
			INSERT INTO tags (name) VALUES ($1)
			ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
			RETURNING id
		`, name).Scan(&tagID)
		if err != nil {
			return nil, fmt.Errorf("get or create tag %q: %w", name, err)
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO task_tags (task_id, tag_id) VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`, taskID, tagID); err != nil {
			return nil, fmt.Errorf("attach tag %q to task: %w", name, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	task := &Task{ID: taskID}
	if err := r.attachTags(ctx, []*Task{task}); err != nil {
		return nil, err
	}
	return task.Tags, nil
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
