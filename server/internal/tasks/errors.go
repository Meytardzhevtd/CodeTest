package tasks

import "errors"

var (
	ErrTaskNotFound      = errors.New("task not found")
	ErrSlugAlreadyExists = errors.New("slug already exists")
	ErrInvalidDifficulty = errors.New("invalid difficulty: must be easy, medium, or hard")
	ErrInvalidLimit      = errors.New("limit must be between 1 and 100")
	ErrInvalidOffset     = errors.New("offset must be non-negative")
	ErrForbidden         = errors.New("forbidden: not the task creator")
	ErrInvalidArchive    = errors.New("invalid tests archive")
	ErrNoTagsProvided    = errors.New("at least one tag must be provided")
	ErrInvalidTagName    = errors.New("invalid tag name: must be 1-50 characters")
	ErrTooManyExamples   = errors.New("at most 3 examples are allowed per task")
)
