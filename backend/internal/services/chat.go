package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/chorus/messenger/internal/models"
	"github.com/lib/pq"
)

type ChatService struct {
	db *sql.DB
}

func NewChatService(db *sql.DB) *ChatService {
	return &ChatService{db: db}
}

func (s *ChatService) Create(createdBy string, req models.CreateChatRequest) (*models.Chat, error) {
	// Validate participants
	if req.Type == "direct" && len(req.Participants) != 1 {
		return nil, errors.New("direct chat must have exactly 1 participant (besides creator)")
	}

	// Check if direct chat already exists
	if req.Type == "direct" {
		existingChat, err := s.FindDirectChat(createdBy, req.Participants[0])
		if err == nil && existingChat != nil {
			return existingChat, nil
		}
	}

	// Create chat
	chat := &models.Chat{}
	settings := map[string]interface{}{"translationEnabled": true}
	settingsJSON, _ := json.Marshal(settings)

	query := `
		INSERT INTO chats (type, name, created_by, settings)
		VALUES ($1, $2, $3, $4)
		RETURNING id, type, name, created_by, settings, created_at
	`

	var settingsBytes []byte
	err := s.db.QueryRow(query, req.Type, req.Name, createdBy, settingsJSON).Scan(
		&chat.ID,
		&chat.Type,
		&chat.Name,
		&chat.CreatedBy,
		&settingsBytes,
		&chat.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	json.Unmarshal(settingsBytes, &chat.Settings)

	// Add creator as admin
	if err := s.AddParticipant(chat.ID, createdBy, "admin"); err != nil {
		return nil, err
	}

	// Add other participants
	for _, participantID := range req.Participants {
		if participantID != createdBy {
			if err := s.AddParticipant(chat.ID, participantID, "member"); err != nil {
				continue
			}
		}
	}

	return chat, nil
}

func (s *ChatService) FindDirectChat(user1ID, user2ID string) (*models.Chat, error) {
	query := `
		SELECT c.id, c.type, c.name, c.created_by, c.settings, c.created_at
		FROM chats c
		INNER JOIN chat_participants cp1 ON c.id = cp1.chat_id AND cp1.user_id = $1
		INNER JOIN chat_participants cp2 ON c.id = cp2.chat_id AND cp2.user_id = $2
		WHERE c.type = 'direct'
		LIMIT 1
	`

	chat := &models.Chat{}
	var settingsBytes []byte

	err := s.db.QueryRow(query, user1ID, user2ID).Scan(
		&chat.ID,
		&chat.Type,
		&chat.Name,
		&chat.CreatedBy,
		&settingsBytes,
		&chat.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	json.Unmarshal(settingsBytes, &chat.Settings)
	return chat, nil
}

func (s *ChatService) GetByID(chatID string) (*models.Chat, error) {
	chat := &models.Chat{}
	query := `
		SELECT id, type, name, created_by, settings, created_at
		FROM chats
		WHERE id = $1
	`

	var settingsBytes []byte
	err := s.db.QueryRow(query, chatID).Scan(
		&chat.ID,
		&chat.Type,
		&chat.Name,
		&chat.CreatedBy,
		&settingsBytes,
		&chat.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	json.Unmarshal(settingsBytes, &chat.Settings)
	return chat, nil
}

func (s *ChatService) GetUserChats(userID string) ([]models.Chat, error) {
	query := `
		SELECT DISTINCT c.id, c.type, c.name, c.created_by, c.settings, c.created_at,
		       cp.archived_at IS NOT NULL, COALESCE(cp.is_muted, FALSE), cp.muted_until
		FROM chats c
		INNER JOIN chat_participants cp ON c.id = cp.chat_id
		LEFT JOIN chat_preferences pref ON pref.chat_id = c.id AND pref.user_id = $1
		WHERE cp.user_id = $1
		ORDER BY c.created_at DESC
	`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	chats := []models.Chat{}
	for rows.Next() {
		chat := models.Chat{}
		var settingsBytes []byte

		err := rows.Scan(
			&chat.ID,
			&chat.Type,
			&chat.Name,
			&chat.CreatedBy,
			&settingsBytes,
			&chat.CreatedAt,
			&chat.IsArchived,
			&chat.IsMuted,
			&chat.MutedUntil,
		)

		if err != nil {
			continue
		}

		json.Unmarshal(settingsBytes, &chat.Settings)
		chats = append(chats, chat)
	}

	return chats, nil
}

func (s *ChatService) GetParticipants(chatID string) ([]models.ChatParticipant, error) {
	query := `
		SELECT id, chat_id, user_id, role, joined_at, last_read_message_id
		FROM chat_participants
		WHERE chat_id = $1
	`

	rows, err := s.db.Query(query, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	participants := []models.ChatParticipant{}
	for rows.Next() {
		p := models.ChatParticipant{}
		err := rows.Scan(&p.ID, &p.ChatID, &p.UserID, &p.Role, &p.JoinedAt, &p.LastReadMessageID)
		if err != nil {
			continue
		}
		participants = append(participants, p)
	}

	return participants, nil
}

func (s *ChatService) AddParticipant(chatID, userID, role string) error {
	query := `
		INSERT INTO chat_participants (chat_id, user_id, role)
		VALUES ($1, $2, $3)
		ON CONFLICT (chat_id, user_id) DO NOTHING
	`

	_, err := s.db.Exec(query, chatID, userID, role)
	return err
}

func (s *ChatService) RemoveParticipant(chatID, userID string) error {
	query := `DELETE FROM chat_participants WHERE chat_id = $1 AND user_id = $2`
	_, err := s.db.Exec(query, chatID, userID)
	return err
}

func (s *ChatService) UpdateChat(chatID string, name *string, settings map[string]interface{}) (*models.Chat, error) {
	settingsJSON, _ := json.Marshal(settings)

	query := `
		UPDATE chats
		SET name = COALESCE($2, name),
		    settings = COALESCE($3, settings)
		WHERE id = $1
		RETURNING id, type, name, created_by, settings, created_at
	`

	chat := &models.Chat{}
	var settingsBytes []byte

	err := s.db.QueryRow(query, chatID, name, settingsJSON).Scan(
		&chat.ID,
		&chat.Type,
		&chat.Name,
		&chat.CreatedBy,
		&settingsBytes,
		&chat.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	json.Unmarshal(settingsBytes, &chat.Settings)
	return chat, nil
}

func (s *ChatService) IsParticipant(chatID, userID string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM chat_participants WHERE chat_id = $1 AND user_id = $2)`
	err := s.db.QueryRow(query, chatID, userID).Scan(&exists)
	return exists, err
}

// IsChatAdmin reports whether the user holds the 'admin' role in the chat
// (used to authorize message deletion, which the sender OR a chat admin may do).
func (s *ChatService) IsChatAdmin(chatID, userID string) (bool, error) {
	var isAdmin bool
	query := `SELECT EXISTS(SELECT 1 FROM chat_participants WHERE chat_id = $1 AND user_id = $2 AND role = 'admin')`
	err := s.db.QueryRow(query, chatID, userID).Scan(&isAdmin)
	return isAdmin, err
}

func (s *ChatService) GetParticipantLanguages(chatID string) (map[string][]string, error) {
	query := `
		SELECT u.id, u.native_language, u.target_languages
		FROM users u
		INNER JOIN chat_participants cp ON u.id = cp.user_id
		WHERE cp.chat_id = $1
	`

	rows, err := s.db.Query(query, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	languages := make(map[string][]string)
	for rows.Next() {
		var userID string
		var nativeLanguage string
		var targetLangs []string
		err := rows.Scan(&userID, &nativeLanguage, pq.Array(&targetLangs))
		if err != nil {
			continue
		}
		// Include native language for comprehension AND target languages for learning
		allLangs := []string{nativeLanguage}
		allLangs = append(allLangs, targetLangs...)
		languages[userID] = allLangs
	}

	return languages, nil
}

// ---------------------------------------------------------------------------
// Archive & mute (task 6.4): per-user, per-chat conversation preferences.
// ---------------------------------------------------------------------------

// ArchiveChat archives or unarchives a conversation for a user. It returns the
// resulting preference row (a one-row upsert in chat_preferences). Because archive
// state is strictly user-scoped, archiving never affects co-participants.
func (s *ChatService) ArchiveChat(ctx context.Context, userID, chatID string, archived bool) (*models.ChatPreference, error) {
	pref := &models.ChatPreference{}
	query := `
		INSERT INTO chat_preferences (user_id, chat_id, archived_at)
		VALUES ($1, $2, CASE WHEN $3 THEN CURRENT_TIMESTAMP ELSE NULL END)
		ON CONFLICT (user_id, chat_id) DO UPDATE SET
			archived_at = EXCLUDED.archived_at,
			updated_at = CURRENT_TIMESTAMP
		RETURNING user_id, chat_id, archived_at, is_muted, muted_until, updated_at
	`
	err := s.db.QueryRowContext(ctx, query, userID, chatID, archived).Scan(
		&pref.UserID,
		&pref.ChatID,
		&pref.ArchivedAt,
		&pref.IsMuted,
		&pref.MutedUntil,
		&pref.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return pref, nil
}

// MuteChat mutes or unmutes a conversation for a user, optionally until a
// timestamp. When muted is true with a nil until, the mute is indefinite; when
// muted is false, any until is cleared. It returns the resulting preference row.
func (s *ChatService) MuteChat(ctx context.Context, userID, chatID string, muted bool, until *time.Time) (*models.ChatPreference, error) {
	pref := &models.ChatPreference{}
	query := `
		INSERT INTO chat_preferences (user_id, chat_id, is_muted, muted_until)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, chat_id) DO UPDATE SET
			is_muted = EXCLUDED.is_muted,
			muted_until = EXCLUDED.muted_until,
			updated_at = CURRENT_TIMESTAMP
		RETURNING user_id, chat_id, archived_at, is_muted, muted_until, updated_at
	`
	err := s.db.QueryRowContext(ctx, query, userID, chatID, muted, until).Scan(
		&pref.UserID,
		&pref.ChatID,
		&pref.ArchivedAt,
		&pref.IsMuted,
		&pref.MutedUntil,
		&pref.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return pref, nil
}

// GetChatPreference returns the per-user preference row for one chat. It returns
// a zero-value preference (not an error) when the user has never archived/muted
// the chat, so callers can always render the default (unarchived, unmuted) state.
func (s *ChatService) GetChatPreference(ctx context.Context, userID, chatID string) (*models.ChatPreference, error) {
	pref := &models.ChatPreference{}
	query := `
		SELECT user_id, chat_id, archived_at, is_muted, muted_until, updated_at
		FROM chat_preferences
		WHERE user_id = $1 AND chat_id = $2
	`
	err := s.db.QueryRowContext(ctx, query, userID, chatID).Scan(
		&pref.UserID,
		&pref.ChatID,
		&pref.ArchivedAt,
		&pref.IsMuted,
		&pref.MutedUntil,
		&pref.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return &models.ChatPreference{UserID: userID, ChatID: chatID}, nil
		}
		return nil, err
	}
	return pref, nil
}

// GetChatPreferences returns every per-user preference row across the user's
// conversations, keyed by chat ID (task 6.4). Chats the user never touched are
// simply absent, so readers fall back to the default state.
func (s *ChatService) GetChatPreferences(ctx context.Context, userID string) (map[string]models.ChatPreference, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT user_id, chat_id, archived_at, is_muted, muted_until, updated_at
		FROM chat_preferences
		WHERE user_id = $1
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	prefs := map[string]models.ChatPreference{}
	for rows.Next() {
		var pref models.ChatPreference
		if err := rows.Scan(&pref.UserID, &pref.ChatID, &pref.ArchivedAt, &pref.IsMuted, &pref.MutedUntil, &pref.UpdatedAt); err != nil {
			continue
		}
		prefs[pref.ChatID] = pref
	}

	return prefs, rows.Err()
}
