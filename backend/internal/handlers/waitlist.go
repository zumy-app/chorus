package handlers

import (
	"log"
	"net/http"
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Please provide your email, spoken languages, and at least one reason."})
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	entry, alreadyJoined, err := h.service.Submit(req)
	if err != nil {
		if err == services.ErrInvalidWaitlistRequest {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		log.Printf("[Waitlist] submit failed for %q: %v", req.Email, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to join the waitlist. Please try again."})
		return
	}

	var subject, html, message string
	status := http.StatusCreated
	if alreadyJoined {
		subject, html = services.UpdatedWaitlistConfirmationEmail(entry.QueuePosition)
		message = "You're already on the waitlist — we've updated your preferences. Your spot is waiting, stay tuned!"
	} else {
		subject, html = services.WaitlistConfirmationEmail(entry.QueuePosition)
		message = "You've joined the waitlist. We'll email you a sign-up link when it's your turn."
	}

	emailSent := false
	if h.email != nil {
		if err := h.email.Send(entry.Email, subject, html); err == nil {
			emailSent = true
		} else {
			log.Printf("[Waitlist] confirmation email to %q failed: %v", entry.Email, err)
		}
	}

	payload := gin.H{
		"entry":          entry,
		"message":        message,
		"alreadyJoined":  alreadyJoined,
		"emailSent":      emailSent,
		"checkSpamNotice": !emailSent,
	}
	if emailSent {
		c.JSON(status, payload)
		return
	}
	if alreadyJoined {
		// Still a success — user is already in, prefs were refreshed.
		c.JSON(http.StatusOK, payload)
		return
	}
	// Fresh signup persisted but the first send attempt failed. The email is
	// queued in the durable outbox and retried automatically.
	payload["message"] = "You've joined the waitlist. Your confirmation email is queued and on its way."
	c.JSON(http.StatusAccepted, payload)
}