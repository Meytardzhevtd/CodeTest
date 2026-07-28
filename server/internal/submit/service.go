package submit

import (
	"context"
	"errors"
	"strings"
)

type Service struct {
	repo RepositoryInterface
}

func NewService(repo RepositoryInterface) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateSubmition(ctx context.Context, userID string, request CreateSubmissionRequest) (CreateSubmissionResponse, error) {
	if err := validateCreateSubmissionRequest(request); err != nil {
		return CreateSubmissionResponse{}, err
	}

	submition := Submission{
		TaskID:   request.TaskID,
		UserID:   userID,
		Code:     request.Code,
		Language: request.Language,
		Status:   StatusPending,
	}

	sub, err := s.repo.Create(ctx, submition)
	if err != nil {
		return CreateSubmissionResponse{}, err
	}
	// TODO: надо отправить соощщение в Kafka о том, что нужно выполнить задчу

	return CreateSubmissionResponse{ID: sub.ID, Status: sub.Status}, nil
}

func (s *Service) GetInfoAboutSubmit(ctx context.Context, userID, submitID string) (GetSubmissionResponse, error) {
	// дергать каждые n секунд, проверять состояние задачи
	sub, err := s.repo.GetByID(ctx, submitID)
	if err != nil {
		return GetSubmissionResponse{}, err
	}

	if sub.UserID != userID {
		return GetSubmissionResponse{}, ErrForbidden
	}

	return GetSubmissionResponse{Submission: sub}, nil
}

func validateCreateSubmissionRequest(req CreateSubmissionRequest) error {
	if strings.TrimSpace(req.TaskID) == "" {
		return errors.New("task_id is required")
	}
	if strings.TrimSpace(req.Code) == "" {
		return errors.New("code is required")
	}
	if strings.TrimSpace(req.Language) == "" {
		return errors.New("language is required")
	}
	return nil
}
