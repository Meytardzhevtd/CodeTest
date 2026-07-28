package kafka

type SubmissionMessage struct {
	SubmissionID string `json:"submission_id"`
	TaskID       string `json:"task_id"`
	Code         string `json:"code"`
	Language     string `json:"language"`
}

type ResponseMessage struct {
	SubmissionID string `json:"submission_id"`
	Status       string `json:"status"`
	Output       string `json:"output"`
	Error        string `json:"error"`
}
