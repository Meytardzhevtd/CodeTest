package submit

import (
	"context"
	"errors"
	"testing"

	"github.com/meytardzhevtd/CodeTest/pkg/kafka"
	"go.uber.org/mock/gomock"
)

func TestService_CreateSubmition_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockRepositoryInterface(ctrl)
	mockProducer := NewMockProducerInterface(ctrl)
	svc := NewService(mockRepo, mockProducer)

	ctx := context.Background()
	userID := "user-1"
	req := CreateSubmissionRequest{
		TaskID:   "task-1",
		Code:     "print('hi')",
		Language: "python",
	}

	expected := Submission{
		ID:       "sub-1",
		TaskID:   req.TaskID,
		UserID:   userID,
		Code:     req.Code,
		Language: req.Language,
		Status:   StatusPending,
	}

	mockRepo.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		Return(expected, nil)
	mockProducer.EXPECT().
		Send(gomock.Any(), gomock.Eq(expected.ID), gomock.Any()).
		Return(nil)

	resp, err := svc.CreateSubmition(ctx, userID, req)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp.ID != expected.ID {
		t.Errorf("expected id %s, got %s", expected.ID, resp.ID)
	}
	if resp.Status != StatusPending {
		t.Errorf("expected status %s, got %s", StatusPending, resp.Status)
	}
}

func TestService_CreateSubmition_ValidationError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockRepositoryInterface(ctrl)
	mockProducer := NewMockProducerInterface(ctrl)
	svc := NewService(mockRepo, mockProducer)

	ctx := context.Background()

	_, err := svc.CreateSubmition(ctx, "user-1", CreateSubmissionRequest{})

	if err == nil {
		t.Error("expected validation error, got nil")
	}
}

func TestService_CreateSubmition_ProducerSendError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockRepositoryInterface(ctrl)
	mockProducer := NewMockProducerInterface(ctrl)
	svc := NewService(mockRepo, mockProducer)

	ctx := context.Background()
	req := CreateSubmissionRequest{TaskID: "task-1", Code: "print('hi')", Language: "python"}
	created := Submission{ID: "sub-1", TaskID: req.TaskID, Code: req.Code, Language: req.Language, Status: StatusPending}
	sendErr := errors.New("broker unavailable")

	mockRepo.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		Return(created, nil)
	mockProducer.EXPECT().
		Send(gomock.Any(), gomock.Eq(created.ID), gomock.Any()).
		Return(sendErr)
	mockRepo.EXPECT().
		UpdateResult(gomock.Any(), gomock.Eq(created.ID), gomock.Eq(StatusError), gomock.Any(), gomock.Any()).
		Return(Submission{}, nil)

	_, err := svc.CreateSubmition(ctx, "user-1", req)

	if !errors.Is(err, sendErr) {
		t.Errorf("expected producer error to propagate, got %v", err)
	}
}

func TestService_GetInfoAboutSubmit_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockRepositoryInterface(ctrl)
	mockProducer := NewMockProducerInterface(ctrl)
	svc := NewService(mockRepo, mockProducer)

	ctx := context.Background()
	userID := "user-1"
	sub := Submission{ID: "sub-1", UserID: userID, Status: StatusPending}

	mockRepo.EXPECT().
		GetByID(gomock.Any(), gomock.Eq(sub.ID)).
		Return(sub, nil)

	resp, err := svc.GetInfoAboutSubmit(ctx, userID, sub.ID)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.Submission.ID != sub.ID {
		t.Errorf("expected id %s, got %s", sub.ID, resp.Submission.ID)
	}
}

func TestService_GetInfoAboutSubmit_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockRepositoryInterface(ctrl)
	mockProducer := NewMockProducerInterface(ctrl)
	svc := NewService(mockRepo, mockProducer)

	ctx := context.Background()

	mockRepo.EXPECT().
		GetByID(gomock.Any(), gomock.Eq("missing")).
		Return(Submission{}, ErrSubmissionNotFound)

	_, err := svc.GetInfoAboutSubmit(ctx, "user-1", "missing")

	if !errors.Is(err, ErrSubmissionNotFound) {
		t.Errorf("expected ErrSubmissionNotFound, got %v", err)
	}
}

func TestService_GetInfoAboutSubmit_Forbidden(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockRepositoryInterface(ctrl)
	mockProducer := NewMockProducerInterface(ctrl)
	svc := NewService(mockRepo, mockProducer)

	ctx := context.Background()
	sub := Submission{ID: "sub-1", UserID: "owner"}

	mockRepo.EXPECT().
		GetByID(gomock.Any(), gomock.Eq(sub.ID)).
		Return(sub, nil)

	_, err := svc.GetInfoAboutSubmit(ctx, "someone-else", sub.ID)

	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestService_HandleResult_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockRepositoryInterface(ctrl)
	mockProducer := NewMockProducerInterface(ctrl)
	svc := NewService(mockRepo, mockProducer)

	ctx := context.Background()
	msg := kafka.ResponseMessage{SubmissionID: "sub-1", Status: "OK", Output: "42"}

	mockRepo.EXPECT().
		UpdateResult(gomock.Any(), gomock.Eq(msg.SubmissionID), gomock.Eq(Status(msg.Status)), gomock.Eq(msg.Output), gomock.Eq(msg.Error)).
		Return(Submission{}, nil)

	if err := svc.HandleResult(ctx, msg); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestService_GetSubmissionHistory_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockRepositoryInterface(ctrl)
	mockProducer := NewMockProducerInterface(ctrl)
	svc := NewService(mockRepo, mockProducer)

	ctx := context.Background()
	userID := "user-1"
	taskID := "task-1"

	subs := []Submission{
		{ID: "sub-1", UserID: userID, TaskID: taskID, Language: "python", Status: StatusWA},
		{ID: "sub-2", UserID: userID, TaskID: taskID, Language: "go", Status: StatusOK},
	}

	mockRepo.EXPECT().
		ListByUserAndTaskID(gomock.Any(), gomock.Eq(userID), gomock.Eq(taskID)).
		Return(subs, nil)

	resp, err := svc.GetSubmissionHistory(ctx, userID, taskID)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(resp.Submissions) != 2 {
		t.Fatalf("expected 2 submissions, got %d", len(resp.Submissions))
	}
	if resp.Submissions[0].Number != 1 || resp.Submissions[0].ID != "sub-1" || resp.Submissions[0].Status != StatusWA {
		t.Errorf("unexpected first item: %+v", resp.Submissions[0])
	}
	if resp.Submissions[1].Number != 2 || resp.Submissions[1].ID != "sub-2" || resp.Submissions[1].Status != StatusOK {
		t.Errorf("unexpected second item: %+v", resp.Submissions[1])
	}
}

func TestService_HandleResult_MissingSubmissionID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockRepositoryInterface(ctrl)
	mockProducer := NewMockProducerInterface(ctrl)
	svc := NewService(mockRepo, mockProducer)

	ctx := context.Background()

	if err := svc.HandleResult(ctx, kafka.ResponseMessage{Status: "OK"}); err == nil {
		t.Error("expected error for missing submission_id, got nil")
	}
}
