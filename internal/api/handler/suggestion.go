package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/saisaravanan/healing-eval/internal/domain"
	"github.com/saisaravanan/healing-eval/internal/storage"
)

type SuggestionHandler struct {
	repo *storage.SuggestionRepo
}

func NewSuggestionHandler(repo *storage.SuggestionRepo) *SuggestionHandler {
	return &SuggestionHandler{repo: repo}
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
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

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
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

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

