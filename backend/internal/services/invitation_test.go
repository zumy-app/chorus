package services

import (
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestInvitationCreateForContact_Email(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	ttl := 7 * 24 * time.Hour // 168 hours
	s := NewInvitationService(db, ttl)

	query := regexp.QuoteMeta(`INSERT INTO invitations
		(waitlist_entry_id, inviter_user_id, email, token_hash, expires_at, channel, recipient, name, status, sent_at)
		VALUES (NULL, $1, $2, $3, CURRENT_TIMESTAMP + $4 * INTERVAL '1 hour', $5, $6, $7, $8,
			CASE WHEN $5 = 'email' THEN CURRENT_TIMESTAMP ELSE NULL END)
		RETURNING id, expires_at`)

	expiry := time.Now().Add(ttl)
	mock.ExpectQuery(query).
		WithArgs("inviter-1", "alice@example.com", sqlmock.AnyArg(), 168, "email", "alice@example.com", "Alice", "sent").
		WillReturnRows(sqlmock.NewRows([]string{"id", "expires_at"}).
			AddRow("invite-1", expiry))

	token, invite, err := s.CreateForContact("inviter-1", "email", "alice@example.com", "alice@example.com", "Alice")
	if err != nil {
		t.Fatalf("CreateForContact failed: %v", err)
	}
	if len(token) != 64 {
		t.Fatalf("expected 64-char token, got %d chars", len(token))
	}
	if invite.Status != "sent" {
		t.Fatalf("expected status 'sent' for email invite, got %q", invite.Status)
	}
	if invite.Recipient != "alice@example.com" {
		t.Fatalf("expected recipient alice@example.com, got %q", invite.Recipient)
	}
	if invite.ExpiresAt.IsZero() {
		t.Fatal("expected a non-zero expiry")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestInvitationCreateForContact_WhatsAppOpen(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := NewInvitationService(db, 7*24*time.Hour)

	query := regexp.QuoteMeta(`INSERT INTO invitations
		(waitlist_entry_id, inviter_user_id, email, token_hash, expires_at, channel, recipient, name, status, sent_at)
		VALUES (NULL, $1, $2, $3, CURRENT_TIMESTAMP + $4 * INTERVAL '1 hour', $5, $6, $7, $8,
			CASE WHEN $5 = 'email' THEN CURRENT_TIMESTAMP ELSE NULL END)
		RETURNING id, expires_at`)

	mock.ExpectQuery(query).
		WithArgs("inviter-1", "", sqlmock.AnyArg(), 168, "whatsapp", "+14085551234", "Alice", "pending").
		WillReturnRows(sqlmock.NewRows([]string{"id", "expires_at"}).
			AddRow("invite-2", time.Now().Add(7*24*time.Hour)))

	token, invite, err := s.CreateForContact("inviter-1", "whatsapp", "+14085551234", "", "Alice")
	if err != nil {
		t.Fatalf("CreateForContact failed: %v", err)
	}
	if len(token) != 64 {
		t.Fatalf("expected 64-char token, got %d chars", len(token))
	}
	if invite.Status != "pending" {
		t.Fatalf("expected status 'pending' for whatsapp invite, got %q", invite.Status)
	}
	if invite.Channel != "whatsapp" {
		t.Fatalf("expected channel whatsapp, got %q", invite.Channel)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestInvitationListForInviter(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := NewInvitationService(db, 7*24*time.Hour)

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
		WithArgs("inviter-1", 50, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "inviter_user_id", "channel", "recipient", "name", "status",
			"expires_at", "created_at", "redeemed_at",
		}).
			AddRow("invite-1", "inviter-1", "email", "alice@example.com", "Alice", "sent",
				time.Now().Add(24*time.Hour), time.Now(), nil).
			AddRow("invite-2", "inviter-1", "whatsapp", "+14085551234", "Bob", "pending",
				time.Now().Add(24*time.Hour), time.Now(), nil))

	invites, err := s.ListForInviter("inviter-1", 50, 0)
	if err != nil {
		t.Fatalf("ListForInviter failed: %v", err)
	}
	if len(invites) != 2 {
		t.Fatalf("expected 2 invites, got %d", len(invites))
	}
	if invites[0].Status != "sent" || invites[1].Status != "pending" {
		t.Fatalf("unexpected statuses: %+v", invites)
	}
	if invites[0].Token != "" {
		t.Fatalf("tokens must not be re-exposed in list: %q", invites[0].Token)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
