package evaluator

import (
	"fmt"

	"github.com/saisaravanan/healing-eval/internal/domain"
)

type BudgetEnforcer struct {
	maxTokensPerEval      int
	maxCostPerEval        float64
	maxTokensPerEvaluator int
}

func NewBudgetEnforcer() *BudgetEnforcer {
	return &BudgetEnforcer{
		maxTokensPerEval:      50000, // Demo: very large, never reached
		maxCostPerEval:        10.0,  // Demo: $10 per eval (very generous)
		maxTokensPerEvaluator: 20000, // Demo: 20K per evaluator
	}
}

// CheckPromptBudget checks if a prompt exceeds per-evaluator token budget
func (b *BudgetEnforcer) CheckPromptBudget(prompt string) error {
	estimatedTokens := EstimateTokens(prompt)
	if estimatedTokens > b.maxTokensPerEvaluator {
		return fmt.Errorf("prompt exceeds token budget: %d > %d",
			estimatedTokens, b.maxTokensPerEvaluator)
	}
	return nil
}

// CheckEvaluationBudget checks if total evaluation exceeds overall budget
func (b *BudgetEnforcer) CheckEvaluationBudget(usage *domain.AggregatedTokenUsage) error {
	if usage.TotalTokens > b.maxTokensPerEval {
		usage.BudgetExceeded = true
		return fmt.Errorf("evaluation exceeded token budget: %d > %d",
			usage.TotalTokens, b.maxTokensPerEval)
	}
	if usage.TotalCost > b.maxCostPerEval {
		usage.BudgetExceeded = true
		return fmt.Errorf("evaluation exceeded cost budget: $%.2f > $%.2f",
			usage.TotalCost, b.maxCostPerEval)
	}
	return nil
}

// CalculateCost estimates cost based on model and token usage
// Note: For self-hosted/open-source models, this returns 0 as cost is not per-token
func CalculateCost(model string, promptTokens, completionTokens int) float64 {
	// Pricing per 1K tokens (input, output)
	prices := map[string]struct{ prompt, completion float64 }{
		"gpt-4o":            {0.0025, 0.010},
		"gpt-4o-mini":       {0.00015, 0.0006},
		"gpt-4":             {0.03, 0.06},
		"gpt-4-turbo":       {0.01, 0.03},
		"gpt-3.5-turbo":     {0.0005, 0.0015},
		"claude-3-opus":     {0.015, 0.075},
		"claude-3-sonnet":   {0.003, 0.015},
		"claude-3-haiku":    {0.00025, 0.00125},
		"claude-3-5-sonnet": {0.003, 0.015},
	}

	p, exists := prices[model]
	if !exists {
		// For unknown models (including self-hosted), return 0
		return 0.0
	}

	promptCost := (float64(promptTokens) / 1000) * p.prompt
	completionCost := (float64(completionTokens) / 1000) * p.completion

	return promptCost + completionCost
}
