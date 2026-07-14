package handlers

import (
	"log"
	"time"

	"github.com/chorus/messenger/internal/models"
	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
)

type MessageHandler struct {
	messageService     *services.MessageService
	chatService        *services.ChatService
	translationService *services.TranslationService
	wsHub              *services.WebSocketHub
}

func NewMessageHandler(
	messageService *services.MessageService,
	chatService *services.ChatService,
	translationService *services.TranslationService,
	wsHub *services.WebSocketHub,
) *MessageHandler {
	return &MessageHandler{
		messageService:     messageService,
		chatService:        chatService,
		translationService: translationService,
		wsHub:              wsHub,
	}
}

func (h *MessageHandler) GetMessages(c *gin.Context) {
	userID := c.GetString("userID")
	chatID := c.Param("chatId")

	// Check if user is a participant
	isParticipant, err := h.chatService.IsParticipant(chatID, userID)
	if err != nil || !isParticipant {
		c.JSON(403, gin.H{"error": "Access denied"})
		return
	}

	limit := 50
	if l := c.Query("limit"); l != "" {
		// Parse limit if provided
	}

	var before *string
	if b := c.Query("before"); b != "" {
		before = &b
	}

	messages, err := h.messageService.GetMessages(chatID, limit, before)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to fetch messages"})
		return
	}

	c.JSON(200, gin.H{
		"messages": messages,
		"hasMore":  len(messages) >= limit,
	})
}

func (h *MessageHandler) SendMessage(c *gin.Context) {
	userID := c.GetString("userID")
	chatID := c.Param("chatId")

	// Check if user is a participant
	isParticipant, err := h.chatService.IsParticipant(chatID, userID)
	if err != nil || !isParticipant {
		c.JSON(403, gin.H{"error": "Access denied"})
		return
	}

	var req models.SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Create message
	message, err := h.messageService.Create(chatID, userID, req.Text, req.ReplyToID)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to send message"})
		return
	}

	// Detect and store the original language based on sender's native language
	participantLangs, err := h.chatService.GetParticipantLanguages(chatID)
	if err == nil {
		if senderLangs, ok := participantLangs[userID]; ok && len(senderLangs) > 0 {
			nativeLang := senderLangs[0]
			if nativeLang != "" {
				// Update the message's original language in DB
				h.messageService.UpdateOriginalLanguage(message.ID, nativeLang)
				message.OriginalLanguage = nativeLang
			}
		}
	}

	// Get participant languages for translation
	participantLangs, err = h.chatService.GetParticipantLanguages(chatID)
	if err == nil && len(participantLangs) > 0 {
		targetLangs := make(map[string]bool)
		for _, langs := range participantLangs {
			for _, lang := range langs {
				targetLangs[lang] = true
			}
		}

		// Phase 1+2: Translate asynchronously (translator-engine fast, then Ollama enhancement)
		go h.translateAndBroadcast(message, targetLangs, chatID)
	}

	// Broadcast new message to ALL chat participants (including sender for multi-device)
	participants, _ := h.chatService.GetParticipants(chatID)
	userIDs := make([]string, 0, len(participants))
	for _, p := range participants {
		userIDs = append(userIDs, p.UserID)
	}

	h.wsHub.SendToChat(chatID, userIDs, "new_message", message)

	c.JSON(201, message)
}

func (h *MessageHandler) translateAndBroadcast(message *models.Message, targetLangs map[string]bool, chatID string) {
	langs := make([]string, 0, len(targetLangs))
	for lang := range targetLangs {
		langs = append(langs, lang)
	}

	quickTranslations := make(map[string]string)
	for _, lang := range langs {
		trans, err := h.translationService.TranslateQuick(message.Text, lang, "auto")
		if err != nil {
			log.Printf("[Translate] fast translation failed for lang %s: %v", lang, err)
		} else if trans != "" {
			quickTranslations[lang] = trans
		}
	}
	if len(quickTranslations) > 0 {
		h.messageService.UpdateTranslations(message.ID, quickTranslations)
		message.Translations = quickTranslations
		message.TranslationEnhanced = false

		participants, _ := h.chatService.GetParticipants(chatID)
		userIDs := make([]string, 0, len(participants))
		for _, p := range participants {
			userIDs = append(userIDs, p.UserID)
		}
		h.wsHub.SendToChat(chatID, userIDs, "message_updated", message)
	}

	go func(msg *models.Message, targetLanguages []string, roomID string) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("background tutor analysis panicked: %v", rec)
			}
		}()
		if len(targetLanguages) == 0 {
			return
		}
		time.Sleep(50 * time.Millisecond)
		backgroundTranslations := make(map[string]string)
		for _, lang := range targetLanguages {
			translated, err := h.translationService.TranslateWithOllama(msg.Text, lang)
			if err != nil {
				continue
			}
			backgroundTranslations[lang] = translated
		}
		if len(backgroundTranslations) == 0 {
			return
		}
		if err := h.messageService.UpdateTranslations(msg.ID, backgroundTranslations); err != nil {
			log.Printf("background translation update failed: %v", err)
			return
		}
		participants, _ := h.chatService.GetParticipants(roomID)
		userIDs := make([]string, 0, len(participants))
		for _, p := range participants {
			userIDs = append(userIDs, p.UserID)
		}
		msg.Translations = backgroundTranslations
		msg.TranslationEnhanced = true
		h.wsHub.SendToChat(roomID, userIDs, "message_updated", msg)
	}(message, langs, chatID)
}

func (h *MessageHandler) MarkAsRead(c *gin.Context) {
	userID := c.GetString("userID")
	chatID := c.Param("chatId")

	// Check if user is a participant
	isParticipant, err := h.chatService.IsParticipant(chatID, userID)
	if err != nil || !isParticipant {
		c.JSON(403, gin.H{"error": "Access denied"})
		return
	}

	var req struct {
		MessageID string `json:"messageId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	if err := h.messageService.MarkAsRead(chatID, userID, req.MessageID); err != nil {
		c.JSON(500, gin.H{"error": "Failed to mark as read"})
		return
	}

	c.JSON(204, nil)
}

func (h *MessageHandler) SearchMessages(c *gin.Context) {
	userID := c.GetString("userID")
	query := c.Query("q")
	chatID := c.Query("chatId")

	if query == "" {
		c.JSON(400, gin.H{"error": "Query parameter 'q' is required"})
		return
	}

	// If chatID is provided, verify user is a participant
	if chatID != "" {
		isParticipant, err := h.chatService.IsParticipant(chatID, userID)
		if err != nil || !isParticipant {
			c.JSON(403, gin.H{"error": "Access denied"})
			return
		}
	}

	limit := 20
	var chatIDPtr *string
	if chatID != "" {
		chatIDPtr = &chatID
	}

	messages, err := h.messageService.Search(query, chatIDPtr, limit)
	if err != nil {
		c.JSON(500, gin.H{"error": "Search failed"})
		return
	}

	c.JSON(200, gin.H{
		"messages": messages,
		"total":    len(messages),
		"hasMore":  len(messages) >= limit,
	})
}
