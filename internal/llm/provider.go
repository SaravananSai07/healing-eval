package llm

import (
	"context"
	"fmt"
	"time"

	"github.com/saisaravanan/healing-eval/internal/config"
)

type Provider interface {
	Name() string
	Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)
}

type CompletionRequest struct {
	Model       string
	Messages    []Message
	MaxTokens   int
	Temperature float64
	JSONMode    bool
}

type Message struct {
	Role    string
	Content string
}

type CompletionResponse struct {
	Content      string
	FinishReason string
	ModelName    string
	Usage        Usage
	Latency      time.Duration
}

type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

type Client struct {
	providers       map[string]Provider
	defaultProvider string
	timeout         time.Duration
}

func NewClient(cfg *config.LLMConfig) (*Client, error) {
	c := &Client{
		providers:       make(map[string]Provider),
		defaultProvider: cfg.DefaultProvider,
		timeout:         cfg.Timeout,
	}

	if cfg.OllamaBaseURL != "" {
		c.providers["ollama"] = NewOllamaProvider(cfg.OllamaBaseURL, cfg.OllamaModel)
	}

	if cfg.OpenAIAPIKey != "" {
		c.providers["openai"] = NewOpenAIProvider(cfg.OpenAIAPIKey)
	}

	if cfg.AnthropicAPIKey != "" {
		c.providers["anthropic"] = NewAnthropicProvider(cfg.AnthropicAPIKey)
	}

	if cfg.OpenRouterAPIKey != "" {
		c.providers["openrouter"] = NewOpenRouterProvider(cfg.OpenRouterAPIKey, cfg.OpenRouterModel, cfg.OpenRouterReasoning)
	}

	if len(c.providers) == 0 {
		return nil, fmt.Errorf("no LLM providers configured")
	}

	if _, ok := c.providers[c.defaultProvider]; !ok {
		for name := range c.providers {
			c.defaultProvider = name
			break
		}
	}

	return c, nil
}

func (c *Client) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	return c.CompleteWithProvider(ctx, c.defaultProvider, req)
}

func (c *Client) CompleteWithProvider(ctx context.Context, providerName string, req *CompletionRequest) (*CompletionResponse, error) {
	provider, ok := c.providers[providerName]
	if !ok {
		return nil, fmt.Errorf("provider %s not found", providerName)
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return provider.Complete(ctx, req)
}

func (c *Client) CompleteWithFallback(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	var lastErr error

	for name, provider := range c.providers {
		resp, err := provider.Complete(ctx, req)
		if err == nil {
			return resp, nil
		}
		lastErr = fmt.Errorf("%s: %w", name, err)
	}

	return nil, fmt.Errorf("all providers failed: %w", lastErr)
}
