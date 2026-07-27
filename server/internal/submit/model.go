package submit

import (
	"time"
)

type Status string

const (
	StatusPending Status = "pending"
	StatusRunning Status = "running"
	StatusOK      Status = "OK"
	StatusWA      Status = "WA"
	StatusRE      Status = "RE"
	StatusCE      Status = "CE"
	StatusTL      Status = "TL"
	StatusML      Status = "ML"
)

type Submission struct {
	ID        string    `json:"id"`
	TaskID    string    `json:"task_id"`
	UserID    string    `json:"user_id"`
	Code      string    `json:"code"`
	Language  string    `json:"language"`
	Status    Status    `json:"status"`
	Output    string    `json:"output,omitempty"`
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateSubmissionRequest struct {
	TaskID   string `json:"task_id"`
	Code     string `json:"code"`
	Language string `json:"language"`
}

type CreateSubmissionResponse struct {
	ID     string `json:"id"`
	Status Status `json:"status"`
}

type GetSubmissionResponse struct {
	Submission Submission `json:"submission"`
}

type ListSubmissionsResponse struct {
	Submissions []Submission `json:"submissions"`
	Total       int          `json:"total"`
}
