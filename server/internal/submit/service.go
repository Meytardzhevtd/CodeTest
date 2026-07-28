package submit

import "context"

type Service struct {
	repo *Repository
}

func NewServer(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateSubmition(ctx context.Context, userID string, request CreateSubmissionRequest) (CreateSubmissionResponse, error) {
	submition := Submission{
		TaskID:   request.TaskID,
		UserID:   userID,
		Code:     request.Code,
		Language: request.Language,
		Status:   "panding",
	}

	sub, err := s.repo.Create(ctx, submition)
	if err != nil {
		return CreateSubmissionResponse{}, err
	}
	// TODO: надо отправить соощщение в Kafka о том, что нужно выполнить задчу

	return CreateSubmissionResponse{ID: sub.ID, Status: sub.Status}, nil
}

func (s *Service) GetInfoAboutSubmit(ctx context.Context, submitID string) (GetSubmissionResponse, error) {
	// дергать каждые n секунд, проверять состояние задачи
	sub, err := s.repo.GetByID(ctx, submitID)
	if err != nil {
		return GetSubmissionResponse{}, err
	}

	return GetSubmissionResponse{Submission: sub}, nil
}
