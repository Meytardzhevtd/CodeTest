package submit

import "errors"

var (
	ErrSubmissionNotFound = errors.New("submission not found")
	ErrForbidden          = errors.New("forbidden")
)
