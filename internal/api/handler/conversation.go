package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/saisaravanan/healing-eval/internal/domain"
	"github.com/saisaravanan/healing-eval/internal/queue"
	"github.com/saisaravanan/healing-eval/internal/storage"
)

const MaxConversationsPerRequest = 30

type ConversationHandler struct {
	repo  *storage.ConversationRepo
	queue *queue.RedisQueue
}

func NewConversationHandler(repo *storage.ConversationRepo, q *queue.RedisQueue) *ConversationHandler {
	return &ConversationHandler{repo: repo, queue: q}
}

type IngestRequest struct {
	Conversations []*domain.Conversation `json:"conversations"`
	Priority      string                 `json:"priority,omitempty"`
}

type IngestResponse struct {
	Accepted int      `json:"accepted"`
	JobID    string   `json:"job_id,omitempty"`
	IDs      []string `json:"ids"`
}

func (h *ConversationHandler) Ingest(c *gin.Context) {
	var req IngestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if len(req.Conversations) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no conversations provided"})
		return
	}

	if len(req.Conversations) > MaxConversationsPerRequest {
		c.JSON(http.StatusBadRequest, gin.H{"error": "exceeds maximum batch size of 30"})
		return
	}

	for _, conv := range req.Conversations {
		if conv.ID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "conversation_id is required"})
			return
		}
		if conv.AgentVersion == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "agent_version is required"})
			return
		}
	}

	if err := h.repo.CreateBatch(c.Request.Context(), req.Conversations); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store conversations"})
		return
	}

	if err := h.queue.PublishBatch(c.Request.Context(), req.Conversations); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to queue conversations"})
		return
	}

	ids := make([]string, len(req.Conversations))
	for i, conv := range req.Conversations {
		ids[i] = conv.ID
	}

	c.JSON(http.StatusAccepted, IngestResponse{
		Accepted: len(req.Conversations),
		IDs:      ids,
	})
}

func (h *ConversationHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}

	conv, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve conversation"})
		return
	}

	if conv == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "conversation not found"})
		return
	}

	c.JSON(http.StatusOK, conv)
}

type UpdateFeedbackRequest struct {
	Feedback *domain.Feedback `json:"feedback"`
}

func (h *ConversationHandler) UpdateFeedback(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}

	var req UpdateFeedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if req.Feedback == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "feedback is required"})
		return
	}

	if err := h.repo.UpdateFeedback(c.Request.Context(), id, req.Feedback); err != nil {
		if err.Error() == "conversation not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "conversation not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update feedback"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "updated"})
}

