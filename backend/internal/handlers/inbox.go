package handlers

import (
	"github.com/chorus/messenger/internal/middleware"
	"github.com/chorus/messenger/internal/models"
	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
)

type InboxHandler struct {
	inbox *services.InboxService
}

func NewInboxHandler(inbox *services.InboxService) *InboxHandler {
	return &InboxHandler{inbox: inbox}
}

func (h *InboxHandler) GetPending(c *gin.Context) {
	userID := c.GetString("userID")
	if h.inbox == nil {
		WriteError(c, middleware.ErrInternal("Inbox unavailable"))
		return
	}
	limit := 100
	msgs, err := h.inbox.GetPendingMessagesForUser(c.Request.Context(), userID, limit)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to fetch pending"))
		return
	}
	if msgs == nil {
		msgs = []models.Message{}
	}
	c.JSON(200, gin.H{"messages": msgs, "count": len(msgs)})
}

func (h *InboxHandler) AckPending(c *gin.Context) {
	userID := c.GetString("userID")
	var req struct {
		MessageIDs []string `json:"messageIds" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.MessageIDs) == 0 {
		WriteError(c, middleware.ErrValidation("messageIds required"))
		return
	}
	if h.inbox == nil {
		WriteError(c, middleware.ErrInternal("Inbox unavailable"))
		return
	}
	if err := h.inbox.MarkPendingDelivered(c.Request.Context(), userID, req.MessageIDs); err != nil {
		WriteError(c, middleware.ErrInternal("Failed to ack"))
		return
	}
	c.JSON(200, gin.H{"acked": len(req.MessageIDs)})
}
