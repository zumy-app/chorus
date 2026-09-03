package handlers

import (
	"context"

	"github.com/chorus/messenger/internal/middleware"
	"github.com/chorus/messenger/internal/models"
	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
)

type ChatHandler struct {
	chatService    *services.ChatService
	userService    *services.UserService
	moderation     *services.ModerationService
	privacyService *services.PrivacyService
	wsHub          *services.WebSocketHub
}

func NewChatHandler(chatService *services.ChatService, userService *services.UserService, moderation *services.ModerationService, wsHub *services.WebSocketHub) *ChatHandler {
	return &ChatHandler{
		chatService: chatService,
		userService: userService,
		moderation:  moderation,
		wsHub:       wsHub,
	}
}

func (h *ChatHandler) SetPrivacyService(p *services.PrivacyService) { h.privacyService = p }

// hydrateParticipants hydrates each participant's profile (via GetMultiple) and
// stamps viewer-relative block status so the chat surfaces render
// Block/Unblock everywhere a user appears (task 7.1).
func (h *ChatHandler) hydrateParticipants(ctx context.Context, viewerID string, participants []models.ChatParticipant) {
	userIDs := make([]string, 0, len(participants))
	for _, p := range participants {
		if p.UserID != "" {
			userIDs = append(userIDs, p.UserID)
		}
	}
	users, err := h.userService.GetMultiple(userIDs)
	if err != nil {
		return
	}
	enriched := make([]*models.User, 0, len(users))
	for i := range participants {
		if user, ok := users[participants[i].UserID]; ok {
			participants[i].User = user
			enriched = append(enriched, user)
		}
	}
	if h.moderation != nil && len(enriched) > 0 {
		_ = h.moderation.EnrichUsers(ctx, viewerID, enriched)
	}
	if h.privacyService != nil {
		for _, u := range enriched {
			h.privacyService.FilterUser(viewerID, u)
		}
	}
}

