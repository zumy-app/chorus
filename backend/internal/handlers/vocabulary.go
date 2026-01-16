package handlers

import (
	"net/http"
	"strconv"

	"github.com/chorus/messenger/internal/models"
	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
)

type VocabularyHandler struct {
	vocabularyService *services.VocabularyService
	messageService    *services.MessageService
}

func NewVocabularyHandler(vocabularyService *services.VocabularyService, messageService *services.MessageService) *VocabularyHandler {
	return &VocabularyHandler{
		vocabularyService: vocabularyService,
		messageService:    messageService,
	}
}

// SaveVocabulary saves a new vocabulary entry from a message
// POST /api/v1/vocabulary
func (h *VocabularyHandler) SaveVocabulary(c *gin.Context) {
	userID := c.GetString("userID")
	
	var req models.SaveVocabularyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Get the message to extract context
	message, err := h.messageService.GetMessageByID(c.Request.Context(), req.MessageID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Message not found"})
		return
	}
	
	// Save vocabulary entry
	entry, err := h.vocabularyService.SaveVocabulary(c.Request.Context(), userID, req, message.Text, message.ChatID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save vocabulary"})
		return
	}
	
	c.JSON(http.StatusCreated, entry)
}

// GetVocabularyDue retrieves vocabulary items due for review
// GET /api/v1/vocabulary/due
func (h *VocabularyHandler) GetVocabularyDue(c *gin.Context) {
	userID := c.GetString("userID")
	
	limit := 20
	if limitParam := c.Query("limit"); limitParam != "" {
		if parsedLimit, err := strconv.Atoi(limitParam); err == nil {
			limit = parsedLimit
		}
	}
	
	entries, err := h.vocabularyService.GetVocabularyDueForReview(c.Request.Context(), userID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get due vocabulary"})
		return
	}
	
	c.JSON(http.StatusOK, entries)
}

// GetUserVocabulary retrieves all vocabulary for a user
// GET /api/v1/vocabulary
func (h *VocabularyHandler) GetUserVocabulary(c *gin.Context) {
	userID := c.GetString("userID")
	language := c.Query("language")
	
	limit := 50
	if limitParam := c.Query("limit"); limitParam != "" {
		if parsedLimit, err := strconv.Atoi(limitParam); err == nil {
			limit = parsedLimit
		}
	}
	
	offset := 0
	if offsetParam := c.Query("offset"); offsetParam != "" {
		if parsedOffset, err := strconv.Atoi(offsetParam); err == nil {
			offset = parsedOffset
		}
	}
	
	entries, err := h.vocabularyService.GetUserVocabulary(c.Request.Context(), userID, language, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get vocabulary"})
		return
	}
	
	c.JSON(http.StatusOK, entries)
}

// UpdatePracticeResult updates learning progress after practice
// POST /api/v1/vocabulary/practice
func (h *VocabularyHandler) UpdatePracticeResult(c *gin.Context) {
	userID := c.GetString("userID")
	
	var req models.PracticeResultRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	err := h.vocabularyService.UpdatePracticeResult(c.Request.Context(), userID, req.VocabularyID, req.Correct)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update practice result"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "Practice result updated successfully"})
}

// DeleteVocabulary removes a vocabulary entry
// DELETE /api/v1/vocabulary/:id
func (h *VocabularyHandler) DeleteVocabulary(c *gin.Context) {
	userID := c.GetString("userID")
	vocabularyID := c.Param("id")
	
	err := h.vocabularyService.DeleteVocabulary(c.Request.Context(), userID, vocabularyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete vocabulary"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "Vocabulary deleted successfully"})
}

// GetLearningProgress retrieves learning statistics
// GET /api/v1/vocabulary/progress
func (h *VocabularyHandler) GetLearningProgress(c *gin.Context) {
	userID := c.GetString("userID")
	
	progress, err := h.vocabularyService.GetLearningProgress(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get learning progress"})
		return
	}
	
	c.JSON(http.StatusOK, progress)
}

// GetVocabularyByID retrieves a specific vocabulary entry
// GET /api/v1/vocabulary/:id
func (h *VocabularyHandler) GetVocabularyByID(c *gin.Context) {
	userID := c.GetString("userID")
	vocabularyID := c.Param("id")
	
	entry, err := h.vocabularyService.GetVocabularyByID(c.Request.Context(), userID, vocabularyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vocabulary not found"})
		return
	}
	
	c.JSON(http.StatusOK, entry)
}

// SearchVocabulary searches user's vocabulary
// GET /api/v1/vocabulary/search
func (h *VocabularyHandler) SearchVocabulary(c *gin.Context) {
	userID := c.GetString("userID")
	query := c.Query("q")
	language := c.Query("language")
	
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query required"})
		return
	}
	
	entries, err := h.vocabularyService.SearchVocabulary(c.Request.Context(), userID, query, language)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search vocabulary"})
		return
	}
	
	c.JSON(http.StatusOK, entries)
}
