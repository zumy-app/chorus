package handlers

import (
	"net/http"
	"strconv"

	"github.com/chorus/messenger/internal/models"
	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
)

type SearchHandler struct {
	searchService *services.SearchService
}

func NewSearchHandler(ss *services.SearchService) *SearchHandler {
	return &SearchHandler{
		searchService: ss,
	}
}

// SearchMessages searches for messages
// GET /api/v1/messages/search
func (h *SearchHandler) SearchMessages(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query required"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	language := c.Query("language")

	// Parse chat IDs if provided
	var chatIDs []string
	if chatID := c.Query("chatId"); chatID != "" {
		chatIDs = append(chatIDs, chatID)
	}

	req := models.SearchRequest{
		Query:    query,
		ChatIDs:  chatIDs,
		Language: language,
		Limit:    limit,
		Offset:   offset,
	}

	result, err := h.searchService.SearchMessages(userID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Search failed"})
		return
	}

	// Record search for suggestions
	h.searchService.RecordSearch(userID, query)

	c.JSON(http.StatusOK, gin.H{
		"data": result,
	})
}

// SearchChats searches chat metadata by name
// GET /api/v1/chats/search
func (h *SearchHandler) SearchChats(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query required"})
		return
	}

	chats, err := h.searchService.SearchChats(userID, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search chats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": chats})
}

// SearchContacts searches contacts by display name/email/username
// GET /api/v1/contacts/search
func (h *SearchHandler) SearchContacts(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query required"})
		return
	}

	contacts, err := h.searchService.SearchContacts(userID, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search contacts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": contacts})
}

// SearchInChat searches within a specific chat
// GET /api/v1/chats/:chatId/search
func (h *SearchHandler) SearchInChat(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	chatID := c.Param("chatId")
	if chatID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chat ID required"})
		return
	}

	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query required"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	messages, err := h.searchService.SearchInChat(userID, chatID, query, limit)
	if err != nil {
		if err.Error() == "not a chat participant" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Not a participant in this chat"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Search failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": messages,
	})
}

// GetSearchSuggestions returns search suggestions
// GET /api/v1/search/suggestions
func (h *SearchHandler) GetSearchSuggestions(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	prefix := c.Query("prefix")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	suggestions, err := h.searchService.GetSearchSuggestions(userID, prefix, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get suggestions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": suggestions,
	})
}

// GetRecentSearches returns recent searches
// GET /api/v1/search/recent
func (h *SearchHandler) GetRecentSearches(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	searches, err := h.searchService.GetRecentSearches(userID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get recent searches"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": searches,
	})
}

// ClearSearchHistory clears search history
// DELETE /api/v1/search/history
func (h *SearchHandler) ClearSearchHistory(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	err := h.searchService.ClearSearchHistory(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Search history cleared",
	})
}

// SearchVocabulary searches vocabulary entries
// GET /api/v1/vocabulary/search
func (h *SearchHandler) SearchVocabulary(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	query := c.Query("q")
	language := c.Query("language")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query required"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	entries, err := h.searchService.SearchVocabularyWithLanguage(userID, query, language, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Search failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": entries,
	})
}
