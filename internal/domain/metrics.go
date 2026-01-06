package domain

import "time"

type EvaluatorAccuracy struct {
	ID               string        `json:"id"`
	EvaluatorType    EvaluatorType `json:"evaluator_type"`
	MetricDate       time.Time     `json:"metric_date"`
	PrecisionScore   float64       `json:"precision_score"`
	RecallScore      float64       `json:"recall_score"`
	F1Score          float64       `json:"f1_score"`
	SampleCount      int           `json:"sample_count"`
	HumanCorrelation float64       `json:"human_correlation"`
	CreatedAt        time.Time     `json:"created_at"`
}

type CalibrationMetrics struct {
	EvaluatorType       EvaluatorType        `json:"evaluator_type"`
	Period              DateRange            `json:"period"`
	PearsonCorrelation  float64              `json:"pearson_correlation"`
	SpearmanCorrelation float64              `json:"spearman_correlation"`
	MeanAbsoluteError   float64              `json:"mean_absolute_error"`
	SampleCount         int                  `json:"sample_count"`
	BreakdownByCategory map[string]CategoryMetrics `json:"breakdown_by_category,omitempty"`
	DriftStatus         string               `json:"drift_status"`
	LastCalibration     time.Time            `json:"last_calibration"`
}

type DateRange struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

type CategoryMetrics struct {
	Correlation        float64 `json:"correlation"`
	MeanAbsoluteError  float64 `json:"mae"`
}

type EvaluatorMetricsResponse struct {
	Evaluators []EvaluatorMetricsSummary `json:"evaluators"`
}

type EvaluatorMetricsSummary struct {
	EvaluatorType  EvaluatorType `json:"evaluator_type"`
	Precision      float64       `json:"precision"`
	Recall         float64       `json:"recall"`
	F1Score        float64       `json:"f1_score"`
	SampleCount    int           `json:"sample_count"`
	FalsePositives int           `json:"false_positives"`
	FalseNegatives int           `json:"false_negatives"`
}

type BlindSpot struct {
	EvaluatorType   EvaluatorType `json:"evaluator_type"`
	IssueType       string        `json:"issue_type"`
	Recall          float64       `json:"recall"`
	SampleCount     int           `json:"sample_count"`
	Examples        []BlindSpotExample `json:"examples,omitempty"`
	SuggestedAction string        `json:"suggested_action"`
}

type BlindSpotExample struct {
	ConversationID string `json:"conversation_id"`
	Description    string `json:"description"`
}

type BlindSpotsResponse struct {
	BlindSpots []BlindSpot `json:"blind_spots"`
}

func CalculatePrecision(tp, fp int) float64 {
	if tp+fp == 0 {
		return 0
	}
	return float64(tp) / float64(tp+fp)
}

func CalculateRecall(tp, fn int) float64 {
	if tp+fn == 0 {
		return 0
	}
	return float64(tp) / float64(tp+fn)
}

func CalculateF1(precision, recall float64) float64 {
	if precision+recall == 0 {
		return 0
	}
	return 2 * (precision * recall) / (precision + recall)
}

