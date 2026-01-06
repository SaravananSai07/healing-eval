package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/saisaravanan/healing-eval/internal/domain"
	"github.com/saisaravanan/healing-eval/internal/storage"
)

type EvaluationHandler struct {
	repo *storage.EvaluationRepo
}

func NewEvaluationHandler(repo *storage.EvaluationRepo) *EvaluationHandler {
	return &EvaluationHandler{repo: repo}
}

func (h *EvaluationHandler) GetByConversationID(c *gin.Context) {
	conversationID := c.Param("conversation_id")
	if conversationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "conversation_id is required"})
		return
	}

	evals, err := h.repo.GetByConversationID(c.Request.Context(), conversationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve evaluations"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"conversation_id": conversationID,
		"evaluations":     evals,
	})
}

func (h *EvaluationHandler) Query(c *gin.Context) {
	var req domain.EvaluationsQueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	resp, err := h.repo.Query(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query evaluations"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

