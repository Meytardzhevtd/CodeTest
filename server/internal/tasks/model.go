package tasks

import (
	"time"
)

type Difficulty string

const (
	DifficultyEasy   Difficulty = "easy"
	DifficultyMedium Difficulty = "medium"
	DifficultyHard   Difficulty = "hard"
)

type Task struct {
	ID            string     `json:"id"`
	Slug          string     `json:"slug"`
	Title         string     `json:"title"`
	Statement     string     `json:"statement"`
	CreatedBy     string     `json:"created_by"`
	CreatedByName string     `json:"created_by_username"`
	Difficulty    Difficulty `json:"difficulty"`
	TimeLimitMs   int        `json:"time_limit_ms"`
	MemoryLimitMb int        `json:"memory_limit_mb"`
	Tags          []string   `json:"tags"`
	Examples      []Example  `json:"examples"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// Tag is a single label a task can be marked with. Name doubles as the
// tag's slug - it's normalized (lowercase, spaces to dashes) and unique.
type Tag struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Example is one sample input/output pair shown on the task page to
// illustrate the expected format. Either side may be blank - a row is only
// dropped entirely (never stored) when both sides are blank.
type Example struct {
	ID     string `json:"id"`
	Input  string `json:"input"`
	Output string `json:"output"`
}

// ExampleInput is one example as submitted by the task creator, before
// blank rows are filtered out and the rest are persisted.
type ExampleInput struct {
	Input  string `json:"input"`
	Output string `json:"output"`
}

type AddTagsRequest struct {
	Tags []string `json:"tags"`
}

type AddTagsResponse struct {
	Tags []string `json:"tags"`
}

type SetExamplesRequest struct {
	Examples []ExampleInput `json:"examples"`
}

type SetExamplesResponse struct {
	Examples []Example `json:"examples"`
}

type CreateTaskRequest struct {
	Slug          string `json:"slug"`
	Title         string `json:"title"`
	Statement     string `json:"statement"`
	Difficulty    string `json:"difficulty"`
	TimeLimitMs   int    `json:"time_limit_ms"`
	MemoryLimitMb int    `json:"memory_limit_mb"`
}

type UpdateTaskRequest struct {
	Title         *string `json:"title,omitempty"`
	Statement     *string `json:"statement,omitempty"`
	Difficulty    *string `json:"difficulty,omitempty"`
	TimeLimitMs   *int    `json:"time_limit_ms,omitempty"`
	MemoryLimitMb *int    `json:"memory_limit_mb,omitempty"`
}

type TaskResponse struct {
	Task Task `json:"task"`
}

type ListTasksResponse struct {
	Tasks []Task `json:"tasks"`
	Total int    `json:"total"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
