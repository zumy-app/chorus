package handlers

import (
	"log"
	"net/http"

	"github.com/chorus/messenger/internal/models"
	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
)

type GrammarHandler struct {
	grammarService *services.GrammarService
	grammarQueue   *services.GrammarQueueService
	messageService *services.MessageService
}

func NewGrammarHandler(gs *services.GrammarService, gq *services.GrammarQueueService, ms *services.MessageService) *GrammarHandler {
	return &GrammarHandler{
		grammarService: gs,
		grammarQueue:   gq,
		messageService: ms,
	}
}

// AnalyzeMessageGrammar analyzes grammar for a specific message using optional target language
// POST /api/v1/grammar/analyze
func (h *GrammarHandler) AnalyzeMessageGrammar(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.GrammarAnalysisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	message, err := h.messageService.GetMessageByID(c.Request.Context(), req.MessageID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Message not found"})
		return
	}

	text := message.Text
	language := message.OriginalLanguage
	if req.TargetLanguage != "" && req.TargetLanguage != message.OriginalLanguage {
		if translation, ok := message.Translations[req.TargetLanguage]; ok {
			text = translation
			language = req.TargetLanguage
		}
	}

	nativeLang := req.NativeLanguage
	if nativeLang == "" {
		nativeLang = "en"
	}
	analysis, err := h.grammarService.AnalyzeGrammar(text, language, nativeLang)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Grammar analysis failed"})
		return
	}

	c.JSON(http.StatusOK, analysis)
}

// AnalyzeText analyzes grammar of arbitrary text
// POST /api/v1/grammar/analyze-text
func (h *GrammarHandler) AnalyzeText(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		Text           string `json:"text" binding:"required"`
		Language       string `json:"language" binding:"required"`
		NativeLanguage string `json:"nativeLanguage"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	if req.NativeLanguage == "" {
		req.NativeLanguage = "en"
	}

	// Perform grammar analysis
	analysis, err := h.grammarService.AnalyzeGrammar(req.Text, req.Language, req.NativeLanguage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Grammar analysis failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"text":     req.Text,
			"language": req.Language,
			"analysis": analysis,
		},
	})
}

// GetDifficultyLevel gets the difficulty level of text
// POST /api/v1/grammar/difficulty
func (h *GrammarHandler) GetDifficultyLevel(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		Text           string `json:"text" binding:"required"`
		Language       string `json:"language" binding:"required"`
		NativeLanguage string `json:"nativeLanguage"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	nativeLang := req.NativeLanguage
	if nativeLang == "" {
		nativeLang = "en"
	}
	analysis, err := h.grammarService.AnalyzeGrammar(req.Text, req.Language, nativeLang)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Analysis failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"difficulty": analysis.Difficulty,
			"patterns":   analysis.Patterns,
		},
	})
}

// GetGrammarSuggestions returns learning suggestions for a language and level
// GET /api/v1/grammar/suggestions
func (h *GrammarHandler) GetGrammarSuggestions(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	level := c.DefaultQuery("level", "B1")
	language := c.Query("language")
	if language == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Language parameter required"})
		return
	}

	suggestions, err := h.grammarService.GetGrammarSuggestions(level, language)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get suggestions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"suggestions": suggestions})
}

// GetGrammarReport returns a grammar progress report for the user
// GET /api/v1/grammar/report
func (h *GrammarHandler) GetGrammarReport(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	language := c.Query("language")
	if language == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Language parameter required"})
		return
	}

	report, err := h.grammarService.GenerateGrammarReport(userID, language)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate report"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": report})
}

// AnalyzeTextWithAI performs AI-powered grammar analysis on arbitrary text.
// Analysis runs asynchronously: the request is enqueued and the job id is
// returned immediately. The result is delivered to the requesting user over the
// WebSocket "grammar_analysis" event and can be polled via GET /grammar/analyze/:jobId.
// Already-cached analyses are returned synchronously without enqueueing.
// POST /api/v1/grammar/analyze-ai
func (h *GrammarHandler) AnalyzeTextWithAI(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		Text           string `json:"text" binding:"required"`
		Language       string `json:"language" binding:"required"`
		NativeLanguage string `json:"nativeLanguage"`
		MessageID      string `json:"messageId"`
		ChatID         string `json:"chatId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	nativeLang := req.NativeLanguage
	if nativeLang == "" {
		nativeLang = "en"
	}

	if h.grammarQueue == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Grammar queue not available"})
		return
	}

	// Hot path: already analyzed on a previous request — return instantly.
	if analysis, provider, ok := h.grammarService.CachedAIAnalysis(req.Text, req.Language, nativeLang); ok {
		c.JSON(http.StatusOK, gin.H{
			"jobId":        "",
			"status":       "done",
			"analysis":     analysis,
			"providerUsed": provider,
		})
		return
	}

	jobID, err := h.grammarQueue.EnqueueForAnalysis(userID, req.ChatID, req.MessageID, req.Text, req.Language, nativeLang)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI grammar analysis failed"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"jobId":     jobID,
		"status":    "queued",
		"messageId": req.MessageID,
	})
}

// GetGrammarJob returns the state and (completed) result of a grammar analysis
// job owned by the requesting user. Used for WebSocket resync/recovery.
// GET /api/v1/grammar/analyze/:jobId
func (h *GrammarHandler) GetGrammarJob(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	jobID := c.Param("jobId")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "jobId is required"})
		return
	}

	job, err := h.grammarQueue.GetJob(jobID, userID)
	if err != nil {
		log.Printf("[Grammar] get job %s for user %s: %v", jobID, userID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Grammar job not found"})
		return
	}

	c.JSON(http.StatusOK, job)
}

// LearnGrammar generates interactive learning content (breakdown, examples, flashcards, custom)
// POST /api/v1/grammar/learn
func (h *GrammarHandler) LearnGrammar(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.LearnRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	content, err := h.grammarService.GenerateLearningContent(req.Text, req.Language, req.NativeLanguage, req.Action, req.CustomQuery)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate learning content"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": content,
	})
}
