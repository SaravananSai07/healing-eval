package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/saisaravanan/healing-eval/internal/storage"
)

type ReviewHandler struct {
	reviewRepo *storage.ReviewQueueRepo
	evalRepo   *storage.EvaluationRepo
	convRepo   *storage.ConversationRepo
}

func NewReviewHandler(reviewRepo *storage.ReviewQueueRepo, evalRepo *storage.EvaluationRepo, convRepo *storage.ConversationRepo) *ReviewHandler {
	return &ReviewHandler{
		reviewRepo: reviewRepo,
		evalRepo:   evalRepo,
		convRepo:   convRepo,
	}
}

// GET /api/v1/reviews/pending
func (h *ReviewHandler) GetPending(c *gin.Context) {
	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	items, err := h.reviewRepo.GetPending(c.Request.Context(), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch pending reviews"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"pending_reviews": items,
		"count":           len(items),
	})
}

// GET /api/v1/reviews/:id
func (h *ReviewHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "review ID is required"})
		return
	}

	item, err := h.reviewRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "review item not found"})
		return
	}

	// Fetch associated conversation and evaluations
	conv, _ := h.convRepo.GetByID(c.Request.Context(), item.ConversationID)
	evals, _ := h.evalRepo.GetByConversationID(c.Request.Context(), item.ConversationID)

	c.JSON(http.StatusOK, gin.H{
		"review":       item,
		"conversation": conv,
		"evaluations":  evals,
	})
}

// POST /api/v1/reviews/:id/complete
func (h *ReviewHandler) CompleteReview(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "review ID is required"})
		return
	}

	var req struct {
		ReviewerNotes string `json:"reviewer_notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := h.reviewRepo.CompleteReview(c.Request.Context(), id, req.ReviewerNotes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to complete review"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "review completed successfully"})
}

// POST /api/v1/reviews/:id/assign
func (h *ReviewHandler) AssignReview(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "review ID is required"})
		return
	}

	var req struct {
		AssignedTo string `json:"assigned_to"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if req.AssignedTo == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "assigned_to is required"})
		return
	}

	if err := h.reviewRepo.AssignReview(c.Request.Context(), id, req.AssignedTo); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to assign review"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "review assigned successfully"})
}

