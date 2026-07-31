package worker

import (
	"context"
	"log"
	"time"

	"github.com/meytardzhevtd/CodeTest/checker/internal/grpc/judgepb"
)

const (
	pollInterval = 50 * time.Millisecond
)

type Worker struct {
	id     string
	client judgepb.CoordinatorClient
	runner *DockerJudge
}

func New(id string, client judgepb.CoordinatorClient, runner *DockerJudge) *Worker {
	return &Worker{id: id, client: client, runner: runner}
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

		result := w.judge(ctx, resp.Task)

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

func (w *Worker) judge(ctx context.Context, task *judgepb.Task) *judgepb.SubmitResultRequest {
	log.Printf("[worker %s] submission %s: выполняю (language=%s, тестов=%d)", w.id, task.SubmissionId, task.Language, len(task.TestCases))
	return w.runner.Judge(ctx, task)
}