func (h *ChatHandler) GetUserChats(c *gin.Context) {
	userID := c.GetString("userID")

	chats, err := h.chatService.GetUserChats(userID)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to fetch chats"))
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
		enriched := make([]*models.User, 0, len(participants))
		for j := range participants {
			if user, ok := users[participants[j].UserID]; ok {
				participants[j].User = user
				enriched = append(enriched, user)
			}
		}
		if h.moderation != nil && len(enriched) > 0 {
			_ = h.moderation.EnrichUsers(c.Request.Context(), userID, enriched)
		}
		if h.privacyService != nil {
			for _, u := range enriched {
				h.privacyService.FilterUser(userID, u)
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
		WriteError(c, middleware.ErrValidation("Invalid request"))
		return
	}

	// Safety rail: a blocked user cannot be added to a new chat.
	if h.moderation != nil && req.Type == "direct" {
		for _, p := range req.Participants {
			if p == userID {
				continue
			}
			blocked, err := h.moderation.IsBlocked(c.Request.Context(), userID, p)
			if err != nil {
				WriteError(c, middleware.ErrInternal("Failed to check blocked users"))
				return
			}
			if blocked {
				WriteError(c, middleware.ErrForbidden("You cannot start a chat with this user"))
				return
			}
		}
	}

	chat, err := h.chatService.Create(userID, req)
	if err != nil {
		WriteError(c, middleware.ErrValidation("Failed to create chat"))
		return
	}

	// Get participants with user details
	participants, _ := h.chatService.GetParticipants(chat.ID)
	userIDs := make([]string, len(participants))
	for i, p := range participants {
		userIDs[i] = p.UserID
	}
	users, _ := h.userService.GetMultiple(userIDs)
	enriched := make([]*models.User, 0, len(participants))
	for i := range participants {
		if user, ok := users[participants[i].UserID]; ok {
			participants[i].User = user
			enriched = append(enriched, user)
		}
	}
	if h.moderation != nil && len(enriched) > 0 {
		_ = h.moderation.EnrichUsers(c.Request.Context(), userID, enriched)
	}
	if h.privacyService != nil {
		for _, u := range enriched {
			h.privacyService.FilterUser(userID, u)
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
		WriteError(c, middleware.ErrForbidden("Access denied"))
		return
	}

	chat, err := h.chatService.GetByID(chatID)
	if err != nil {
		WriteError(c, middleware.ErrNotFound("Chat not found"))
		return
	}

	// Get participants with user details
	participants, _ := h.chatService.GetParticipants(chatID)
	userIDs := make([]string, len(participants))
	for i, p := range participants {
		userIDs[i] = p.UserID
	}
	users, _ := h.userService.GetMultiple(userIDs)
	enriched := make([]*models.User, 0, len(participants))
	for i := range participants {
		if user, ok := users[participants[i].UserID]; ok {
			participants[i].User = user
			enriched = append(enriched, user)
		}
	}
	if h.moderation != nil && len(enriched) > 0 {
		_ = h.moderation.EnrichUsers(c.Request.Context(), userID, enriched)
	}
	if h.privacyService != nil {
		for _, u := range enriched {
			h.privacyService.FilterUser(userID, u)
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
		WriteError(c, middleware.ErrForbidden("Access denied"))
		return
	}

	var req struct {
		Name     string                 `json:"name"`
		Settings map[string]interface{} `json:"settings"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		WriteError(c, middleware.ErrValidation("Invalid request"))
		return
	}

	var name *string
	if req.Name != "" {
		name = &req.Name
	}

	chat, err := h.chatService.UpdateChat(chatID, name, req.Settings)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to update chat"))
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
		WriteError(c, middleware.ErrForbidden("Access denied"))
		return
	}

	var req struct {
		UserID string `json:"userId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		WriteError(c, middleware.ErrValidation("Invalid request"))
		return
	}

	// Safety rail: never add a user who has a block relationship with an
	// existing participant or the actor.
	if h.moderation != nil {
		participants, err := h.chatService.GetParticipants(chatID)
		if err != nil {
			WriteError(c, middleware.ErrInternal("Failed to load participants"))
			return
		}
		others := []string{userID}
		for _, p := range participants {
			others = append(others, p.UserID)
		}
		for _, other := range others {
			blocked, err := h.moderation.IsBlocked(c.Request.Context(), req.UserID, other)
			if err != nil {
				WriteError(c, middleware.ErrInternal("Failed to check blocked users"))
				return
			}
			if blocked {
				WriteError(c, middleware.ErrForbidden("You cannot add this user"))
				return
			}
		}
	}

	if err := h.chatService.AddParticipant(chatID, req.UserID, "member"); err != nil {
		WriteError(c, middleware.ErrValidation("Failed to add participant"))
		return
	}

	c.JSON(201, gin.H{"message": "Participant added successfully"})
}

func (h *ChatHandler) RemoveParticipant(c *gin.Context) {
	userID := c.GetString("userID")
	chatID := c.Param("chatId")
	targetUserID := c.Param("userId")

	// Check if user is a participant
	isParticipant, err := h.chatService.IsParticipant(chatID, userID)
	if err != nil || !isParticipant {
		WriteError(c, middleware.ErrForbidden("Access denied"))
		return
	}

	if err := h.chatService.RemoveParticipant(chatID, targetUserID); err != nil {
		WriteError(c, middleware.ErrValidation("Failed to remove participant"))
		return
	}

	c.JSON(204, nil)
}

func (h *ChatHandler) LeaveChat(c *gin.Context) {
	userID := c.GetString("userID")
	chatID := c.Param("chatId")

	if err := h.chatService.RemoveParticipant(chatID, userID); err != nil {
		WriteError(c, middleware.ErrValidation("Failed to leave chat"))
		return
	}

	c.JSON(204, nil)
}

// ---------------------------------------------------------------------------
// Archive & mute (task 6.4): per-user, per-chat conversation preferences.
// ---------------------------------------------------------------------------

// ArchiveChat archives (or, with archived=false, unarchives) a conversation for
// the calling user. The caller must be a participant. The preference is
// strictly user-scoped, so no co-participants are notified.
func (h *ChatHandler) ArchiveChat(c *gin.Context) {
	userID := c.GetString("userID")
	chatID := c.Param("chatId")

	isParticipant, err := h.chatService.IsParticipant(chatID, userID)
	if err != nil || !isParticipant {
		WriteError(c, middleware.ErrForbidden("Access denied"))
		return
	}

	var req models.ArchiveChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteError(c, middleware.ErrValidation("Invalid request"))
		return
	}

	// Default to archiving when the flag is omitted.
	archived := true
	if req.Archived != nil {
		archived = *req.Archived
	}

	pref, err := h.chatService.ArchiveChat(c.Request.Context(), userID, chatID, archived)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to update archive state"))
		return
	}

	h.wsHub.SendToUser(userID, "chat_preferences_updated", pref)

	c.JSON(200, pref)
}

// UnarchiveChat unarchives a conversation for the calling user (task 6.4).
func (h *ChatHandler) UnarchiveChat(c *gin.Context) {
	userID := c.GetString("userID")
	chatID := c.Param("chatId")

	isParticipant, err := h.chatService.IsParticipant(chatID, userID)
	if err != nil || !isParticipant {
		WriteError(c, middleware.ErrForbidden("Access denied"))
		return
	}

	pref, err := h.chatService.ArchiveChat(c.Request.Context(), userID, chatID, false)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to update archive state"))
		return
	}

	h.wsHub.SendToUser(userID, "chat_preferences_updated", pref)

	c.JSON(200, pref)
}

