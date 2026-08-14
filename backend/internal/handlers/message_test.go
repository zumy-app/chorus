package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

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
	h := NewMessageHandler(nil, chatService, nil, nil, nil, moderation, nil, nil)
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
