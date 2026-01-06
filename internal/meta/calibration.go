package meta

import (
	"context"
	"math"
	"time"

	"github.com/saisaravanan/healing-eval/internal/domain"
)

type CalibrationService struct{}

func NewCalibrationService() *CalibrationService {
	return &CalibrationService{}
}

type ComparisonPair struct {
	EvaluatorScore float64
	HumanScore     float64
}

func (s *CalibrationService) CalculateCalibration(
	ctx context.Context,
	evaluatorType domain.EvaluatorType,
	pairs []ComparisonPair,
	dateFrom, dateTo time.Time,
) *domain.CalibrationMetrics {
	if len(pairs) < 2 {
		return &domain.CalibrationMetrics{
			EvaluatorType: evaluatorType,
			Period:        domain.DateRange{From: dateFrom, To: dateTo},
			DriftStatus:   "insufficient_data",
			SampleCount:   len(pairs),
		}
	}

	pearson := s.pearsonCorrelation(pairs)
	spearman := s.spearmanCorrelation(pairs)
	mae := s.meanAbsoluteError(pairs)

	driftStatus := "stable"
	if pearson < 0.6 {
		driftStatus = "critical"
	} else if pearson < 0.75 {
		driftStatus = "warning"
	}

	return &domain.CalibrationMetrics{
		EvaluatorType:       evaluatorType,
		Period:              domain.DateRange{From: dateFrom, To: dateTo},
		PearsonCorrelation:  pearson,
		SpearmanCorrelation: spearman,
		MeanAbsoluteError:   mae,
		SampleCount:         len(pairs),
		DriftStatus:         driftStatus,
		LastCalibration:     time.Now(),
	}
}

func (s *CalibrationService) pearsonCorrelation(pairs []ComparisonPair) float64 {
	n := float64(len(pairs))
	if n < 2 {
		return 0
	}

	var sumX, sumY, sumXY, sumX2, sumY2 float64
	for _, p := range pairs {
		sumX += p.EvaluatorScore
		sumY += p.HumanScore
		sumXY += p.EvaluatorScore * p.HumanScore
		sumX2 += p.EvaluatorScore * p.EvaluatorScore
		sumY2 += p.HumanScore * p.HumanScore
	}

	numerator := n*sumXY - sumX*sumY
	denominator := math.Sqrt((n*sumX2 - sumX*sumX) * (n*sumY2 - sumY*sumY))

	if denominator == 0 {
		return 0
	}

	return numerator / denominator
}

func (s *CalibrationService) spearmanCorrelation(pairs []ComparisonPair) float64 {
	n := len(pairs)
	if n < 2 {
		return 0
	}

	evalRanks := s.rank(pairs, func(p ComparisonPair) float64 { return p.EvaluatorScore })
	humanRanks := s.rank(pairs, func(p ComparisonPair) float64 { return p.HumanScore })

	var sumD2 float64
	for i := range pairs {
		d := evalRanks[i] - humanRanks[i]
		sumD2 += d * d
	}

	nf := float64(n)
	return 1 - (6*sumD2)/(nf*(nf*nf-1))
}

func (s *CalibrationService) rank(pairs []ComparisonPair, getValue func(ComparisonPair) float64) []float64 {
	n := len(pairs)
	ranks := make([]float64, n)

	indices := make([]int, n)
	for i := range indices {
		indices[i] = i
	}

	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			if getValue(pairs[indices[i]]) > getValue(pairs[indices[j]]) {
				indices[i], indices[j] = indices[j], indices[i]
			}
		}
	}

	for rank, idx := range indices {
		ranks[idx] = float64(rank + 1)
	}

	return ranks
}

func (s *CalibrationService) meanAbsoluteError(pairs []ComparisonPair) float64 {
	if len(pairs) == 0 {
		return 0
	}

	var sum float64
	for _, p := range pairs {
		sum += math.Abs(p.EvaluatorScore - p.HumanScore)
	}

	return sum / float64(len(pairs))
}

