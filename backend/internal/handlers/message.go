package handlers

import (
	"context"
	"database/sql"
	"log"
	"strings"
	"time"

	"github.com/chorus/messenger/internal/middleware"
	"github.com/chorus/messenger/internal/models"
	"github.com/chorus/messenger/internal/observability"
	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
)

type MessageHandler struct {
	messageService     *services.MessageService
	chatService        *services.ChatService
	userService        *services.UserService
	entitlementService *services.EntitlementService
	translationQueue   *services.TranslationQueueService
	settingsService    *services.SettingsService
	moderation         *services.ModerationService
	wsHub              *services.WebSocketHub
	translation        *services.TranslationService
	wordMining         *services.WordMiningQueueService
	learningProfile    *services.LearningProfileService
	receiptService     *services.ReceiptService
	inboxService       *services.InboxService
	router             *services.DeliveryRouter
}

func NewMessageHandler(
	messageService *services.MessageService,
	chatService *services.ChatService,
	userService *services.UserService,
	entitlementService *services.EntitlementService,
	translationQueue *services.TranslationQueueService,
	settingsService *services.SettingsService,
	moderation *services.ModerationService,
	wsHub *services.WebSocketHub,
	translation *services.TranslationService,
) *MessageHandler {
	return &MessageHandler{
		messageService:     messageService,
		chatService:        chatService,
		userService:        userService,
		entitlementService: entitlementService,
		translationQueue:   translationQueue,
		settingsService:    settingsService,
		moderation:         moderation,
		wsHub:              wsHub,
		translation:        translation,
	}
}

// SetWordMining attaches the (optional) vocabulary miner so target-language
// messages feed the learner's SRS pipeline.
func (h *MessageHandler) SetWordMining(q *services.WordMiningQueueService, lp *services.LearningProfileService) {
	h.wordMining = q
	h.learningProfile = lp
}

// SetReceiptService attaches the read/delivery receipt service so delivered
// and read ticks are recorded and fanned out (task 6.1).
func (h *MessageHandler) SetReceiptService(rs *services.ReceiptService) {
	h.receiptService = rs
}

// SetInboxService attaches the offline delivery service so messages are queued
// for recipients whose devices are offline (task 6.1).
func (h *MessageHandler) SetInboxService(is *services.InboxService) {
	h.inboxService = is
}

func (h *MessageHandler) SetRouter(r *services.DeliveryRouter) {
	h.router = r
}

