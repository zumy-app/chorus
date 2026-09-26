package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/chorus/messenger/internal/models"
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
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, chat_id, sender_id, text, COALESCE(original_language, ''), COALESCE(translations, '{}'::jsonb), delivery_status, reply_to_id, created_at, deleted_at, forwarded_from_message_id, forwarded_from_chat_id, forwarded_from_sender_id FROM messages WHERE id = $1`)).
		WithArgs("msg-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "sender_id", "text", "original_language", "translations", "delivery_status", "reply_to_id", "created_at", "deleted_at", "forwarded_from_message_id", "forwarded_from_chat_id", "forwarded_from_sender_id"}).
			AddRow("msg-1", "chat-1", "sender-1", "hola", "es", []byte("{}"), "delivered", nil, time.Now(), nil, nil, nil, nil))
	// 3. Feature settings: translation is disabled for this account.
	mock.ExpectExec(`INSERT INTO user_settings \(user_id\) VALUES \(\$1\) ON CONFLICT \(user_id\) DO NOTHING`).
		WithArgs("u1").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT ` + settingsTestColumns + ` FROM user_settings WHERE user_id = $1`)).
		WithArgs("u1").
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "grammar_enabled", "vocabulary_enabled", "difficulty_level", "transcript_recording", "message_retention_days", "translation_enabled", "grammar_auto", "highlights_enabled", "last_seen_visibility", "profile_photo_visibility", "contacts_visibility", "updated_at"}).
			AddRow("u1", true, true, "intermediate", true, 365, false, true, true, "everyone", "everyone", "everyone", time.Now()))

	body, _ := json.Marshal(map[string]string{"targetLang": "es"})
	w := serveChat(t, h.TranslateMessage, http.MethodPost, "/chats/:chatId/messages/:messageId/translate", "/chats/chat-1/messages/msg-1/translate", "u1", bytes.NewBuffer(body))

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 when translation is disabled, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMarkAsRead_WithReceiptService(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	chatService := services.NewChatService(db)
	messageService := services.NewMessageService(db, nil)
	receiptService := services.NewReceiptService(nil, nil, messageService, chatService)

	h := NewMessageHandler(messageService, chatService, nil, nil, nil, nil, nil, nil, nil)
	h.SetReceiptService(receiptService)

	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM chat_participants WHERE chat_id = \$1 AND user_id = \$2\)`).
		WithArgs("chat-1", "u1").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectExec(`UPDATE message_receipts SET read_at = COALESCE\(read_at, CURRENT_TIMESTAMP\), received_at = COALESCE\(received_at, CURRENT_TIMESTAMP\) WHERE message_id = \$1 AND user_id = \$2 AND chat_id = \$3 AND read_at IS NULL`).
		WithArgs("msg-10", "u1", "chat-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE chat_participants SET last_read_message_id = \$1 WHERE chat_id = \$2 AND user_id = \$3`).
		WithArgs("msg-10", "chat-1", "u1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	body, _ := json.Marshal(map[string]string{"messageId": "msg-10"})
	w := serveChat(t, h.MarkAsRead, http.MethodPut, "/chats/:chatId/read", "/chats/chat-1/read", "u1", bytes.NewBuffer(body))

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
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

func TestForwardMessage_NotTargetParticipant(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	h := NewMessageHandler(
		services.NewMessageService(db, nil),
		services.NewChatService(db),
		nil, nil, nil, nil, nil, nil, nil,
	)

	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM chat_participants WHERE chat_id = \$1 AND user_id = \$2\)`).
		WithArgs("chat-1", "u1").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM chat_participants WHERE chat_id = \$1 AND user_id = \$2\)`).
		WithArgs("chat-2", "u1").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	body, _ := json.Marshal(map[string]string{"targetChatId": "chat-2"})
	w := serveChat(t, h.ForwardMessage, http.MethodPost, "/chats/:chatId/messages/:messageId/forward", "/chats/chat-1/messages/msg-1/forward", "u1", bytes.NewBuffer(body))

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestForwardMessage_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	h := NewMessageHandler(
		services.NewMessageService(db, nil),
		services.NewChatService(db),
		nil, nil, nil, nil, nil, nil, nil,
	)
	translationsJSON, _ := json.Marshal(map[string]string{})

	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM chat_participants WHERE chat_id = \$1 AND user_id = \$2\)`).
		WithArgs("chat-1", "u1").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM chat_participants WHERE chat_id = \$1 AND user_id = \$2\)`).
		WithArgs("chat-2", "u1").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`SELECT text, COALESCE\(original_language, ''\), sender_id FROM messages WHERE id = \$1 AND chat_id = \$2 AND deleted_at IS NULL`).
		WithArgs("msg-1", "chat-1").
		WillReturnRows(sqlmock.NewRows([]string{"text", "original_language", "sender_id"}).
			AddRow("Hola mundo", "es", "sender-1"))
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO messages (chat_id, sender_id, text, delivery_status, original_language, forwarded_from_message_id, forwarded_from_chat_id, forwarded_from_sender_id) VALUES ($1, $2, $3, 'sent', $4, $5, $6, $7) RETURNING id, chat_id, sender_id, text, COALESCE(original_language, ''), COALESCE(translations, '{}'::jsonb), delivery_status, reply_to_id, created_at, deleted_at, forwarded_from_message_id, forwarded_from_chat_id, forwarded_from_sender_id`)).
		WithArgs("chat-2", "u1", "Hola mundo", "es", "msg-1", "chat-1", "sender-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "sender_id", "text", "original_language", "translations", "delivery_status", "reply_to_id", "created_at", "deleted_at", "forwarded_from_message_id", "forwarded_from_chat_id", "forwarded_from_sender_id"}).
			AddRow("fwd-1", "chat-2", "u1", "Hola mundo", "es", translationsJSON, "sent", nil, time.Now(), nil, "msg-1", "chat-1", "sender-1"))
	mock.ExpectQuery(`SELECT id, chat_id, user_id, role, joined_at, last_read_message_id FROM chat_participants WHERE chat_id = \$1`).
		WithArgs("chat-2").WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "user_id", "role", "joined_at", "last_read_message_id"}))

	body, _ := json.Marshal(map[string]string{"targetChatId": "chat-2"})
	w := serveChat(t, h.ForwardMessage, http.MethodPost, "/chats/:chatId/messages/:messageId/forward", "/chats/chat-1/messages/msg-1/forward", "u1", bytes.NewBuffer(body))

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestDeleteMessage_Success_Author(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	h := NewMessageHandler(
		services.NewMessageService(db, nil),
		services.NewChatService(db),
		nil, nil, nil, nil, nil, nil, nil,
	)

	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM chat_participants WHERE chat_id = \$1 AND user_id = \$2\)`).
		WithArgs("chat-1", "u1").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, chat_id, sender_id, text, COALESCE(original_language, ''), COALESCE(translations, '{}'::jsonb), delivery_status, reply_to_id, created_at, deleted_at, forwarded_from_message_id, forwarded_from_chat_id, forwarded_from_sender_id FROM messages WHERE id = $1`)).
		WithArgs("msg-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "sender_id", "text", "original_language", "translations", "delivery_status", "reply_to_id", "created_at", "deleted_at", "forwarded_from_message_id", "forwarded_from_chat_id", "forwarded_from_sender_id"}).
			AddRow("msg-1", "chat-1", "u1", "hello", "en", []byte("{}"), "sent", nil, time.Now(), nil, nil, nil, nil))
	mock.ExpectExec(`UPDATE messages SET deleted_at = COALESCE\(deleted_at, CURRENT_TIMESTAMP\) WHERE id = \$1 AND chat_id = \$2`).
		WithArgs("msg-1", "chat-1").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT id, chat_id, user_id, role, joined_at, last_read_message_id FROM chat_participants WHERE chat_id = \$1`).
		WithArgs("chat-1").WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "user_id", "role", "joined_at", "last_read_message_id"}))

	w := serveChat(t, h.DeleteMessage, http.MethodDelete, "/chats/:chatId/messages/:messageId", "/chats/chat-1/messages/msg-1", "u1", nil)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestDeleteMessage_Forbidden_NotAuthorNotAdmin(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	h := NewMessageHandler(
		services.NewMessageService(db, nil),
		services.NewChatService(db),
		nil, nil, nil, nil, nil, nil, nil,
	)

	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM chat_participants WHERE chat_id = \$1 AND user_id = \$2\)`).
		WithArgs("chat-1", "u1").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, chat_id, sender_id, text, COALESCE(original_language, ''), COALESCE(translations, '{}'::jsonb), delivery_status, reply_to_id, created_at, deleted_at, forwarded_from_message_id, forwarded_from_chat_id, forwarded_from_sender_id FROM messages WHERE id = $1`)).
		WithArgs("msg-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "sender_id", "text", "original_language", "translations", "delivery_status", "reply_to_id", "created_at", "deleted_at", "forwarded_from_message_id", "forwarded_from_chat_id", "forwarded_from_sender_id"}).
			AddRow("msg-1", "chat-1", "someone-else", "hello", "en", []byte("{}"), "sent", nil, time.Now(), nil, nil, nil, nil))
	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM chat_participants WHERE chat_id = \$1 AND user_id = \$2 AND role = 'admin'\)`).
		WithArgs("chat-1", "u1").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	w := serveChat(t, h.DeleteMessage, http.MethodDelete, "/chats/:chatId/messages/:messageId", "/chats/chat-1/messages/msg-1", "u1", nil)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestDeleteMessage_Success_Admin(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	h := NewMessageHandler(
		services.NewMessageService(db, nil),
		services.NewChatService(db),
		nil, nil, nil, nil, nil, nil, nil,
	)

	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM chat_participants WHERE chat_id = \$1 AND user_id = \$2\)`).
		WithArgs("chat-1", "u1").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, chat_id, sender_id, text, COALESCE(original_language, ''), COALESCE(translations, '{}'::jsonb), delivery_status, reply_to_id, created_at, deleted_at, forwarded_from_message_id, forwarded_from_chat_id, forwarded_from_sender_id FROM messages WHERE id = $1`)).
		WithArgs("msg-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "sender_id", "text", "original_language", "translations", "delivery_status", "reply_to_id", "created_at", "deleted_at", "forwarded_from_message_id", "forwarded_from_chat_id", "forwarded_from_sender_id"}).
			AddRow("msg-1", "chat-1", "someone-else", "hello", "en", []byte("{}"), "sent", nil, time.Now(), nil, nil, nil, nil))
	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM chat_participants WHERE chat_id = \$1 AND user_id = \$2 AND role = 'admin'\)`).
		WithArgs("chat-1", "u1").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectExec(`UPDATE messages SET deleted_at = COALESCE\(deleted_at, CURRENT_TIMESTAMP\) WHERE id = \$1 AND chat_id = \$2`).
		WithArgs("msg-1", "chat-1").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT id, chat_id, user_id, role, joined_at, last_read_message_id FROM chat_participants WHERE chat_id = \$1`).
		WithArgs("chat-1").WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "user_id", "role", "joined_at", "last_read_message_id"}))

	w := serveChat(t, h.DeleteMessage, http.MethodDelete, "/chats/:chatId/messages/:messageId", "/chats/chat-1/messages/msg-1", "u1", nil)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestPinMessage_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	h := NewMessageHandler(
		services.NewMessageService(db, nil),
		services.NewChatService(db),
		nil, nil, nil, nil, nil, nil, nil,
	)

	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM chat_participants WHERE chat_id = \$1 AND user_id = \$2\)`).
		WithArgs("chat-1", "u1").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, chat_id, sender_id, text, COALESCE(original_language, ''), COALESCE(translations, '{}'::jsonb), delivery_status, reply_to_id, created_at, deleted_at, forwarded_from_message_id, forwarded_from_chat_id, forwarded_from_sender_id FROM messages WHERE id = $1`)).
		WithArgs("msg-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "sender_id", "text", "original_language", "translations", "delivery_status", "reply_to_id", "created_at", "deleted_at", "forwarded_from_message_id", "forwarded_from_chat_id", "forwarded_from_sender_id"}).
			AddRow("msg-1", "chat-1", "sender-1", "hola", "es", []byte("{}"), "sent", nil, time.Now(), nil, nil, nil, nil))
	mock.ExpectExec(`INSERT INTO pinned_messages \(chat_id, message_id, pinned_by\) VALUES \(\$1, \$2, \$3\) ON CONFLICT \(chat_id, message_id\) DO NOTHING`).
		WithArgs("chat-1", "msg-1", "u1").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT id, chat_id, user_id, role, joined_at, last_read_message_id FROM chat_participants WHERE chat_id = \$1`).
		WithArgs("chat-1").WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "user_id", "role", "joined_at", "last_read_message_id"}))

	body, _ := json.Marshal(map[string]string{"messageId": "msg-1"})
	w := serveChat(t, h.PinMessage, http.MethodPost, "/chats/:chatId/pins", "/chats/chat-1/pins", "u1", bytes.NewBuffer(body))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestUnpinMessage_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	h := NewMessageHandler(
		services.NewMessageService(db, nil),
		services.NewChatService(db),
		nil, nil, nil, nil, nil, nil, nil,
	)

	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM chat_participants WHERE chat_id = \$1 AND user_id = \$2\)`).
		WithArgs("chat-1", "u1").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectExec(`DELETE FROM pinned_messages WHERE chat_id = \$1 AND message_id = \$2`).
		WithArgs("chat-1", "msg-1").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT id, chat_id, user_id, role, joined_at, last_read_message_id FROM chat_participants WHERE chat_id = \$1`).
		WithArgs("chat-1").WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "user_id", "role", "joined_at", "last_read_message_id"}))

	w := serveChat(t, h.UnpinMessage, http.MethodDelete, "/chats/:chatId/pins/:messageId", "/chats/chat-1/pins/msg-1", "u1", nil)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestGetPinnedMessages_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	h := NewMessageHandler(
		services.NewMessageService(db, nil),
		services.NewChatService(db),
		nil, nil, nil, nil, nil, nil, nil,
	)
	translationsJSON, _ := json.Marshal(map[string]string{})

	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM chat_participants WHERE chat_id = \$1 AND user_id = \$2\)`).
		WithArgs("chat-1", "u1").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`SELECT pm.pinned_by, pm.created_at, m.id, m.chat_id, m.sender_id, m.text, COALESCE\(m.original_language, ''\), COALESCE\(m.translations, '{}'::jsonb\), m.delivery_status, m.reply_to_id, m.created_at, m.deleted_at, m.forwarded_from_message_id, m.forwarded_from_chat_id, m.forwarded_from_sender_id FROM pinned_messages pm JOIN messages m ON m.id = pm.message_id WHERE pm.chat_id = \$1 AND m.deleted_at IS NULL ORDER BY pm.created_at DESC`).
		WithArgs("chat-1").
		WillReturnRows(sqlmock.NewRows([]string{"pinned_by", "pinned_at", "id", "chat_id", "sender_id", "text", "original_language", "translations", "delivery_status", "reply_to_id", "created_at", "deleted_at", "forwarded_from_message_id", "forwarded_from_chat_id", "forwarded_from_sender_id"}).
			AddRow("user-1", time.Now(), "msg-1", "chat-1", "sender-1", "hola", "es", translationsJSON, "sent", nil, time.Now(), nil, nil, nil, nil))

	w := serveChat(t, h.GetPinnedMessages, http.MethodGet, "/chats/:chatId/pins", "/chats/chat-1/pins", "u1", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Pins []models.PinnedMessage `json:"pins"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if len(resp.Pins) != 1 || resp.Pins[0].Message.ID != "msg-1" {
		t.Fatalf("unexpected pins: %+v", resp.Pins)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
