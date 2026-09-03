package handlers

import (
	"encoding/base64"
	"errors"
	"net/http"
	"strconv"

	"github.com/chorus/messenger/internal/middleware"
	"github.com/chorus/messenger/internal/models"
	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
)

type CallHandler struct {
	callService   *services.CallService
	reviewService *services.CaptionReviewService
}

func NewCallHandler(callService *services.CallService) *CallHandler {
	return &CallHandler{callService: callService}
}

func (h *CallHandler) SetReviewService(s *services.CaptionReviewService) { h.reviewService = s }

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
	var body struct {
		Phrase string `json:"phrase"`
		Text   string `json:"text"`
		Term   string `json:"term"`
	}
	_ = c.ShouldBindJSON(&body)
	phrase := body.Phrase
	if phrase == "" {
		phrase = body.Text
	}
	if phrase == "" {
		phrase = body.Term
	}
	entry, err := h.callService.BookmarkCaptionWithPhrase(c.Request.Context(), callID, userID, idx, phrase)
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

func (h *CallHandler) TranscribeCaption(c *gin.Context) {
	callID := c.Param("callId")
	userID := c.GetString("userID")
	var req struct {
		Audio    string `json:"audio" binding:"required"`
		Language string `json:"language"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteError(c, middleware.ErrValidation("Audio (base64) required"))
		return
	}
	audioData, err := base64.StdEncoding.DecodeString(req.Audio)
	if err != nil {
		WriteError(c, middleware.ErrValidation("Invalid audio encoding"))
		return
	}
	seg, err := h.callService.TranscribeAndPublish(c.Request.Context(), callID, userID, audioData, req.Language)
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

func (h *CallHandler) ReviewCaption(c *gin.Context) {
	if h.reviewService == nil {
		WriteError(c, middleware.ErrInternal("Review service unavailable"))
		return
	}
	callID := c.Param("callId")
	idx, _ := strconv.Atoi(c.Param("index"))
	userID := c.GetString("userID")
	var req models.CaptionReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteError(c, middleware.ErrValidation("Invalid review payload"))
		return
	}
	review, err := h.reviewService.SubmitReview(c.Request.Context(), callID, idx, userID, req)
	if err != nil {
		if err.Error() == "rating must be between 1 and 5" || err.Error() == "feedback too long" || err.Error() == "corrected text too long" {
			WriteError(c, middleware.ErrValidation(err.Error()))
			return
		}
		if errors.Is(err, services.ErrCallNotFound) {
			WriteError(c, middleware.ErrNotFound("Call not found"))
			return
		}
		if err.Error() == "transcript not found" || err.Error() == "segment index out of range" {
			WriteError(c, middleware.ErrValidation(err.Error()))
			return
		}
		WriteError(c, middleware.ErrInternal("Failed to save review"))
		return
	}
	c.JSON(http.StatusCreated, review)
}

func (h *CallHandler) GetCaptionReviews(c *gin.Context) {
	if h.reviewService == nil {
		WriteError(c, middleware.ErrInternal("Review service unavailable"))
		return
	}
	callID := c.Param("callId")
	idx, _ := strconv.Atoi(c.Param("index"))
	reviews, err := h.reviewService.GetReviews(c.Request.Context(), callID, idx)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to fetch reviews"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"reviews": reviews})
}

func (h *CallHandler) GetReviewQueue(c *gin.Context) {
	if h.reviewService == nil {
		WriteError(c, middleware.ErrInternal("Review service unavailable"))
		return
	}
	userID := c.GetString("userID")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	items, total, err := h.reviewService.GetReviewQueue(c.Request.Context(), userID, limit, offset)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to fetch queue"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total, "hasMore": offset+len(items) < total})
}

func (h *CallHandler) GetCaptionQualityStats(c *gin.Context) {
	if h.reviewService == nil {
		WriteError(c, middleware.ErrInternal("Review service unavailable"))
		return
	}
	stats, err := h.reviewService.GetQualityStats(c.Request.Context())
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to fetch stats"))
		return
	}
	c.JSON(http.StatusOK, stats)
}
