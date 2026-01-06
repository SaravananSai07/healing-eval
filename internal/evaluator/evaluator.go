package evaluator

import (
	"context"

	"github.com/saisaravanan/healing-eval/internal/domain"
)

type Evaluator interface {
	Name() string
	Type() domain.EvaluatorType
	Evaluate(ctx context.Context, conv *domain.Conversation) (*domain.Evaluation, error)
	Weight() float64
}

type EvaluatorConfig struct {
	Enabled bool
	Weight  float64
}

