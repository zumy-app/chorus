package handlers

import (
	"net/http"
	"strconv"

	"github.com/chorus/messenger/internal/middleware"
	"github.com/chorus/messenger/internal/models"
	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
)

// GalleryHandler serves the chat-scoped media gallery (task 6.5).
type GalleryHandler struct {
	galleryService *services.GalleryService
	moderation     *services.ModerationService
}

func NewGalleryHandler(gs *services.GalleryService) *GalleryHandler {
	return &GalleryHandler{galleryService: gs}
}

func NewGalleryHandlerWithModeration(gs *services.GalleryService, moderation *services.ModerationService) *GalleryHandler {
	return &GalleryHandler{galleryService: gs, moderation: moderation}
}

// GetChatGallery returns the media gallery for a chat (task 6.5): the photos,
// videos, audio, documents and links shared in a specific chat. An optional
// `type` query param filters to a tab (media / docs / links) or a comma-separated
// list of concrete types; `limit`/`offset` paginate and the response includes
// per-type counts for tab badges.
// GET /api/v1/chats/:chatId/gallery
func (h *GalleryHandler) GetChatGallery(c *gin.Context) {
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

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "30"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	typeFilter := c.Query("type")

	result, err := h.galleryService.GetChatGallery(c.Request.Context(), userID, chatID, typeFilter, limit, offset)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to load gallery"))
		return
	}
	if h.moderation != nil && len(result.Items) > 0 {
		senders := make([]*models.User, 0, len(result.Items))
		for i := range result.Items {
			if result.Items[i].Sender != nil && result.Items[i].Sender.ID != userID {
				senders = append(senders, result.Items[i].Sender)
			}
		}
		if len(senders) > 0 {
			_ = h.moderation.EnrichUsers(c.Request.Context(), userID, senders)
		}
	}

	c.JSON(http.StatusOK, result)
}
