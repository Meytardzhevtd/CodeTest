package tasks

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"
)

func TestService_CreateTask_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockRepositoryInterface(ctrl)
	svc := NewService(mockRepo, nil)

	ctx := context.Background()
	req := CreateTaskRequest{
		Slug:          "test-task",
		Title:         "Test Task",
		Statement:     "Some statement",
		Difficulty:    "easy",
		TimeLimitMs:   1000,
		MemoryLimitMb: 256,
	}

	expectedTask := Task{
		Slug:          "test-task",
		Title:         "Test Task",
		Statement:     "Some statement",
		CreatedBy:     "user-1",
		Difficulty:    DifficultyEasy,
		TimeLimitMs:   1000,
		MemoryLimitMb: 256,
	}

	mockRepo.EXPECT().
		CreateTask(gomock.Any(), gomock.Any()).
		Return(expectedTask, nil)

	created, err := svc.CreateTask(ctx, "user-1", req)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if created.Slug != expectedTask.Slug {
		t.Errorf("expected slug %s, got %s", expectedTask.Slug, created.Slug)
	}
}

func TestService_CreateTask_SlugAlreadyExists(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockRepositoryInterface(ctrl)
	svc := NewService(mockRepo, nil)

	ctx := context.Background()
	req := CreateTaskRequest{
		Slug:          "existing-slug",
		Title:         "Test Task",
		Statement:     "Some statement",
		Difficulty:    "easy",
		TimeLimitMs:   1000,
		MemoryLimitMb: 256,
	}

	mockRepo.EXPECT().
		CreateTask(gomock.Any(), gomock.Any()).
		Return(Task{}, ErrSlugAlreadyExists)

	_, err := svc.CreateTask(ctx, "user-1", req)

	if !errors.Is(err, ErrSlugAlreadyExists) {
		t.Errorf("expected ErrSlugAlreadyExists, got %v", err)
	}
}

func TestService_CreateTask_ValidationError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockRepositoryInterface(ctrl)
	svc := NewService(mockRepo, nil)

	ctx := context.Background()
	req := CreateTaskRequest{
		Slug:      "",
		Title:     "",
		Statement: "",
	}

	_, err := svc.CreateTask(ctx, "user-1", req)

	if err == nil {
		t.Error("expected validation error, got nil")
	}
}

func TestService_GetTaskByID_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockRepositoryInterface(ctrl)
	svc := NewService(mockRepo, nil)

	ctx := context.Background()
	taskID := "550e8400-e29b-41d4-a716-446655440000"

	expectedTask := Task{
		ID:            taskID,
		Slug:          "test-task",
		Title:         "Test Task",
		Statement:     "Some statement",
		Difficulty:    DifficultyEasy,
		TimeLimitMs:   1000,
		MemoryLimitMb: 256,
	}

	mockRepo.EXPECT().
		GetTaskByID(gomock.Any(), gomock.Eq(taskID)).
		Return(expectedTask, nil)

	task, err := svc.GetTaskByID(ctx, taskID)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if task.ID != expectedTask.ID {
		t.Errorf("expected id %s, got %s", expectedTask.ID, task.ID)
	}
}

func TestService_GetTaskByID_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockRepositoryInterface(ctrl)
	svc := NewService(mockRepo, nil)

	ctx := context.Background()
	taskID := "non-existent-id"

	mockRepo.EXPECT().
		GetTaskByID(gomock.Any(), gomock.Eq(taskID)).
		Return(Task{}, ErrTaskNotFound)

	_, err := svc.GetTaskByID(ctx, taskID)

	if !errors.Is(err, ErrTaskNotFound) {
		t.Errorf("expected ErrTaskNotFound, got %v", err)
	}
}

func TestService_GetTaskByID_EmptyID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockRepositoryInterface(ctrl)
	svc := NewService(mockRepo, nil)

	ctx := context.Background()

	_, err := svc.GetTaskByID(ctx, "")

	if err == nil {
		t.Error("expected error for empty id, got nil")
	}
}

