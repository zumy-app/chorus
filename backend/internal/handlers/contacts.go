package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/chorus/messenger/internal/middleware"
	"github.com/chorus/messenger/internal/models"
	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
)

// ContactsHandler implements the Contacts & Invites epic (REQ 2.4 / FR-22-23):
// a permission-gated, privacy-preserving contact scan that detects on-platform
// users from hashed device contacts, and invitation creation/tracking for
// off-platform contacts via email, SMS, or WhatsApp.
type ContactsHandler struct {
	contactService    *services.ContactService
	invitationService *services.InvitationService
	notifications     services.EmailSender
	inviteBaseURL     string
}

func NewContactsHandler(
	cs *services.ContactService,
	inv *services.InvitationService,
	notifications services.EmailSender,
	inviteBaseURL string,
) *ContactsHandler {
	return &ContactsHandler{
		contactService:    cs,
		invitationService: inv,
		notifications:     notifications,
		inviteBaseURL:     strings.TrimRight(inviteBaseURL, "?"),
	}
}

// ScanContacts detects which of the caller's (hashed) contacts are already on
// Chorus. Raw contacts never reach the server — only SHA-256 hashes of
// normalized emails are uploaded (FR-22/23 "on-platform detect (hashed)").
// POST /api/v1/contacts/scan
func (h *ContactsHandler) ScanContacts(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		WriteError(c, middleware.ErrAuth("Unauthorized"))
		return
	}

	var req models.ContactScanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteError(c, middleware.ErrValidation("Provide at least one contact hash."))
		return
	}

	matches, err := h.contactService.ScanHashed(userID, req.Hashes)
	if err != nil {
		if errors.Is(err, services.ErrTooManyContactHashes) {
			WriteError(c, middleware.ErrValidation("Too many contact hashes in one request."))
			return
		}
		WriteError(c, middleware.ErrInternal("Failed to scan contacts."))
		return
	}

	if matches == nil {
		matches = []models.ContactMatch{}
	}
	c.JSON(http.StatusOK, gin.H{"data": matches})
}

// CreateInvite issues a single-use, expiring invite for an off-platform contact.
// Email invites are dispatched durably via the notification outbox; SMS/WhatsApp
// invites return a shareable link for the client to hand off on-device.
// POST /api/v1/contacts/invites
func (h *ContactsHandler) CreateInvite(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		WriteError(c, middleware.ErrAuth("Unauthorized"))
		return
	}

	var req models.ContactInviteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteError(c, middleware.ErrValidation("Provide a valid channel and contact."))
		return
	}

	var recipient, email string
	switch req.Channel {
	case "email":
		email = strings.ToLower(strings.TrimSpace(req.Contact.Email))
		if email == "" {
			WriteError(c, middleware.ErrValidation("An email is required for an email invite."))
			return
		}
		recipient = email
	case "sms", "whatsapp":
		recipient = strings.TrimSpace(req.Contact.Phone)
		if recipient == "" {
			WriteError(c, middleware.ErrValidation("A phone number is required for SMS/WhatsApp invites."))
			return
		}
		// Bind to the contact's email when known; otherwise the invite is open
		// and may be redeemed with whatever email the recipient registers with.
		email = strings.ToLower(strings.TrimSpace(req.Contact.Email))
	default:
		WriteError(c, middleware.ErrValidation("Channel must be email, sms, or whatsapp."))
		return
	}

	token, invite, err := h.invitationService.CreateForContact(
		userID, req.Channel, recipient, email, req.Contact.Name)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to create invitation."))
		return
	}

	link := h.inviteBaseURL + "?invite=" + token
	invite.Link = link

	// Email invites are persisted to the durable outbox so delivery is retried.
	if req.Channel == "email" {
		subject, html := services.InvitationEmail(link)
		if err := h.notifications.Send(recipient, subject, html); err != nil {
			WriteError(c, middleware.ErrInternal("Invitation created, but email delivery is pending retry."))
			return
		}
	}

	c.JSON(http.StatusCreated, gin.H{"data": invite})
}

// ListInvites returns the caller's outbound invites with a live status (pending,
// sent, redeemed, or expired) for status tracking (REQ 2.4).
// GET /api/v1/contacts/invites
func (h *ContactsHandler) ListInvites(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		WriteError(c, middleware.ErrAuth("Unauthorized"))
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	invites, err := h.invitationService.ListForInviter(userID, limit, offset)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to list invitations."))
		return
	}

	if invites == nil {
		invites = []models.ContactInvite{}
	}
	c.JSON(http.StatusOK, gin.H{"data": invites})
}
