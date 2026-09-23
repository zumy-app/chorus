package handlers

import (
	"net/http"

	"github.com/chorus/messenger/internal/middleware"
	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
)

// SparkyHandler serves async Sparky answers. POST /sparky/ask returns 202 +
// job id instantly; the answer arrives per-user over the WebSocket
// ("sparky_result") and is resyncable via GET /sparky/jobs/:jobId.
type SparkyHandler struct {
	queue *services.SparkyQueueService
}

func NewSparkyHandler(q *services.SparkyQueueService) *SparkyHandler {
	return &SparkyHandler{queue: q}
}

// AskSparkyRequest is the body for POST /sparky/ask.
type AskSparkyRequest struct {
	Context        string `json:"context" binding:"required"`
	Query          string `json:"query" binding:"required"`
	Language       string `json:"language" binding:"required"`
	NativeLanguage string `json:"nativeLanguage" binding:"required"`
	ChatID         string `json:"chatId"`
}

// AskSparky enqueues a Sparky question and returns the job id immediately.
func (h *SparkyHandler) AskSparky(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		WriteError(c, middleware.ErrAuth("Unauthorized"))
		return
	}

	var req AskSparkyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteError(c, middleware.ErrValidation("Invalid request body"))
		return
	}

	if h.queue == nil {
		WriteError(c, middleware.ErrUnavailable("Sparky queue not available"))
		return
	}

	jobID, err := h.queue.EnqueueForAsk(userID, req.ChatID, req.Context, req.Language, req.NativeLanguage, req.Query)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to queue Sparky question"))
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"jobId":  jobID,
		"status": "queued",
	})
}

// GetSparkyJob returns the state and (completed) answer of a Sparky job owned
// by the requesting user. Used for resync after reconnect/backgrounding.
// GET /api/v1/sparky/jobs/:jobId
func (h *SparkyHandler) GetSparkyJob(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		WriteError(c, middleware.ErrAuth("Unauthorized"))
		return
	}

	jobID := c.Param("jobId")
	if jobID == "" {
		WriteError(c, middleware.ErrValidation("job id is required"))
		return
	}

	if h.queue == nil {
		WriteError(c, middleware.ErrUnavailable("Sparky queue not available"))
		return
	}

	job, err := h.queue.GetJob(jobID, userID)
	if err != nil {
		WriteError(c, middleware.ErrNotFound("Sparky job not found"))
		return
	}

	c.JSON(http.StatusOK, job)
}
