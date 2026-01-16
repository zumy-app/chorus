package handlers

import (
	"net/http"
	"strconv"

	"github.com/chorus/messenger/internal/models"
	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
)

type VocabularyHandler struct {
	vocabularyService  *services.VocabularyService
	messageService     *services.MessageService
	translationService *services.TranslationService
}

func NewVocabularyHandler(vs *services.VocabularyService, ms *services.MessageService, ts *services.TranslationService) *VocabularyHandler {
	return &VocabularyHandler{
		vocabularyService:  vs,
		messageService:     ms,
		translationService: ts,
	}
}

// SaveVocabulary saves a word to the user's vocabulary (compat wrapper)
// POST /api/v1/vocabulary
func (h *VocabularyHandler) SaveVocabulary(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.SaveVocabularyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	entry, err := h.vocabularyService.SaveWord(userID, req, h.messageService, h.translationService)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save vocabulary"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data": entry,
	})
}

// SaveWord keeps backward compatibility with older routers
func (h *VocabularyHandler) SaveWord(c *gin.Context) {
	h.SaveVocabulary(c)
}

// GetVocabulary returns user's vocabulary list
// GET /api/v1/vocabulary
func (h *VocabularyHandler) GetVocabulary(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	language := c.Query("language")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	entries, total, err := h.vocabularyService.GetUserVocabulary(userID, language, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get vocabulary"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"entries": entries,
			"total":   total,
			"limit":   limit,
			"offset":  offset,
			"hasMore": offset+len(entries) < total,
		},
	})
}

// GetDueVocabulary returns vocabulary due for review
// GET /api/v1/vocabulary/due
func (h *VocabularyHandler) GetDueVocabulary(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	entries, err := h.vocabularyService.GetDueVocabulary(userID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get due vocabulary"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": entries,
	})
}

// SearchVocabulary searches user's vocabulary by term/translation/definition
// GET /api/v1/vocabulary/search
func (h *VocabularyHandler) SearchVocabulary(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	query := c.Query("q")
	language := c.Query("language")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query required"})
		return
	}

	entries, err := h.vocabularyService.SearchVocabulary(userID, query, language, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search vocabulary"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": entries})
}

// RecordPractice records a practice session result
// POST /api/v1/vocabulary/practice
func (h *VocabularyHandler) RecordPractice(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.PracticeResultRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	err := h.vocabularyService.RecordPracticeResult(userID, req.VocabularyID, req.Correct)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record practice"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Practice recorded successfully",
	})
}

// UpdatePracticeResult is a compatibility alias for RecordPractice
// POST /api/v1/vocabulary/practice
func (h *VocabularyHandler) UpdatePracticeResult(c *gin.Context) {
	h.RecordPractice(c)
}

// GetProgress returns learning progress statistics
// GET /api/v1/vocabulary/progress
func (h *VocabularyHandler) GetProgress(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	progress, err := h.vocabularyService.GetLearningProgress(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get progress"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": progress,
	})
}

// DeleteVocabulary deletes a vocabulary entry
// DELETE /api/v1/vocabulary/:id
func (h *VocabularyHandler) DeleteVocabulary(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	vocabularyID := c.Param("id")
	if vocabularyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vocabulary ID required"})
		return
	}

	err := h.vocabularyService.DeleteVocabulary(userID, vocabularyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete vocabulary"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Vocabulary deleted successfully",
	})
}

// GetVocabularyByID returns a specific vocabulary entry
// GET /api/v1/vocabulary/:id
func (h *VocabularyHandler) GetVocabularyByID(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	vid := c.Param("id")
	entry, err := h.vocabularyService.GetVocabularyByID(userID, vid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vocabulary not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": entry})
}
