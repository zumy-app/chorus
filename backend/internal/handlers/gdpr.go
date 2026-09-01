package handlers

import (
	"net/http"

	"github.com/chorus/messenger/internal/middleware"
	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
)

type GDPRHandler struct {
	gdprService      *services.GDPRService
	retentionService *services.RetentionService
}

func NewGDPRHandler(gdprService *services.GDPRService, retentionService *services.RetentionService) *GDPRHandler {
	return &GDPRHandler{gdprService: gdprService, retentionService: retentionService}
}

func (h *GDPRHandler) ExportMyData(c *gin.Context) {
	userID := c.GetString("userID")
	data, err := h.gdprService.ExportUserData(userID)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to export data"))
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *GDPRHandler) DeleteMyAccount(c *gin.Context) {
	userID := c.GetString("userID")
	if err := h.gdprService.EraseUser(userID); err != nil {
		WriteError(c, middleware.ErrInternal("Failed to delete account"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Account deleted", "userId": userID})
}

func (h *GDPRHandler) GetRetentionPolicy(c *gin.Context) {
	policy := h.retentionService.GetPolicy()
	c.JSON(http.StatusOK, policy)
}

func (h *GDPRHandler) PurgeExpired(c *gin.Context) {
	result, err := h.retentionService.PurgeExpired(c.Request.Context())
	if err != nil {
		WriteError(c, middleware.ErrInternal("Purge failed"))
		return
	}
	c.JSON(http.StatusOK, result)
}
