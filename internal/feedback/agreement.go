package feedback

import (
	"github.com/saisaravanan/healing-eval/internal/domain"
)

type AgreementCalculator struct{}

func NewAgreementCalculator() *AgreementCalculator {
	return &AgreementCalculator{}
}

func (c *AgreementCalculator) Calculate(annotations []domain.Annotation) *domain.AgreementMetrics {
	metrics := &domain.AgreementMetrics{}
	
	if len(annotations) < 2 {
		metrics.PercentAgree = 1.0
		metrics.CohenKappa = 1.0
		metrics.FleissKappa = 1.0
		metrics.NeedsReview = false
		return metrics
	}

	labelCounts := make(map[string]int)
	for _, a := range annotations {
		labelCounts[a.Label]++
	}

	maxCount := 0
	for _, count := range labelCounts {
		if count > maxCount {
			maxCount = count
		}
	}
	metrics.PercentAgree = float64(maxCount) / float64(len(annotations))

	if len(annotations) == 2 {
		metrics.CohenKappa = c.cohenKappa(annotations)
		metrics.FleissKappa = metrics.CohenKappa
	} else {
		metrics.FleissKappa = c.fleissKappa(annotations)
		metrics.CohenKappa = metrics.FleissKappa
	}

	metrics.NeedsReview = metrics.FleissKappa < 0.6

	return metrics
}

func (c *AgreementCalculator) cohenKappa(annotations []domain.Annotation) float64 {
	if len(annotations) != 2 {
		return 0
	}

	labels := c.uniqueLabels(annotations)
	n := 1

	labelToIdx := make(map[string]int)
	for i, l := range labels {
		labelToIdx[l] = i
	}

	k := len(labels)
	matrix := make([][]int, k)
	for i := range matrix {
		matrix[i] = make([]int, k)
	}

	i1, i2 := labelToIdx[annotations[0].Label], labelToIdx[annotations[1].Label]
	matrix[i1][i2]++

	po := 0.0
	if annotations[0].Label == annotations[1].Label {
		po = 1.0
	}

	row := make([]float64, k)
	col := make([]float64, k)
	for i := 0; i < k; i++ {
		for j := 0; j < k; j++ {
			row[i] += float64(matrix[i][j])
			col[j] += float64(matrix[i][j])
		}
	}

	pe := 0.0
	total := float64(n)
	for i := 0; i < k; i++ {
		pe += (row[i] / total) * (col[i] / total)
	}

	if pe >= 1.0 {
		return 1.0
	}

	return (po - pe) / (1.0 - pe)
}

func (c *AgreementCalculator) fleissKappa(annotations []domain.Annotation) float64 {
	n := len(annotations)
	if n < 2 {
		return 1.0
	}

	labels := c.uniqueLabels(annotations)
	k := len(labels)
	if k <= 1 {
		return 1.0
	}

	labelToIdx := make(map[string]int)
	for i, l := range labels {
		labelToIdx[l] = i
	}

	counts := make([]int, k)
	for _, a := range annotations {
		counts[labelToIdx[a.Label]]++
	}

	sumSquares := 0.0
	for _, count := range counts {
		sumSquares += float64(count * count)
	}

	pBar := (sumSquares - float64(n)) / float64(n*(n-1))

	pe := 0.0
	for _, count := range counts {
		pj := float64(count) / float64(n)
		pe += pj * pj
	}

	if pe >= 1.0 {
		return 1.0
	}

	return (pBar - pe) / (1.0 - pe)
}

func (c *AgreementCalculator) uniqueLabels(annotations []domain.Annotation) []string {
	seen := make(map[string]bool)
	var labels []string
	for _, a := range annotations {
		if !seen[a.Label] {
			seen[a.Label] = true
			labels = append(labels, a.Label)
		}
	}
	return labels
}

