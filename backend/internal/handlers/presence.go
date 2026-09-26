package handlers

import (
	"net/http"

	"github.com/chorus/messenger/internal/middleware"
	"github.com/chorus/messenger/internal/models"
	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
)

type PresenceHandler struct {
	presenceService *services.PresenceService
	privacyService  *services.PrivacyService
}

func NewPresenceHandler(ps *services.PresenceService) *PresenceHandler {
	return &PresenceHandler{
		presenceService: ps,
	}
}

func NewPresenceHandlerWithPrivacy(ps *services.PresenceService, priv *services.PrivacyService) *PresenceHandler {
	return &PresenceHandler{
		presenceService: ps,
		privacyService:  priv,
	}
}

func (h *PresenceHandler) GetPresence(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		WriteError(c, middleware.ErrAuth("Unauthorized"))
		return
	}
	targetUserID := c.Param("userId")
	if targetUserID == "" {
		WriteError(c, middleware.ErrValidation("User ID required"))
		return
	}
	presence, err := h.presenceService.GetPresence(targetUserID)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to get presence"))
		return
	}
	if h.privacyService != nil && !h.privacyService.CanViewLastSeen(userID, targetUserID) {
		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"userId": presence.UserID,
				"status": "offline",
			},
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": presence,
	})
}

func (h *PresenceHandler) GetMultiplePresence(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		WriteError(c, middleware.ErrAuth("Unauthorized"))
		return
	}
	var req struct {
		UserIDs []string `json:"userIds" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteError(c, middleware.ErrValidation("Invalid request body"))
		return
	}
	if len(req.UserIDs) > 100 {
		WriteError(c, middleware.ErrValidation("Maximum 100 users per request"))
		return
	}
	presence, err := h.presenceService.GetMultiplePresence(req.UserIDs)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to get presence"))
		return
	}
	if h.privacyService != nil {
		for uid, p := range presence {
			if !h.privacyService.CanViewLastSeen(userID, uid) {
				presence[uid] = &models.PresenceStatus{UserID: p.UserID, Status: "offline"}
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"data": presence,
	})
}

func (h *PresenceHandler) UpdatePresence(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		WriteError(c, middleware.ErrAuth("Unauthorized"))
		return
	}
	var req struct {
		Status     string `json:"status" binding:"required,oneof=online offline away"`
		DeviceType string `json:"deviceType"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteError(c, middleware.ErrValidation("Invalid request body"))
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
		WriteError(c, middleware.ErrInternal("Failed to update presence"))
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "Presence updated successfully",
	})
}

func (h *PresenceHandler) Heartbeat(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		WriteError(c, middleware.ErrAuth("Unauthorized"))
		return
	}
	deviceType := c.DefaultQuery("deviceType", "web")
	err := h.presenceService.Heartbeat(userID, deviceType)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to update heartbeat"))
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "Heartbeat recorded",
	})
}

func (h *PresenceHandler) UpdateActivity(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		WriteError(c, middleware.ErrAuth("Unauthorized"))
		return
	}
	if err := h.presenceService.UpdateUserActivity(userID); err != nil {
		WriteError(c, middleware.ErrInternal("Failed to update activity"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Activity recorded"})
}

func (h *PresenceHandler) GetOnlineCount(c *gin.Context) {
	count, err := h.presenceService.GetOnlineUserCount()
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to get online count"))
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"count": count,
		},
	})
}

func (h *PresenceHandler) GetPresenceStats(c *gin.Context) {
	stats, err := h.presenceService.GetPresenceStats()
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to get stats"))
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": stats,
	})
}
