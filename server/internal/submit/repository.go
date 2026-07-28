package submit

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RepositoryInterface interface {
	Create(ctx context.Context, submission Submission) (Submission, error)
	GetByID(ctx context.Context, id string) (Submission, error)
	ListByUserID(ctx context.Context, userID string, limit, offset int) ([]Submission, int, error)
	ListByTaskID(ctx context.Context, taskID string, limit, offset int) ([]Submission, int, error)
	UpdateResult(ctx context.Context, id string, status Status, output, errMsg string) (Submission, error)
	Delete(ctx context.Context, id string) error
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, submission Submission) (Submission, error) {
	var created Submission
	err := r.pool.QueryRow(ctx, `
		INSERT INTO submissions (task_id, user_id, code, language, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, task_id, user_id, code, language, status, output, error, created_at, updated_at
	`, submission.TaskID, submission.UserID, submission.Code, submission.Language, submission.Status).Scan(
		&created.ID,
		&created.TaskID,
		&created.UserID,
		&created.Code,
		&created.Language,
		&created.Status,
		&created.Output,
		&created.Error,
		&created.CreatedAt,
		&created.UpdatedAt,
	)
	if err != nil {
		return Submission{}, fmt.Errorf("create submission: %w", err)
	}
	return created, nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (Submission, error) {
	var submission Submission
	err := r.pool.QueryRow(ctx, `
		SELECT id, task_id, user_id, code, language, status, output, error, created_at, updated_at
		FROM submissions
		WHERE id = $1
	`, id).Scan(
		&submission.ID,
		&submission.TaskID,
		&submission.UserID,
		&submission.Code,
		&submission.Language,
		&submission.Status,
		&submission.Output,
		&submission.Error,
		&submission.CreatedAt,
		&submission.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return Submission{}, ErrSubmissionNotFound
		}
		return Submission{}, fmt.Errorf("get submission by id: %w", err)
	}
	return submission, nil
}

func (r *Repository) ListByUserID(ctx context.Context, userID string, limit, offset int) ([]Submission, int, error) {
	submissions := []Submission{}

	rows, err := r.pool.Query(ctx, `
		SELECT id, task_id, user_id, code, language, status, output, error, created_at, updated_at
		FROM submissions
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list submissions by user: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var s Submission
		if err := rows.Scan(
			&s.ID,
			&s.TaskID,
			&s.UserID,
			&s.Code,
			&s.Language,
			&s.Status,
			&s.Output,
			&s.Error,
			&s.CreatedAt,
			&s.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan submission: %w", err)
		}
		submissions = append(submissions, s)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows error: %w", err)
	}

	var total int
	err = r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM submissions WHERE user_id = $1`, userID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count submissions: %w", err)
	}

	return submissions, total, nil
}

func (r *Repository) ListByTaskID(ctx context.Context, taskID string, limit, offset int) ([]Submission, int, error) {
	submissions := []Submission{}

	rows, err := r.pool.Query(ctx, `
		SELECT id, task_id, user_id, code, language, status, output, error, created_at, updated_at
		FROM submissions
		WHERE task_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, taskID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list submissions by task: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var s Submission
		if err := rows.Scan(
			&s.ID,
			&s.TaskID,
			&s.UserID,
			&s.Code,
			&s.Language,
			&s.Status,
			&s.Output,
			&s.Error,
			&s.CreatedAt,
			&s.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan submission: %w", err)
		}
		submissions = append(submissions, s)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows error: %w", err)
	}

	var total int
	err = r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM submissions WHERE task_id = $1`, taskID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count submissions: %w", err)
	}

	return submissions, total, nil
}

func (r *Repository) UpdateResult(ctx context.Context, id string, status Status, output, errMsg string) (Submission, error) {
	var submission Submission
	err := r.pool.QueryRow(ctx, `
		UPDATE submissions
		SET
			status = $2,
			output = $3,
			error = $4,
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, task_id, user_id, code, language, status, output, error, created_at, updated_at
	`, id, status, output, errMsg).Scan(
		&submission.ID,
		&submission.TaskID,
		&submission.UserID,
		&submission.Code,
		&submission.Language,
		&submission.Status,
		&submission.Output,
		&submission.Error,
		&submission.CreatedAt,
		&submission.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return Submission{}, ErrSubmissionNotFound
		}
		return Submission{}, fmt.Errorf("update submission result: %w", err)
	}
	return submission, nil
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	cmdTag, err := r.pool.Exec(ctx, `DELETE FROM submissions WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete submission: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrSubmissionNotFound
	}
	return nil
}
