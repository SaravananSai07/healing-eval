package improvement

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

type Suggester struct {
	llmClient *llm.Client
}

func NewSuggester(client *llm.Client) *Suggester {
	return &Suggester{llmClient: client}
}

func (s *Suggester) GenerateSuggestions(ctx context.Context, patterns []*domain.FailurePattern) ([]*domain.Suggestion, error) {
	var suggestions []*domain.Suggestion

	for _, pattern := range patterns {
		suggestion, err := s.generateForPattern(ctx, pattern)
		if err != nil {
			continue
		}
		suggestions = append(suggestions, suggestion)
	}

	return suggestions, nil
}

func (s *Suggester) generateForPattern(ctx context.Context, pattern *domain.FailurePattern) (*domain.Suggestion, error) {
	prompt := s.buildPrompt(pattern)

	resp, err := s.llmClient.Complete(ctx, &llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "system", Content: "You are an expert at improving AI agent prompts and tools. Generate actionable improvement suggestions."},
			{Role: "user", Content: prompt},
		},
		MaxTokens:   512,
		Temperature: 0.3,
		JSONMode:    true,
	})

	if err != nil {
		return nil, fmt.Errorf("llm completion: %w", err)
	}

	result, err := s.parseResponse(resp.Content)
	if err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	now := time.Now()
	return &domain.Suggestion{
		ID:         uuid.New().String(),
		PatternID:  pattern.ID,
		Type:       result.Type,
		Target:     result.Target,
		Suggestion: result.Suggestion,
		Rationale:  result.Rationale,
		Confidence: result.Confidence,
		Status:     domain.SuggestionStatusPending,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

func (s *Suggester) buildPrompt(pattern *domain.FailurePattern) string {
	var sb strings.Builder

	sb.WriteString("Analyze this failure pattern and suggest an improvement:\n\n")
	sb.WriteString(fmt.Sprintf("Pattern Type: %s\n", pattern.Type))
	sb.WriteString(fmt.Sprintf("Description: %s\n", pattern.Description))
	sb.WriteString(fmt.Sprintf("Occurrence Count: %d\n", pattern.Count))
	sb.WriteString("\nExamples:\n")
	for i, ex := range pattern.Examples {
		if i >= 3 {
			break
		}
		sb.WriteString(fmt.Sprintf("- %s\n", ex))
	}

	sb.WriteString(`
Generate an improvement suggestion. Respond with JSON:
{
  "suggestion_type": "prompt" or "tool",
  "target": "<what to modify - prompt name or tool name>",
  "suggestion": "<specific, actionable suggestion>",
  "rationale": "<why this will help>",
  "confidence": <float 0-1>
}`)

	return sb.String()
}

type suggestionResponse struct {
	Type       domain.SuggestionType `json:"suggestion_type"`
	Target     string                `json:"target"`
	Suggestion string                `json:"suggestion"`
	Rationale  string                `json:"rationale"`
	Confidence float64               `json:"confidence"`
}

func (s *Suggester) parseResponse(content string) (*suggestionResponse, error) {
	var result suggestionResponse
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	if result.Confidence == 0 {
		result.Confidence = 0.7
	}

	return &result, nil
}

