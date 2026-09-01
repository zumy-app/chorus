package handlers

import (
	"context"
	"log"
	"net/http"

	"github.com/chorus/messenger/internal/middleware"
	"github.com/chorus/messenger/internal/models"
	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
)

// LocationHandler serves the location-sharing surface (task 6.7): a validated
// lat/lng pair (plus an optional label) creates a message backed by a
// media_attachments row of type 'location', then fans the message out over
// WebSockets — with per-user receipts and offline-inbox queueing — exactly like
// a text or attachment message. The map pin is rendered by the client from the
// lat/lng; the attachment URL carries a keyless map embed link.
type LocationHandler struct {
	locationService *services.LocationService
	chatService     *services.ChatService
	messageService  *services.MessageService
	inboxService    *services.InboxService
	wsHub           *services.WebSocketHub
}

// NewLocationHandler creates a LocationHandler.
func NewLocationHandler(
	locationService *services.LocationService,
	chatService *services.ChatService,
	messageService *services.MessageService,
	inboxService *services.InboxService,
	wsHub *services.WebSocketHub,
) *LocationHandler {
	return &LocationHandler{
		locationService: locationService,
		chatService:     chatService,
		messageService:  messageService,
		inboxService:    inboxService,
		wsHub:           wsHub,
	}
}

// SendLocation shares a location pin into a chat (task 6.7).
//
// Request JSON:
//
//	latitude   (required) [-90, 90]
//	longitude  (required) [-180, 180]
//	label      (optional) a human-readable place name
//	replyToId  (optional) reply to an existing message
//
// Response: 201 with the created message (media[0] populated with the pin).
// POST /api/v1/chats/:chatId/location
func (h *LocationHandler) SendLocation(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		WriteError(c, middleware.ErrAuth("Unauthorized"))
		return
	}

	chatID := c.Param("chatId")
	if chatID == "" {
		WriteError(c, middleware.ErrValidation("Chat ID required"))
		return
	}

	// Only participants may share a location into a chat.
	isParticipant, err := h.chatService.IsParticipant(chatID, userID)
	if err != nil || !isParticipant {
		WriteError(c, middleware.ErrForbidden("Access denied"))
		return
	}

	var req models.SendLocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteError(c, middleware.ErrValidation("Invalid request"))
		return
	}

	message, err := h.locationService.SendLocation(
		c.Request.Context(), chatID, userID, req.Latitude, req.Longitude, req.Label, req.ReplyToID,
	)
	if err != nil {
		log.Printf("[Location] send to chat %s by %s failed: %v", chatID, userID, err)
		WriteError(c, middleware.ErrValidation("Failed to share location: "+err.Error()))
		return
	}

	h.broadcastLocationMessage(c.Request.Context(), message, chatID)

	c.JSON(http.StatusCreated, message)
}

// broadcastLocationMessage fans a freshly-created location message out to a
// chat's participants: per-recipient 'sent' receipts, offline-inbox queueing,
// and the real-time new_message event (mirrors SendMessage's fan-out).
func (h *LocationHandler) broadcastLocationMessage(ctx context.Context, message *models.Message, chatID string) {
	if message == nil {
		return
	}

	participants, _ := h.chatService.GetParticipants(chatID)
	userIDs := make([]string, 0, len(participants))
	for _, p := range participants {
		userIDs = append(userIDs, p.UserID)
	}

	if h.messageService != nil {
		_ = h.messageService.InitializeReceipts(ctx, message, userIDs)
	}
	if h.inboxService != nil {
		_ = h.inboxService.QueueMessageForOfflineClients(message, userIDs)
	}
	h.wsHub.SendToChat(chatID, userIDs, "new_message", message)
}
