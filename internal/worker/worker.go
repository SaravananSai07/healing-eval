package worker

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/saisaravanan/healing-eval/internal/domain"
	"github.com/saisaravanan/healing-eval/internal/evaluator"
	"github.com/saisaravanan/healing-eval/internal/feedback"
	"github.com/saisaravanan/healing-eval/internal/meta"
	"github.com/saisaravanan/healing-eval/internal/queue"
	"github.com/saisaravanan/healing-eval/internal/storage"
)

type Worker struct {
	queue              *queue.RedisQueue
	convRepo           *storage.ConversationRepo
	evalRepo           *storage.EvaluationRepo
	reviewQueueRepo    *storage.ReviewQueueRepo
	orchestrator       *evaluator.Orchestrator
	agreementCalc      *feedback.AgreementCalculator
	accuracyTracker    *meta.AccuracyTracker
	calibrationService *meta.CalibrationService
	confidenceRouter   *feedback.ConfidenceRouter
	concurrency        int
	batchSize          int
}

func New(
	q *queue.RedisQueue,
	convRepo *storage.ConversationRepo,
	evalRepo *storage.EvaluationRepo,
	reviewQueueRepo *storage.ReviewQueueRepo,
	orchestrator *evaluator.Orchestrator,
	concurrency int,
	batchSize int,
) *Worker {
	return &Worker{
		queue:              q,
		convRepo:           convRepo,
		evalRepo:           evalRepo,
		reviewQueueRepo:    reviewQueueRepo,
		orchestrator:       orchestrator,
		agreementCalc:      feedback.NewAgreementCalculator(),
		accuracyTracker:    meta.NewAccuracyTracker(),
		calibrationService: meta.NewCalibrationService(),
		confidenceRouter:   feedback.NewConfidenceRouter(),
		concurrency:        concurrency,
		batchSize:          batchSize,
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

	// Orchestrator ALWAYS returns result, never returns error
	result, _ := w.orchestrator.Evaluate(ctx, conv)

	// Store ALL evaluations (including failed ones)
	evals := make([]*domain.Evaluation, len(result.Evaluations))
	for i := range result.Evaluations {
		evals[i] = &result.Evaluations[i]
	}

	if err := w.evalRepo.CreateBatch(ctx, evals); err != nil {
		return fmt.Errorf("store evaluations: %w", err)
	}

	// Log comprehensive token usage and status
	log.Printf("Conversation %s: Status=%s, Tokens=%d, Cost=$%.4f, Success=%d/%d",
		conv.ID, result.Status, result.TokenUsage.TotalTokens, result.TokenUsage.TotalCost,
		result.SuccessfulCount, result.ExpectedCount)

	// Log failed evaluators if any
	if len(result.FailedEvaluators) > 0 {
		log.Printf("Failed evaluators for %s:", conv.ID)
		for _, failure := range result.FailedEvaluators {
			log.Printf("  - %s: %s (retryable=%v)",
				failure.EvaluatorType, failure.ErrorMessage, failure.Retryable)
		}
	}

	// Log token usage by evaluator
	if len(result.TokenUsage.ByEvaluator) > 0 {
		log.Printf("Token usage by evaluator for %s:", conv.ID)
		for evalType, usage := range result.TokenUsage.ByEvaluator {
			log.Printf("  - %s: %d tokens ($%.4f) [%s]",
				evalType, usage.TotalTokens, usage.EstimatedCost, usage.ModelName)
		}
	}

	// Process feedback and annotations if present
	if conv.Feedback != nil && len(conv.Feedback.Annotations) > 0 {
		w.processFeedback(ctx, conv, result)
	}

	// Route to human review if needed
	if w.shouldRouteToHumanReview(result) {
		if err := w.addToReviewQueue(ctx, conv, result); err != nil {
			log.Printf("Failed to add to review queue: %v", err)
			// Don't fail the entire job if review queue insertion fails
		}
	}

	// Mark as processed with status
	if err := w.convRepo.MarkProcessedWithStatus(ctx, conv.ID, string(result.Status)); err != nil {
		return err
	}

	log.Printf("Completed evaluation for %s: overall=%.2f, issues=%d",
		conv.ID, result.Scores.Overall, len(result.Issues))

	return nil
}

func (w *Worker) processFeedback(ctx context.Context, conv *domain.Conversation, result *domain.AggregatedEvaluation) {
	annotations := conv.Feedback.Annotations
	
	// Calculate annotator agreement
	if len(annotations) >= 2 {
		agreement := w.agreementCalc.Calculate(annotations)
		log.Printf("Annotator agreement for %s: Fleiss Kappa=%.2f, Needs Review=%v",
			conv.ID, agreement.FleissKappa, agreement.NeedsReview)
		
		if agreement.NeedsReview {
			log.Printf("Low agreement detected for %s - flagging for human review", conv.ID)
		}
	}

	// Compare evaluator predictions with human annotations
	// This provides data for meta-evaluation
	for _, eval := range result.Evaluations {
		w.compareWithAnnotations(eval, annotations)
	}
}

func (w *Worker) compareWithAnnotations(eval domain.Evaluation, annotations []domain.Annotation) {
	// Group annotations by type to compare with evaluator findings
	annotationsByType := make(map[string][]domain.Annotation)
	for _, ann := range annotations {
		annotationsByType[ann.Type] = append(annotationsByType[ann.Type], ann)
	}

	// Compare evaluator issues with human annotations
	for _, issue := range eval.Issues {
		if anns, ok := annotationsByType[issue.Type]; ok {
			// Human annotators also found this issue type
			log.Printf("Evaluator %s correctly identified %s issue (validated by %d annotators)",
				eval.EvaluatorType, issue.Type, len(anns))
		}
	}
	
	// Log for future meta-evaluation
	// In a production system, this would be stored in the database
	// for later analysis of evaluator accuracy
}

func (w *Worker) shouldRouteToHumanReview(result *domain.AggregatedEvaluation) bool {
	// Route if evaluation failed or partial
	if result.Status != domain.AggregatedStatusSuccess {
		return true
	}

	// Route if low overall confidence
	if len(result.Evaluations) > 0 {
		avgConfidence := 0.0
		for _, eval := range result.Evaluations {
			avgConfidence += eval.Confidence
		}
		avgConfidence /= float64(len(result.Evaluations))

		if w.confidenceRouter.NeedsHumanReview(avgConfidence) {
			return true
		}
	}

	// Route if overall score is very low
	if result.Scores.Overall < 0.5 {
		return true
	}

	return false
}

func (w *Worker) addToReviewQueue(ctx context.Context, conv *domain.Conversation, result *domain.AggregatedEvaluation) error {
	reason := w.determineReviewReason(result)
	priority := w.determineReviewPriority(result)

	// Calculate routing confidence
	avgConfidence := 0.0
	if len(result.Evaluations) > 0 {
		for _, eval := range result.Evaluations {
			avgConfidence += eval.Confidence
		}
		avgConfidence /= float64(len(result.Evaluations))
	}

	item := &domain.ReviewQueueItem{
		ConversationID:    conv.ID,
		Reason:            reason,
		Priority:          priority,
		Status:            "pending",
		RoutingConfidence: avgConfidence,
	}

	log.Printf("Adding conversation %s to human review queue: %s (priority=%d)", conv.ID, reason, priority)

	return w.reviewQueueRepo.AddToQueue(ctx, item)
}

func (w *Worker) determineReviewReason(result *domain.AggregatedEvaluation) string {
	switch result.Status {
	case domain.AggregatedStatusFailed:
		return "evaluation_failed"
	case domain.AggregatedStatusPartial:
		return fmt.Sprintf("partial_evaluation_%d_%d", result.SuccessfulCount, result.ExpectedCount)
	default:
		if result.Scores.Overall < 0.5 {
			return "low_quality_score"
		}
		// Check confidence
		avgConfidence := 0.0
		if len(result.Evaluations) > 0 {
			for _, eval := range result.Evaluations {
				avgConfidence += eval.Confidence
			}
			avgConfidence /= float64(len(result.Evaluations))
		}
		if avgConfidence < 0.6 {
			return "low_confidence"
		}
		return "quality_review"
	}
}

func (w *Worker) determineReviewPriority(result *domain.AggregatedEvaluation) int {
	// Priority: 1 = high, 2 = medium, 3 = low
	if result.Status == domain.AggregatedStatusFailed {
		return 1 // High priority for complete failures
	}
	if result.Status == domain.AggregatedStatusPartial {
		return 2 // Medium priority for partial evaluations
	}
	if result.Scores.Overall < 0.3 {
		return 1 // High priority for very low scores
	}
	if result.Scores.Overall < 0.5 {
		return 2 // Medium priority for low scores
	}
	return 3 // Low priority for other cases
}

