package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/chorus/messenger/internal/services"
)

// newMessageEnforcementTestService builds a MessageHandler with shared
// sqlmock-backed chat + moderation services; the message/entitlement/translation
// services and WS hub are unused by the enforcement branches under test.
func newMessageEnforcementTestService(t *testing.T) (*MessageHandler, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	chatService := services.NewChatService(db)
	moderation := services.NewModerationService(db)
	h := NewMessageHandler(nil, chatService, nil, nil, nil, nil, moderation, nil, nil)
	return h, mock, func() { db.Close() }
}

func TestSendMessage_BlockedByParticipant(t *testing.T) {
	h, mock, cleanup := newMessageEnforcementTestService(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM chat_participants WHERE chat_id = \$1 AND user_id = \$2\)`).
		WithArgs("chat-1", "u1").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`SELECT EXISTS\(\s*SELECT 1\s*FROM blocked_users b\s*INNER JOIN chat_participants`).
		WithArgs("chat-1", "u1").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	body, _ := json.Marshal(map[string]string{"text": "hello"})
	w := serveChat(t, h.SendMessage, http.MethodPost, "/chats/:chatId/messages", "/chats/chat-1/messages", "u1", bytes.NewBuffer(body))

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestSendMessage_BlockCheckError(t *testing.T) {
	h, mock, cleanup := newMessageEnforcementTestService(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM chat_participants WHERE chat_id = \$1 AND user_id = \$2\)`).
		WithArgs("chat-1", "u1").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`SELECT EXISTS\(\s*SELECT 1\s*FROM blocked_users b\s*INNER JOIN chat_participants`).
		WithArgs("chat-1", "u1").WillReturnError(errTestDB)

	body, _ := json.Marshal(map[string]string{"text": "hello"})
	w := serveChat(t, h.SendMessage, http.MethodPost, "/chats/:chatId/messages", "/chats/chat-1/messages", "u1", bytes.NewBuffer(body))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestTranslateMessage_TranslationDisabled verifies the FR-25 acceptance
// criterion "toggle off -> no translation job is enqueued". A viewer whose
// per-account translation_enabled is off gets an explicit 403 BEFORE the
// manual translate enqueue path is reached, so no translation job is created.
func TestTranslateMessage_TranslationDisabled(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	h := NewMessageHandler(
		services.NewMessageService(db, nil),
		services.NewChatService(db),
		nil,
		nil,
		nil,
		services.NewSettingsService(db),
		nil,
		nil,
		nil,
	)

	// 1. Viewer is a participant of the chat.
	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM chat_participants WHERE chat_id = \$1 AND user_id = \$2\)`).
		WithArgs("chat-1", "u1").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	// 2. The target message exists and has no prior translation for "es".
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, chat_id, sender_id, text, COALESCE(original_language, ''), COALESCE(translations, '{}'::jsonb), delivery_status, reply_to_id, created_at FROM messages WHERE id = $1`)).
		WithArgs("msg-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "sender_id", "text", "original_language", "translations", "delivery_status", "reply_to_id", "created_at"}).
			AddRow("msg-1", "chat-1", "sender-1", "hola", "es", []byte("{}"), "delivered", nil, time.Now()))
	// 3. Feature settings: translation is disabled for this account.
	mock.ExpectExec(`INSERT INTO user_settings \(user_id\) VALUES \(\$1\) ON CONFLICT \(user_id\) DO NOTHING`).
		WithArgs("u1").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT ` + settingsTestColumns + ` FROM user_settings WHERE user_id = $1`)).
		WithArgs("u1").
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "grammar_enabled", "vocabulary_enabled", "difficulty_level", "transcript_recording", "message_retention_days", "translation_enabled", "grammar_auto", "highlights_enabled", "updated_at"}).
			AddRow("u1", true, true, "intermediate", true, 365, false, true, true, time.Now()))

	body, _ := json.Marshal(map[string]string{"targetLang": "es"})
	w := serveChat(t, h.TranslateMessage, http.MethodPost, "/chats/:chatId/messages/:messageId/translate", "/chats/chat-1/messages/msg-1/translate", "u1", bytes.NewBuffer(body))

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 when translation is disabled, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestSendMessage_NotParticipant(t *testing.T) {
	h, mock, cleanup := newMessageEnforcementTestService(t)
	defer cleanup()

	// The participation check short-circuits before the block check, so no
	// blocked_users query is expected.
	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM chat_participants WHERE chat_id = \$1 AND user_id = \$2\)`).
		WithArgs("chat-1", "stranger").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	body, _ := json.Marshal(map[string]string{"text": "hello"})
	w := serveChat(t, h.SendMessage, http.MethodPost, "/chats/:chatId/messages", "/chats/chat-1/messages", "stranger", bytes.NewBuffer(body))

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
