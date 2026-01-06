package evaluator

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/saisaravanan/healing-eval/internal/domain"
)

type HeuristicEvaluator struct {
	latencyThreshold int
	weight           float64
}

func NewHeuristicEvaluator(latencyThreshold int) *HeuristicEvaluator {
	if latencyThreshold == 0 {
		latencyThreshold = 1000
	}
	return &HeuristicEvaluator{
		latencyThreshold: latencyThreshold,
		weight:           0.2,
	}
}

func (e *HeuristicEvaluator) Name() string {
	return "heuristic"
}

func (e *HeuristicEvaluator) Type() domain.EvaluatorType {
	return domain.EvaluatorTypeHeuristic
}

func (e *HeuristicEvaluator) Weight() float64 {
	return e.weight
}

func (e *HeuristicEvaluator) Evaluate(ctx context.Context, conv *domain.Conversation) (*domain.Evaluation, error) {
	start := time.Now()

	var issues []domain.Issue
	var scores domain.Scores

	latencyScore := e.checkLatency(conv, &issues)
	formatScore := e.checkFormat(conv, &issues)
	toolExecutionScore := e.checkToolExecution(conv, &issues)

	scores.Overall = (latencyScore + formatScore + toolExecutionScore) / 3
	scores.ResponseQuality = formatScore
	scores.ToolAccuracy = toolExecutionScore

	return &domain.Evaluation{
		ID:             uuid.New().String(),
		ConversationID: conv.ID,
		EvaluatorType:  domain.EvaluatorTypeHeuristic,
		Scores:         scores,
		Issues:         issues,
		Confidence:     0.95,
		LatencyMs:      int(time.Since(start).Milliseconds()),
		CreatedAt:      time.Now(),
	}, nil
}

func (e *HeuristicEvaluator) checkLatency(conv *domain.Conversation, issues *[]domain.Issue) float64 {
	totalLatency := conv.TotalLatencyMs()

	if totalLatency > e.latencyThreshold*2 {
		*issues = append(*issues, domain.Issue{
			Type:        "latency",
			Severity:    "error",
			Description: "Response latency significantly exceeds threshold",
		})
		return 0.3
	}

	if totalLatency > e.latencyThreshold {
		*issues = append(*issues, domain.Issue{
			Type:        "latency",
			Severity:    "warning",
			Description: "Response latency exceeds threshold",
		})
		return 0.7
	}

	return 1.0
}

func (e *HeuristicEvaluator) checkFormat(conv *domain.Conversation, issues *[]domain.Issue) float64 {
	score := 1.0

	for _, turn := range conv.Turns {
		if turn.Role == "assistant" && turn.Content == "" {
			*issues = append(*issues, domain.Issue{
				Type:        "format",
				Severity:    "error",
				Description: "Empty assistant response",
				TurnID:      &turn.TurnID,
			})
			score -= 0.3
		}
	}

	if score < 0 {
		score = 0
	}
	return score
}

func (e *HeuristicEvaluator) checkToolExecution(conv *domain.Conversation, issues *[]domain.Issue) float64 {
	var totalCalls, successCalls int

	for _, turn := range conv.Turns {
		for _, tc := range turn.ToolCalls {
			totalCalls++
			if tc.Result != nil && tc.Result.Status == "success" {
				successCalls++
			} else if tc.Result != nil && tc.Result.Status == "error" {
				*issues = append(*issues, domain.Issue{
					Type:        "tool_execution",
					Severity:    "error",
					Description: "Tool execution failed: " + tc.ToolName,
					TurnID:      &turn.TurnID,
				})
			}
		}
	}

	if totalCalls == 0 {
		return 1.0
	}

	return float64(successCalls) / float64(totalCalls)
}

