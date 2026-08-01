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

// AddTagsToTask attaches tags to a task, creating any that don't already
// exist. Only the task's creator may do this.
func (s *Service) AddTagsToTask(ctx context.Context, userID, taskID string, tagNames []string) ([]string, error) {
	if taskID == "" {
		return nil, errors.New("task id cannot be empty")
	}

	names, err := normalizeTagNames(tagNames)
	if err != nil {
		return nil, err
	}
	if len(names) == 0 {
		return nil, ErrNoTagsProvided
	}

	task, err := s.repo.GetTaskByID(ctx, taskID)
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			return nil, ErrTaskNotFound
		}
		return nil, fmt.Errorf("get task by id: %w", err)
	}
	if task.CreatedBy != userID {
		return nil, ErrForbidden
	}

	tags, err := s.repo.AddTagsToTask(ctx, taskID, names)
	if err != nil {
		return nil, fmt.Errorf("add tags to task: %w", err)
	}

	return tags, nil
}

// MaxExamplesPerTask caps how many sample input/output pairs a task can show.
const MaxExamplesPerTask = 3

// SetExamples replaces a task's sample input/output pairs. Rows where both
// sides are blank are dropped silently; a row with only one side filled in
// is kept as-is. Only the task's creator may do this.
func (s *Service) SetExamples(ctx context.Context, userID, taskID string, inputs []ExampleInput) ([]Example, error) {
	if taskID == "" {
		return nil, errors.New("task id cannot be empty")
	}

	task, err := s.repo.GetTaskByID(ctx, taskID)
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			return nil, ErrTaskNotFound
		}
		return nil, fmt.Errorf("get task by id: %w", err)
	}
	if task.CreatedBy != userID {
		return nil, ErrForbidden
	}

	normalized := normalizeExamples(inputs)
	if len(normalized) > MaxExamplesPerTask {
		return nil, ErrTooManyExamples
	}

	examples, err := s.repo.SetExamples(ctx, taskID, normalized)
	if err != nil {
		return nil, fmt.Errorf("set examples: %w", err)
	}

	return examples, nil
}

// normalizeExamples drops rows where both input and output are blank -
// those represent an example slot the task creator left untouched.
func normalizeExamples(inputs []ExampleInput) []ExampleInput {
	normalized := make([]ExampleInput, 0, len(inputs))
	for _, ex := range inputs {
		if strings.TrimSpace(ex.Input) == "" && strings.TrimSpace(ex.Output) == "" {
			continue
		}
		normalized = append(normalized, ex)
	}
	return normalized
}

func normalizeTagNames(names []string) ([]string, error) {
	seen := make(map[string]struct{}, len(names))
	normalized := make([]string, 0, len(names))

	for _, raw := range names {
		name := normalizeSlug(raw)
		if name == "" {
			continue
		}
		if len(name) > 50 {
			return nil, ErrInvalidTagName
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		normalized = append(normalized, name)
	}

	return normalized, nil
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
