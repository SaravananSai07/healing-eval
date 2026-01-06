package domain

import (
	"encoding/json"
	"time"
)

type Conversation struct {
	ID           string          `json:"conversation_id"`
	AgentVersion string          `json:"agent_version"`
	Turns        []Turn          `json:"turns"`
	Feedback     *Feedback       `json:"feedback,omitempty"`
	Metadata     json.RawMessage `json:"metadata,omitempty"`
	CreatedAt    time.Time       `json:"created_at,omitempty"`
	ProcessedAt  *time.Time      `json:"processed_at,omitempty"`
}

type Turn struct {
	TurnID    int        `json:"turn_id"`
	Role      string     `json:"role"`
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	Timestamp time.Time  `json:"timestamp"`
}

type ToolCall struct {
	ToolName   string          `json:"tool_name"`
	Parameters json.RawMessage `json:"parameters"`
	Result     *ToolResult     `json:"result,omitempty"`
	LatencyMs  int             `json:"latency_ms,omitempty"`
}

type ToolResult struct {
	Status string          `json:"status"`
	Data   json.RawMessage `json:"data,omitempty"`
	Error  string          `json:"error,omitempty"`
}

type Feedback struct {
	UserRating  *int            `json:"user_rating,omitempty"`
	OpsReview   *OpsReview      `json:"ops_review,omitempty"`
	Annotations []Annotation    `json:"annotations,omitempty"`
}

type OpsReview struct {
	Quality string `json:"quality"`
	Notes   string `json:"notes,omitempty"`
}

func (c *Conversation) TotalLatencyMs() int {
	var total int
	for _, turn := range c.Turns {
		for _, tc := range turn.ToolCalls {
			total += tc.LatencyMs
		}
	}
	return total
}

func (c *Conversation) HasToolCalls() bool {
	for _, turn := range c.Turns {
		if len(turn.ToolCalls) > 0 {
			return true
		}
	}
	return false
}

func (c *Conversation) GetAssistantTurns() []Turn {
	var turns []Turn
	for _, t := range c.Turns {
		if t.Role == "assistant" {
			turns = append(turns, t)
		}
	}
	return turns
}

