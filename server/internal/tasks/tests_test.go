package tasks

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"go.uber.org/mock/gomock"
)

func buildTestZip(t *testing.T, files map[string]string) (*bytes.Reader, int64) {
	t.Helper()

	buf := &bytes.Buffer{}
	zw := zip.NewWriter(buf)
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("create zip entry %q: %v", name, err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatalf("write zip entry %q: %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}

	b := buf.Bytes()
	return bytes.NewReader(b), int64(len(b))
}

func TestService_UploadTests_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockRepositoryInterface(ctrl)
	mockStore := NewMockWriter(ctrl)
	svc := NewService(mockRepo, mockStore)

	ctx := context.Background()
	taskID := "task-1"
	userID := "user-1"

	archive, size := buildTestZip(t, map[string]string{
		"test_001.in":  "in-1",
		"test_001.out": "out-1",
		"test_002.in":  "in-2",
		"test_002.out": "out-2",
	})

	mockRepo.EXPECT().
		GetTaskByID(gomock.Any(), gomock.Eq(taskID)).
		Return(Task{ID: taskID, CreatedBy: userID}, nil)

	mockStore.EXPECT().DeleteTask(gomock.Any(), gomock.Eq(taskID)).Return(nil)

	wantContent := map[int]struct{ in, out string }{
		1: {"in-1", "out-1"},
		2: {"in-2", "out-2"},
	}
	for num, want := range wantContent {
		mockStore.EXPECT().
			UploadTest(gomock.Any(), gomock.Eq(taskID), gomock.Eq(num), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, _ int, in io.Reader, _ int64, out io.Reader, _ int64) error {
				gotIn, _ := io.ReadAll(in)
				gotOut, _ := io.ReadAll(out)
				if string(gotIn) != want.in {
					t.Errorf("test %d: input = %q, want %q", num, gotIn, want.in)
				}
				if string(gotOut) != want.out {
					t.Errorf("test %d: output = %q, want %q", num, gotOut, want.out)
				}
				return nil
			})
	}

	count, err := svc.UploadTests(ctx, userID, taskID, archive, size)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 tests uploaded, got %d", count)
	}
}

func TestService_UploadTests_Forbidden(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockRepositoryInterface(ctrl)
	mockStore := NewMockWriter(ctrl)
	svc := NewService(mockRepo, mockStore)

	taskID := "task-1"
	archive, size := buildTestZip(t, map[string]string{
		"test_001.in":  "in",
		"test_001.out": "out",
	})

	mockRepo.EXPECT().
		GetTaskByID(gomock.Any(), gomock.Eq(taskID)).
		Return(Task{ID: taskID, CreatedBy: "someone-else"}, nil)
	// No store calls expected: ownership is checked before touching storage.

	_, err := svc.UploadTests(context.Background(), "user-1", taskID, archive, size)
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestService_UploadTests_TaskNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockRepositoryInterface(ctrl)
	mockStore := NewMockWriter(ctrl)
	svc := NewService(mockRepo, mockStore)

	taskID := "missing"
	archive, size := buildTestZip(t, map[string]string{
		"test_001.in":  "in",
		"test_001.out": "out",
	})

	mockRepo.EXPECT().
		GetTaskByID(gomock.Any(), gomock.Eq(taskID)).
		Return(Task{}, ErrTaskNotFound)

	_, err := svc.UploadTests(context.Background(), "user-1", taskID, archive, size)
	if !errors.Is(err, ErrTaskNotFound) {
		t.Errorf("expected ErrTaskNotFound, got %v", err)
	}
}

func TestService_UploadTests_NotAZip(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockRepositoryInterface(ctrl)
	mockStore := NewMockWriter(ctrl)
	svc := NewService(mockRepo, mockStore)

	taskID := "task-1"
	garbage := bytes.NewReader([]byte("not a zip file"))

	mockRepo.EXPECT().
		GetTaskByID(gomock.Any(), gomock.Eq(taskID)).
		Return(Task{ID: taskID, CreatedBy: "user-1"}, nil)

	_, err := svc.UploadTests(context.Background(), "user-1", taskID, garbage, int64(garbage.Len()))
	if !errors.Is(err, ErrInvalidArchive) {
		t.Errorf("expected ErrInvalidArchive, got %v", err)
	}
}

