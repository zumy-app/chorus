package handlers

import (
	"github.com/chorus/messenger/internal/models"
	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
)

type ChatHandler struct {
	chatService *services.ChatService
	userService *services.UserService
	keyService  *services.KeyService
	wsHub       *services.WebSocketHub
}

func NewChatHandler(chatService *services.ChatService, userService *services.UserService, keyService *services.KeyService, wsHub *services.WebSocketHub) *ChatHandler {
	return &ChatHandler{
		chatService: chatService,
		userService: userService,
		keyService:  keyService,
		wsHub:       wsHub,
	}
}

func (h *ChatHandler) GetUserChats(c *gin.Context) {
	userID := c.GetString("userID")

	chats, err := h.chatService.GetUserChats(userID)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to fetch chats"})
		return
	}

	// Enrich chats with participants
	for i := range chats {
		participants, err := h.chatService.GetParticipants(chats[i].ID)
		if err != nil {
			continue
		}

		// Get user details for each participant
		userIDs := make([]string, len(participants))
		for j, p := range participants {
			userIDs[j] = p.UserID
		}

		users, err := h.userService.GetMultiple(userIDs)
		if err != nil {
			continue
		}

		// Attach user details to participants
		for j := range participants {
			if user, ok := users[participants[j].UserID]; ok {
				participants[j].User = user
			}
		}

		chats[i].Participants = participants
	}

	c.JSON(200, gin.H{
		"chats":   chats,
		"total":   len(chats),
		"hasMore": false,
	})
}

func (h *ChatHandler) CreateChat(c *gin.Context) {
	userID := c.GetString("userID")

	var req models.CreateChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	chat, err := h.chatService.Create(userID, req)
	if err != nil {
		c.JSON(400, gin.H{"error": "Failed to create chat: " + err.Error()})
		return
	}
	if h.keyService != nil && len(req.RecipientKeys) > 0 {
		if err := h.keyService.StoreChatRecipientKeys(chat.ID, req.RecipientKeys); err != nil {
			c.JSON(400, gin.H{"error": "Failed to store recipient keys: " + err.Error()})
			return
		}
	}

	// Get participants with user details
	participants, _ := h.chatService.GetParticipants(chat.ID)
	userIDs := make([]string, len(participants))
	for i, p := range participants {
		userIDs[i] = p.UserID
	}
	users, _ := h.userService.GetMultiple(userIDs)
	for i := range participants {
		if user, ok := users[participants[i].UserID]; ok {
			participants[i].User = user
		}
	}
	chat.Participants = participants

	// Broadcast chat_updated to all participants so their chat lists refresh
	// (the other participant needs to know they were added to a new chat)
	h.wsHub.SendToChat(chat.ID, userIDs, "chat_updated", chat)

	c.JSON(201, chat)
}

func (h *ChatHandler) GetChat(c *gin.Context) {
	userID := c.GetString("userID")
	chatID := c.Param("chatId")

	// Check if user is a participant
	isParticipant, err := h.chatService.IsParticipant(chatID, userID)
	if err != nil || !isParticipant {
		c.JSON(403, gin.H{"error": "Access denied"})
		return
	}

	chat, err := h.chatService.GetByID(chatID)
	if err != nil {
		c.JSON(404, gin.H{"error": "Chat not found"})
		return
	}

	// Get participants with user details
	participants, _ := h.chatService.GetParticipants(chatID)
	userIDs := make([]string, len(participants))
	for i, p := range participants {
		userIDs[i] = p.UserID
	}
	users, _ := h.userService.GetMultiple(userIDs)
	for i := range participants {
		if user, ok := users[participants[i].UserID]; ok {
			participants[i].User = user
		}
	}
	chat.Participants = participants

	c.JSON(200, chat)
}

func (h *ChatHandler) UpdateChat(c *gin.Context) {
	userID := c.GetString("userID")
	chatID := c.Param("chatId")

	// Check if user is a participant
	isParticipant, err := h.chatService.IsParticipant(chatID, userID)
	if err != nil || !isParticipant {
		c.JSON(403, gin.H{"error": "Access denied"})
		return
	}

	var req struct {
		Name     string                 `json:"name"`
		Settings map[string]interface{} `json:"settings"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	var name *string
	if req.Name != "" {
		name = &req.Name
	}

	chat, err := h.chatService.UpdateChat(chatID, name, req.Settings)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to update chat"})
		return
	}

	c.JSON(200, chat)
}

func (h *ChatHandler) AddParticipant(c *gin.Context) {
	userID := c.GetString("userID")
	chatID := c.Param("chatId")

	// Check if user is a participant
	isParticipant, err := h.chatService.IsParticipant(chatID, userID)
	if err != nil || !isParticipant {
		c.JSON(403, gin.H{"error": "Access denied"})
		return
	}

	var req struct {
		UserID string `json:"userId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	if err := h.chatService.AddParticipant(chatID, req.UserID, "member"); err != nil {
		c.JSON(400, gin.H{"error": "Failed to add participant"})
		return
	}

	c.JSON(202, gin.H{
		"message":       "Participant added; clients must re-key before sending new encrypted messages",
		"rekeyRequired": true,
	})
}

func (h *ChatHandler) RemoveParticipant(c *gin.Context) {
	userID := c.GetString("userID")
	chatID := c.Param("chatId")
	targetUserID := c.Param("userId")

	// Check if user is a participant
	isParticipant, err := h.chatService.IsParticipant(chatID, userID)
	if err != nil || !isParticipant {
		c.JSON(403, gin.H{"error": "Access denied"})
		return
	}

	if err := h.chatService.RemoveParticipant(chatID, targetUserID); err != nil {
		c.JSON(400, gin.H{"error": "Failed to remove participant"})
		return
	}

	c.JSON(202, gin.H{
		"message":       "Participant removed; clients must re-key before sending new encrypted messages",
		"rekeyRequired": true,
	})
}

func (h *ChatHandler) LeaveChat(c *gin.Context) {
	userID := c.GetString("userID")
	chatID := c.Param("chatId")

	if err := h.chatService.RemoveParticipant(chatID, userID); err != nil {
		c.JSON(400, gin.H{"error": "Failed to leave chat"})
		return
	}

	c.JSON(204, nil)
}
