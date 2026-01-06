package evaluator

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/saisaravanan/healing-eval/internal/domain"
	"github.com/saisaravanan/healing-eval/internal/llm"
)

type LLMJudgeEvaluator struct {
	client *llm.Client
	weight float64
}

func NewLLMJudgeEvaluator(client *llm.Client) *LLMJudgeEvaluator {
	return &LLMJudgeEvaluator{
		client: client,
		weight: 0.4,
	}
}

func (e *LLMJudgeEvaluator) Name() string {
	return "llm_judge"
}

func (e *LLMJudgeEvaluator) Type() domain.EvaluatorType {
	return domain.EvaluatorTypeLLMJudge
}

func (e *LLMJudgeEvaluator) Weight() float64 {
	return e.weight
}

func (e *LLMJudgeEvaluator) Evaluate(ctx context.Context, conv *domain.Conversation) (*domain.Evaluation, error) {
	start := time.Now()

	prompt := e.buildPrompt(conv)

	resp, err := e.client.Complete(ctx, &llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "system", Content: "You are an expert AI response evaluator. Always respond with valid JSON."},
			{Role: "user", Content: prompt},
		},
		MaxTokens:   1024,
		Temperature: 0.1,
		JSONMode:    true,
	})

	if err != nil {
		return nil, fmt.Errorf("llm completion: %w", err)
	}

	result, err := e.parseResponse(resp.Content)
	if err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &domain.Evaluation{
		ID:             uuid.New().String(),
		ConversationID: conv.ID,
		EvaluatorType:  domain.EvaluatorTypeLLMJudge,
		Scores: domain.Scores{
			Overall:         result.Overall,
			ResponseQuality: result.ResponseQuality,
			Helpfulness:     result.Helpfulness,
			Factuality:      result.Factuality,
		},
		Issues:     result.Issues,
		Confidence: result.Confidence,
		RawOutput:  json.RawMessage(resp.Content),
		LatencyMs:  int(time.Since(start).Milliseconds()),
		CreatedAt:  time.Now(),
	}, nil
}

func (e *LLMJudgeEvaluator) buildPrompt(conv *domain.Conversation) string {
	var sb strings.Builder

	sb.WriteString("Evaluate this AI assistant conversation:\n\n")

	for _, turn := range conv.Turns {
		role := strings.ToUpper(turn.Role)
		sb.WriteString(fmt.Sprintf("[%s] (Turn %d): %s\n", role, turn.TurnID, turn.Content))

		if len(turn.ToolCalls) > 0 {
			sb.WriteString("Tool Calls:\n")
			for _, tc := range turn.ToolCalls {
				sb.WriteString(fmt.Sprintf("- %s: %s\n", tc.ToolName, string(tc.Parameters)))
				if tc.Result != nil {
					sb.WriteString(fmt.Sprintf("  Result: %s\n", tc.Result.Status))
				}
			}
		}
		sb.WriteString("\n")
	}

	sb.WriteString(`
Evaluate the assistant's performance on:
1. Response Quality (0-1): Is the response well-structured and appropriate?
2. Helpfulness (0-1): Does it effectively address the user's needs?
3. Factuality (0-1): Are claims accurate based on context?

Respond with JSON:
{
  "response_quality": <float>,
  "helpfulness": <float>,
  "factuality": <float>,
  "overall": <float>,
  "confidence": <float>,
  "issues": [{"type": "...", "severity": "error|warning|info", "description": "...", "turn_id": <int or null>}],
  "reasoning": "..."
}`)

	return sb.String()
}

type llmJudgeResponse struct {
	ResponseQuality float64        `json:"response_quality"`
	Helpfulness     float64        `json:"helpfulness"`
	Factuality      float64        `json:"factuality"`
	Overall         float64        `json:"overall"`
	Confidence      float64        `json:"confidence"`
	Issues          []domain.Issue `json:"issues"`
	Reasoning       string         `json:"reasoning"`
}

func (e *LLMJudgeEvaluator) parseResponse(content string) (*llmJudgeResponse, error) {
	var result llmJudgeResponse
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	if result.Overall == 0 && result.ResponseQuality > 0 {
		result.Overall = (result.ResponseQuality + result.Helpfulness + result.Factuality) / 3
	}

	if result.Confidence == 0 {
		result.Confidence = 0.8
	}

	return &result, nil
}

