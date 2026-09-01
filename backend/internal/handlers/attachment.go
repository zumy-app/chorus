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

// AttachmentHandler serves the file / document sharing surface (task 6.6): a
// multipart upload creates a persistent message with a media_attachments row,
// stores the bytes, and fans the message out over WebSockets — with per-user
// receipts and offline-inbox queueing — exactly like a text message. The file
// can then be reached through the media gallery, media search, and its public
// URL, so a shared PDF/doc/xlsx is fully first-class.
type AttachmentHandler struct {
	attachmentService *services.AttachmentService
	chatService       *services.ChatService
	messageService    *services.MessageService
	inboxService      *services.InboxService
	wsHub             *services.WebSocketHub
	router            *services.DeliveryRouter
}

// NewAttachmentHandler creates an AttachmentHandler.
func NewAttachmentHandler(
	attachmentService *services.AttachmentService,
	chatService *services.ChatService,
	messageService *services.MessageService,
	inboxService *services.InboxService,
	wsHub *services.WebSocketHub,
) *AttachmentHandler {
	return &AttachmentHandler{
		attachmentService: attachmentService,
		chatService:       chatService,
		messageService:    messageService,
		inboxService:      inboxService,
		wsHub:             wsHub,
	}
}

func (h *AttachmentHandler) SetRouter(r *services.DeliveryRouter) {
	h.router = r
}

// SendAttachment accepts a multipart/file upload and creates a media message.
// Form fields:
//
//	file        (required) the uploaded bytes
//	type        (optional) explicit media type: image|video|audio|document
//	caption     (optional) a text caption; the message text falls back to the
//	            file name when omitted
//
// Response: 201 with the created message (media[0] populated).
// POST /api/v1/chats/:chatId/attachments
func (h *AttachmentHandler) SendAttachment(c *gin.Context) {
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

	// Only participants may share into a chat.
	isParticipant, err := h.chatService.IsParticipant(chatID, userID)
	if err != nil || !isParticipant {
		WriteError(c, middleware.ErrForbidden("Access denied"))
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		WriteError(c, middleware.ErrValidation("File field 'file' is required"))
		return
	}
	defer file.Close()

	if header.Size <= 0 {
		WriteError(c, middleware.ErrValidation("Uploaded file is empty"))
		return
	}
	if max := h.attachmentService.MaxBytes(); header.Size > max {
		WriteError(c, middleware.ErrTooLarge("File exceeds the upload limit"))
		return
	}

	mediaType := c.PostForm("type")
	caption := c.PostForm("caption")

	message, err := h.attachmentService.SendFile(
		c.Request.Context(), chatID, userID, caption, header.Filename, mediaType, file, header.Size,
	)
	if err != nil {
		log.Printf("[Attachment] send to chat %s by %s failed: %v", chatID, userID, err)
		WriteError(c, middleware.ErrValidation("Failed to process upload: "+err.Error()))
		return
	}

	h.broadcastMediaMessage(c.Request.Context(), message, chatID)

	c.JSON(http.StatusCreated, message)
}

// broadcastMediaMessage fans a freshly-created media message out to a chat's
// participants: per-recipient 'sent' receipts, offline-inbox queueing, and the
// real-time new_message event (mirrors SendMessage's fan-out).
func (h *AttachmentHandler) broadcastMediaMessage(ctx context.Context, message *models.Message, chatID string) {
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
	if h.router != nil {
		h.router.RouteMessage(ctx, message, userIDs)
	} else {
		h.wsHub.SendToChat(chatID, userIDs, "new_message", message)
	}
}