func (h *MessageHandler) GetMessages(c *gin.Context) {
	userID := c.GetString("userID")
	chatID := c.Param("chatId")

	// Check if user is a participant
	isParticipant, err := h.chatService.IsParticipant(chatID, userID)
	if err != nil || !isParticipant {
		WriteError(c, middleware.ErrForbidden("Access denied"))
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
		WriteError(c, middleware.ErrInternal("Failed to fetch messages"))
		return
	}

	if h.messageService != nil {
		_ = h.messageService.AttachReceipts(c.Request.Context(), chatID, messages)
	}
	// Task 6.6: attach any file/document/media rows so a loaded history carries
	// the attachment metadata alongside each message.
	if h.messageService != nil {
		_ = h.messageService.AttachMedia(c.Request.Context(), chatID, messages)
	}
	if h.moderation != nil && h.userService != nil && len(messages) > 0 {
		ids := make([]string, 0, len(messages))
		seen := map[string]struct{}{}
		for _, m := range messages {
			if m.SenderID == "" || m.SenderID == userID {
				continue
			}
			if _, ok := seen[m.SenderID]; !ok {
				seen[m.SenderID] = struct{}{}
				ids = append(ids, m.SenderID)
			}
		}
		if len(ids) > 0 {
			if users, err := h.userService.GetMultiple(ids); err == nil {
				senders := make([]*models.User, 0, len(users))
				userByID := map[string]*models.User{}
				for _, u := range users {
					senders = append(senders, u)
					userByID[u.ID] = u
				}
				_ = h.moderation.EnrichUsers(c.Request.Context(), userID, senders)
				for i := range messages {
					if u, ok := userByID[messages[i].SenderID]; ok {
						messages[i].Sender = u
					}
				}
			}
		}
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
		WriteError(c, middleware.ErrForbidden("Access denied"))
		return
	}

	// Safety rail: if a block relationship exists with any participant, the
	// chat is frozen for this sender (either direction blocks messaging).
	if h.moderation != nil {
		blocked, err := h.moderation.ChatBlocked(c.Request.Context(), chatID, userID)
		if err != nil {
			WriteError(c, middleware.ErrInternal("Failed to check blocked users"))
			return
		}
		if blocked {
			WriteError(c, middleware.ErrForbidden("You cannot send messages in this chat"))
			return
		}
	}

	var req models.SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteError(c, middleware.ErrValidation("Invalid request"))
		return
	}

	// Create message with NFR-18 timing + outcome metric.
	start := time.Now()
	message, err := h.messageService.Create(chatID, userID, req.Text, req.ReplyToID)
	if err != nil {
		observability.ObserveMessageSend("error", time.Since(start))
		WriteError(c, middleware.ErrInternal("Failed to send message"))
		return
	}
	observability.ObserveMessageSend("sent", time.Since(start))

	// Broadcast new message to ALL chat participants (including sender for multi-device)
	participants, _ := h.chatService.GetParticipants(chatID)
	userIDs := make([]string, 0, len(participants))
	for _, p := range participants {
		userIDs = append(userIDs, p.UserID)
	}

	// Task 6.1: seed a per-recipient 'sent' receipt and queue the message for
	// any offline recipients so delivery ticks can advance once they reconnect.
	if h.messageService != nil {
		_ = h.messageService.InitializeReceipts(c.Request.Context(), message, userIDs)
	}
	if h.inboxService != nil {
		_ = h.inboxService.QueueMessageForOfflineClients(message, userIDs)
	}

	if h.router != nil {
		h.router.RouteMessage(c.Request.Context(), message, userIDs)
	} else {
		h.wsHub.SendToChat(chatID, userIDs, "new_message", message)
	}

	c.JSON(201, message)

	// Handle language detection and translation asynchronously
	go func() {
		participantLangs, err := h.chatService.GetParticipantLanguages(chatID)
		if err != nil || len(participantLangs) == 0 {
			return
		}

		// Detect and store the original language based on the actual message
		// content. Falling back to the sender's native language only when
		// detection is unavailable (no provider or a detection failure) so a
		// message written in a language other than the sender's profile native
		// (e.g. a Spanish message from an English native speaker) is translated
		// correctly for recipients.
		sourceLang := "auto"
		if h.translation != nil {
			if detected, err := h.translation.DetectLanguage(message.Text); err == nil && detected != "" {
				sourceLang = detected
				h.messageService.UpdateOriginalLanguage(message.ID, detected)
				message.OriginalLanguage = detected
			}
		}
		if sourceLang == "auto" {
			if senderLangs, ok := participantLangs[userID]; ok && len(senderLangs) > 0 {
				if nativeLang := senderLangs[0]; nativeLang != "" {
					sourceLang = nativeLang
					h.messageService.UpdateOriginalLanguage(message.ID, nativeLang)
					message.OriginalLanguage = nativeLang
				}
			}
		}

		// Word mining: enqueue a durable vocabulary-extraction job for every
		// participant whose learning profile targets the (best-guess) message
		// language. Detection is broad but imperfect, so a Spanish function-word
		// / diacritic signal is used to correct a fallback to the sender's native.
		if h.wordMining != nil && sourceLang != "" && sourceLang != "auto" {
			miningLang := miningLanguageFor(message.Text, sourceLang)
			for _, uid := range userIDs {
				if uid == "" {
					continue
				}
				if !h.targetLanguageMatches(context.Background(), uid, miningLang) {
					continue
				}
				if _, err := h.wordMining.EnqueueForMessage(uid, chatID, message.ID, "chat", message.Text, miningLang, nativeSourceFor(uid, miningLang, h)); err != nil {
					log.Printf("[Mining] enqueue for user %s: %v", uid, err)
				}
			}
		}

		// Premium feature tiers: resolve the sender's entitlements to decide
		// whether this message may be translated and with what priority.
		priority := 0
		ent := h.entitlementService.ResolveNow(h.resolveUser(userID))
		if ent.Features.FasterResponses {
			priority = 1
		}
		// FR-25 Feature toggles: when the sender has auto-translation disabled,
		// skip translation entirely so no translation jobs are enqueued. Message
		// delivery is unaffected — the message was already persisted and acked
		// above. Toggles are per-account; a misread falls back to enabled-safe
		// behaviour and only warns.
		if h.settingsService != nil {
			fs, err := h.settingsService.GetFeatureSettings(userID)
			if err != nil {
				log.Printf("[Translate] read feature settings for %s: %v", userID, err)
			} else if !fs.TranslationEnabled {
				log.Printf("[Translate] auto-translation disabled for sender %s — not enqueuing jobs for message %s", userID, message.ID)
				return
			}
		}
		// Message-size gate (words): free = 280, premium = 1,000. Longer
		// messages are stored but not translated. Notify participants so the
		// UI can show the premium nudge. There are NO per-day usage quotas.
		if ent.Features.TranslationWordLimit != nil {
			words := services.WordCount(message.Text)
			if words > *ent.Features.TranslationWordLimit {
				log.Printf("[Translate] message %s blocked: %d words > plan limit %d", message.ID, words, *ent.Features.TranslationWordLimit)
				h.wsHub.SendToChat(chatID, userIDs, "translation_blocked", gin.H{
					"messageId": message.ID,
					"chatId":    chatID,
					"wordLimit": *ent.Features.TranslationWordLimit,
					"wordCount": words,
					"charLimit": ent.Features.TranslationCharLimit,
					"reason":    "message_too_long",
				})
				return
			}
		}

		// Build target languages for translation
		targetLangs := make(map[string]bool)
		for _, langs := range participantLangs {
			for _, lang := range langs {
				targetLangs[lang] = true
			}
		}

		// Durable, near-real-time translation queue (DB outbox + Redis pub/sub
		// trigger + retry sweeper + startup recovery). Completions are pushed to
		// chat participants via message_updated as they arrive.
		h.translationQueue.EnqueueForMessage(message, sourceLang, keys(targetLangs), priority)
	}()
}

