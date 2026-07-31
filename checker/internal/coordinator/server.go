package coordinator

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/meytardzhevtd/CodeTest/checker/internal/grpc/judgepb"
	"github.com/meytardzhevtd/CodeTest/pkg/kafka"
	"github.com/meytardzhevtd/CodeTest/pkg/storage"
)

type Server struct {
	judgepb.UnimplementedCoordinatorServer
	queue    *Queue
	producer *kafka.Producer
	tests    storage.Reader
}

func NewServer(queue *Queue, producer *kafka.Producer, tests storage.Reader) *Server {
	return &Server{queue: queue, producer: producer, tests: tests}
}

func (s *Server) PullTask(ctx context.Context, req *judgepb.PullTaskRequest) (*judgepb.PullTaskResponse, error) {
	task, taskID := s.queue.Lease()
	if task == nil {
		return &judgepb.PullTaskResponse{}, nil
	}

	testCases, err := loadTestCases(ctx, s.tests, taskID)
	if err != nil {
		log.Printf("[coordinator] submission %s: не удалось прочитать тесты задачи %s: %v", task.SubmissionId, taskID, err)
		return nil, fmt.Errorf("load test cases for task %s: %w", taskID, err)
	}
	task.TestCases = testCases

	return &judgepb.PullTaskResponse{Task: task}, nil
}

// loadTestCases читает все тест-кейсы задачи из object storage заново на
// каждый вызов — кеширования пока нет, каждая выдача задачи воркеру означает
// полный поход в MinIO.
func loadTestCases(ctx context.Context, reader storage.Reader, taskID string) ([]*judgepb.TestCase, error) {
	list, err := reader.ListTests(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("list tests: %w", err)
	}

	cases := make([]*judgepb.TestCase, 0, len(list))
	for _, tc := range list {
		input, output, err := reader.GetTest(ctx, tc)
		if err != nil {
			return nil, fmt.Errorf("get test %03d: %w", tc.Number, err)
		}

		inputBytes, err := io.ReadAll(input)
		input.Close()
		if err != nil {
			output.Close()
			return nil, fmt.Errorf("read input for test %03d: %w", tc.Number, err)
		}

		outputBytes, err := io.ReadAll(output)
		output.Close()
		if err != nil {
			return nil, fmt.Errorf("read expected output for test %03d: %w", tc.Number, err)
		}

		cases = append(cases, &judgepb.TestCase{
			Number:         int32(tc.Number),
			Input:          string(inputBytes),
			ExpectedOutput: string(outputBytes),
		})
	}

	return cases, nil
}

func (s *Server) SubmitResult(ctx context.Context, req *judgepb.SubmitResultRequest) (*judgepb.SubmitResultResponse, error) {
	if !s.queue.Complete(req.SubmissionId, req) {
		return &judgepb.SubmitResultResponse{Accepted: false}, nil
	}

	msg := kafka.ResponseMessage{
		SubmissionID: req.SubmissionId,
		Status:       statusToString(req.Status),
		Output:       req.Output,
		Error:        req.Error,
	}
	if err := s.producer.Send(ctx, req.SubmissionId, msg); err != nil {
		log.Printf("[coordinator] submission %s: не удалось опубликовать результат в Kafka: %v", req.SubmissionId, err)
		return nil, err
	}

	return &judgepb.SubmitResultResponse{Accepted: true}, nil
}

func statusToString(s judgepb.Status) string {
	switch s {
	case judgepb.Status_STATUS_OK:
		return "OK"
	case judgepb.Status_STATUS_WA:
		return "WA"
	case judgepb.Status_STATUS_RE:
		return "RE"
	case judgepb.Status_STATUS_CE:
		return "CE"
	case judgepb.Status_STATUS_TL:
		return "TL"
	case judgepb.Status_STATUS_ML:
		return "ML"
	default:
		return "ERROR"
	}
}
