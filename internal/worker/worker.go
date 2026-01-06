package worker

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/saisaravanan/healing-eval/internal/domain"
	"github.com/saisaravanan/healing-eval/internal/evaluator"
	"github.com/saisaravanan/healing-eval/internal/queue"
	"github.com/saisaravanan/healing-eval/internal/storage"
)

type Worker struct {
	queue        *queue.RedisQueue
	convRepo     *storage.ConversationRepo
	evalRepo     *storage.EvaluationRepo
	orchestrator *evaluator.Orchestrator
	concurrency  int
	batchSize    int
}

func New(
	q *queue.RedisQueue,
	convRepo *storage.ConversationRepo,
	evalRepo *storage.EvaluationRepo,
	orchestrator *evaluator.Orchestrator,
	concurrency int,
	batchSize int,
) *Worker {
	return &Worker{
		queue:        q,
		convRepo:     convRepo,
		evalRepo:     evalRepo,
		orchestrator: orchestrator,
		concurrency:  concurrency,
		batchSize:    batchSize,
	}
}

func (w *Worker) Start(ctx context.Context) error {
	log.Printf("Starting worker with concurrency=%d, batchSize=%d", w.concurrency, w.batchSize)

	jobs := make(chan queue.Message, w.concurrency*2)
	var wg sync.WaitGroup

	for i := 0; i < w.concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			w.processJobs(ctx, workerID, jobs)
		}(i)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				close(jobs)
				return
			default:
				messages, err := w.queue.Consume(ctx, int64(w.batchSize), 5*time.Second)
				if err != nil {
					log.Printf("Error consuming messages: %v", err)
					time.Sleep(time.Second)
					continue
				}

				for _, msg := range messages {
					select {
					case jobs <- msg:
					case <-ctx.Done():
						close(jobs)
						return
					}
				}
			}
		}
	}()

	wg.Wait()
	return nil
}

func (w *Worker) processJobs(ctx context.Context, workerID int, jobs <-chan queue.Message) {
	for msg := range jobs {
		if err := w.processConversation(ctx, msg); err != nil {
			log.Printf("Worker %d: error processing %s: %v", workerID, msg.Conversation.ID, err)
			continue
		}

		if err := w.queue.Ack(ctx, msg.ID); err != nil {
			log.Printf("Worker %d: error acking %s: %v", workerID, msg.ID, err)
		}
	}
}

func (w *Worker) processConversation(ctx context.Context, msg queue.Message) error {
	conv := msg.Conversation
	log.Printf("Processing conversation: %s", conv.ID)

	result, err := w.orchestrator.Evaluate(ctx, conv)
	if err != nil {
		return err
	}

	evals := make([]*domain.Evaluation, len(result.Evaluations))
	for i := range result.Evaluations {
		evals[i] = &result.Evaluations[i]
	}

	if err := w.evalRepo.CreateBatch(ctx, evals); err != nil {
		return err
	}

	if err := w.convRepo.MarkProcessed(ctx, conv.ID); err != nil {
		return err
	}

	log.Printf("Completed evaluation for %s: overall=%.2f, issues=%d",
		conv.ID, result.Scores.Overall, len(result.Issues))

	return nil
}

