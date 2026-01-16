package handlers

import (
	"net/http"

	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
)

type PresenceHandler struct {
	presenceService *services.PresenceService
}

func NewPresenceHandler(presenceService *services.PresenceService) *PresenceHandler {
	return &PresenceHandler{
		presenceService: presenceService,
	}
}

func (h *PresenceHandler) UpdatePresence(c *gin.Context) {
	userID := c.GetString("userID")

	var req struct {
		Status string `json:"status" binding:"required"` // "online" or "offline"
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var err error
	if req.Status == "online" {
		err = h.presenceService.SetUserOnline(userID)
	} else {
		err = h.presenceService.SetUserOffline(userID)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update presence"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func (h *PresenceHandler) GetPresence(c *gin.Context) {
	targetUserID := c.Param("userID")

	isOnline, err := h.presenceService.IsUserOnline(targetUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get presence"})
		return
	}

	lastSeen, err := h.presenceService.GetUserLastSeen(targetUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get last seen"})
		return
	}

	response := gin.H{
		"user_id":  targetUserID,
		"is_online": isOnline,
	}

	if lastSeen != nil {
		response["last_seen"] = lastSeen
	}

	c.JSON(http.StatusOK, response)
}

func (h *PresenceHandler) UpdateActivity(c *gin.Context) {
	userID := c.GetString("userID")

	err := h.presenceService.UpdateUserActivity(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update activity"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}
