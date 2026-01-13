package handler

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/saisaravanan/healing-eval/internal/domain"
	"github.com/saisaravanan/healing-eval/internal/improvement"
	"github.com/saisaravanan/healing-eval/internal/llm"
	"github.com/saisaravanan/healing-eval/internal/storage"
)

type SuggestionHandler struct {
	repo            *storage.SuggestionRepo
	evalRepo        *storage.EvaluationRepo
	llmClient       *llm.Client
	patternDetector *improvement.PatternDetector
	suggester       *improvement.Suggester
}

func NewSuggestionHandler(
	repo *storage.SuggestionRepo,
	evalRepo *storage.EvaluationRepo,
	llmClient *llm.Client,
) *SuggestionHandler {
	return &SuggestionHandler{
		repo:            repo,
		evalRepo:        evalRepo,
		llmClient:       llmClient,
		patternDetector: improvement.NewPatternDetector(),
		suggester:       improvement.NewSuggester(llmClient),
	}
}

func (h *SuggestionHandler) List(c *gin.Context) {
	statusParam := c.Query("status")
	var status *domain.SuggestionStatus
	if statusParam != "" {
		s := domain.SuggestionStatus(statusParam)
		status = &s
	}

	suggestions, err := h.repo.List(c.Request.Context(), status, 100, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list suggestions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"suggestions": suggestions})
}

func (h *SuggestionHandler) GetByID(c *gin.Context) {
	id := c.Param("id")

	suggestion, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve suggestion"})
		return
	}

	if suggestion == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "suggestion not found"})
		return
	}

	c.JSON(http.StatusOK, suggestion)
}

func (h *SuggestionHandler) Approve(c *gin.Context) {
	id := c.Param("id")

	var req domain.ApproveRequest
	c.ShouldBindJSON(&req)
	c.ShouldBind(&req)

	if err := h.repo.UpdateStatus(c.Request.Context(), id, domain.SuggestionStatusApproved); err != nil {
		if err.Error() == "suggestion not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "suggestion not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to approve suggestion"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "approved"})
}

func (h *SuggestionHandler) Reject(c *gin.Context) {
	id := c.Param("id")

	var req domain.RejectRequest
	c.ShouldBindJSON(&req)
	c.ShouldBind(&req)

	if err := h.repo.UpdateStatus(c.Request.Context(), id, domain.SuggestionStatusRejected); err != nil {
		if err.Error() == "suggestion not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "suggestion not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reject suggestion"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "rejected", "reason": req.Reason})
}

func (h *SuggestionHandler) GetImpact(c *gin.Context) {
	id := c.Param("id")

	suggestion, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve suggestion"})
		return
	}

	if suggestion == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "suggestion not found"})
		return
	}

	if suggestion.ImpactMeasured == nil {
		c.JSON(http.StatusOK, gin.H{
			"suggestion_id": id,
			"status":        suggestion.Status,
			"impact":        nil,
			"message":       "Impact not yet measured",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"suggestion_id": id,
		"status":        suggestion.Status,
		"impact":        suggestion.ImpactMeasured,
	})
}

func (h *SuggestionHandler) Generate(c *gin.Context) {
	var req struct {
		HoursBack      int     `json:"hours_back"`
		MaxScore       float64 `json:"max_score"`
		MinOccurrences int     `json:"min_occurrences"`
	}

	// Set defaults
	req.HoursBack = 24
	req.MaxScore = 0.7
	req.MinOccurrences = 5

	if err := c.ShouldBindJSON(&req); err != nil {
		// Use defaults if no body provided
		log.Printf("Using default parameters for suggestion generation")
	}

	log.Printf("Generating suggestions for last %d hours, max_score=%.2f", req.HoursBack, req.MaxScore)

	// Query recent low-scoring evaluations
	dateTo := time.Now()
	dateFrom := dateTo.Add(-time.Duration(req.HoursBack) * time.Hour)

	evalReq := &domain.EvaluationsQueryRequest{
		DateFrom:        &dateFrom,
		DateTo:          &dateTo,
		MaxOverallScore: &req.MaxScore,
		Limit:           500,
	}

	evalResp, err := h.evalRepo.Query(c.Request.Context(), evalReq)
	if err != nil {
		log.Printf("Error querying evaluations: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query evaluations"})
		return
	}

	// Convert to pointer slice
	evalPtrs := make([]*domain.Evaluation, len(evalResp.Evaluations))
	for i := range evalResp.Evaluations {
		evalPtrs[i] = &evalResp.Evaluations[i]
	}

	// Detect patterns
	patterns := h.patternDetector.DetectPatterns(c.Request.Context(), evalPtrs)
	log.Printf("Detected %d failure patterns", len(patterns))

	if len(patterns) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"patterns_detected":     0,
			"suggestions_generated": 0,
			"suggestions_stored":    0,
			"message":               "No significant failure patterns detected",
		})
		return
	}

	// Generate suggestions
	suggestions, err := h.suggester.GenerateSuggestions(c.Request.Context(), patterns)
	if err != nil {
		log.Printf("Error generating suggestions: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate suggestions"})
		return
	}

	// Store suggestions (skip duplicates)
	stored := 0
	for _, suggestion := range suggestions {
		// Check if suggestion already exists for this pattern
		existing, err := h.repo.GetByPatternID(c.Request.Context(), suggestion.PatternID)
		if err == nil && existing != nil {
			log.Printf("Suggestion already exists for pattern %s, skipping", suggestion.PatternID)
			continue
		}

		if err := h.repo.Create(c.Request.Context(), suggestion); err != nil {
			log.Printf("Error storing suggestion: %v", err)
			continue
		}
		stored++
	}

	c.JSON(http.StatusOK, gin.H{
		"patterns_detected":     len(patterns),
		"suggestions_generated": len(suggestions),
		"suggestions_stored":    stored,
		"message":               "Suggestion generation completed successfully",
	})
}
