package domain

import (
	"encoding/json"
	"time"
)

type Annotation struct {
	ID             string          `json:"id,omitempty"`
	ConversationID string          `json:"conversation_id"`
	TurnID         *int            `json:"turn_id,omitempty"`
	AnnotatorID    string          `json:"annotator_id"`
	Type           string          `json:"type"`
	Label          string          `json:"label"`
	Confidence     float64         `json:"confidence,omitempty"`
	Metadata       json.RawMessage `json:"metadata,omitempty"`
	CreatedAt      time.Time       `json:"created_at,omitempty"`
}

type Annotator struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	ReliabilityScore float64   `json:"reliability_score"`
	TotalAnnotations int       `json:"total_annotations"`
	AgreementRate    float64   `json:"agreement_rate"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type AgreementMetrics struct {
	CohenKappa   float64 `json:"cohen_kappa"`
	FleissKappa  float64 `json:"fleiss_kappa"`
	PercentAgree float64 `json:"percent_agree"`
	NeedsReview  bool    `json:"needs_review"`
}

func (m *AgreementMetrics) Calculate(annotations []Annotation) {
	if len(annotations) < 2 {
		m.PercentAgree = 1.0
		m.NeedsReview = false
		return
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

	m.PercentAgree = float64(maxCount) / float64(len(annotations))
	
	if len(annotations) == 2 {
		m.CohenKappa = calculateCohenKappa(annotations)
		m.FleissKappa = m.CohenKappa
	} else {
		m.FleissKappa = calculateFleissKappa(annotations)
		m.CohenKappa = m.FleissKappa
	}

	m.NeedsReview = m.FleissKappa < 0.6
}

func calculateCohenKappa(annotations []Annotation) float64 {
	if len(annotations) != 2 {
		return 0
	}

	if annotations[0].Label == annotations[1].Label {
		return 1.0
	}

	return 0.0
}

func calculateFleissKappa(annotations []Annotation) float64 {
	n := len(annotations)
	if n < 2 {
		return 1.0
	}

	labels := make(map[string]int)
	for _, a := range annotations {
		labels[a.Label]++
	}

	k := len(labels)
	if k <= 1 {
		return 1.0
	}

	pBar := 0.0
	for _, count := range labels {
		pBar += float64(count*count - count)
	}
	pBar = pBar / float64(n*(n-1))

	pe := 0.0
	for _, count := range labels {
		pj := float64(count) / float64(n)
		pe += pj * pj
	}

	if pe >= 1.0 {
		return 1.0
	}

	return (pBar - pe) / (1.0 - pe)
}

type ConfidenceLevel string

const (
	ConfidenceHigh   ConfidenceLevel = "high"
	ConfidenceMedium ConfidenceLevel = "medium"
	ConfidenceLow    ConfidenceLevel = "low"
)

func GetConfidenceLevel(confidence float64) ConfidenceLevel {
	if confidence >= 0.85 {
		return ConfidenceHigh
	}
	if confidence >= 0.60 {
		return ConfidenceMedium
	}
	return ConfidenceLow
}

type ReviewQueueItem struct {
	ConversationID string           `json:"conversation_id"`
	EvaluationID   string           `json:"evaluation_id"`
	Confidence     float64          `json:"confidence"`
	Reason         string           `json:"reason"`
	Annotations    []Annotation     `json:"annotations,omitempty"`
	Agreement      *AgreementMetrics `json:"agreement,omitempty"`
	CreatedAt      time.Time        `json:"created_at"`
}

