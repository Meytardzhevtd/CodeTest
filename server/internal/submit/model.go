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
	// StatusError помечает сабмишн, который не удалось поставить в очередь
	// или обработать по инфраструктурной причине (не связанной с самим решением).
	StatusError Status = "ERROR"
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

// SubmissionHistoryItem — краткая сводка по одной посылке для истории по задаче.
// Number — это не ID из БД, а порядковый номер посылки пользователя по этой
// задаче (1..n в хронологическом порядке), удобный для отображения в UI.
type SubmissionHistoryItem struct {
	Number   int    `json:"number"`
	ID       string `json:"id"`
	Status   Status `json:"status"`
	Language string `json:"language"`
}

type SubmissionHistoryResponse struct {
	Submissions []SubmissionHistoryItem `json:"submissions"`
}
