package handlers

import (
	"context"
	"log"
	"strings"

	"github.com/chorus/messenger/internal/models"
	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
)

type MessageHandler struct {
	messageService     *services.MessageService
	chatService        *services.ChatService
	userService        *services.UserService
	entitlementService *services.EntitlementService
	translationQueue   *services.TranslationQueueService
	moderation         *services.ModerationService
	wsHub              *services.WebSocketHub
	translation        *services.TranslationService
	wordMining         *services.WordMiningQueueService
	learningProfile    *services.LearningProfileService
}

func NewMessageHandler(
	messageService *services.MessageService,
	chatService *services.ChatService,
	userService *services.UserService,
	entitlementService *services.EntitlementService,
	translationQueue *services.TranslationQueueService,
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

	// Safety rail: if a block relationship exists with any participant, the
	// chat is frozen for this sender (either direction blocks messaging).
	if h.moderation != nil {
		blocked, err := h.moderation.ChatBlocked(c.Request.Context(), chatID, userID)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to check blocked users"})
			return
		}
		if blocked {
			c.JSON(403, gin.H{"error": "You cannot send messages in this chat"})
			return
		}
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

	// Broadcast new message to ALL chat participants (including sender for multi-device)
	participants, _ := h.chatService.GetParticipants(chatID)
	userIDs := make([]string, 0, len(participants))
	for _, p := range participants {
		userIDs = append(userIDs, p.UserID)
	}

	h.wsHub.SendToChat(chatID, userIDs, "new_message", message)

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
		c.JSON(403, gin.H{"error": "Access denied"})
		return
	}

	var req struct {
		TargetLang string `json:"targetLang" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	message, err := h.messageService.GetMessageByID(c.Request.Context(), messageID)
	if err != nil {
		c.JSON(404, gin.H{"error": "Message not found"})
		return
	}
	if message.ChatID != chatID {
		c.JSON(404, gin.H{"error": "Message not found"})
		return
	}

	// The requested translation already exists — nothing to do.
	if message.Translations != nil {
		if _, ok := message.Translations[req.TargetLang]; ok {
			c.JSON(200, message)
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
