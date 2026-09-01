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
	return NewChatHandler(chatService, userService, moderation, services.NewWebSocketHub(nil)), mock, func() { db.Close() }
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

// --- Archive & mute (task 6.4) ---

func TestArchiveChat_Archives(t *testing.T) {
	h, mock, cleanup := newChatEnforcementTestService(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM chat_participants WHERE chat_id = \$1 AND user_id = \$2\)`).
		WithArgs("chat-1", "u1").WillReturnRows(isParticipantRow(true))
	mock.ExpectQuery(`INSERT INTO chat_preferences \(user_id, chat_id, archived_at\) VALUES \(\$1, \$2, CASE WHEN \$3 THEN CURRENT_TIMESTAMP ELSE NULL END\) ON CONFLICT \(user_id, chat_id\) DO UPDATE SET archived_at = EXCLUDED.archived_at, updated_at = CURRENT_TIMESTAMP RETURNING user_id, chat_id, archived_at, is_muted, muted_until, updated_at`).
		WithArgs("u1", "chat-1", true).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "chat_id", "archived_at", "is_muted", "muted_until", "updated_at"}).
			AddRow("u1", "chat-1", time.Now(), false, nil, time.Now()))

	w := serveChat(t, h.ArchiveChat, http.MethodPost, "/chats/:chatId/archive", "/chats/chat-1/archive", "u1", bytes.NewBufferString(`{"archived":true}`))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp["userId"] != "u1" || resp["chatId"] != "chat-1" {
		t.Fatalf("unexpected preference response: %v", resp)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestUnarchiveChat_Unarchives(t *testing.T) {
	h, mock, cleanup := newChatEnforcementTestService(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM chat_participants WHERE chat_id = \$1 AND user_id = \$2\)`).
		WithArgs("chat-1", "u1").WillReturnRows(isParticipantRow(true))
	mock.ExpectQuery(`INSERT INTO chat_preferences \(user_id, chat_id, archived_at\) VALUES \(\$1, \$2, CASE WHEN \$3 THEN CURRENT_TIMESTAMP ELSE NULL END\) ON CONFLICT \(user_id, chat_id\) DO UPDATE SET archived_at = EXCLUDED.archived_at, updated_at = CURRENT_TIMESTAMP RETURNING user_id, chat_id, archived_at, is_muted, muted_until, updated_at`).
		WithArgs("u1", "chat-1", false).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "chat_id", "archived_at", "is_muted", "muted_until", "updated_at"}).
			AddRow("u1", "chat-1", nil, false, nil, time.Now()))

	w := serveChat(t, h.UnarchiveChat, http.MethodDelete, "/chats/:chatId/archive", "/chats/chat-1/archive", "u1", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMuteChat_MutesUntil(t *testing.T) {
	h, mock, cleanup := newChatEnforcementTestService(t)
	defer cleanup()

	until := time.Now().Add(8 * time.Hour).UTC().Truncate(time.Millisecond)

	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM chat_participants WHERE chat_id = \$1 AND user_id = \$2\)`).
		WithArgs("chat-1", "u1").WillReturnRows(isParticipantRow(true))
	mock.ExpectQuery(`INSERT INTO chat_preferences \(user_id, chat_id, is_muted, muted_until\) VALUES \(\$1, \$2, \$3, \$4\) ON CONFLICT \(user_id, chat_id\) DO UPDATE SET is_muted = EXCLUDED.is_muted, muted_until = EXCLUDED.muted_until, updated_at = CURRENT_TIMESTAMP RETURNING user_id, chat_id, archived_at, is_muted, muted_until, updated_at`).
		WithArgs("u1", "chat-1", true, until).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "chat_id", "archived_at", "is_muted", "muted_until", "updated_at"}).
			AddRow("u1", "chat-1", nil, true, until, time.Now()))

	body, _ := json.Marshal(map[string]interface{}{"muted": true, "until": until})
	w := serveChat(t, h.MuteChat, http.MethodPost, "/chats/:chatId/mute", "/chats/chat-1/mute", "u1", bytes.NewBuffer(body))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestUnmuteChat_Unmutes(t *testing.T) {
	h, mock, cleanup := newChatEnforcementTestService(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM chat_participants WHERE chat_id = \$1 AND user_id = \$2\)`).
		WithArgs("chat-1", "u1").WillReturnRows(isParticipantRow(true))
	mock.ExpectQuery(`INSERT INTO chat_preferences \(user_id, chat_id, is_muted, muted_until\) VALUES \(\$1, \$2, \$3, \$4\) ON CONFLICT \(user_id, chat_id\) DO UPDATE SET is_muted = EXCLUDED.is_muted, muted_until = EXCLUDED.muted_until, updated_at = CURRENT_TIMESTAMP RETURNING user_id, chat_id, archived_at, is_muted, muted_until, updated_at`).
		WithArgs("u1", "chat-1", false, nil).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "chat_id", "archived_at", "is_muted", "muted_until", "updated_at"}).
			AddRow("u1", "chat-1", time.Now(), false, nil, time.Now()))

	w := serveChat(t, h.UnmuteChat, http.MethodDelete, "/chats/:chatId/mute", "/chats/chat-1/mute", "u1", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestGetChatPreference_Default(t *testing.T) {
	h, mock, cleanup := newChatEnforcementTestService(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM chat_participants WHERE chat_id = \$1 AND user_id = \$2\)`).
		WithArgs("chat-1", "u1").WillReturnRows(isParticipantRow(true))
	mock.ExpectQuery(`SELECT user_id, chat_id, archived_at, is_muted, muted_until, updated_at FROM chat_preferences WHERE user_id = \$1 AND chat_id = \$2`).
		WithArgs("u1", "chat-1").WillReturnError(errTestDB)

	w := serveChat(t, h.GetChatPreference, http.MethodGet, "/chats/:chatId/preferences", "/chats/chat-1/preferences", "u1", nil)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestArchiveChat_NonParticipant(t *testing.T) {
	h, mock, cleanup := newChatEnforcementTestService(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM chat_participants WHERE chat_id = \$1 AND user_id = \$2\)`).
		WithArgs("chat-1", "u1").WillReturnRows(isParticipantRow(false))

	w := serveChat(t, h.ArchiveChat, http.MethodPost, "/chats/:chatId/archive", "/chats/chat-1/archive", "u1", bytes.NewBufferString(`{"archived":true}`))

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
