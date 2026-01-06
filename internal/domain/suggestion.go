package domain

import (
	"encoding/json"
	"time"
)

type SuggestionType string

const (
	SuggestionTypePrompt SuggestionType = "prompt"
	SuggestionTypeTool   SuggestionType = "tool"
)

type SuggestionStatus string

const (
	SuggestionStatusPending  SuggestionStatus = "pending"
	SuggestionStatusApproved SuggestionStatus = "approved"
	SuggestionStatusRejected SuggestionStatus = "rejected"
	SuggestionStatusApplied  SuggestionStatus = "applied"
)

type Suggestion struct {
	ID             string           `json:"id"`
	PatternID      string           `json:"pattern_id,omitempty"`
	Type           SuggestionType   `json:"suggestion_type"`
	Target         string           `json:"target"`
	Suggestion     string           `json:"suggestion"`
	Rationale      string           `json:"rationale"`
	Confidence     float64          `json:"confidence"`
	Status         SuggestionStatus `json:"status"`
	ImpactMeasured *ImpactMetrics   `json:"impact_measured,omitempty"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
}

type ImpactMetrics struct {
	BeforeErrorRate float64   `json:"before_error_rate"`
	AfterErrorRate  float64   `json:"after_error_rate"`
	ImprovementPct  float64   `json:"improvement_percent"`
	SampleSize      int       `json:"sample_size"`
	Significant     bool      `json:"statistically_significant"`
	MeasuredAt      time.Time `json:"measured_at"`
}

type ApproveRequest struct {
	ApplyStrategy string `json:"apply_strategy"`
	ABTestPercent int    `json:"ab_test_percent,omitempty"`
	Notes         string `json:"notes,omitempty"`
}

type RejectRequest struct {
	Reason string `json:"reason"`
	Notes  string `json:"notes,omitempty"`
}

type FailurePattern struct {
	ID           string          `json:"id"`
	Type         string          `json:"type"`
	Description  string          `json:"description"`
	Count        int             `json:"count"`
	Examples     []string        `json:"examples"`
	Metadata     json.RawMessage `json:"metadata,omitempty"`
	FirstSeenAt  time.Time       `json:"first_seen_at"`
	LastSeenAt   time.Time       `json:"last_seen_at"`
}