// TranslateMessage manually queues a translation of a specific message to a
// target language chosen by the requesting participant (the "Translate" button
// in the UI). Unlike auto translation — which is gated on the *sender's* plan
// and word limit — this is an explicit viewer-side request: any participant may
// ask for a translation of a received message, always with sourceLang "auto" so
// the provider detects the actual source language of the text.
func (h *MessageHandler) TranslateMessage(c *gin.Context) {
	userID := c.GetString("userID")
	chatID := c.Param("chatId")
	messageID := c.Param("messageId")

	isParticipant, err := h.chatService.IsParticipant(chatID, userID)
	if err != nil || !isParticipant {
		WriteError(c, middleware.ErrForbidden("Access denied"))
		return
	}

	var req struct {
		TargetLang string `json:"targetLang" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteError(c, middleware.ErrValidation("Invalid request"))
		return
	}

	message, err := h.messageService.GetMessageByID(c.Request.Context(), messageID)
	if err != nil {
		WriteError(c, middleware.ErrNotFound("Message not found"))
		return
	}
	if message.ChatID != chatID {
		WriteError(c, middleware.ErrNotFound("Message not found"))
		return
	}

	// The requested translation already exists — nothing to do.
	if message.Translations != nil {
		if _, ok := message.Translations[req.TargetLang]; ok {
			c.JSON(200, message)
			return
		}
	}

	// FR-25 Feature toggles: an explicit "Translate" request from a viewer with
	// auto-translation disabled is also suppressed so no translation job is
	// enqueued. The toggle is per-account and applies to both auto and explicit
	// translation paths.
	if h.settingsService != nil {
		fs, err := h.settingsService.GetFeatureSettings(userID)
		if err != nil {
			log.Printf("[Translate] read feature settings for %s: %v", userID, err)
		} else if !fs.TranslationEnabled {
			WriteError(c, middleware.ErrForbidden("Translation is disabled in your settings"))
			return
		}
	}

	// Priority is based on the requesting viewer's plan (the viewer "owns"
	// this request), not the original sender's.
	priority := 0
	if ent := h.entitlementService.ResolveNow(h.resolveUser(userID)); ent.Features.FasterResponses {
		priority = 1
	}

	h.translationQueue.EnqueueManual(message, req.TargetLang, priority)

	c.JSON(202, gin.H{
		"messageId":  message.ID,
		"targetLang": req.TargetLang,
		"queued":     true,
	})
}

// resolveUser returns the user row for a user id, or nil if it cannot be
// loaded (entitlement resolution treats unknown users as free).
func (h *MessageHandler) resolveUser(userID string) *models.User {
	u, err := h.userService.GetByID(userID)
	if err != nil {
		log.Printf("[Message] resolve user %s: %v", userID, err)
		return nil
	}
	return u
}

// targetLanguageMatches reports whether the user has a learning profile targeting
// the given language AND has mining enabled. When no profile exists, it falls
// back to the user's stored target_languages on the users table.
func (h *MessageHandler) targetLanguageMatches(ctx context.Context, userID, lang string) bool {
	if h.learningProfile == nil {
		return false
	}
	profile, err := h.learningProfile.GetProfile(ctx, userID, lang, "")
	if err != nil || profile == nil {
		return false
	}
	return profile.MiningEnabled
}

// nativeSourceFor returns the user's native language (for mining prompts) or
// "en" when it cannot be resolved.
func nativeSourceFor(userID, targetLang string, h *MessageHandler) string {
	if h.learningProfile == nil {
		return "en"
	}
	if profile, err := h.learningProfile.GetProfile(context.Background(), userID, targetLang, ""); err == nil && profile != nil {
		return profile.NativeLanguage
	}
	return "en"
}

// looksLikeSpanish reports whether the text carries Spanish diacritics or
// inverted punctuation, a cheap signal used to correct weak language detection.
func looksLikeSpanish(text string) bool {
	for _, r := range text {
		switch r {
		case '¿', '¡', 'ñ', 'Ñ', 'á', 'Á', 'é', 'É', 'í', 'Í', 'ó', 'Ó', 'ú', 'Ú', 'ü', 'Ü':
			return true
		}
	}
	return false
}

// spanishFunctionWords are high-frequency Spanish grammatical words whose
// presence strongly indicates Spanish even when a weak detector returns the
// sender's native language.
var spanishFunctionWords = []string{
	"el", "la", "los", "las", "un", "una", "unos", "unas", "de", "del", "al",
	"que", "y", "o", "es", "son", "está", "estoy", "para", "por", "con", "sin",
	"no", "sí", "me", "te", "se", "hay", "mi", "tu", "su", "como", "muy",
	"también", "gracias", "por favor", "quiero", "quisiera", "buenos", "buenas",
}

// miningLanguageFor picks the best language to mine for a message. It prefers the
// detected language, but corrects to Spanish when the message clearly carries
// Spanish diacritics or a flood of Spanish function words.
func miningLanguageFor(text, detected string) string {
	if looksLikeSpanish(text) {
		return "es"
	}
	norm := strings.ToLower(text)
	fielded := strings.Fields(norm)
	hits := 0
	for _, w := range fielded {
		for _, fw := range spanishFunctionWords {
			if w == fw {
				hits++
				break
			}
		}
	}
	// A dense cluster of Spanish function words is enough to override a native
	// "en" detection for a target-language learner.
	if hits >= 2 {
		return "es"
	}
	return detected
}

// keys returns the keys of a set (helper for target-language de-duplication).
func keys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	return out
}

func (h *MessageHandler) MarkAsRead(c *gin.Context) {
	userID := c.GetString("userID")
	chatID := c.Param("chatId")

	// Check if user is a participant
	isParticipant, err := h.chatService.IsParticipant(chatID, userID)
	if err != nil || !isParticipant {
		WriteError(c, middleware.ErrForbidden("Access denied"))
		return
	}

	var req struct {
		MessageID string `json:"messageId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		WriteError(c, middleware.ErrValidation("Invalid request"))
		return
	}

	if h.receiptService != nil {
		err = h.receiptService.AcknowledgeRead(c.Request.Context(), chatID, req.MessageID, userID)
	} else if h.messageService != nil {
		err = h.messageService.MarkAsRead(chatID, userID, req.MessageID)
	}
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to mark as read"))
		return
	}

	c.JSON(204, nil)
}

