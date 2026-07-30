package coordinator

import (
	"context"
	"log"

	"github.com/meytardzhevtd/CodeTest/checker/internal/grpc/judgepb"
	"github.com/meytardzhevtd/CodeTest/pkg/kafka"
)

type Server struct {
	judgepb.UnimplementedCoordinatorServer
	queue    *Queue
	producer *kafka.Producer
}

func NewServer(queue *Queue, producer *kafka.Producer) *Server {
	return &Server{queue: queue, producer: producer}
}

func (s *Server) PullTask(ctx context.Context, req *judgepb.PullTaskRequest) (*judgepb.PullTaskResponse, error) {
	return &judgepb.PullTaskResponse{Task: s.queue.Lease()}, nil
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
