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

type ToolCallEvaluator struct {
	client *llm.Client
	weight float64
}

func NewToolCallEvaluator(client *llm.Client) *ToolCallEvaluator {
	return &ToolCallEvaluator{
		client: client,
		weight: 0.25,
	}
}

func (e *ToolCallEvaluator) Name() string {
	return "tool_call"
}

func (e *ToolCallEvaluator) Type() domain.EvaluatorType {
	return domain.EvaluatorTypeToolCall
}

func (e *ToolCallEvaluator) Weight() float64 {
	return e.weight
}

func (e *ToolCallEvaluator) Evaluate(ctx context.Context, conv *domain.Conversation) (*domain.Evaluation, error) {
	start := time.Now()

	if !conv.HasToolCalls() {
		return &domain.Evaluation{
			ID:             uuid.New().String(),
			ConversationID: conv.ID,
			EvaluatorType:  domain.EvaluatorTypeToolCall,
			Scores: domain.Scores{
				Overall:           1.0,
				ToolAccuracy:      1.0,
				SelectionAccuracy: 1.0,
				ParameterAccuracy: 1.0,
			},
			Confidence: 1.0,
			LatencyMs:  int(time.Since(start).Milliseconds()),
			CreatedAt:  time.Now(),
		}, nil
	}

	prompt := e.buildPrompt(conv)

	resp, err := e.client.Complete(ctx, &llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "system", Content: "You are an expert at evaluating AI tool usage. Always respond with valid JSON."},
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
		EvaluatorType:  domain.EvaluatorTypeToolCall,
		Scores: domain.Scores{
			Overall:           result.Overall,
			ToolAccuracy:      result.Overall,
			SelectionAccuracy: result.SelectionAccuracy,
			ParameterAccuracy: result.ParameterAccuracy,
		},
		Issues:     result.Issues,
		Confidence: result.Confidence,
		RawOutput:  json.RawMessage(resp.Content),
		LatencyMs:  int(time.Since(start).Milliseconds()),
		CreatedAt:  time.Now(),
	}, nil
}

func (e *ToolCallEvaluator) buildPrompt(conv *domain.Conversation) string {
	var sb strings.Builder

	sb.WriteString("Evaluate the tool calls in this conversation:\n\n")

	for _, turn := range conv.Turns {
		if turn.Role == "user" {
			sb.WriteString(fmt.Sprintf("[USER] (Turn %d): %s\n\n", turn.TurnID, turn.Content))
		}

		if turn.Role == "assistant" && len(turn.ToolCalls) > 0 {
			sb.WriteString(fmt.Sprintf("[ASSISTANT] (Turn %d):\n", turn.TurnID))
			sb.WriteString(fmt.Sprintf("Response: %s\n", turn.Content))
			sb.WriteString("Tool Calls:\n")

			for _, tc := range turn.ToolCalls {
				sb.WriteString(fmt.Sprintf("- Tool: %s\n", tc.ToolName))
				sb.WriteString(fmt.Sprintf("  Parameters: %s\n", string(tc.Parameters)))
				if tc.Result != nil {
					sb.WriteString(fmt.Sprintf("  Result Status: %s\n", tc.Result.Status))
					if tc.Result.Error != "" {
						sb.WriteString(fmt.Sprintf("  Error: %s\n", tc.Result.Error))
					}
				}
			}
			sb.WriteString("\n")
		}
	}

	sb.WriteString(`
Evaluate the tool usage:
1. Selection Accuracy (0-1): Was the correct tool chosen for the task?
2. Parameter Accuracy (0-1): Were parameters extracted correctly from context?
3. Check for hallucinated parameters (made-up values not in context)

Respond with JSON:
{
  "selection_accuracy": <float>,
  "parameter_accuracy": <float>,
  "overall": <float>,
  "confidence": <float>,
  "hallucinated_params": ["param1", "param2"],
  "issues": [{"type": "...", "severity": "error|warning|info", "description": "...", "turn_id": <int or null>}],
  "reasoning": "..."
}`)

	return sb.String()
}

type toolCallResponse struct {
	SelectionAccuracy  float64        `json:"selection_accuracy"`
	ParameterAccuracy  float64        `json:"parameter_accuracy"`
	Overall            float64        `json:"overall"`
	Confidence         float64        `json:"confidence"`
	HallucinatedParams []string       `json:"hallucinated_params"`
	Issues             []domain.Issue `json:"issues"`
	Reasoning          string         `json:"reasoning"`
}

func (e *ToolCallEvaluator) parseResponse(content string) (*toolCallResponse, error) {
	var result toolCallResponse
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	if result.Overall == 0 {
		result.Overall = (result.SelectionAccuracy + result.ParameterAccuracy) / 2
	}

	if result.Confidence == 0 {
		result.Confidence = 0.8
	}

	if len(result.HallucinatedParams) > 0 {
		result.Issues = append(result.Issues, domain.Issue{
			Type:        "hallucination",
			Severity:    "error",
			Description: fmt.Sprintf("Hallucinated parameters detected: %v", result.HallucinatedParams),
		})
	}

	return &result, nil
}

