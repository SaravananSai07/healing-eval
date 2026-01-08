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

type CoherenceEvaluator struct {
	client     *llm.Client
	weight     float64
	windowSize int
}

func NewCoherenceEvaluator(client *llm.Client) *CoherenceEvaluator {
	return &CoherenceEvaluator{
		client:     client,
		weight:     0.15,
		windowSize: 10,
	}
}

func (e *CoherenceEvaluator) Name() string {
	return "coherence"
}

func (e *CoherenceEvaluator) Type() domain.EvaluatorType {
	return domain.EvaluatorTypeCoherence
}

func (e *CoherenceEvaluator) Weight() float64 {
	return e.weight
}

func (e *CoherenceEvaluator) Evaluate(ctx context.Context, conv *domain.Conversation) (*domain.Evaluation, error) {
	start := time.Now()

	if len(conv.Turns) < 3 {
		return &domain.Evaluation{
			ID:             uuid.New().String(),
			ConversationID: conv.ID,
			EvaluatorType:  domain.EvaluatorTypeCoherence,
			Status:         domain.EvalStatusSuccess,
			Scores: domain.Scores{
				Overall:     1.0,
				Coherence:   1.0,
				Consistency: 1.0,
			},
			Confidence: 1.0,
			LatencyMs:  int(time.Since(start).Milliseconds()),
			CreatedAt:  time.Now(),
		}, nil
	}

	prompt := e.buildPrompt(conv)

	resp, err := e.client.Complete(ctx, &llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "system", Content: "You are an expert at evaluating conversation coherence and consistency. Always respond with valid JSON."},
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

	// Calculate cost
	cost := CalculateCost(resp.ModelName, resp.Usage.PromptTokens, resp.Usage.CompletionTokens)

	return &domain.Evaluation{
		ID:               uuid.New().String(),
		ConversationID:   conv.ID,
		EvaluatorType:    domain.EvaluatorTypeCoherence,
		Status:           domain.EvalStatusSuccess,
		ModelName:        resp.ModelName,
		PromptTokens:     resp.Usage.PromptTokens,
		CompletionTokens: resp.Usage.CompletionTokens,
		TotalTokens:      resp.Usage.TotalTokens,
		EstimatedCostUSD: cost,
		Scores: domain.Scores{
			Overall:     result.Overall,
			Coherence:   result.Coherence,
			Consistency: result.Consistency,
		},
		Issues:     result.Issues,
		Confidence: result.Confidence,
		RawOutput:  json.RawMessage(resp.Content),
		LatencyMs:  int(time.Since(start).Milliseconds()),
		CreatedAt:  time.Now(),
	}, nil
}

func (e *CoherenceEvaluator) buildPrompt(conv *domain.Conversation) string {
	var sb strings.Builder

	sb.WriteString("Evaluate coherence and consistency in this multi-turn conversation:\n\n")

	// Sanitize conversation to prevent prompt injection and handle large messages
	sanitizer := NewMessageSanitizer()
	turns := sanitizer.PrepareConversationForEval(conv.Turns)

	if len(turns) > e.windowSize*2 {
		sb.WriteString("[Earlier conversation summarized]\n")
		sb.WriteString(e.summarizeEarlierTurns(turns[:len(turns)-e.windowSize]))
		sb.WriteString("\n[Recent turns in full]\n\n")
		turns = turns[len(turns)-e.windowSize:]
	}

	for _, turn := range turns {
		role := strings.ToUpper(turn.Role)
		sb.WriteString(fmt.Sprintf("[%s] (Turn %d): %s\n\n", role, turn.TurnID, turn.Content))
	}

	sb.WriteString(`
Evaluate:
1. Coherence (0-1): Does the assistant maintain context across turns?
2. Consistency (0-1): Are there any contradictions in assistant responses?
3. Reference handling: Does the assistant properly resolve pronouns and references?

Look for:
- Context loss (forgetting earlier information)
- Contradictions between responses
- Improper handling of references to earlier turns

Respond with JSON:
{
  "coherence": <float>,
  "consistency": <float>,
  "overall": <float>,
  "confidence": <float>,
  "context_losses": [{"turn_id": <int>, "description": "..."}],
  "contradictions": [{"turn_ids": [<int>, <int>], "description": "..."}],
  "issues": [{"type": "...", "severity": "error|warning|info", "description": "...", "turn_id": <int or null>}],
  "reasoning": "..."
}`)

	return sb.String()
}

