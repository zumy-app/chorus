package handlers

import (
	"net/http"

	"github.com/chorus/messenger/internal/models"
	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
)

type GrammarHandler struct {
	grammarService *services.GrammarService
	messageService *services.MessageService
}

func NewGrammarHandler(grammarService *services.GrammarService, messageService *services.MessageService) *GrammarHandler {
	return &GrammarHandler{
		grammarService: grammarService,
		messageService: messageService,
	}
}

// AnalyzeMessageGrammar analyzes grammar for a specific message
// POST /api/v1/grammar/analyze
func (h *GrammarHandler) AnalyzeMessageGrammar(c *gin.Context) {
	var req models.GrammarAnalysisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Get the message
	message, err := h.messageService.GetMessageByID(c.Request.Context(), req.MessageID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Message not found"})
		return
	}
	
	// Determine which text to analyze
	textToAnalyze := message.Text
	languageToAnalyze := message.OriginalLanguage
	
	// If target language is specified and different from original, use translation
	if req.TargetLanguage != "" && req.TargetLanguage != message.OriginalLanguage {
		if translation, exists := message.Translations[req.TargetLanguage]; exists {
			textToAnalyze = translation
			languageToAnalyze = req.TargetLanguage
		}
	}
	
	// Perform grammar analysis
	analysis, err := h.grammarService.AnalyzeGrammar(c.Request.Context(), textToAnalyze, languageToAnalyze)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to analyze grammar"})
		return
	}
	
	c.JSON(http.StatusOK, analysis)
}

// AnalyzeText analyzes grammar for arbitrary text
// POST /api/v1/grammar/analyze-text
func (h *GrammarHandler) AnalyzeText(c *gin.Context) {
	var req struct {
		Text     string `json:"text" binding:"required"`
		Language string `json:"language" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	analysis, err := h.grammarService.AnalyzeGrammar(c.Request.Context(), req.Text, req.Language)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to analyze grammar"})
		return
	}
	
	c.JSON(http.StatusOK, analysis)
}

// GetGrammarSuggestions provides learning suggestions based on user level
// GET /api/v1/grammar/suggestions
func (h *GrammarHandler) GetGrammarSuggestions(c *gin.Context) {
	userLevel := c.Query("level")
	targetLanguage := c.Query("language")
	
	if userLevel == "" {
		userLevel = "B1" // Default to intermediate
	}
	
	if targetLanguage == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Language parameter required"})
		return
	}
	
	suggestions, err := h.grammarService.GetGrammarSuggestions(c.Request.Context(), userLevel, targetLanguage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get suggestions"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"suggestions": suggestions})
}

// GetGrammarReport generates a grammar learning report for the user
// GET /api/v1/grammar/report
func (h *GrammarHandler) GetGrammarReport(c *gin.Context) {
	userID := c.GetString("userID")
	language := c.Query("language")
	
	if language == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Language parameter required"})
		return
	}
	
	report, err := h.grammarService.GenerateGrammarReport(c.Request.Context(), userID, language)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate report"})
		return
	}
	
	c.JSON(http.StatusOK, report)
}
