package evaluator

import (
	"context"
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
}

func (o *Orchestrator) Evaluate(ctx context.Context, conv *domain.Conversation) (*domain.AggregatedEvaluation, error) {
	results := make(chan evaluationResult, len(o.evaluators))
	var wg sync.WaitGroup

	for _, eval := range o.evaluators {
		wg.Add(1)
		go func(e Evaluator) {
			defer wg.Done()
			result, err := e.Evaluate(ctx, conv)
			results <- evaluationResult{evaluation: result, err: err}
		}(eval)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var evaluations []*domain.Evaluation
	var allIssues []domain.Issue

	for r := range results {
		if r.err != nil {
			continue
		}
		if r.evaluation != nil {
			evaluations = append(evaluations, r.evaluation)
			allIssues = append(allIssues, r.evaluation.Issues...)
		}
	}

	aggregated := &domain.AggregatedEvaluation{
		ConversationID: conv.ID,
		Scores:         o.aggregateScores(evaluations),
		Issues:         allIssues,
		Evaluations:    o.toSlice(evaluations),
		CreatedAt:      time.Now(),
	}

	aggregated.ToolEvaluation = o.extractToolEvaluation(evaluations)

	return aggregated, nil
}

func (o *Orchestrator) aggregateScores(evals []*domain.Evaluation) domain.Scores {
	if len(evals) == 0 {
		return domain.Scores{}
	}

	var totalWeight float64
	scores := domain.Scores{}

	for _, e := range evals {
		weight := o.getWeight(e.EvaluatorType)
		totalWeight += weight

		scores.Overall += e.Scores.Overall * weight
		scores.ResponseQuality += e.Scores.ResponseQuality * weight
		scores.Helpfulness += e.Scores.Helpfulness * weight
		scores.Factuality += e.Scores.Factuality * weight
		scores.ToolAccuracy += e.Scores.ToolAccuracy * weight
		scores.Coherence += e.Scores.Coherence * weight
		scores.Consistency += e.Scores.Consistency * weight
	}

	if totalWeight > 0 {
		scores.Overall /= totalWeight
		scores.ResponseQuality /= totalWeight
		scores.Helpfulness /= totalWeight
		scores.Factuality /= totalWeight
		scores.ToolAccuracy /= totalWeight
		scores.Coherence /= totalWeight
		scores.Consistency /= totalWeight
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

