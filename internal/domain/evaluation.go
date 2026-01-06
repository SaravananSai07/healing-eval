package domain

import (
	"encoding/json"
	"time"
)

type EvaluatorType string

const (
	EvaluatorTypeLLMJudge  EvaluatorType = "llm_judge"
	EvaluatorTypeToolCall  EvaluatorType = "tool_call"
	EvaluatorTypeCoherence EvaluatorType = "coherence"
	EvaluatorTypeHeuristic EvaluatorType = "heuristic"
)

type Evaluation struct {
	ID             string          `json:"id"`
	ConversationID string          `json:"conversation_id"`
	EvaluatorType  EvaluatorType   `json:"evaluator_type"`
	Scores         Scores          `json:"scores"`
	Issues         []Issue         `json:"issues,omitempty"`
	Confidence     float64         `json:"confidence"`
	RawOutput      json.RawMessage `json:"raw_output,omitempty"`
	LatencyMs      int             `json:"latency_ms"`
	CreatedAt      time.Time       `json:"created_at"`
}

type Scores struct {
	Overall          float64 `json:"overall"`
	ResponseQuality  float64 `json:"response_quality,omitempty"`
	Helpfulness      float64 `json:"helpfulness,omitempty"`
	Factuality       float64 `json:"factuality,omitempty"`
	ToolAccuracy     float64 `json:"tool_accuracy,omitempty"`
	SelectionAccuracy float64 `json:"selection_accuracy,omitempty"`
	ParameterAccuracy float64 `json:"parameter_accuracy,omitempty"`
	Coherence        float64 `json:"coherence,omitempty"`
	Consistency      float64 `json:"consistency,omitempty"`
}

type Issue struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	TurnID      *int   `json:"turn_id,omitempty"`
}

type AggregatedEvaluation struct {
	ConversationID string       `json:"conversation_id"`
	Scores         Scores       `json:"scores"`
	ToolEvaluation *ToolEvaluation `json:"tool_evaluation,omitempty"`
	Issues         []Issue      `json:"issues_detected"`
	Evaluations    []Evaluation `json:"evaluations"`
	CreatedAt      time.Time    `json:"created_at"`
}

type ToolEvaluation struct {
	SelectionAccuracy float64 `json:"selection_accuracy"`
	ParameterAccuracy float64 `json:"parameter_accuracy"`
	ExecutionSuccess  bool    `json:"execution_success"`
	HallucinatedParams []string `json:"hallucinated_params,omitempty"`
}

type EvaluationsQueryRequest struct {
	ConversationIDs []string       `json:"conversation_ids,omitempty"`
	AgentVersions   []string       `json:"agent_versions,omitempty"`
	EvaluatorTypes  []EvaluatorType `json:"evaluator_types,omitempty"`
	DateFrom        *time.Time     `json:"date_from,omitempty"`
	DateTo          *time.Time     `json:"date_to,omitempty"`
	MinOverallScore *float64       `json:"min_overall_score,omitempty"`
	MaxOverallScore *float64       `json:"max_overall_score,omitempty"`
	HasIssues       *bool          `json:"has_issues,omitempty"`
	IssueTypes      []string       `json:"issue_types,omitempty"`
	Limit           int            `json:"limit,omitempty"`
	Offset          int            `json:"offset,omitempty"`
	SortBy          string         `json:"sort_by,omitempty"`
	SortOrder       string         `json:"sort_order,omitempty"`
}

type EvaluationsQueryResponse struct {
	Evaluations []Evaluation `json:"evaluations"`
	Total       int          `json:"total"`
	Limit       int          `json:"limit"`
	Offset      int          `json:"offset"`
	HasMore     bool         `json:"has_more"`
}

func (r *EvaluationsQueryRequest) SetDefaults() {
	if r.Limit <= 0 || r.Limit > 500 {
		r.Limit = 50
	}
	if r.SortBy == "" {
		r.SortBy = "created_at"
	}
	if r.SortOrder == "" {
		r.SortOrder = "desc"
	}
}

