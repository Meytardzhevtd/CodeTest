package coordinator

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/meytardzhevtd/CodeTest/checker/internal/grpc/judgepb"
)

const (
	leaseTimeout    = 90 * time.Second
	cleanupInterval = 30 * time.Second
)

type entry struct {
	taskID      string
	task        *judgepb.Task
	leasedUntil time.Time
	done        bool
	result      *judgepb.SubmitResultRequest
}

type Queue struct {
	mu      sync.Mutex
	entries []*entry
}

func NewQueue(ctx context.Context) *Queue {
	q := &Queue{}
	go q.runCleanup(ctx, cleanupInterval)
	return q
}

func (q *Queue) Enqueue(taskID string, task *judgepb.Task) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.entries = append(q.entries, &entry{taskID: taskID, task: task})
	log.Printf("[queue] submission %s поставлена в очередь (language=%s)", task.SubmissionId, task.Language)
}

func (q *Queue) Lease() (*judgepb.Task, string) {
	q.mu.Lock()
	defer q.mu.Unlock()

	now := time.Now()
	for _, e := range q.entries {
		if e.done || e.leasedUntil.After(now) {
			continue
		}

		e.leasedUntil = now.Add(leaseTimeout)
		log.Printf("[queue] submission %s выдана воркеру (lease до %s)", e.task.SubmissionId, e.leasedUntil.Format(time.RFC3339))
		return e.task, e.taskID
	}

	return nil, ""
}

func (q *Queue) Complete(submissionID string, result *judgepb.SubmitResultRequest) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	for _, e := range q.entries {
		if e.task.SubmissionId != submissionID {
			continue
		}

		if e.done {
			log.Printf("[queue] submission %s: результат уже был принят ранее, игнорирую дубль", submissionID)
			return false
		}

		e.done = true
		e.result = result
		log.Printf("[queue] submission %s: результат принят (%s)", submissionID, result.Status)
		return true
	}

	log.Printf("[queue] submission %s: задача не найдена, результат отброшен", submissionID)
	return false
}

func (q *Queue) runCleanup(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			q.cleanup()
		}
	}
}

func (q *Queue) cleanup() {
	q.mu.Lock()
	defer q.mu.Unlock()

	kept := q.entries[:0]
	removed := 0
	for _, e := range q.entries {
		if e.done {
			removed++
			continue
		}
		kept = append(kept, e)
	}
	q.entries = kept

	if removed > 0 {
		log.Printf("[queue] очистка: удалено %d завершённых задач, осталось %d", removed, len(q.entries))
	}
}