// MuteChat mutes (or, with muted=false, unmutes) a conversation for the calling
// user (task 6.4). Until, when set with muted=true, makes the mute timed; an
// omitted until with muted=true mutes indefinitely.
func (h *ChatHandler) MuteChat(c *gin.Context) {
	userID := c.GetString("userID")
	chatID := c.Param("chatId")

	isParticipant, err := h.chatService.IsParticipant(chatID, userID)
	if err != nil || !isParticipant {
		WriteError(c, middleware.ErrForbidden("Access denied"))
		return
	}

	var req models.MuteChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteError(c, middleware.ErrValidation("Invalid request"))
		return
	}

	// Default to muting when the flag is omitted.
	muted := true
	if req.Muted != nil {
		muted = *req.Muted
	}

	pref, err := h.chatService.MuteChat(c.Request.Context(), userID, chatID, muted, req.Until)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to update mute state"))
		return
	}

	h.wsHub.SendToUser(userID, "chat_preferences_updated", pref)

	c.JSON(200, pref)
}

// UnmuteChat unmutes a conversation for the calling user (task 6.4).
func (h *ChatHandler) UnmuteChat(c *gin.Context) {
	userID := c.GetString("userID")
	chatID := c.Param("chatId")

	isParticipant, err := h.chatService.IsParticipant(chatID, userID)
	if err != nil || !isParticipant {
		WriteError(c, middleware.ErrForbidden("Access denied"))
		return
	}

	pref, err := h.chatService.MuteChat(c.Request.Context(), userID, chatID, false, nil)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to update mute state"))
		return
	}

	h.wsHub.SendToUser(userID, "chat_preferences_updated", pref)

	c.JSON(200, pref)
}

// GetChatPreferences returns the calling user's per-chat preferences (archive &
// mute state, task 6.4) across all of their conversations, keyed by chat ID.
func (h *ChatHandler) GetChatPreferences(c *gin.Context) {
	userID := c.GetString("userID")

	prefs, err := h.chatService.GetChatPreferences(c.Request.Context(), userID)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to fetch chat preferences"))
		return
	}

	c.JSON(200, gin.H{
		"preferences": prefs,
	})
}

// GetChatPreference returns the calling user's preference row for a single chat
// (task 6.4). It always returns a default (unarchived, unmuted) preference when
// the user has never touched this chat.
func (h *ChatHandler) GetChatPreference(c *gin.Context) {
	userID := c.GetString("userID")
	chatID := c.Param("chatId")

	isParticipant, err := h.chatService.IsParticipant(chatID, userID)
	if err != nil || !isParticipant {
		WriteError(c, middleware.ErrForbidden("Access denied"))
		return
	}

	pref, err := h.chatService.GetChatPreference(c.Request.Context(), userID, chatID)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to fetch chat preference"))
		return
	}

	c.JSON(200, pref)
}