func TestService_UploadTests_IncompletePair(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockRepositoryInterface(ctrl)
	mockStore := NewMockWriter(ctrl)
	svc := NewService(mockRepo, mockStore)

	taskID := "task-1"
	archive, size := buildTestZip(t, map[string]string{
		"test_001.in": "in-only",
	})

	mockRepo.EXPECT().
		GetTaskByID(gomock.Any(), gomock.Eq(taskID)).
		Return(Task{ID: taskID, CreatedBy: "user-1"}, nil)

	_, err := svc.UploadTests(context.Background(), "user-1", taskID, archive, size)
	if !errors.Is(err, ErrInvalidArchive) {
		t.Errorf("expected ErrInvalidArchive, got %v", err)
	}
}

func TestService_UploadTests_EmptyArchive(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockRepositoryInterface(ctrl)
	mockStore := NewMockWriter(ctrl)
	svc := NewService(mockRepo, mockStore)

	taskID := "task-1"
	archive, size := buildTestZip(t, map[string]string{
		"readme.txt": "not a test file",
	})

	mockRepo.EXPECT().
		GetTaskByID(gomock.Any(), gomock.Eq(taskID)).
		Return(Task{ID: taskID, CreatedBy: "user-1"}, nil)

	_, err := svc.UploadTests(context.Background(), "user-1", taskID, archive, size)
	if !errors.Is(err, ErrInvalidArchive) {
		t.Errorf("expected ErrInvalidArchive, got %v", err)
	}
}

func TestService_UploadTests_PathTraversal(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockRepositoryInterface(ctrl)
	mockStore := NewMockWriter(ctrl)
	svc := NewService(mockRepo, mockStore)

	taskID := "task-1"
	archive, size := buildTestZip(t, map[string]string{
		"../../etc/test_001.in":  "in",
		"../../etc/test_001.out": "out",
	})

	mockRepo.EXPECT().
		GetTaskByID(gomock.Any(), gomock.Eq(taskID)).
		Return(Task{ID: taskID, CreatedBy: "user-1"}, nil)

	_, err := svc.UploadTests(context.Background(), "user-1", taskID, archive, size)
	if !errors.Is(err, ErrInvalidArchive) {
		t.Errorf("expected ErrInvalidArchive, got %v", err)
	}
}

// Regression guard: if an upload fails partway through a multi-test batch,
// the service must roll back (delete) whatever it already wrote instead of
// leaving a mix of old/new tests behind.
func TestService_UploadTests_RollsBackOnPartialFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockRepositoryInterface(ctrl)
	mockStore := NewMockWriter(ctrl)
	svc := NewService(mockRepo, mockStore)

	ctx := context.Background()
	taskID := "task-1"
	userID := "user-1"

	archive, size := buildTestZip(t, map[string]string{
		"test_001.in":  "in-1",
		"test_001.out": "out-1",
		"test_002.in":  "in-2",
		"test_002.out": "out-2",
	})

	mockRepo.EXPECT().
		GetTaskByID(gomock.Any(), gomock.Eq(taskID)).
		Return(Task{ID: taskID, CreatedBy: userID}, nil)

	// Initial clear + best-effort rollback after the failure below.
	mockStore.EXPECT().DeleteTask(gomock.Any(), gomock.Eq(taskID)).Return(nil).Times(2)

	mockStore.EXPECT().
		UploadTest(gomock.Any(), gomock.Eq(taskID), gomock.Eq(1), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)
	mockStore.EXPECT().
		UploadTest(gomock.Any(), gomock.Eq(taskID), gomock.Eq(2), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(errors.New("minio unavailable"))

	_, err := svc.UploadTests(ctx, userID, taskID, archive, size)
	if err == nil {
		t.Fatal("expected an error from the failed upload, got nil")
	}
}
