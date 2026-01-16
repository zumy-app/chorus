package handlers

import (
	"net/http"
	"strconv"

	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
)

type SearchHandler struct {
	searchService *services.SearchService
}

func NewSearchHandler(searchService *services.SearchService) *SearchHandler {
	return &SearchHandler{
		searchService: searchService,
	}
}

func (h *SearchHandler) SearchMessages(c *gin.Context) {
	userID := c.GetString("userID")
	query := c.Query("q")
	chatID := c.Query("chat_id")
	
	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query is required"})
		return
	}

	messages, err := h.searchService.SearchMessages(userID, chatID, query, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search messages"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"messages": messages})
}

func (h *SearchHandler) SearchChats(c *gin.Context) {
	userID := c.GetString("userID")
	query := c.Query("q")

	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query is required"})
		return
	}

	chats, err := h.searchService.SearchChats(userID, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search chats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"chats": chats})
}

func (h *SearchHandler) SearchContacts(c *gin.Context) {
	userID := c.GetString("userID")
	query := c.Query("q")

	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query is required"})
		return
	}

	contacts, err := h.searchService.SearchContacts(userID, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search contacts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"contacts": contacts})
}

func (h *SearchHandler) SearchVocabulary(c *gin.Context) {
	userID := c.GetString("userID")
	query := c.Query("q")

	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query is required"})
		return
	}

	vocabulary, err := h.searchService.SearchVocabulary(userID, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search vocabulary"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"vocabulary": vocabulary})
}
