package handlers

import (
	"net/http"

	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
)

type PresenceHandler struct {
	presenceService *services.PresenceService
}

func NewPresenceHandler(ps *services.PresenceService) *PresenceHandler {
	return &PresenceHandler{
		presenceService: ps,
	}
}

// GetPresence gets the presence status of a user
// GET /api/v1/presence/:userId
func (h *PresenceHandler) GetPresence(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	targetUserID := c.Param("userId")
	if targetUserID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID required"})
		return
	}

	presence, err := h.presenceService.GetPresence(targetUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get presence"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": presence,
	})
}

// GetMultiplePresence gets presence status of multiple users
// POST /api/v1/presence/batch
func (h *PresenceHandler) GetMultiplePresence(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		UserIDs []string `json:"userIds" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if len(req.UserIDs) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Maximum 100 users per request"})
		return
	}

	presence, err := h.presenceService.GetMultiplePresence(req.UserIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get presence"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": presence,
	})
}

// UpdatePresence updates the current user's presence
// PUT /api/v1/presence
func (h *PresenceHandler) UpdatePresence(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		Status     string `json:"status" binding:"required,oneof=online offline away"`
		DeviceType string `json:"deviceType"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	var err error
	switch req.Status {
	case "online":
		err = h.presenceService.SetOnline(userID, req.DeviceType)
	case "offline":
		err = h.presenceService.SetOffline(userID)
	case "away":
		err = h.presenceService.SetAway(userID)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update presence"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Presence updated successfully",
	})
}

// Heartbeat updates the user's last activity
// POST /api/v1/presence/heartbeat
func (h *PresenceHandler) Heartbeat(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	deviceType := c.DefaultQuery("deviceType", "web")

	err := h.presenceService.Heartbeat(userID, deviceType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update heartbeat"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Heartbeat recorded",
	})
}

// UpdateActivity updates last-seen timestamp without changing status
// POST /api/v1/presence/activity
func (h *PresenceHandler) UpdateActivity(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if err := h.presenceService.UpdateUserActivity(userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update activity"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Activity recorded"})
}

// GetOnlineCount returns the count of online users
// GET /api/v1/presence/online/count
func (h *PresenceHandler) GetOnlineCount(c *gin.Context) {
	count, err := h.presenceService.GetOnlineUserCount()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get online count"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"count": count,
		},
	})
}

// GetPresenceStats returns presence statistics (admin only)
// GET /api/v1/admin/presence/stats
func (h *PresenceHandler) GetPresenceStats(c *gin.Context) {
	stats, err := h.presenceService.GetPresenceStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": stats,
	})
}
