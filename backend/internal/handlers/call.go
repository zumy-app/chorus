package handlers

import (
	"net/http"
	"strconv"

	"github.com/chorus/messenger/internal/models"
	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
)

type CallHandler struct {
	callService *services.CallService
}

func NewCallHandler(callService *services.CallService) *CallHandler {
	return &CallHandler{
		callService: callService,
	}
}

// InitiateCall initiates a new voice or video call
// POST /api/v1/calls/initiate
func (h *CallHandler) InitiateCall(c *gin.Context) {
	userID := c.GetString("userID")
	
	var req models.InitiateCallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Create call session
	session, err := h.callService.InitiateCall(c.Request.Context(), req.ChatID, userID, req.Type)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initiate call"})
		return
	}
	
	// Generate WebRTC offer
	offer, err := h.callService.GenerateWebRTCOffer(c.Request.Context(), session.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate WebRTC offer"})
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{
		"session": session,
		"offer":   offer,
	})
}

// EndCall ends an active call
// POST /api/v1/calls/:callId/end
func (h *CallHandler) EndCall(c *gin.Context) {
	callID := c.Param("callId")
	
	err := h.callService.EndCall(c.Request.Context(), callID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to end call"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "Call ended successfully"})
}

// GetCallSession retrieves call session details
// GET /api/v1/calls/:callId
func (h *CallHandler) GetCallSession(c *gin.Context) {
	callID := c.Param("callId")
	
	session, err := h.callService.GetCallSession(c.Request.Context(), callID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Call session not found"})
		return
	}
	
	c.JSON(http.StatusOK, session)
}

// GetCallTranscript retrieves the transcript for a call
// GET /api/v1/calls/:callId/transcript
func (h *CallHandler) GetCallTranscript(c *gin.Context) {
	callID := c.Param("callId")
	
	transcript, err := h.callService.GetCallTranscript(c.Request.Context(), callID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transcript not found"})
		return
	}
	
	c.JSON(http.StatusOK, transcript)
}

// GetCallHistory retrieves call history for the user
// GET /api/v1/calls/history
func (h *CallHandler) GetCallHistory(c *gin.Context) {
	userID := c.GetString("userID")
	
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
	
	history, err := h.callService.GetUserCallHistory(c.Request.Context(), userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get call history"})
		return
	}
	
	c.JSON(http.StatusOK, history)
}

// DeleteCallTranscript deletes a call transcript
// DELETE /api/v1/calls/:callId/transcript
func (h *CallHandler) DeleteCallTranscript(c *gin.Context) {
	userID := c.GetString("userID")
	callID := c.Param("callId")
	
	err := h.callService.DeleteCallTranscript(c.Request.Context(), callID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete transcript"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "Transcript deleted successfully"})
}

// SearchTranscripts searches through call transcripts
// GET /api/v1/calls/transcripts/search
func (h *CallHandler) SearchTranscripts(c *gin.Context) {
	userID := c.GetString("userID")
	query := c.Query("q")
	language := c.Query("language")
	
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query required"})
		return
	}
	
	transcripts, err := h.callService.SearchTranscripts(c.Request.Context(), userID, query, language)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search transcripts"})
		return
	}
	
	c.JSON(http.StatusOK, transcripts)
}

// HandleWebRTCSignaling handles WebRTC signaling (offer, answer, ICE candidates)
// POST /api/v1/calls/:callId/signal
func (h *CallHandler) HandleWebRTCSignaling(c *gin.Context) {
	callID := c.Param("callId")
	
	var signal struct {
		Type      string `json:"type" binding:"required"` // offer, answer, ice-candidate
		SDP       string `json:"sdp"`
		Candidate string `json:"candidate"`
	}
	
	if err := c.ShouldBindJSON(&signal); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// In a real implementation, this would:
	// 1. Validate the call session
	// 2. Forward signaling data to other participants via WebSocket
	// 3. Handle ICE candidate exchange
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Signal received",
		"callId":  callID,
	})
}
