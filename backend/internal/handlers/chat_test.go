package handlers

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
)

// newChatEnforcementTestService builds a ChatHandler whose chat + moderation
// services share one sqlmock-backed DB.
func newChatEnforcementTestService(t *testing.T) (*ChatHandler, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	chatService := services.NewChatService(db)
	userService := services.NewUserService(db)
	moderation := services.NewModerationService(db)
	return NewChatHandler(chatService, userService, moderation, nil), mock, func() { db.Close() }
}

func serveChat(t *testing.T, handler func(*gin.Context), method, routePattern, path, userID string, body io.Reader) *httptest.ResponseRecorder {
	t.Helper()
	router := setupTestRouter()
	router.Handle(method, routePattern, func(c *gin.Context) {
		c.Set("userID", userID)
		handler(c)
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(method, path, body)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	router.ServeHTTP(w, req)
	return w
}

func isParticipantRow(exists bool) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"exists"}).AddRow(exists)
}

func participantsRows(rows [][]driver.Value) *sqlmock.Rows {
	out := sqlmock.NewRows([]string{"id", "chat_id", "user_id", "role", "joined_at", "last_read_message_id"})
	for _, r := range rows {
		out.AddRow(r...)
	}
	return out
}

// --- CreateChat block enforcement ---

func TestCreateChat_BlockedDirectParticipant(t *testing.T) {
	h, mock, cleanup := newChatEnforcementTestService(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT EXISTS\(\s*SELECT 1 FROM blocked_users`).
		WithArgs("u1", "u2").WillReturnRows(isParticipantRow(true))

	body, _ := json.Marshal(map[string]interface{}{"type": "direct", "participants": []string{"u2"}})
	w := serveChat(t, h.CreateChat, http.MethodPost, "/chats", "/chats", "u1", bytes.NewBuffer(body))

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCreateChat_BlockedSelfSkipped(t *testing.T) {
	h, mock, cleanup := newChatEnforcementTestService(t)
	defer cleanup()

	// The actor is in the participant list; that entry must be skipped by the
	// enforcement loop (no IsBlocked query for self).
	mock.ExpectQuery(`SELECT EXISTS\(\s*SELECT 1 FROM blocked_users`).
		WithArgs("u1", "u2").WillReturnRows(isParticipantRow(true))

	body, _ := json.Marshal(map[string]interface{}{"type": "direct", "participants": []string{"u1", "u2"}})
	w := serveChat(t, h.CreateChat, http.MethodPost, "/chats", "/chats", "u1", bytes.NewBuffer(body))

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCreateChat_BlockCheckError(t *testing.T) {
	h, mock, cleanup := newChatEnforcementTestService(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT EXISTS\(\s*SELECT 1 FROM blocked_users`).
		WithArgs("u1", "u2").WillReturnError(errTestDB)

	body, _ := json.Marshal(map[string]interface{}{"type": "direct", "participants": []string{"u2"}})
	w := serveChat(t, h.CreateChat, http.MethodPost, "/chats", "/chats", "u1", bytes.NewBuffer(body))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// --- AddParticipant block enforcement ---

func TestAddParticipant_BlockedAgainstExistingMember(t *testing.T) {
	h, mock, cleanup := newChatEnforcementTestService(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM chat_participants WHERE chat_id = \$1 AND user_id = \$2\)`).
		WithArgs("chat-1", "u1").WillReturnRows(isParticipantRow(true))
	mock.ExpectQuery(`SELECT id, chat_id, user_id, role, joined_at, last_read_message_id FROM chat_participants WHERE chat_id = \$1`).
		WithArgs("chat-1").WillReturnRows(participantsRows([][]driver.Value{
		{"cp1", "chat-1", "p2", "member", time.Now(), nil},
	}))
	// Actor checked first (allowed), then the existing member (blocked).
	mock.ExpectQuery(`SELECT EXISTS\(\s*SELECT 1 FROM blocked_users`).
		WithArgs("u2", "u1").WillReturnRows(isParticipantRow(false))
	mock.ExpectQuery(`SELECT EXISTS\(\s*SELECT 1 FROM blocked_users`).
		WithArgs("u2", "p2").WillReturnRows(isParticipantRow(true))

	body, _ := json.Marshal(map[string]string{"userId": "u2"})
	w := serveChat(t, h.AddParticipant, http.MethodPost, "/chats/:chatId/participants", "/chats/chat-1/participants", "u1", bytes.NewBuffer(body))

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestAddParticipant_NotBlocked(t *testing.T) {
	h, mock, cleanup := newChatEnforcementTestService(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM chat_participants WHERE chat_id = \$1 AND user_id = \$2\)`).
		WithArgs("chat-1", "u1").WillReturnRows(isParticipantRow(true))
	// No existing members → only the actor is checked.
	mock.ExpectQuery(`SELECT id, chat_id, user_id, role, joined_at, last_read_message_id FROM chat_participants WHERE chat_id = \$1`).
		WithArgs("chat-1").WillReturnRows(participantsRows([][]driver.Value{}))
	mock.ExpectQuery(`SELECT EXISTS\(\s*SELECT 1 FROM blocked_users`).
		WithArgs("u2", "u1").WillReturnRows(isParticipantRow(false))
	mock.ExpectExec(`INSERT INTO chat_participants \(chat_id, user_id, role\) VALUES \(\$1, \$2, \$3\)`).
		WithArgs("chat-1", "u2", "member").WillReturnResult(sqlmock.NewResult(1, 1))

	body, _ := json.Marshal(map[string]string{"userId": "u2"})
	w := serveChat(t, h.AddParticipant, http.MethodPost, "/chats/:chatId/participants", "/chats/chat-1/participants", "u1", bytes.NewBuffer(body))

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestAddParticipant_BlockCheckError(t *testing.T) {
	h, mock, cleanup := newChatEnforcementTestService(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM chat_participants WHERE chat_id = \$1 AND user_id = \$2\)`).
		WithArgs("chat-1", "u1").WillReturnRows(isParticipantRow(true))
	mock.ExpectQuery(`SELECT id, chat_id, user_id, role, joined_at, last_read_message_id FROM chat_participants WHERE chat_id = \$1`).
		WithArgs("chat-1").WillReturnRows(participantsRows([][]driver.Value{}))
	mock.ExpectQuery(`SELECT EXISTS\(\s*SELECT 1 FROM blocked_users`).
		WithArgs("u2", "u1").WillReturnError(errTestDB)

	body, _ := json.Marshal(map[string]string{"userId": "u2"})
	w := serveChat(t, h.AddParticipant, http.MethodPost, "/chats/:chatId/participants", "/chats/chat-1/participants", "u1", bytes.NewBuffer(body))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
