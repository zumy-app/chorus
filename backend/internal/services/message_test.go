package services

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestMessageCreate(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := NewMessageService(db, nil)

	translationsJSON, _ := json.Marshal(map[string]string{})

	mock.ExpectQuery(`INSERT INTO messages \(chat_id, sender_id, text, delivery_status, reply_to_id\) VALUES \(\$1, \$2, \$3, 'sent', \$4\) RETURNING id, chat_id, sender_id, text, COALESCE\(original_language, ''\), COALESCE\(translations, '{}'::jsonb\), delivery_status, reply_to_id, created_at`).
		WithArgs("chat-1", "user-1", "Hello, World!", nil).
		WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "sender_id", "text", "original_language", "translations", "delivery_status", "reply_to_id", "created_at"}).
			AddRow("msg-1", "chat-1", "user-1", "Hello, World!", "en", translationsJSON, "sent", nil, time.Now()))

	msg, err := s.Create("chat-1", "user-1", "Hello, World!", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if msg.ID != "msg-1" {
		t.Fatalf("expected msg-1, got %s", msg.ID)
	}
	if msg.Text != "Hello, World!" {
		t.Fatalf("expected 'Hello, World!', got '%s'", msg.Text)
	}
	if msg.DeliveryStatus != "sent" {
		t.Fatalf("expected 'sent', got '%s'", msg.DeliveryStatus)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMessageCreate_WithReply(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := NewMessageService(db, nil)
	replyToID := "original-msg-1"
	translationsJSON, _ := json.Marshal(map[string]string{})

	mock.ExpectQuery(`INSERT INTO messages \(chat_id, sender_id, text, delivery_status, reply_to_id\) VALUES \(\$1, \$2, \$3, 'sent', \$4\) RETURNING id, chat_id, sender_id, text, COALESCE\(original_language, ''\), COALESCE\(translations, '{}'::jsonb\), delivery_status, reply_to_id, created_at`).
		WithArgs("chat-1", "user-1", "Reply message", &replyToID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "sender_id", "text", "original_language", "translations", "delivery_status", "reply_to_id", "created_at"}).
			AddRow("msg-2", "chat-1", "user-1", "Reply message", "en", translationsJSON, "sent", &replyToID, time.Now()))

	msg, err := s.Create("chat-1", "user-1", "Reply message", &replyToID)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if msg.ReplyToID == nil || *msg.ReplyToID != "original-msg-1" {
		t.Fatalf("expected reply to original-msg-1")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMessageGetMessages(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := NewMessageService(db, nil)
	translationsJSON, _ := json.Marshal(map[string]string{})

	mock.ExpectQuery(`SELECT id, chat_id, sender_id, text, COALESCE\(original_language, ''\), COALESCE\(translations, '{}'::jsonb\), delivery_status, reply_to_id, created_at FROM messages WHERE chat_id = \$1 ORDER BY created_at DESC LIMIT \$2`).
		WithArgs("chat-1", 50).
		WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "sender_id", "text", "original_language", "translations", "delivery_status", "reply_to_id", "created_at"}).
			AddRow("msg-1", "chat-1", "user-1", "Hello", "en", translationsJSON, "sent", nil, time.Now()).
			AddRow("msg-2", "chat-1", "user-2", "Hi back", "es", translationsJSON, "sent", nil, time.Now()))

	messages, err := s.GetMessages("chat-1", 50, nil)
	if err != nil {
		t.Fatalf("GetMessages failed: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}
	if messages[0].Text != "Hello" {
		t.Fatalf("expected 'Hello', got '%s'", messages[0].Text)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMessageGetMessages_BeforeCursor(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := NewMessageService(db, nil)
	translationsJSON, _ := json.Marshal(map[string]string{})
	before := "msg-10"

	mock.ExpectQuery(`SELECT id, chat_id, sender_id, text, COALESCE\(original_language, ''\), COALESCE\(translations, '{}'::jsonb\), delivery_status, reply_to_id, created_at FROM messages WHERE chat_id = \$1 AND created_at < \(SELECT created_at FROM messages WHERE id = \$2\) ORDER BY created_at DESC LIMIT \$3`).
		WithArgs("chat-1", before, 20).
		WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "sender_id", "text", "original_language", "translations", "delivery_status", "reply_to_id", "created_at"}).
			AddRow("msg-1", "chat-1", "user-1", "Older message", "en", translationsJSON, "sent", nil, time.Now()))

	messages, err := s.GetMessages("chat-1", 20, &before)
	if err != nil {
		t.Fatalf("GetMessages failed: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMessageGetMessageByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := NewMessageService(db, nil)
	translationsJSON, _ := json.Marshal(map[string]string{})

	mock.ExpectQuery(`SELECT id, chat_id, sender_id, text, COALESCE\(original_language, ''\), COALESCE\(translations, '{}'::jsonb\), delivery_status, reply_to_id, created_at FROM messages WHERE id = \$1`).
		WithArgs("msg-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "sender_id", "text", "original_language", "translations", "delivery_status", "reply_to_id", "created_at"}).
			AddRow("msg-1", "chat-1", "user-1", "Hello", "en", translationsJSON, "sent", nil, time.Now()))

	msg, err := s.GetMessageByID(context.Background(), "msg-1")
	if err != nil {
		t.Fatalf("GetMessageByID failed: %v", err)
	}
	if msg == nil {
		t.Fatal("expected non-nil message")
	}
	if msg.ID != "msg-1" {
		t.Fatalf("expected msg-1, got %s", msg.ID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMessageGetMessageByID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := NewMessageService(db, nil)

	mock.ExpectQuery(`SELECT id, chat_id, sender_id, text, COALESCE\(original_language, ''\), COALESCE\(translations, '{}'::jsonb\), delivery_status, reply_to_id, created_at FROM messages WHERE id = \$1`).
		WithArgs("nonexistent").
		WillReturnError(sqlmock.ErrCancelled)

	msg, err := s.GetMessageByID(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent message")
	}
	if msg != nil {
		t.Fatal("expected nil message for error")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMessageUpdateTranslations(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := NewMessageService(db, nil)

	translations := map[string]string{"es": "Hola", "fr": "Bonjour"}

	mock.ExpectExec(`UPDATE messages SET translations = \$1 WHERE id = \$2`).
		WithArgs(sqlmock.AnyArg(), "msg-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = s.UpdateTranslations("msg-1", translations)
	if err != nil {
		t.Fatalf("UpdateTranslations failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMessageMarkAsRead(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := NewMessageService(db, nil)

	mock.ExpectExec(`UPDATE chat_participants SET last_read_message_id = \$1 WHERE chat_id = \$2 AND user_id = \$3`).
		WithArgs("msg-10", "chat-1", "user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = s.MarkAsRead("chat-1", "user-1", "msg-10")
	if err != nil {
		t.Fatalf("MarkAsRead failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
