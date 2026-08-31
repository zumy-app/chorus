package handlers

import (
	"net/http"
	"strconv"

	"github.com/chorus/messenger/internal/middleware"
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
		WriteError(c, middleware.ErrAuth("Unauthorized"))
		return
	}

	query := c.Query("q")
	if query == "" {
		WriteError(c, middleware.ErrValidation("Search query required"))
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
		WriteError(c, middleware.ErrInternal("Search failed"))
		return
	}

	// Record search for suggestions
	h.searchService.RecordSearch(userID, query)

	c.JSON(http.StatusOK, result)
}

// SearchChats searches chat metadata by name
// GET /api/v1/chats/search
func (h *SearchHandler) SearchChats(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		WriteError(c, middleware.ErrAuth("Unauthorized"))
		return
	}

	query := c.Query("q")
	if query == "" {
		WriteError(c, middleware.ErrValidation("Search query required"))
		return
	}

	chats, err := h.searchService.SearchChats(userID, query)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to search chats"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": chats})
}

// SearchContacts searches contacts by display name/email/username
// GET /api/v1/contacts/search
func (h *SearchHandler) SearchContacts(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		WriteError(c, middleware.ErrAuth("Unauthorized"))
		return
	}

	query := c.Query("q")
	if query == "" {
		WriteError(c, middleware.ErrValidation("Search query required"))
		return
	}

	contacts, err := h.searchService.SearchContacts(userID, query)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to search contacts"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": contacts})
}

// SearchInChat searches within a specific chat
// GET /api/v1/chats/:chatId/search
func (h *SearchHandler) SearchInChat(c *gin.Context) {
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

	query := c.Query("q")
	if query == "" {
		WriteError(c, middleware.ErrValidation("Search query required"))
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	messages, err := h.searchService.SearchInChat(userID, chatID, query, limit)
	if err != nil {
		if err.Error() == "not a chat participant" {
			WriteError(c, middleware.ErrForbidden("Not a participant in this chat"))
			return
		}
		WriteError(c, middleware.ErrInternal("Search failed"))
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
		WriteError(c, middleware.ErrAuth("Unauthorized"))
		return
	}

	prefix := c.Query("prefix")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	suggestions, err := h.searchService.GetSearchSuggestions(userID, prefix, limit)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to get suggestions"))
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
		WriteError(c, middleware.ErrAuth("Unauthorized"))
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	searches, err := h.searchService.GetRecentSearches(userID, limit)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to get recent searches"))
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
		WriteError(c, middleware.ErrAuth("Unauthorized"))
		return
	}

	err := h.searchService.ClearSearchHistory(userID)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to clear history"))
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
		WriteError(c, middleware.ErrAuth("Unauthorized"))
		return
	}

	query := c.Query("q")
	language := c.Query("language")
	if query == "" {
		WriteError(c, middleware.ErrValidation("Search query required"))
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	entries, err := h.searchService.SearchVocabularyWithLanguage(userID, query, language, limit)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Search failed"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": entries,
	})
}