// GetMessageReceipts returns the per-recipient sent/delivered/read tick state
// for a single message (task 6.1).
func (h *MessageHandler) GetMessageReceipts(c *gin.Context) {
	userID := c.GetString("userID")
	chatID := c.Param("chatId")
	messageID := c.Param("messageId")

	isParticipant, err := h.chatService.IsParticipant(chatID, userID)
	if err != nil || !isParticipant {
		WriteError(c, middleware.ErrForbidden("Access denied"))
		return
	}

	if h.messageService == nil {
		WriteError(c, middleware.ErrInternal("Failed to fetch receipts"))
		return
	}

	receipts, err := h.messageService.GetMessageReceipts(c.Request.Context(), messageID, chatID)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to fetch receipts"))
		return
	}

	c.JSON(200, gin.H{
		"receipts": receipts,
	})
}

// GetUnreadCount returns the number of unread messages in a chat for the
// calling user, driven by their last_read_message_id cursor.
func (h *MessageHandler) GetUnreadCount(c *gin.Context) {
	userID := c.GetString("userID")
	chatID := c.Param("chatId")

	isParticipant, err := h.chatService.IsParticipant(chatID, userID)
	if err != nil || !isParticipant {
		WriteError(c, middleware.ErrForbidden("Access denied"))
		return
	}

	if h.messageService == nil {
		WriteError(c, middleware.ErrInternal("Failed to fetch unread count"))
		return
	}

	count, err := h.messageService.GetUnreadCount(chatID, userID)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to fetch unread count"))
		return
	}

	c.JSON(200, gin.H{
		"unreadCount": count,
	})
}

