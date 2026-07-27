package tasks

import "errors"

var (
	ErrTaskNotFound      = errors.New("task not found")
	ErrSlugAlreadyExists = errors.New("slug already exists")
	ErrInvalidDifficulty = errors.New("invalid difficulty: must be easy, medium, or hard")
	ErrInvalidLimit      = errors.New("limit must be between 1 and 100")
	ErrInvalidOffset     = errors.New("offset must be non-negative")
)
