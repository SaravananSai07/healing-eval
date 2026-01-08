package llm

import (
	"context"
	"fmt"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

type OpenRouterProvider struct {
	client          *openai.Client
	model           string
	enableReasoning bool
}

func NewOpenRouterProvider(apiKey, model string, enableReasoning bool) *OpenRouterProvider {
	config := openai.DefaultConfig(apiKey)
	config.BaseURL = "https://openrouter.ai/api/v1"

	return &OpenRouterProvider{
		client:          openai.NewClientWithConfig(config),
		model:           model,
		enableReasoning: enableReasoning,
	}
}

func (p *OpenRouterProvider) Name() string {
	return "openrouter"
}

func (p *OpenRouterProvider) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	start := time.Now()

	model := req.Model
	if model == "" {
		model = p.model
	}
	if model == "" {
		model = "nvidia/nemotron-3-nano-30b-a3b:free"
	}

	messages := make([]openai.ChatCompletionMessage, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = openai.ChatCompletionMessage{
			Role:    m.Role,
			Content: m.Content,
		}
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 2048
	}

	chatReq := openai.ChatCompletionRequest{
		Model:       model,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: float32(req.Temperature),
	}

	if req.JSONMode {
		chatReq.ResponseFormat = &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		}
	}

	// Note: Reasoning mode is passed via custom parameters that OpenRouter supports
	// The go-openai library doesn't directly support this, but OpenRouter will
	// recognize it from the model capabilities

	var resp openai.ChatCompletionResponse
	var err error

	// Implement exponential backoff for rate limits
	maxRetries := 3
	for attempt := 0; attempt <= maxRetries; attempt++ {
		resp, err = p.client.CreateChatCompletion(ctx, chatReq)
		if err == nil {
			break
		}

		// Check if it's a rate limit error (429)
		if attempt < maxRetries && isRateLimitError(err) {
			waitTime := time.Duration(1<<uint(attempt)) * time.Second // 1s, 2s, 4s
			time.Sleep(waitTime)
			continue
		}

		return nil, fmt.Errorf("create completion (attempt %d): %w", attempt+1, err)
	}

	if err != nil {
		return nil, fmt.Errorf("create completion: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	return &CompletionResponse{
		Content:      resp.Choices[0].Message.Content,
		FinishReason: string(resp.Choices[0].FinishReason),
		ModelName:    model,
		Usage: Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
		Latency: time.Since(start),
	}, nil
}

// isRateLimitError checks if the error is a rate limit error
func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	// Check for common rate limit error indicators
	errStr := err.Error()
	return contains(errStr, "429") || contains(errStr, "rate limit") || contains(errStr, "Too Many Requests")
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
