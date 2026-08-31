package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/chorus/messenger/internal/models"
	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
)

type mockEmailSender struct {
	gotTo      string
	gotSubject string
	gotHTML    string
	sent       bool
}

func (m *mockEmailSender) Send(to, subject, html string) error {
	m.sent = true
	m.gotTo = to
	m.gotSubject = subject
	m.gotHTML = html
	return nil
}

func withUserID(c *gin.Context, userID string) {
	c.Set("userID", userID)
}

func TestContactsScanHandler(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	contactService := services.NewContactService(db)
	h := NewContactsHandler(contactService, nil, &mockEmailSender{}, "http://localhost:3000/register")

	aliceHash := services.HashIdentifier("alice@example.com")
	bobHash := services.HashIdentifier("bob@example.com")

	query := regexp.QuoteMeta(`
		SELECT id, username, email, display_name, native_language, target_languages
		FROM users
		WHERE id != $1 AND deleted_at IS NULL AND suspended_at IS NULL`)
	mock.ExpectQuery(query).
		WithArgs("user-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "username", "email", "display_name", "native_language", "target_languages",
		}).
			AddRow("user-alice", "alice", "alice@example.com", "Alice", "en", pq.Array([]string{"es"})).
			AddRow("user-bob", "bob", "bob@example.com", "Bob", "en", pq.Array([]string{"fr"})))

	router := gin.New()
	router.POST("/api/v1/contacts/scan", func(c *gin.Context) {
		withUserID(c, "user-1")
		h.ScanContacts(c)
	})

	body, _ := json.Marshal(map[string]interface{}{
		"hashes": []string{aliceHash, bobHash, services.HashIdentifier("nobody@example.com")},
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/contacts/scan", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data []models.ContactMatch `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if len(resp.Data) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(resp.Data))
	}
	if resp.Data[0].EmailHash == "" || resp.Data[1].EmailHash == "" {
		t.Fatal("expected each match to echo its emailHash for correlation")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestContactsScanHandler_NoHashes(t *testing.T) {
	contactService := services.NewContactService(nil)
	h := NewContactsHandler(contactService, nil, &mockEmailSender{}, "http://localhost:3000/register")

	router := gin.New()
	router.POST("/api/v1/contacts/scan", func(c *gin.Context) {
		withUserID(c, "user-1")
		h.ScanContacts(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/contacts/scan", bytes.NewBuffer([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestContactsCreateInvite_Email(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	invitationService := services.NewInvitationService(db, 7*24*time.Hour)
	emailSender := &mockEmailSender{}
	h := NewContactsHandler(nil, invitationService, emailSender, "http://localhost:3000/register")

	query := regexp.QuoteMeta(`INSERT INTO invitations
		(waitlist_entry_id, inviter_user_id, email, token_hash, expires_at, channel, recipient, name, status, sent_at)
		VALUES (NULL, $1, $2, $3, CURRENT_TIMESTAMP + $4 * INTERVAL '1 hour', $5, $6, $7, $8,
			CASE WHEN $5 = 'email' THEN CURRENT_TIMESTAMP ELSE NULL END)
		RETURNING id, expires_at`)
	mock.ExpectQuery(query).
		WithArgs("user-1", "alice@example.com", sqlmock.AnyArg(), 168, "email", "alice@example.com", "Alice", "sent").
		WillReturnRows(sqlmock.NewRows([]string{"id", "expires_at"}).
			AddRow("invite-1", time.Now().Add(7*24*time.Hour)))

	router := gin.New()
	router.POST("/api/v1/contacts/invites", func(c *gin.Context) {
		withUserID(c, "user-1")
		h.CreateInvite(c)
	})

	body, _ := json.Marshal(map[string]interface{}{
		"channel": "email",
		"contact": map[string]interface{}{"name": "Alice", "email": "alice@example.com"},
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/contacts/invites", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data models.ContactInvite `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.Data.Status != "sent" {
		t.Fatalf("expected status sent, got %q", resp.Data.Status)
	}
	if resp.Data.Link == "" || !strings.Contains(resp.Data.Link, "?invite=") {
		t.Fatalf("expected shareable invite link, got %q", resp.Data.Link)
	}
	if resp.Data.Token == "" {
		t.Fatal("expected token in creation response")
	}
	if !emailSender.sent || emailSender.gotTo != "alice@example.com" {
		t.Fatalf("expected invite email sent to alice@example.com, got to=%q sent=%v", emailSender.gotTo, emailSender.sent)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestContactsCreateInvite_WhatsAppOpen(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	invitationService := services.NewInvitationService(db, 7*24*time.Hour)
	emailSender := &mockEmailSender{}
	h := NewContactsHandler(nil, invitationService, emailSender, "http://localhost:3000/register")

	query := regexp.QuoteMeta(`INSERT INTO invitations
		(waitlist_entry_id, inviter_user_id, email, token_hash, expires_at, channel, recipient, name, status, sent_at)
		VALUES (NULL, $1, $2, $3, CURRENT_TIMESTAMP + $4 * INTERVAL '1 hour', $5, $6, $7, $8,
			CASE WHEN $5 = 'email' THEN CURRENT_TIMESTAMP ELSE NULL END)
		RETURNING id, expires_at`)
	mock.ExpectQuery(query).
		WithArgs("user-1", "", sqlmock.AnyArg(), 168, "whatsapp", "+14085551234", "Alice", "pending").
		WillReturnRows(sqlmock.NewRows([]string{"id", "expires_at"}).
			AddRow("invite-2", time.Now().Add(7*24*time.Hour)))

	router := gin.New()
	router.POST("/api/v1/contacts/invites", func(c *gin.Context) {
		withUserID(c, "user-1")
		h.CreateInvite(c)
	})

	body, _ := json.Marshal(map[string]interface{}{
		"channel": "whatsapp",
		"contact": map[string]interface{}{"name": "Alice", "phone": "+14085551234"},
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/contacts/invites", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data models.ContactInvite `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.Data.Status != "pending" {
		t.Fatalf("expected status pending, got %q", resp.Data.Status)
	}
	if resp.Data.Link == "" || !strings.Contains(resp.Data.Link, "?invite=") {
		t.Fatalf("expected shareable invite link, got %q", resp.Data.Link)
	}
	if emailSender.sent {
		t.Fatal("whatsapp invite must not trigger an email send")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestContactsListInvites(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	invitationService := services.NewInvitationService(db, 7*24*time.Hour)
	h := NewContactsHandler(nil, invitationService, &mockEmailSender{}, "http://localhost:3000/register")

	query := regexp.QuoteMeta(`SELECT id, inviter_user_id, channel, recipient, COALESCE(name, ''),
			CASE
				WHEN redeemed_at IS NOT NULL THEN 'redeemed'
				WHEN expires_at < CURRENT_TIMESTAMP THEN 'expired'
				ELSE status
			END,
			expires_at, created_at, redeemed_at
		FROM invitations
		WHERE inviter_user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`)
	mock.ExpectQuery(query).
		WithArgs("user-1", 50, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "inviter_user_id", "channel", "recipient", "name", "status",
			"expires_at", "created_at", "redeemed_at",
		}).
			AddRow("invite-1", "user-1", "email", "alice@example.com", "Alice", "redeemed",
				time.Now().Add(24*time.Hour), time.Now(), time.Now()))

	router := gin.New()
	router.GET("/api/v1/contacts/invites", func(c *gin.Context) {
		withUserID(c, "user-1")
		h.ListInvites(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/contacts/invites", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data []models.ContactInvite `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 invite, got %d", len(resp.Data))
	}
	if resp.Data[0].Status != "redeemed" {
		t.Fatalf("expected status redeemed, got %q", resp.Data[0].Status)
	}
	if resp.Data[0].Token != "" {
		t.Fatalf("tokens must not be re-exposed in list: %q", resp.Data[0].Token)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