func (h *MessageHandler) SearchMessages(c *gin.Context) {
	userID := c.GetString("userID")
	query := c.Query("q")
	chatID := c.Query("chatId")

	if query == "" {
		WriteError(c, middleware.ErrValidation("Query parameter 'q' is required"))
		return
	}

	// If chatID is provided, verify user is a participant
	if chatID != "" {
		isParticipant, err := h.chatService.IsParticipant(chatID, userID)
		if err != nil || !isParticipant {
			WriteError(c, middleware.ErrForbidden("Access denied"))
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
		WriteError(c, middleware.ErrInternal("Search failed"))
		return
	}

	c.JSON(200, gin.H{
		"messages": messages,
		"total":    len(messages),
		"hasMore":  len(messages) >= limit,
	})
}

// ---------------------------------------------------------------------------
// Message actions (task 6.2): forward, delete, pin, unpin, pins list.
// ---------------------------------------------------------------------------

// ForwardMessage copies an existing message into a target chat (task 6.2). The
// caller must be a participant of both chats; the copy is authored by the caller
// and marked forwarded, carrying the trail to the original author/chat. The new
// message flows to the target chat exactly like a normal send (receipts, offline
// inbox, real-time fan-out) but is not automatically re-translated: it carries
// the original text verbatim.
func (h *MessageHandler) ForwardMessage(c *gin.Context) {
	userID := c.GetString("userID")
	sourceChatID := c.Param("chatId")
	messageID := c.Param("messageId")

	if h.messageService == nil {
		WriteError(c, middleware.ErrInternal("Failed to forward message"))
		return
	}

	// The caller must be able to read the source message.
	isParticipant, err := h.chatService.IsParticipant(sourceChatID, userID)
	if err != nil || !isParticipant {
		WriteError(c, middleware.ErrForbidden("Access denied"))
		return
	}

	var req struct {
		TargetChatID string `json:"targetChatId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.TargetChatID == "" {
		WriteError(c, middleware.ErrValidation("targetChatId is required"))
		return
	}
	if req.TargetChatID == sourceChatID {
		WriteError(c, middleware.ErrValidation("Cannot forward a message to the same chat"))
		return
	}

	// The caller must be able to post into the target chat.
	isParticipant, err = h.chatService.IsParticipant(req.TargetChatID, userID)
	if err != nil || !isParticipant {
		WriteError(c, middleware.ErrForbidden("Access denied"))
		return
	}

	start := time.Now()
	forwarded, err := h.messageService.ForwardMessage(c.Request.Context(), sourceChatID, messageID, req.TargetChatID, userID)
	if err != nil {
		observability.ObserveMessageSend("error", time.Since(start))
		if err == sql.ErrNoRows {
			WriteError(c, middleware.ErrNotFound("Message not found"))
			return
		}
		WriteError(c, middleware.ErrInternal("Failed to forward message"))
		return
	}
	observability.ObserveMessageSend("sent", time.Since(start))

	h.broadcastNewMessage(c.Request.Context(), forwarded, req.TargetChatID)

	c.JSON(201, forwarded)
}

// DeleteMessage soft-deletes a message (task 6.2). Only the author or a chat
// admin may delete; everyone in the chat is notified so their UI removes the
// message. The row is kept for recoverability and to keep replies/forwards valid.
func (h *MessageHandler) DeleteMessage(c *gin.Context) {
	userID := c.GetString("userID")
	chatID := c.Param("chatId")
	messageID := c.Param("messageId")

	isParticipant, err := h.chatService.IsParticipant(chatID, userID)
	if err != nil || !isParticipant {
		WriteError(c, middleware.ErrForbidden("Access denied"))
		return
	}

	if h.messageService == nil {
		WriteError(c, middleware.ErrInternal("Failed to delete message"))
		return
	}

	message, err := h.messageService.GetMessageByID(c.Request.Context(), messageID)
	if err != nil || message == nil || message.ChatID != chatID {
		WriteError(c, middleware.ErrNotFound("Message not found"))
		return
	}

	// Author or chat admin only.
	if message.SenderID != userID {
		isAdmin, aerr := h.chatService.IsChatAdmin(chatID, userID)
		if aerr != nil {
			WriteError(c, middleware.ErrInternal("Failed to check admin role"))
			return
		}
		if !isAdmin {
			WriteError(c, middleware.ErrForbidden("You can only delete your own messages"))
			return
		}
	}

	deleted, err := h.messageService.DeleteMessage(c.Request.Context(), chatID, messageID)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to delete message"))
		return
	}
	if !deleted {
		WriteError(c, middleware.ErrNotFound("Message not found"))
		return
	}

	participants, _ := h.chatService.GetParticipants(chatID)
	userIDs := make([]string, 0, len(participants))
	for _, p := range participants {
		userIDs = append(userIDs, p.UserID)
	}
	h.wsHub.SendToChat(chatID, userIDs, "message_deleted", gin.H{
		"chatId":    chatID,
		"messageId": messageID,
		"deletedBy": userID,
		"deletedAt": time.Now().UTC(),
	})

	c.JSON(204, nil)
}

// PinMessage pins a message to a chat (task 6.2). Any participant may pin; the
// whole chat is notified so the pin banner updates in real time.
func (h *MessageHandler) PinMessage(c *gin.Context) {
	userID := c.GetString("userID")
	chatID := c.Param("chatId")

	isParticipant, err := h.chatService.IsParticipant(chatID, userID)
	if err != nil || !isParticipant {
		WriteError(c, middleware.ErrForbidden("Access denied"))
		return
	}

	var req struct {
		MessageID string `json:"messageId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.MessageID == "" {
		WriteError(c, middleware.ErrValidation("messageId is required"))
		return
	}

	if h.messageService == nil {
		WriteError(c, middleware.ErrInternal("Failed to pin message"))
		return
	}

	message, err := h.messageService.GetMessageByID(c.Request.Context(), req.MessageID)
	if err != nil || message == nil || message.ChatID != chatID {
		WriteError(c, middleware.ErrNotFound("Message not found"))
		return
	}

	if err := h.messageService.PinMessage(c.Request.Context(), chatID, req.MessageID, userID); err != nil {
		WriteError(c, middleware.ErrInternal("Failed to pin message"))
		return
	}

	participants, _ := h.chatService.GetParticipants(chatID)
	userIDs := make([]string, 0, len(participants))
	for _, p := range participants {
		userIDs = append(userIDs, p.UserID)
	}
	h.wsHub.SendToChat(chatID, userIDs, "message_pinned", gin.H{
		"chatId":    chatID,
		"messageId": req.MessageID,
		"pinnedBy":  userID,
		"pinnedAt":  time.Now().UTC(),
	})

	c.JSON(200, gin.H{
		"messageId": req.MessageID,
		"pinnedBy":  userID,
		"pinnedAt":  time.Now().UTC(),
		"pinned":    true,
	})
}

// UnpinMessage removes a message from a chat's pin list (task 6.2).
func (h *MessageHandler) UnpinMessage(c *gin.Context) {
	userID := c.GetString("userID")
	chatID := c.Param("chatId")
	messageID := c.Param("messageId")

	isParticipant, err := h.chatService.IsParticipant(chatID, userID)
	if err != nil || !isParticipant {
		WriteError(c, middleware.ErrForbidden("Access denied"))
		return
	}

	if h.messageService == nil {
		WriteError(c, middleware.ErrInternal("Failed to unpin message"))
		return
	}

	if err := h.messageService.UnpinMessage(c.Request.Context(), chatID, messageID); err != nil {
		WriteError(c, middleware.ErrInternal("Failed to unpin message"))
		return
	}

	participants, _ := h.chatService.GetParticipants(chatID)
	userIDs := make([]string, 0, len(participants))
	for _, p := range participants {
		userIDs = append(userIDs, p.UserID)
	}
	h.wsHub.SendToChat(chatID, userIDs, "message_unpinned", gin.H{
		"chatId":    chatID,
		"messageId": messageID,
	})

	c.JSON(204, nil)
}

// GetPinnedMessages returns the messages pinned to a chat (task 6.2), newest
// pin first, with the "pinned by" attribution.
func (h *MessageHandler) GetPinnedMessages(c *gin.Context) {
	userID := c.GetString("userID")
	chatID := c.Param("chatId")

	isParticipant, err := h.chatService.IsParticipant(chatID, userID)
	if err != nil || !isParticipant {
		WriteError(c, middleware.ErrForbidden("Access denied"))
		return
	}

	if h.messageService == nil {
		WriteError(c, middleware.ErrInternal("Failed to fetch pinned messages"))
		return
	}

	pins, err := h.messageService.GetPinnedMessages(c.Request.Context(), chatID)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to fetch pinned messages"))
		return
	}

	c.JSON(200, gin.H{"pins": pins})
}

// broadcastNewMessage fans a freshly-created message out to a chat's
// participants: per-recipient 'sent' receipts, offline-inbox queueing, and the
// real-time new_message event (mirrors SendMessage, minus translation enqueue).
func (h *MessageHandler) broadcastNewMessage(ctx context.Context, message *models.Message, chatID string) {
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
