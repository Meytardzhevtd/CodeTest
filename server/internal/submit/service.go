package submit

import (
	"context"
	"errors"
	"log"
	"strings"

	"github.com/meytardzhevtd/CodeTest/pkg/kafka"
)

type Service struct {
	repo     RepositoryInterface
	producer ProducerInterface
}

func NewService(repo RepositoryInterface, producer ProducerInterface) *Service {
	return &Service{repo: repo, producer: producer}
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

	msg := kafka.SubmissionMessage{
		SubmissionID: sub.ID,
		TaskID:       sub.TaskID,
		Code:         sub.Code,
		Language:     sub.Language,
	}
	if err := s.producer.Send(ctx, sub.ID, msg); err != nil {
		if _, updErr := s.repo.UpdateResult(ctx, sub.ID, StatusError, "", "failed to queue submission for checking"); updErr != nil {
			log.Printf("[submit] failed to mark submission %s as errored after queue failure: %v", sub.ID, updErr)
		}
		return CreateSubmissionResponse{}, err
	}

	return CreateSubmissionResponse{ID: sub.ID, Status: sub.Status}, nil
}

func (s *Service) HandleResult(ctx context.Context, msg kafka.ResponseMessage) error {
	if strings.TrimSpace(msg.SubmissionID) == "" {
		return errors.New("result message missing submission_id")
	}

	if _, err := s.repo.UpdateResult(ctx, msg.SubmissionID, Status(msg.Status), msg.Output, msg.Error); err != nil {
		return err
	}

	return nil
}

func (s *Service) GetInfoAboutSubmit(ctx context.Context, userID, submitID string) (GetSubmissionResponse, error) {
	sub, err := s.repo.GetByID(ctx, submitID)
	if err != nil {
		return GetSubmissionResponse{}, err
	}

	if sub.UserID != userID {
		return GetSubmissionResponse{}, ErrForbidden
	}

	return GetSubmissionResponse{Submission: sub}, nil
}

func (s *Service) GetSubmissionHistory(ctx context.Context, userID, taskID string) (SubmissionHistoryResponse, error) {
	subs, err := s.repo.ListByUserAndTaskID(ctx, userID, taskID)
	if err != nil {
		return SubmissionHistoryResponse{}, err
	}

	items := make([]SubmissionHistoryItem, len(subs))
	for i, sub := range subs {
		items[i] = SubmissionHistoryItem{
			Number:   i + 1,
			ID:       sub.ID,
			Status:   sub.Status,
			Language: sub.Language,
		}
	}

	return SubmissionHistoryResponse{Submissions: items}, nil
}

var allowedLanguages = map[string]bool{
	"python": true,
	"cpp":    true,
	"go":     true,
}

func validateCreateSubmissionRequest(req CreateSubmissionRequest) error {
	if strings.TrimSpace(req.TaskID) == "" {
		return errors.New("task_id is required")
	}
	if strings.TrimSpace(req.Code) == "" {
		return errors.New("code is required")
	}
	if !allowedLanguages[req.Language] {
		return errors.New("unsupported language")
	}
	return nil
}
