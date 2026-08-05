package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/chorus/messenger/internal/models"
	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
)

type WaitlistHandler struct {
	service *services.WaitlistService
	email   services.EmailSender
}

func NewWaitlistHandler(service *services.WaitlistService, senders ...services.EmailSender) *WaitlistHandler {
	var email services.EmailSender
	if len(senders) > 0 {
		email = senders[0]
	}
	return &WaitlistHandler{service: service, email: email}
}

func (h *WaitlistHandler) Submit(c *gin.Context) {
	var req models.WaitlistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Please provide your email, spoken language, learning languages, and at least one reason."})
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	entry, err := h.service.Submit(req)
	if err != nil {
		if err == services.ErrInvalidWaitlistRequest {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to join the waitlist. Please try again."})
		return
	}
	if h.email != nil {
		if err := h.email.Send(entry.Email, "You're on the Chorus waitlist",
			"<p>Thanks for joining the Chorus waitlist. Your place is #"+strconv.Itoa(entry.QueuePosition)+".</p>"); err != nil {
			c.JSON(http.StatusAccepted, gin.H{"entry": entry, "message": "You have been added to the waitlist, but we could not send a confirmation email."})
			return
		}
	}
	c.JSON(http.StatusCreated, gin.H{"entry": entry, "message": "You have been added to the waitlist."})
}
