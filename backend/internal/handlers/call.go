package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/chorus/messenger/internal/middleware"
	"github.com/chorus/messenger/internal/models"
	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
)

type CallHandler struct {
	callService *services.CallService
}

func NewCallHandler(callService *services.CallService) *CallHandler {
	return &CallHandler{callService: callService}
}

func (h *CallHandler) InitiateCall(c *gin.Context) {
	userID := c.GetString("userID")
	var req models.InitiateCallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteError(c, middleware.ErrValidation("Invalid request"))
		return
	}
	session, err := h.callService.InitiateCall(c.Request.Context(), req.ChatID, userID, req.Type)
	if err != nil {
		if errors.Is(err, services.ErrNotParticipant) {
			WriteError(c, middleware.ErrForbidden("Not a participant of this chat"))
			return
		}
		if errors.Is(err, services.ErrChatNotFound) {
			WriteError(c, middleware.ErrNotFound("Chat not found"))
			return
		}
		WriteError(c, middleware.ErrInternal("Failed to initiate call"))
		return
	}
	offer, err := h.callService.GenerateWebRTCOffer(c.Request.Context(), session.ID)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to generate WebRTC offer"))
		return
	}
	c.JSON(http.StatusCreated, gin.H{"session": session, "offer": offer})
}

func (h *CallHandler) EndCall(c *gin.Context) {
	callID := c.Param("callId")
	userID := c.GetString("userID")
	err := h.callService.EndCallAs(c.Request.Context(), callID, userID)
	if err != nil {
		if errors.Is(err, services.ErrCallNotFound) {
			WriteError(c, middleware.ErrNotFound("Call session not found"))
			return
		}
		if errors.Is(err, services.ErrNotParticipant) {
			WriteError(c, middleware.ErrForbidden("Not a participant of this call"))
			return
		}
		if errors.Is(err, services.ErrCallAlreadyEnded) {
			WriteError(c, middleware.ErrValidation("Call already ended"))
			return
		}
		WriteError(c, middleware.ErrInternal("Failed to end call"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Call ended successfully"})
}

func (h *CallHandler) GetCallSession(c *gin.Context) {
	callID := c.Param("callId")
	userID := c.GetString("userID")
	session, err := h.callService.GetCallSession(c.Request.Context(), callID)
	if err != nil {
		if errors.Is(err, services.ErrCallNotFound) {
			WriteError(c, middleware.ErrNotFound("Call session not found"))
			return
		}
		WriteError(c, middleware.ErrNotFound("Call session not found"))
		return
	}
	isPart := false
	for _, p := range session.Participants {
		if p == userID {
			isPart = true
			break
		}
	}
	if !isPart {
		WriteError(c, middleware.ErrForbidden("Not a participant of this call"))
		return
	}
	c.JSON(http.StatusOK, session)
}

func (h *CallHandler) GetCallTranscript(c *gin.Context) {
	callID := c.Param("callId")
	userID := c.GetString("userID")
	session, err := h.callService.GetCallSession(c.Request.Context(), callID)
	if err == nil {
		isPart := false
		for _, p := range session.Participants {
			if p == userID {
				isPart = true
				break
			}
		}
		if !isPart {
			WriteError(c, middleware.ErrForbidden("Not a participant of this call"))
			return
		}
	}
	transcript, err := h.callService.GetCallTranscript(c.Request.Context(), callID)
	if err != nil {
		WriteError(c, middleware.ErrNotFound("Transcript not found"))
		return
	}
	c.JSON(http.StatusOK, transcript)
}

func (h *CallHandler) GetCallHistory(c *gin.Context) {
	userID := c.GetString("userID")
	limit := 50
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	offset := 0
	if v := c.Query("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			offset = n
		}
	}
	history, err := h.callService.GetUserCallHistory(c.Request.Context(), userID, limit, offset)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to get call history"))
		return
	}
	c.JSON(http.StatusOK, history)
}

func (h *CallHandler) DeleteCallTranscript(c *gin.Context) {
	userID := c.GetString("userID")
	callID := c.Param("callId")
	err := h.callService.DeleteCallTranscript(c.Request.Context(), callID, userID)
	if err != nil {
		if errors.Is(err, services.ErrNotParticipant) {
			WriteError(c, middleware.ErrForbidden("Not a participant of this call"))
			return
		}
		if errors.Is(err, services.ErrCallNotFound) {
			WriteError(c, middleware.ErrNotFound("Call session not found"))
			return
		}
		WriteError(c, middleware.ErrInternal("Failed to delete transcript"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Transcript deleted successfully"})
}

func (h *CallHandler) SearchTranscripts(c *gin.Context) {
	userID := c.GetString("userID")
	query := c.Query("q")
	language := c.Query("language")
	if query == "" {
		WriteError(c, middleware.ErrValidation("Search query required"))
		return
	}
	transcripts, err := h.callService.SearchTranscripts(c.Request.Context(), userID, query, language)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to search transcripts"))
		return
	}
	c.JSON(http.StatusOK, transcripts)
}

func (h *CallHandler) PostCaption(c *gin.Context) {
	callID := c.Param("callId")
	userID := c.GetString("userID")
	var req struct {
		Text     string `json:"text" binding:"required"`
		Language string `json:"language"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteError(c, middleware.ErrValidation("Invalid request"))
		return
	}
	seg, err := h.callService.PublishLiveCaption(c.Request.Context(), callID, userID, req.Text, req.Language)
	if err != nil {
		if errors.Is(err, services.ErrNotParticipant) {
			WriteError(c, middleware.ErrForbidden("Not a participant of this call"))
			return
		}
		if errors.Is(err, services.ErrCallAlreadyEnded) {
			WriteError(c, middleware.ErrValidation("Call already ended"))
			return
		}
		if errors.Is(err, services.ErrCallNotFound) {
			WriteError(c, middleware.ErrNotFound("Call session not found"))
			return
		}
		WriteError(c, middleware.ErrValidation(err.Error()))
		return
	}
	c.JSON(http.StatusCreated, seg)
}

func (h *CallHandler) GetCaptions(c *gin.Context) {
	callID := c.Param("callId")
	userID := c.GetString("userID")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	segments, total, err := h.callService.GetCaptionsPaginated(c.Request.Context(), callID, userID, limit, offset)
	if err != nil {
		if errors.Is(err, services.ErrNotParticipant) {
			WriteError(c, middleware.ErrForbidden("Not a participant of this call"))
			return
		}
		if errors.Is(err, services.ErrCallNotFound) {
			WriteError(c, middleware.ErrNotFound("Call session not found"))
			return
		}
		WriteError(c, middleware.ErrInternal("Failed to get captions"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"segments": segments, "total": total, "hasMore": offset+len(segments) < total})
}

func (h *CallHandler) BookmarkCaption(c *gin.Context) {
	callID := c.Param("callId")
	userID := c.GetString("userID")
	idxStr := c.Param("index")
	idx, err := strconv.Atoi(idxStr)
	if err != nil {
		WriteError(c, middleware.ErrValidation("Invalid caption index"))
		return
	}
	entry, err := h.callService.BookmarkCaption(c.Request.Context(), callID, userID, idx)
	if err != nil {
		if errors.Is(err, services.ErrNotParticipant) {
			WriteError(c, middleware.ErrForbidden("Not a participant of this call"))
			return
		}
		WriteError(c, middleware.ErrValidation(err.Error()))
		return
	}
	c.JSON(http.StatusCreated, entry)
}

func (h *CallHandler) HandleWebRTCSignaling(c *gin.Context) {
	callID := c.Param("callId")
	userID := c.GetString("userID")
	var signal struct {
		Type      string                 `json:"type" binding:"required"`
		SDP       string                 `json:"sdp"`
		Candidate string                 `json:"candidate"`
		Data      map[string]interface{} `json:"data"`
	}
	if err := c.ShouldBindJSON(&signal); err != nil {
		WriteError(c, middleware.ErrValidation("Invalid request"))
		return
	}
	extra := map[string]interface{}{}
	if signal.Data != nil {
		extra = signal.Data
		if signal.SDP == "" {
			if v, ok := extra["sdp"].(string); ok {
				signal.SDP = v
			}
		}
		if signal.Candidate == "" {
			if v, ok := extra["candidate"]; ok {
				switch val := v.(type) {
				case string:
					signal.Candidate = val
				}
			}
		}
	}
	err := h.callService.HandleSignal(c.Request.Context(), callID, userID, signal.Type, signal.SDP, signal.Candidate, extra)
	if err != nil {
		if errors.Is(err, services.ErrInvalidSignal) {
			WriteError(c, middleware.ErrValidation("Invalid signal type: must be offer, answer, ice-candidate, screen-share-start, screen-share-stop, or video-toggle"))
			return
		}
		if errors.Is(err, services.ErrCallNotFound) {
			WriteError(c, middleware.ErrNotFound("Call session not found"))
			return
		}
		if errors.Is(err, services.ErrNotParticipant) {
			WriteError(c, middleware.ErrForbidden("Not a participant of this call"))
			return
		}
		if errors.Is(err, services.ErrCallAlreadyEnded) {
			WriteError(c, middleware.ErrValidation("Call has already ended"))
			return
		}
		WriteError(c, middleware.ErrValidation(err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Signal forwarded", "callId": callID})
}