func (e *CoherenceEvaluator) summarizeEarlierTurns(turns []domain.Turn) string {
	// Try LLM-based summarization first
	summary, err := e.summarizeWithLLM(turns)
	if err == nil && summary != "" {
		return summary
	}

	// Fallback to simple summarization
	return e.summarizeSimple(turns)
}

func (e *CoherenceEvaluator) summarizeWithLLM(turns []domain.Turn) (string, error) {
	var sb strings.Builder
	sb.WriteString("Summarize the following conversation turns, focusing on:\n")
	sb.WriteString("1. Key topics discussed\n")
	sb.WriteString("2. Important entities (names, places, dates, etc.)\n")
	sb.WriteString("3. User requests and assistant commitments\n")
	sb.WriteString("4. Any context that would be needed to understand later turns\n\n")

	// Include all turns but keep it concise
	for i, turn := range turns {
		if i >= 20 {
			sb.WriteString(fmt.Sprintf("... and %d more turns\n", len(turns)-i))
			break
		}
		role := strings.ToUpper(turn.Role)
		content := turn.Content
		if len(content) > 200 {
			content = content[:200] + "..."
		}
		sb.WriteString(fmt.Sprintf("[%s] (Turn %d): %s\n", role, turn.TurnID, content))
	}

	sb.WriteString("\nProvide a concise summary (3-5 sentences):")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := e.client.Complete(ctx, &llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "system", Content: "You are a helpful assistant that creates concise, informative summaries of conversations."},
			{Role: "user", Content: sb.String()},
		},
		MaxTokens:   300,
		Temperature: 0.3,
	})

	if err != nil {
		return "", err
	}

	return "Summary: " + resp.Content + "\n", nil
}

func (e *CoherenceEvaluator) summarizeSimple(turns []domain.Turn) string {
	var sb strings.Builder
	sb.WriteString("Key points from earlier conversation:\n")

	userMsgCount := 0
	for _, turn := range turns {
		if turn.Role == "user" {
			userMsgCount++
			// Include first few user messages
			if userMsgCount <= 5 {
				content := turn.Content
				if len(content) > 100 {
					content = content[:100] + "..."
				}
				sb.WriteString(fmt.Sprintf("- User (Turn %d): %s\n", turn.TurnID, content))
			}
		}
	}

	if userMsgCount > 5 {
		sb.WriteString(fmt.Sprintf("... and %d more user messages\n", userMsgCount-5))
	}

	return sb.String()
}

type coherenceResponse struct {
	Coherence      float64        `json:"coherence"`
	Consistency    float64        `json:"consistency"`
	Overall        float64        `json:"overall"`
	Confidence     float64        `json:"confidence"`
	ContextLosses  []contextLoss  `json:"context_losses"`
	Contradictions []contradiction `json:"contradictions"`
	Issues         []domain.Issue `json:"issues"`
	Reasoning      string         `json:"reasoning"`
}

type contextLoss struct {
	TurnID      int    `json:"turn_id"`
	Description string `json:"description"`
}

type contradiction struct {
	TurnIDs     []int  `json:"turn_ids"`
	Description string `json:"description"`
}

func (e *CoherenceEvaluator) parseResponse(content string) (*coherenceResponse, error) {
	var result coherenceResponse
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	if result.Overall == 0 {
		result.Overall = (result.Coherence + result.Consistency) / 2
	}

	if result.Confidence == 0 {
		result.Confidence = 0.8
	}

	for _, cl := range result.ContextLosses {
		result.Issues = append(result.Issues, domain.Issue{
			Type:        "context_loss",
			Severity:    "warning",
			Description: cl.Description,
			TurnID:      &cl.TurnID,
		})
	}

	for _, c := range result.Contradictions {
		var turnID *int
		if len(c.TurnIDs) > 0 {
			turnID = &c.TurnIDs[0]
		}
		result.Issues = append(result.Issues, domain.Issue{
			Type:        "contradiction",
			Severity:    "error",
			Description: c.Description,
			TurnID:      turnID,
		})
	}

	return &result, nil
}

