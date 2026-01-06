package feedback

import (
	"github.com/saisaravanan/healing-eval/internal/domain"
)

type ConfidenceRouter struct {
	highThreshold   float64
	mediumThreshold float64
}

func NewConfidenceRouter() *ConfidenceRouter {
	return &ConfidenceRouter{
		highThreshold:   0.85,
		mediumThreshold: 0.60,
	}
}

type RoutingDecision string

const (
	DecisionAutoLabel   RoutingDecision = "auto_label"
	DecisionSpotCheck   RoutingDecision = "spot_check"
	DecisionHumanReview RoutingDecision = "human_review"
)

type RoutingResult struct {
	Decision   RoutingDecision
	Confidence float64
	Reason     string
}

func (r *ConfidenceRouter) Route(eval *domain.Evaluation) *RoutingResult {
	if eval.Confidence >= r.highThreshold {
		return &RoutingResult{
			Decision:   DecisionAutoLabel,
			Confidence: eval.Confidence,
			Reason:     "High confidence evaluation",
		}
	}

	if eval.Confidence >= r.mediumThreshold {
		return &RoutingResult{
			Decision:   DecisionSpotCheck,
			Confidence: eval.Confidence,
			Reason:     "Medium confidence - sample for review",
		}
	}

	return &RoutingResult{
		Decision:   DecisionHumanReview,
		Confidence: eval.Confidence,
		Reason:     "Low confidence - requires human review",
	}
}

func (r *ConfidenceRouter) RouteByAgreement(metrics *domain.AgreementMetrics) *RoutingResult {
	if metrics.FleissKappa >= 0.8 {
		return &RoutingResult{
			Decision:   DecisionAutoLabel,
			Confidence: metrics.FleissKappa,
			Reason:     "Strong annotator agreement",
		}
	}

	if metrics.FleissKappa >= 0.6 {
		return &RoutingResult{
			Decision:   DecisionSpotCheck,
			Confidence: metrics.FleissKappa,
			Reason:     "Moderate annotator agreement",
		}
	}

	return &RoutingResult{
		Decision:   DecisionHumanReview,
		Confidence: metrics.FleissKappa,
		Reason:     "Low annotator agreement - requires tiebreaker",
	}
}

func (r *ConfidenceRouter) ShouldAutoLabel(confidence float64) bool {
	return confidence >= r.highThreshold
}

func (r *ConfidenceRouter) NeedsHumanReview(confidence float64) bool {
	return confidence < r.mediumThreshold
}

