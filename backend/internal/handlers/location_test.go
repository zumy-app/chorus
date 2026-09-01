package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/chorus/messenger/internal/services"
)

// newLocationTestHandler builds a LocationHandler backed by a sqlmock DB shared
// by the chat + message + location services (task 6.7). The inbox service is
// left nil (offline queueing is a no-op under test) and the WS hub has no
// connections, so the real-time fan-out is a harmless no-op.
func newLocationTestHandler(t *testing.T) (*LocationHandler, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	chatService := services.NewChatService(db)
	messageService := services.NewMessageService(db, nil)
	locationService := services.NewLocationService(db)
	h := NewLocationHandler(locationService, chatService, messageService, nil, services.NewWebSocketHub(nil))
	return h, mock, func() { db.Close() }
}

// TestSendLocation_Success verifies the full share path (task 6.7): a valid
// lat/lng for an authenticated participant creates the message + location
// attachment, seeds a 'sent' receipt for the co-participant, and returns 201
// with the message carrying the populated pin.
func TestSendLocation_Success(t *testing.T) {
	h, mock, cleanup := newLocationTestHandler(t)
	defer cleanup()

	// 1. Sender is a participant of the chat.
	mock.ExpectQuery(`SELECT EXISTS\(\s*SELECT 1 FROM chat_participants`).
		WithArgs("chat-9", "user-1").WillReturnRows(isParticipantRow(true))
	// 2. Persist the message + location attachment atomically.
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO messages`).
		WithArgs("chat-9", "user-1", "NYC").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "chat_id", "sender_id", "text", "original_language",
			"translations", "delivery_status", "reply_to_id", "created_at",
		}).AddRow("msg-loc", "chat-9", "user-1", "NYC", "", "{}", "sent", nil, time.Now()))
	mock.ExpectExec(`INSERT INTO media_attachments`).
		WithArgs(sqlmock.AnyArg(), "msg-loc", "location", "location", int64(0),
			"application/vnd.chorus.location", sqlmock.AnyArg(), 40.7128, -74.0060, "NYC").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	// 3. Fan-out: load participants, then seed the co-participant's 'sent' receipt.
	mock.ExpectQuery(`SELECT id, chat_id, user_id, role, joined_at, last_read_message_id FROM chat_participants WHERE chat_id = \$1`).
		WithArgs("chat-9").
		WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "user_id", "role", "joined_at", "last_read_message_id"}).
			AddRow("cp-1", "chat-9", "user-2", "member", time.Now(), nil))
	mock.ExpectExec(`INSERT INTO message_receipts`).
		WithArgs("msg-loc", "user-2", "chat-9").
		WillReturnResult(sqlmock.NewResult(1, 1))

	body, _ := json.Marshal(map[string]interface{}{
		"latitude":  40.7128,
		"longitude": -74.0060,
		"label":     "NYC",
	})
	w := serveChat(t, h.SendLocation, http.MethodPost, "/chats/:chatId/location", "/chats/chat-9/location", "user-1", bytes.NewBuffer(body))

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if body := w.Body.String(); body == "" || !bytes.Contains([]byte(body), []byte("\"location\"")) {
		t.Fatalf("expected a location message payload, got %s", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestSendLocation_Unauthorized ensures a missing userID is rejected before any
// DB access.
func TestSendLocation_Unauthorized(t *testing.T) {
	h, _, cleanup := newLocationTestHandler(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]interface{}{"latitude": 40.7128, "longitude": -74.0060})
	w := serveChat(t, h.SendLocation, http.MethodPost, "/chats/:chatId/location", "/chats/chat-9/location", "", bytes.NewBuffer(body))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

// TestSendLocation_NotParticipant ensures a non-participant is rejected with 403.
func TestSendLocation_NotParticipant(t *testing.T) {
	h, mock, cleanup := newLocationTestHandler(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT EXISTS\(\s*SELECT 1 FROM chat_participants`).
		WithArgs("chat-9", "stranger").WillReturnRows(isParticipantRow(false))

	body, _ := json.Marshal(map[string]interface{}{"latitude": 40.7128, "longitude": -74.0060})
	w := serveChat(t, h.SendLocation, http.MethodPost, "/chats/:chatId/location", "/chats/chat-9/location", "stranger", bytes.NewBuffer(body))
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestSendLocation_MissingChatID ensures a missing chatId is a validation error.
func TestSendLocation_MissingChatID(t *testing.T) {
	h, _, cleanup := newLocationTestHandler(t)
	defer cleanup()

	// Mount without a :chatId segment so the guard branch is exercised.
	body, _ := json.Marshal(map[string]interface{}{"latitude": 40.7128, "longitude": -74.0060})
	w := serveChat(t, h.SendLocation, http.MethodPost, "/location", "/location", "user-1", bytes.NewBuffer(body))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// TestSendLocation_InvalidBinding ensures a request missing a required coordinate
// is rejected with 400 (no DB write happens).
func TestSendLocation_InvalidBinding(t *testing.T) {
	h, mock, cleanup := newLocationTestHandler(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT EXISTS\(\s*SELECT 1 FROM chat_participants`).
		WithArgs("chat-9", "user-1").WillReturnRows(isParticipantRow(true))

	body, _ := json.Marshal(map[string]interface{}{"longitude": -74.0060})
	w := serveChat(t, h.SendLocation, http.MethodPost, "/chats/:chatId/location", "/chats/chat-9/location", "user-1", bytes.NewBuffer(body))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
