package improvement

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/saisaravanan/healing-eval/internal/domain"
)

type PatternDetector struct {
	minOccurrences int
	windowDays     int
}

func NewPatternDetector() *PatternDetector {
	return &PatternDetector{
		minOccurrences: 5,
		windowDays:     7,
	}
}

type IssueAggregate struct {
	Type        string
	Count       int
	Examples    []string
	ConvIDs     []string
	FirstSeen   time.Time
	LastSeen    time.Time
}

func (d *PatternDetector) DetectPatterns(ctx context.Context, evaluations []*domain.Evaluation) []*domain.FailurePattern {
	issueMap := make(map[string]*IssueAggregate)

	for _, eval := range evaluations {
		for _, issue := range eval.Issues {
			key := issue.Type + ":" + issue.Severity

			if agg, exists := issueMap[key]; exists {
				agg.Count++
				if len(agg.Examples) < 5 {
					agg.Examples = append(agg.Examples, issue.Description)
				}
				if len(agg.ConvIDs) < 10 {
					agg.ConvIDs = append(agg.ConvIDs, eval.ConversationID)
				}
				if eval.CreatedAt.Before(agg.FirstSeen) {
					agg.FirstSeen = eval.CreatedAt
				}
				if eval.CreatedAt.After(agg.LastSeen) {
					agg.LastSeen = eval.CreatedAt
				}
			} else {
				issueMap[key] = &IssueAggregate{
					Type:      issue.Type,
					Count:     1,
					Examples:  []string{issue.Description},
					ConvIDs:   []string{eval.ConversationID},
					FirstSeen: eval.CreatedAt,
					LastSeen:  eval.CreatedAt,
				}
			}
		}
	}

	var patterns []*domain.FailurePattern
	for _, agg := range issueMap {
		if agg.Count >= d.minOccurrences {
			patterns = append(patterns, &domain.FailurePattern{
				ID:          uuid.New().String(),
				Type:        agg.Type,
				Description: d.generateDescription(agg),
				Count:       agg.Count,
				Examples:    agg.Examples,
				FirstSeenAt: agg.FirstSeen,
				LastSeenAt:  agg.LastSeen,
			})
		}
	}

	return patterns
}

func (d *PatternDetector) generateDescription(agg *IssueAggregate) string {
	switch agg.Type {
	case "latency":
		return "Latency issues detected in multiple conversations"
	case "tool_execution":
		return "Tool execution failures detected"
	case "hallucination":
		return "Parameter hallucination detected in tool calls"
	case "context_loss":
		return "Context loss detected in multi-turn conversations"
	case "contradiction":
		return "Contradictions detected in assistant responses"
	default:
		return "Pattern detected: " + agg.Type
	}
}

