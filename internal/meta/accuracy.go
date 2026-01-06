package meta

import (
	"github.com/saisaravanan/healing-eval/internal/domain"
)

type AccuracyTracker struct{}

func NewAccuracyTracker() *AccuracyTracker {
	return &AccuracyTracker{}
}

type Prediction struct {
	EvaluatorType domain.EvaluatorType
	Predicted     bool
	Actual        bool
	Category      string
}

type AccuracyResult struct {
	EvaluatorType  domain.EvaluatorType
	TruePositives  int
	FalsePositives int
	TrueNegatives  int
	FalseNegatives int
	Precision      float64
	Recall         float64
	F1Score        float64
}

func (t *AccuracyTracker) Calculate(predictions []Prediction) *AccuracyResult {
	if len(predictions) == 0 {
		return &AccuracyResult{}
	}

	result := &AccuracyResult{
		EvaluatorType: predictions[0].EvaluatorType,
	}

	for _, p := range predictions {
		if p.Predicted && p.Actual {
			result.TruePositives++
		} else if p.Predicted && !p.Actual {
			result.FalsePositives++
		} else if !p.Predicted && p.Actual {
			result.FalseNegatives++
		} else {
			result.TrueNegatives++
		}
	}

	result.Precision = domain.CalculatePrecision(result.TruePositives, result.FalsePositives)
	result.Recall = domain.CalculateRecall(result.TruePositives, result.FalseNegatives)
	result.F1Score = domain.CalculateF1(result.Precision, result.Recall)

	return result
}

func (t *AccuracyTracker) CalculateByCategory(predictions []Prediction) map[string]*AccuracyResult {
	categoryPreds := make(map[string][]Prediction)
	for _, p := range predictions {
		categoryPreds[p.Category] = append(categoryPreds[p.Category], p)
	}

	results := make(map[string]*AccuracyResult)
	for category, preds := range categoryPreds {
		results[category] = t.Calculate(preds)
	}

	return results
}

type BlindSpotAnalysis struct {
	Category        string
	Recall          float64
	MissedCount     int
	TotalActual     int
	SuggestedAction string
}

func (t *AccuracyTracker) DetectBlindSpots(predictions []Prediction, recallThreshold float64) []BlindSpotAnalysis {
	byCategory := t.CalculateByCategory(predictions)

	var blindSpots []BlindSpotAnalysis
	for category, result := range byCategory {
		if result.Recall < recallThreshold {
			blindSpots = append(blindSpots, BlindSpotAnalysis{
				Category:        category,
				Recall:          result.Recall,
				MissedCount:     result.FalseNegatives,
				TotalActual:     result.TruePositives + result.FalseNegatives,
				SuggestedAction: t.suggestAction(category, result),
			})
		}
	}

	return blindSpots
}

func (t *AccuracyTracker) suggestAction(category string, result *AccuracyResult) string {
	if result.Recall < 0.3 {
		return "Critical: Consider adding dedicated evaluator for " + category
	}
	if result.Recall < 0.5 {
		return "Add explicit checks for " + category + " in evaluation prompt"
	}
	return "Enhance sensitivity for " + category + " detection"
}

