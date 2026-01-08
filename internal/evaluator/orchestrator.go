package evaluator

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/saisaravanan/healing-eval/internal/domain"
)

type Orchestrator struct {
	evaluators []Evaluator
}

func NewOrchestrator(evaluators ...Evaluator) *Orchestrator {
	return &Orchestrator{evaluators: evaluators}
}

func (o *Orchestrator) AddEvaluator(e Evaluator) {
	o.evaluators = append(o.evaluators, e)
}

type evaluationResult struct {
	evaluation *domain.Evaluation
	err        error
	evalType   domain.EvaluatorType
}

func (o *Orchestrator) Evaluate(ctx context.Context, conv *domain.Conversation) (*domain.AggregatedEvaluation, error) {
	results := make(chan evaluationResult, len(o.evaluators))
	var wg sync.WaitGroup

	expectedCount := len(o.evaluators)

	// Run all evaluators in parallel with per-evaluator timeout
	for _, eval := range o.evaluators {
		wg.Add(1)
		go func(e Evaluator) {
			defer wg.Done()
			result, err := o.evaluateWithTimeout(ctx, e, conv, 30*time.Second)
			results <- evaluationResult{
				evaluation: result,
				err:        err,
				evalType:   e.Type(),
			}
		}(eval)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results and track failures
	var successful []*domain.Evaluation
	var failures []domain.EvaluatorFailure
	tokenUsage := &domain.AggregatedTokenUsage{
		ByEvaluator:      make(map[domain.EvaluatorType]domain.TokenUsage),
		MaxBudgetPerEval: 50000, // Demo: very large budget
	}

	for r := range results {
		if r.err != nil {
			// RECORD FAILURE with details
			failures = append(failures, domain.EvaluatorFailure{
				EvaluatorType: r.evalType,
				ErrorMessage:  r.err.Error(),
				Retryable:     isRetryable(r.err),
			})
			log.Printf("Evaluator %s failed: %v", r.evalType, r.err)
			continue
		}

		if r.evaluation != nil {
			successful = append(successful, r.evaluation)

			// Aggregate token usage
			tokenUsage.TotalTokens += r.evaluation.TotalTokens
			tokenUsage.TotalCost += r.evaluation.EstimatedCostUSD
			tokenUsage.ByEvaluator[r.evalType] = domain.TokenUsage{
				PromptTokens:     r.evaluation.PromptTokens,
				CompletionTokens: r.evaluation.CompletionTokens,
				TotalTokens:      r.evaluation.TotalTokens,
				EstimatedCost:    r.evaluation.EstimatedCostUSD,
				ModelName:        r.evaluation.ModelName,
			}
		}
	}

	// Determine overall status
	status := o.determineStatus(len(successful), len(failures), expectedCount)

	// Smart scoring that accounts for missing evaluators
	scores := o.aggregateScoresWithFailures(successful, failures, expectedCount)

	// Check budget (won't trigger with demo limits, but logs if exceeded)
	budgetEnforcer := NewBudgetEnforcer()
	if err := budgetEnforcer.CheckEvaluationBudget(tokenUsage); err != nil {
		log.Printf("Budget check: %v (continuing with result)", err)
	}

	aggregated := &domain.AggregatedEvaluation{
		ConversationID:   conv.ID,
		Status:           status,
		Scores:           scores,
		TokenUsage:       *tokenUsage,
		FailedEvaluators: failures,
		SuccessfulCount:  len(successful),
		ExpectedCount:    expectedCount,
		Issues:           o.collectIssues(successful),
		Evaluations:      o.toSlice(successful),
		ToolEvaluation:   o.extractToolEvaluation(successful),
		CreatedAt:        time.Now(),
	}

	return aggregated, nil
}

func (o *Orchestrator) evaluateWithTimeout(ctx context.Context, e Evaluator, conv *domain.Conversation, timeout time.Duration) (*domain.Evaluation, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	resultCh := make(chan struct {
		eval *domain.Evaluation
		err  error
	}, 1)

	go func() {
		eval, err := e.Evaluate(ctx, conv)
		resultCh <- struct {
			eval *domain.Evaluation
			err  error
		}{eval, err}
	}()

	select {
	case result := <-resultCh:
		return result.eval, result.err
	case <-ctx.Done():
		return nil, fmt.Errorf("evaluation timeout after %v", timeout)
	}
}

func (o *Orchestrator) determineStatus(successful, failed, expected int) domain.EvaluationStatus {
	if failed == 0 {
		return domain.AggregatedStatusSuccess
	}
	if successful == 0 {
		return domain.AggregatedStatusFailed
	}
	return domain.AggregatedStatusPartial
}

func (o *Orchestrator) aggregateScoresWithFailures(
	evals []*domain.Evaluation,
	failures []domain.EvaluatorFailure,
	expectedCount int,
) domain.Scores {
	if len(evals) == 0 {
		return domain.Scores{Overall: 0}
	}

	// Calculate weighted average using ACTUAL weights of successful evaluators
	// Then scale by completeness ratio
	scores := domain.Scores{}
	actualWeight := 0.0
	expectedWeight := 0.0

	// Get expected total weight
	for _, e := range o.evaluators {
		expectedWeight += e.Weight()
	}

	// Calculate scores from successful evaluators
	for _, eval := range evals {
		weight := o.getWeight(eval.EvaluatorType)
		actualWeight += weight

		scores.Overall += eval.Scores.Overall * weight
		scores.ResponseQuality += eval.Scores.ResponseQuality * weight
		scores.Helpfulness += eval.Scores.Helpfulness * weight
		scores.Factuality += eval.Scores.Factuality * weight
		scores.ToolAccuracy += eval.Scores.ToolAccuracy * weight
		scores.SelectionAccuracy += eval.Scores.SelectionAccuracy * weight
		scores.ParameterAccuracy += eval.Scores.ParameterAccuracy * weight
		scores.Coherence += eval.Scores.Coherence * weight
		scores.Consistency += eval.Scores.Consistency * weight
	}

	// Normalize by actual weight
	if actualWeight > 0 {
		scores.Overall /= actualWeight
		scores.ResponseQuality /= actualWeight
		scores.Helpfulness /= actualWeight
		scores.Factuality /= actualWeight
		scores.ToolAccuracy /= actualWeight
		scores.SelectionAccuracy /= actualWeight
		scores.ParameterAccuracy /= actualWeight
		scores.Coherence /= actualWeight
		scores.Consistency /= actualWeight
	}

	// Apply completeness penalty
	completeness := actualWeight / expectedWeight
	scores.Overall *= completeness

	// Mark as degraded if < 80% complete
	if completeness < 0.8 {
		scores.Overall *= 0.9 // Additional 10% penalty for significant missing data
	}

	return scores
}

func (o *Orchestrator) getWeight(evalType domain.EvaluatorType) float64 {
	for _, e := range o.evaluators {
		if e.Type() == evalType {
			return e.Weight()
		}
	}
	return 1.0
}

func (o *Orchestrator) collectIssues(evals []*domain.Evaluation) []domain.Issue {
	var allIssues []domain.Issue
	for _, eval := range evals {
		allIssues = append(allIssues, eval.Issues...)
	}
	return allIssues
}

func (o *Orchestrator) extractToolEvaluation(evals []*domain.Evaluation) *domain.ToolEvaluation {
	for _, e := range evals {
		if e.EvaluatorType == domain.EvaluatorTypeToolCall {
			return &domain.ToolEvaluation{
				SelectionAccuracy: e.Scores.SelectionAccuracy,
				ParameterAccuracy: e.Scores.ParameterAccuracy,
				ExecutionSuccess:  e.Scores.ToolAccuracy >= 0.9,
			}
		}
	}
	return nil
}

func (o *Orchestrator) toSlice(evals []*domain.Evaluation) []domain.Evaluation {
	result := make([]domain.Evaluation, len(evals))
	for i, e := range evals {
		result[i] = *e
	}
	return result
}

func isRetryable(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())
	retryablePatterns := []string{
		"timeout",
		"rate limit",
		"429",
		"503",
		"connection",
		"temporary",
		"deadline exceeded",
		"context deadline",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}
	return false
}
