package evaluator

import (
	"log"
	"sync"

	"github.com/saisaravanan/healing-eval/internal/domain"
)

// TokenTracker tracks token usage and costs per evaluation
type TokenTracker struct {
	mu    sync.RWMutex
	usage map[domain.EvaluatorType]*TokenUsage
}

type TokenUsage struct {
	TotalTokens      int
	PromptTokens     int
	CompletionTokens int
	EvaluationCount  int
	EstimatedCostUSD float64
}

func NewTokenTracker() *TokenTracker {
	return &TokenTracker{
		usage: make(map[domain.EvaluatorType]*TokenUsage),
	}
}

// EstimateTokens provides a rough estimate of tokens in text
// Using approximation: ~4 characters per token
func EstimateTokens(text string) int {
	return len(text) / 4
}

// RecordUsage records token usage for an evaluator
func (t *TokenTracker) RecordUsage(evalType domain.EvaluatorType, promptTokens, completionTokens int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.usage[evalType] == nil {
		t.usage[evalType] = &TokenUsage{}
	}

	usage := t.usage[evalType]
	usage.PromptTokens += promptTokens
	usage.CompletionTokens += completionTokens
	usage.TotalTokens += (promptTokens + completionTokens)
	usage.EvaluationCount++

	// Cost estimation (GPT-4 pricing as baseline)
	// Adjust based on actual provider
	const promptCostPer1kTokens = 0.03
	const completionCostPer1kTokens = 0.06

	usage.EstimatedCostUSD += (float64(promptTokens) / 1000.0 * promptCostPer1kTokens)
	usage.EstimatedCostUSD += (float64(completionTokens) / 1000.0 * completionCostPer1kTokens)
}

// GetUsage returns usage stats for an evaluator
func (t *TokenTracker) GetUsage(evalType domain.EvaluatorType) *TokenUsage {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if usage, ok := t.usage[evalType]; ok {
		// Return a copy
		return &TokenUsage{
			TotalTokens:      usage.TotalTokens,
			PromptTokens:     usage.PromptTokens,
			CompletionTokens: usage.CompletionTokens,
			EvaluationCount:  usage.EvaluationCount,
			EstimatedCostUSD: usage.EstimatedCostUSD,
		}
	}
	return &TokenUsage{}
}

// GetAllUsage returns usage stats for all evaluators
func (t *TokenTracker) GetAllUsage() map[domain.EvaluatorType]*TokenUsage {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make(map[domain.EvaluatorType]*TokenUsage)
	for k, v := range t.usage {
		result[k] = &TokenUsage{
			TotalTokens:      v.TotalTokens,
			PromptTokens:     v.PromptTokens,
			CompletionTokens: v.CompletionTokens,
			EvaluationCount:  v.EvaluationCount,
			EstimatedCostUSD: v.EstimatedCostUSD,
		}
	}
	return result
}

// LogUsageSummary logs a summary of token usage
func (t *TokenTracker) LogUsageSummary() {
	t.mu.RLock()
	defer t.mu.RUnlock()

	log.Println("=== Token Usage Summary ===")
	totalTokens := 0
	totalCost := 0.0

	for evalType, usage := range t.usage {
		log.Printf("%s: %d tokens (avg: %d/eval), $%.4f", 
			evalType, 
			usage.TotalTokens,
			usage.TotalTokens/max(usage.EvaluationCount, 1),
			usage.EstimatedCostUSD)
		totalTokens += usage.TotalTokens
		totalCost += usage.EstimatedCostUSD
	}

	log.Printf("TOTAL: %d tokens, $%.4f", totalTokens, totalCost)
	log.Println("===========================")
}

// CheckBudget warns if approaching context limits
func CheckBudget(promptText string, maxTokens int) bool {
	estimated := EstimateTokens(promptText)
	if estimated > maxTokens*8/10 {
		log.Printf("WARNING: Prompt approaching token limit: %d/%d tokens", estimated, maxTokens)
		return false
	}
	return true
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

