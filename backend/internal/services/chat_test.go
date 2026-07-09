package services

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/chorus/messenger/internal/models"
	"github.com/DATA-DOG/go-sqlmock"
)

func TestChatCreate_Direct(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := NewChatService(db)

	// Create chat
	settings := map[string]interface{}{"translationEnabled": true}
	settingsJSON, _ := json.Marshal(settings)

	mock.ExpectQuery(`INSERT INTO chats \(type, name, created_by, settings\) VALUES \(\$1, \$2, \$3, \$4\) RETURNING id, type, name, created_by, settings, created_at`).
		WithArgs("direct", "", "creator-1", settingsJSON).
		WillReturnRows(sqlmock.NewRows([]string{"id", "type", "name", "created_by", "settings", "created_at"}).
			AddRow("chat-1", "direct", "", "creator-1", settingsJSON, time.Now()))

	// Add creator as admin
	mock.ExpectExec(`INSERT INTO chat_participants \(chat_id, user_id, role\) VALUES \(\$1, \$2, \$3\)`).
		WithArgs("chat-1", "creator-1", "admin").
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Add participant
	mock.ExpectExec(`INSERT INTO chat_participants \(chat_id, user_id, role\) VALUES \(\$1, \$2, \$3\)`).
		WithArgs("chat-1", "participant-1", "member").
		WillReturnResult(sqlmock.NewResult(2, 1))

	chat, err := s.Create("creator-1", models.CreateChatRequest{
		Type:         "direct",
		Participants: []string{"participant-1"},
		Name:         "",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if chat.ID != "chat-1" {
		t.Fatalf("expected chat-1, got %s", chat.ID)
	}
	if chat.Type != "direct" {
		t.Fatalf("expected direct, got %s", chat.Type)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestChatCreate_Group(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := NewChatService(db)

	settings := map[string]interface{}{"translationEnabled": true}
	settingsJSON, _ := json.Marshal(settings)

	mock.ExpectQuery(`INSERT INTO chats \(type, name, created_by, settings\) VALUES \(\$1, \$2, \$3, \$4\) RETURNING id, type, name, created_by, settings, created_at`).
		WithArgs("group", "Test Group", "creator-1", settingsJSON).
		WillReturnRows(sqlmock.NewRows([]string{"id", "type", "name", "created_by", "settings", "created_at"}).
			AddRow("chat-1", "group", "Test Group", "creator-1", settingsJSON, time.Now()))

	mock.ExpectExec(`INSERT INTO chat_participants \(chat_id, user_id, role\) VALUES \(\$1, \$2, \$3\)`).
		WithArgs("chat-1", "creator-1", "admin").
		WillReturnResult(sqlmock.NewResult(1, 1))

	for _, p := range []string{"p1", "p2"} {
		mock.ExpectExec(`INSERT INTO chat_participants \(chat_id, user_id, role\) VALUES \(\$1, \$2, \$3\)`).
			WithArgs("chat-1", p, "member").
			WillReturnResult(sqlmock.NewResult(1, 1))
	}

	chat, err := s.Create("creator-1", models.CreateChatRequest{
		Type:         "group",
		Participants: []string{"p1", "p2"},
		Name:         "Test Group",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if chat.Name != "Test Group" {
		t.Fatalf("expected 'Test Group', got '%s'", chat.Name)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestChatGetByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := NewChatService(db)

	settings := map[string]interface{}{"translationEnabled": true}
	settingsJSON, _ := json.Marshal(settings)

	mock.ExpectQuery(`SELECT id, type, name, created_by, settings, created_at FROM chats WHERE id = \$1`).
		WithArgs("chat-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "type", "name", "created_by", "settings", "created_at"}).
			AddRow("chat-1", "direct", "", "creator-1", settingsJSON, time.Now()))

	chat, err := s.GetByID("chat-1")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if chat.ID != "chat-1" {
		t.Fatalf("expected chat-1, got %s", chat.ID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestChatGetUserChats(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := NewChatService(db)

	settings := map[string]interface{}{"translationEnabled": true}
	settingsJSON, _ := json.Marshal(settings)

	mock.ExpectQuery(`SELECT DISTINCT c.id, c.type, c.name, c.created_by, c.settings, c.created_at FROM chats c INNER JOIN chat_participants cp ON c.id = cp.chat_id WHERE cp.user_id = \$1 ORDER BY c.created_at DESC`).
		WithArgs("user-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "type", "name", "created_by", "settings", "created_at"}).
			AddRow("chat-1", "direct", "", "creator-1", settingsJSON, time.Now()).
			AddRow("chat-2", "group", "Group Chat", "creator-1", settingsJSON, time.Now()))

	chats, err := s.GetUserChats("user-1")
	if err != nil {
		t.Fatalf("GetUserChats failed: %v", err)
	}
	if len(chats) != 2 {
		t.Fatalf("expected 2 chats, got %d", len(chats))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestChatGetParticipants(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := NewChatService(db)

	mock.ExpectQuery(`SELECT id, chat_id, user_id, role, joined_at, last_read_message_id FROM chat_participants WHERE chat_id = \$1`).
		WithArgs("chat-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "user_id", "role", "joined_at", "last_read_message_id"}).
			AddRow("cp-1", "chat-1", "user-1", "admin", time.Now(), nil).
			AddRow("cp-2", "chat-1", "user-2", "member", time.Now(), nil))

	participants, err := s.GetParticipants("chat-1")
	if err != nil {
		t.Fatalf("GetParticipants failed: %v", err)
	}
	if len(participants) != 2 {
		t.Fatalf("expected 2 participants, got %d", len(participants))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestFindDirectChat_Exists(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := NewChatService(db)

	settings := map[string]interface{}{"translationEnabled": true}
	settingsJSON, _ := json.Marshal(settings)

	mock.ExpectQuery(`SELECT c.id, c.type, c.name, c.created_by, c.settings, c.created_at FROM chats c INNER JOIN chat_participants cp1 ON c.id = cp1.chat_id AND cp1.user_id = \$1 INNER JOIN chat_participants cp2 ON c.id = cp2.chat_id AND cp2.user_id = \$2 WHERE c.type = 'direct' LIMIT 1`).
		WithArgs("user-1", "user-2").
		WillReturnRows(sqlmock.NewRows([]string{"id", "type", "name", "created_by", "settings", "created_at"}).
			AddRow("existing-chat", "direct", "", "user-1", settingsJSON, time.Now()))

	chat, err := s.FindDirectChat("user-1", "user-2")
	if err != nil {
		t.Fatalf("FindDirectChat failed: %v", err)
	}
	if chat.ID != "existing-chat" {
		t.Fatalf("expected existing-chat, got %s", chat.ID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestFindDirectChat_NotExists(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := NewChatService(db)

	mock.ExpectQuery(`SELECT c.id, c.type, c.name, c.created_by, c.settings, c.created_at FROM chats c INNER JOIN chat_participants cp1 ON c.id = cp1.chat_id AND cp1.user_id = \$1 INNER JOIN chat_participants cp2 ON c.id = cp2.chat_id AND cp2.user_id = \$2 WHERE c.type = 'direct' LIMIT 1`).
		WithArgs("user-1", "user-2").
		WillReturnError(sqlmock.ErrCancelled)

	_, err = s.FindDirectChat("user-1", "user-2")
	if err == nil {
		t.Fatal("expected error when direct chat not found")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
