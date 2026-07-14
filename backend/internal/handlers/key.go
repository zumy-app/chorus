package handlers

import (
	"github.com/chorus/messenger/internal/models"
	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
)

type KeyHandler struct {
	keyService  *services.KeyService
	chatService *services.ChatService
}

func NewKeyHandler(keyService *services.KeyService, chatService *services.ChatService) *KeyHandler {
	return &KeyHandler{keyService: keyService, chatService: chatService}
}

func (h *KeyHandler) RegisterDeviceKeys(c *gin.Context) {
	userID := c.GetString("userID")

	var req models.RegisterDeviceKeysRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	bundle, err := h.keyService.RegisterDeviceKeys(userID, req)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to register device keys"})
		return
	}

	c.JSON(201, bundle)
}

func (h *KeyHandler) GetUserDeviceKeys(c *gin.Context) {
	userID := c.Param("userId")

	bundles, err := h.keyService.GetUserDeviceKeys(userID)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to fetch device keys"})
		return
	}

	c.JSON(200, gin.H{"devices": bundles})
}

func (h *KeyHandler) GetChatRecipientKey(c *gin.Context) {
	userID := c.GetString("userID")
	chatID := c.Param("chatId")
	deviceID := c.Query("deviceId")
	if deviceID == "" {
		c.JSON(400, gin.H{"error": "deviceId query parameter is required"})
		return
	}

	isParticipant, err := h.chatService.IsParticipant(chatID, userID)
	if err != nil || !isParticipant {
		c.JSON(403, gin.H{"error": "Access denied"})
		return
	}

	envelope, err := h.keyService.GetChatRecipientKey(chatID, userID, deviceID)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to fetch recipient key"})
		return
	}
	if envelope == nil {
		c.JSON(404, gin.H{"error": "Recipient key not found"})
		return
	}

	c.JSON(200, envelope)
}
