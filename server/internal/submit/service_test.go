package submit

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"
)

func TestService_CreateSubmition_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockRepositoryInterface(ctrl)
	svc := NewService(mockRepo)

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
	svc := NewService(mockRepo)

	ctx := context.Background()

	_, err := svc.CreateSubmition(ctx, "user-1", CreateSubmissionRequest{})

	if err == nil {
		t.Error("expected validation error, got nil")
	}
}

func TestService_GetInfoAboutSubmit_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockRepositoryInterface(ctrl)
	svc := NewService(mockRepo)

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
	svc := NewService(mockRepo)

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
	svc := NewService(mockRepo)

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
