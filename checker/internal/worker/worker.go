package worker

import (
	"context"
	"log"
	"time"

	"github.com/meytardzhevtd/CodeTest/checker/internal/grpc/judgepb"
)

const (
	pollInterval = 50 * time.Millisecond
	judgeDelay   = 2 * time.Second
)

type Worker struct {
	id     string
	client judgepb.CoordinatorClient
}

func New(id string, client judgepb.CoordinatorClient) *Worker {
	return &Worker{id: id, client: client}
}

func (w *Worker) Run(ctx context.Context) {
	log.Printf("[worker %s] запущен, жду задачи", w.id)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		resp, err := w.client.PullTask(ctx, &judgepb.PullTaskRequest{})
		if err != nil {
			log.Printf("[worker %s] pull task: %v", w.id, err)
			time.Sleep(pollInterval)
			continue
		}

		if resp.Task == nil {
			time.Sleep(pollInterval)
			continue
		}

		log.Printf("[worker %s] submission %s: получена (language=%s)", w.id, resp.Task.SubmissionId, resp.Task.Language)

		result := w.judge(resp.Task)

		submitResp, err := w.client.SubmitResult(ctx, result)
		if err != nil {
			log.Printf("[worker %s] submission %s: не удалось отправить результат: %v", w.id, result.SubmissionId, err)
			continue
		}
		if !submitResp.Accepted {
			log.Printf("[worker %s] submission %s: результат не принят (уже отчитался кто-то другой)", w.id, result.SubmissionId)
			continue
		}

		log.Printf("[worker %s] submission %s: результат принят координатором", w.id, result.SubmissionId)
	}
}

func (w *Worker) judge(task *judgepb.Task) *judgepb.SubmitResultRequest {
	// TODO: реальное выполнение — поднять контейнер по task.Language через
	// Docker SDK, прогнать все task.TestCases и вернуть настоящий вердикт.
	log.Printf("[worker %s] submission %s: выполняю (имитация, %s)", w.id, task.SubmissionId, judgeDelay)
	time.Sleep(judgeDelay)

	return &judgepb.SubmitResultRequest{
		SubmissionId: task.SubmissionId,
		Status:       judgepb.Status_STATUS_OK,
	}
}
