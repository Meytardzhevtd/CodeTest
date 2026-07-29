package tasks

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/meytardzhevtd/CodeTest/pkg/storage"
)

type Service struct {
	repo  RepositoryInterface
	store storage.Writer
}

func NewService(repo RepositoryInterface, store storage.Writer) *Service {
	return &Service{repo: repo, store: store}
}

func (s *Service) CreateTask(ctx context.Context, userID string, req CreateTaskRequest) (Task, error) {
	if err := validateCreateTask(req); err != nil {
		return Task{}, err
	}

	slug := normalizeSlug(req.Slug)
	title := strings.TrimSpace(req.Title)
	statement := strings.TrimSpace(req.Statement)
	difficulty := Difficulty(req.Difficulty)

	task := Task{
		Slug:          slug,
		Title:         title,
		Statement:     statement,
		CreatedBy:     userID,
		Difficulty:    difficulty,
		TimeLimitMs:   req.TimeLimitMs,
		MemoryLimitMb: req.MemoryLimitMb,
	}

	created, err := s.repo.CreateTask(ctx, task)
	if err != nil {
		if errors.Is(err, ErrSlugAlreadyExists) {
			return Task{}, ErrSlugAlreadyExists
		}
		return Task{}, fmt.Errorf("create task: %w", err)
	}

	return created, nil
}

func (s *Service) GetTaskByID(ctx context.Context, id string) (Task, error) {
	if id == "" {
		return Task{}, errors.New("task id cannot be empty")
	}

	task, err := s.repo.GetTaskByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			return Task{}, ErrTaskNotFound
		}
		return Task{}, fmt.Errorf("get task by id: %w", err)
	}

	return task, nil
}

func (s *Service) GetTaskBySlug(ctx context.Context, slug string) (Task, error) {
	if slug == "" {
		return Task{}, errors.New("task slug cannot be empty")
	}

	slug = normalizeSlug(slug)
	task, err := s.repo.GetTaskBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			return Task{}, ErrTaskNotFound
		}
		return Task{}, fmt.Errorf("get task by slug: %w", err)
	}

	return task, nil
}

func (s *Service) ListTasks(ctx context.Context, limit, offset int) ([]Task, int, error) {
	if limit < 1 || limit > 100 {
		return nil, 0, ErrInvalidLimit
	}
	if offset < 0 {
		return nil, 0, ErrInvalidOffset
	}

	tasks, total, err := s.repo.ListTasks(ctx, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list tasks: %w", err)
	}

	return tasks, total, nil
}

func (s *Service) DeleteTask(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("task id cannot be empty")
	}

	if err := s.repo.DeleteTask(ctx, id); err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			return ErrTaskNotFound
		}
		return fmt.Errorf("delete task: %w", err)
	}

	return nil
}

func validateCreateTask(req CreateTaskRequest) error {
	if strings.TrimSpace(req.Slug) == "" {
		return errors.New("slug is required")
	}
	if strings.TrimSpace(req.Title) == "" {
		return errors.New("title is required")
	}
	if strings.TrimSpace(req.Statement) == "" {
		return errors.New("statement is required")
	}
	if req.Difficulty == "" {
		return errors.New("difficulty is required")
	}

	difficulty := Difficulty(req.Difficulty)
	if difficulty != DifficultyEasy && difficulty != DifficultyMedium && difficulty != DifficultyHard {
		return ErrInvalidDifficulty
	}

	if req.TimeLimitMs < 1 {
		return errors.New("time_limit_ms must be greater than 0")
	}
	if req.MemoryLimitMb < 1 {
		return errors.New("memory_limit_mb must be greater than 0")
	}

	return nil
}

func validateTask(task Task) error {
	if strings.TrimSpace(task.Slug) == "" {
		return errors.New("slug is required")
	}
	if strings.TrimSpace(task.Title) == "" {
		return errors.New("title is required")
	}
	if strings.TrimSpace(task.Statement) == "" {
		return errors.New("statement is required")
	}
	if task.Difficulty != DifficultyEasy && task.Difficulty != DifficultyMedium && task.Difficulty != DifficultyHard {
		return ErrInvalidDifficulty
	}
	if task.TimeLimitMs < 1 {
		return errors.New("time_limit_ms must be greater than 0")
	}
	if task.MemoryLimitMb < 1 {
		return errors.New("memory_limit_mb must be greater than 0")
	}

	return nil
}

func normalizeSlug(slug string) string {
	slug = strings.TrimSpace(strings.ToLower(slug))
	slug = strings.ReplaceAll(slug, " ", "-")
	return slug
}