func TestService_GetTaskBySlug_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockRepositoryInterface(ctrl)
	svc := NewService(mockRepo, nil)

	ctx := context.Background()
	slug := "test-task"

	expectedTask := Task{
		ID:            "550e8400-e29b-41d4-a716-446655440000",
		Slug:          slug,
		Title:         "Test Task",
		Statement:     "Some statement",
		Difficulty:    DifficultyEasy,
		TimeLimitMs:   1000,
		MemoryLimitMb: 256,
	}

	mockRepo.EXPECT().
		GetTaskBySlug(gomock.Any(), gomock.Eq(slug)).
		Return(expectedTask, nil)

	task, err := svc.GetTaskBySlug(ctx, slug)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if task.Slug != expectedTask.Slug {
		t.Errorf("expected slug %s, got %s", expectedTask.Slug, task.Slug)
	}
}

func TestService_GetTaskBySlug_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockRepositoryInterface(ctrl)
	svc := NewService(mockRepo, nil)

	ctx := context.Background()
	slug := "non-existent-slug"

	mockRepo.EXPECT().
		GetTaskBySlug(gomock.Any(), gomock.Eq(slug)).
		Return(Task{}, ErrTaskNotFound)

	_, err := svc.GetTaskBySlug(ctx, slug)

	if !errors.Is(err, ErrTaskNotFound) {
		t.Errorf("expected ErrTaskNotFound, got %v", err)
	}
}

func TestService_DeleteTask_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockRepositoryInterface(ctrl)
	svc := NewService(mockRepo, nil)

	ctx := context.Background()
	taskID := "550e8400-e29b-41d4-a716-446655440000"

	mockRepo.EXPECT().
		DeleteTask(gomock.Any(), gomock.Eq(taskID)).
		Return(nil)

	err := svc.DeleteTask(ctx, taskID)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestService_DeleteTask_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockRepositoryInterface(ctrl)
	svc := NewService(mockRepo, nil)

	ctx := context.Background()
	taskID := "non-existent-id"

	mockRepo.EXPECT().
		DeleteTask(gomock.Any(), gomock.Eq(taskID)).
		Return(ErrTaskNotFound)

	err := svc.DeleteTask(ctx, taskID)

	if !errors.Is(err, ErrTaskNotFound) {
		t.Errorf("expected ErrTaskNotFound, got %v", err)
	}
}

func TestService_ListTasks_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockRepositoryInterface(ctrl)
	svc := NewService(mockRepo, nil)

	ctx := context.Background()
	expectedTasks := []Task{
		{
			ID:            "550e8400-e29b-41d4-a716-446655440000",
			Slug:          "task-1",
			Title:         "Task 1",
			Statement:     "Statement 1",
			Difficulty:    DifficultyEasy,
			TimeLimitMs:   1000,
			MemoryLimitMb: 256,
		},
	}
	total := 1

	mockRepo.EXPECT().
		ListTasks(gomock.Any(), gomock.Eq(10), gomock.Eq(0), gomock.Eq("")).
		Return(expectedTasks, total, nil)

	result, count, err := svc.ListTasks(ctx, 10, 0, "")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if count != total {
		t.Errorf("expected total %d, got %d", total, count)
	}
	if len(result) != len(expectedTasks) {
		t.Errorf("expected %d tasks, got %d", len(expectedTasks), len(result))
	}
}

func TestService_ListTasks_InvalidLimit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockRepositoryInterface(ctrl)
	svc := NewService(mockRepo, nil)

	ctx := context.Background()

	_, _, err := svc.ListTasks(ctx, 101, 0, "")

	if err == nil {
		t.Error("expected error for invalid limit, got nil")
	}
}

func TestService_ListTasks_InvalidOffset(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockRepositoryInterface(ctrl)
	svc := NewService(mockRepo, nil)

	ctx := context.Background()

	_, _, err := svc.ListTasks(ctx, 10, -1, "")

	if err == nil {
		t.Error("expected error for invalid offset, got nil")
	}
}

func TestService_ListTasks_WithSearch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockRepositoryInterface(ctrl)
	svc := NewService(mockRepo, nil)

	ctx := context.Background()
	expectedTasks := []Task{
		{
			ID:    "550e8400-e29b-41d4-a716-446655440002",
			Slug:  "two-sum",
			Title: "Two Sum",
		},
	}

	mockRepo.EXPECT().
		ListTasks(gomock.Any(), gomock.Eq(10), gomock.Eq(0), gomock.Eq("two sum")).
		Return(expectedTasks, 1, nil)

	result, count, err := svc.ListTasks(ctx, 10, 0, "  two sum  ")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if count != 1 {
		t.Errorf("expected total 1, got %d", count)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 task, got %d", len(result))
	}
}
