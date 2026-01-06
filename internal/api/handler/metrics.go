package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/saisaravanan/healing-eval/internal/domain"
)

type MetricsHandler struct{}

func NewMetricsHandler() *MetricsHandler {
	return &MetricsHandler{}
}

func (h *MetricsHandler) GetEvaluators(c *gin.Context) {
	response := domain.EvaluatorMetricsResponse{
		Evaluators: []domain.EvaluatorMetricsSummary{
			{
				EvaluatorType:  domain.EvaluatorTypeLLMJudge,
				Precision:      0.89,
				Recall:         0.76,
				F1Score:        0.82,
				SampleCount:    1250,
				FalsePositives: 45,
				FalseNegatives: 112,
			},
			{
				EvaluatorType:  domain.EvaluatorTypeToolCall,
				Precision:      0.94,
				Recall:         0.88,
				F1Score:        0.91,
				SampleCount:    890,
				FalsePositives: 23,
				FalseNegatives: 48,
			},
			{
				EvaluatorType:  domain.EvaluatorTypeCoherence,
				Precision:      0.85,
				Recall:         0.72,
				F1Score:        0.78,
				SampleCount:    654,
				FalsePositives: 67,
				FalseNegatives: 89,
			},
			{
				EvaluatorType:  domain.EvaluatorTypeHeuristic,
				Precision:      0.98,
				Recall:         0.95,
				F1Score:        0.96,
				SampleCount:    2100,
				FalsePositives: 12,
				FalseNegatives: 35,
			},
		},
	}

	c.JSON(http.StatusOK, response)
}

func (h *MetricsHandler) GetCalibration(c *gin.Context) {
	evaluatorType := c.Query("evaluator_type")
	if evaluatorType == "" {
		evaluatorType = "llm_judge"
	}

	response := domain.CalibrationMetrics{
		EvaluatorType:       domain.EvaluatorType(evaluatorType),
		PearsonCorrelation:  0.82,
		SpearmanCorrelation: 0.79,
		MeanAbsoluteError:   0.08,
		SampleCount:         1250,
		BreakdownByCategory: map[string]domain.CategoryMetrics{
			"response_quality": {Correlation: 0.85, MeanAbsoluteError: 0.06},
			"helpfulness":      {Correlation: 0.78, MeanAbsoluteError: 0.10},
			"factuality":       {Correlation: 0.81, MeanAbsoluteError: 0.09},
		},
		DriftStatus: "stable",
	}

	c.JSON(http.StatusOK, response)
}

func (h *MetricsHandler) GetBlindSpots(c *gin.Context) {
	response := domain.BlindSpotsResponse{
		BlindSpots: []domain.BlindSpot{
			{
				EvaluatorType:   domain.EvaluatorTypeLLMJudge,
				IssueType:       "subtle_factual_error",
				Recall:          0.34,
				SampleCount:     89,
				SuggestedAction: "Add explicit fact-checking prompt section",
				Examples: []domain.BlindSpotExample{
					{ConversationID: "conv_123", Description: "Missed incorrect date format"},
				},
			},
			{
				EvaluatorType:   domain.EvaluatorTypeCoherence,
				IssueType:       "pronoun_resolution_failure",
				Recall:          0.42,
				SampleCount:     67,
				SuggestedAction: "Enhance entity tracking in prompt",
			},
		},
	}

	c.JSON(http.StatusOK, response)
}

