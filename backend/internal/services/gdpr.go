package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/chorus/messenger/internal/models"
)

type GDPRDataExport struct {
	User           *models.User          `json:"user"`
	Settings       *models.UserSettings  `json:"settings"`
	Chats          []models.Chat         `json:"chats"`
	Messages       []models.Message      `json:"messages"`
	Vocabulary     []models.VocabularyEntry `json:"vocabulary"`
	ExportedAt     time.Time             `json:"exportedAt"`
	RetentionPolicy RetentionPolicy      `json:"retentionPolicy"`
}

type GDPRService struct {
	db               *sql.DB
	retentionService *RetentionService
}

func NewGDPRService(db *sql.DB, retentionService *RetentionService) *GDPRService {
	if retentionService == nil {
		retentionService = NewRetentionService(db)
	}
	return &GDPRService{db: db, retentionService: retentionService}
}

func (s *GDPRService) ExportUserData(userID string) (*GDPRDataExport, error) {
	user, err := NewUserService(s.db).GetByID(userID)
	if err != nil {
		return nil, err
	}
	settings, _ := NewSettingsService(s.db).GetSettings(userID)

	chats, _ := s.exportChats(userID)
	msgs, _ := s.exportMessages(userID)
	vocab, _ := s.exportVocabulary(userID)

	return &GDPRDataExport{
		User:           user,
		Settings:       settings,
		Chats:          chats,
		Messages:       msgs,
		Vocabulary:     vocab,
		ExportedAt:     time.Now().UTC(),
		RetentionPolicy: s.retentionService.GetPolicy(),
	}, nil
}

func (s *GDPRService) exportChats(userID string) ([]models.Chat, error) {
	rows, err := s.db.Query(`
		SELECT c.id, c.type, c.name, c.created_by, c.created_at
		FROM chats c
		JOIN chat_participants cp ON cp.chat_id = c.id
		WHERE cp.user_id = $1
		ORDER BY c.created_at DESC
		LIMIT 500`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var chats []models.Chat
	for rows.Next() {
		var c models.Chat
		var name sql.NullString
		var settings json.RawMessage
		_ = settings
		if err := rows.Scan(&c.ID, &c.Type, &name, &c.CreatedBy, &c.CreatedAt); err != nil {
			continue
		}
		if name.Valid {
			c.Name = name.String
		}
		chats = append(chats, c)
	}
	return chats, rows.Err()
}

func (s *GDPRService) exportMessages(userID string) ([]models.Message, error) {
	rows, err := s.db.Query(`
		SELECT id, chat_id, sender_id, text, original_language, translations, delivery_status, created_at, deleted_at
		FROM messages
		WHERE sender_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT 1000`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var msgs []models.Message
	for rows.Next() {
		var m models.Message
		var translations json.RawMessage
		var origLang sql.NullString
		var deletedAt sql.NullTime
		if err := rows.Scan(&m.ID, &m.ChatID, &m.SenderID, &m.Text, &origLang, &translations, &m.DeliveryStatus, &m.CreatedAt, &deletedAt); err != nil {
			continue
		}
		if origLang.Valid {
			m.OriginalLanguage = origLang.String
		}
		if deletedAt.Valid {
			t := deletedAt.Time
			m.DeletedAt = &t
		}
		if len(translations) > 0 {
			_ = json.Unmarshal(translations, &m.Translations)
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

func (s *GDPRService) exportVocabulary(userID string) ([]models.VocabularyEntry, error) {
	rows, err := s.db.Query(`
		SELECT id, term, language, translation, context_sentence, created_at
		FROM vocabulary
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT 1000`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var vocab []models.VocabularyEntry
	for rows.Next() {
		var v models.VocabularyEntry
		var ctx sql.NullString
		if err := rows.Scan(&v.ID, &v.Term, &v.Language, &v.Translation, &ctx, &v.CreatedAt); err != nil {
			continue
		}
		if ctx.Valid {
			v.Context.Sentence = ctx.String
		}
		v.UserID = userID
		vocab = append(vocab, v)
	}
	return vocab, rows.Err()
}

func (s *GDPRService) EraseUser(userID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	anonymizedEmail := fmt.Sprintf("deleted_%s@deleted.local", userID)
	anonymizedUsername := fmt.Sprintf("deleted_%s", userID[:8])

	_, err = tx.Exec(`
		UPDATE users
		SET email = $2,
		    username = $3,
		    display_name = 'Deleted User',
		    first_name = '',
		    last_name = '',
		    phone = NULL,
		    phone_verified = false,
		    avatar_url = NULL,
		    target_languages = '{}',
		    deleted_at = CURRENT_TIMESTAMP,
		    suspended_at = NULL
		WHERE id = $1`, userID, anonymizedEmail, anonymizedUsername)
	if err != nil {
		return err
	}

	_, _ = tx.Exec(`DELETE FROM refresh_tokens WHERE user_id = $1`, userID)
	_, _ = tx.Exec(`DELETE FROM password_resets WHERE user_id = $1`, userID)
	_, _ = tx.Exec(`DELETE FROM phone_otps WHERE user_id = $1`, userID)
	_, _ = tx.Exec(`DELETE FROM clients WHERE user_id = $1`, userID)
	_, _ = tx.Exec(`DELETE FROM user_settings WHERE user_id = $1`, userID)
	_, _ = tx.Exec(`DELETE FROM blocked_users WHERE blocker_id = $1 OR blocked_id = $1`, userID)
	_, _ = tx.Exec(`DELETE FROM reports WHERE reporter_id = $1 OR reported_user_id = $1`, userID)
	_, _ = tx.Exec(`DELETE FROM vocabulary WHERE user_id = $1`, userID)
	_, _ = tx.Exec(`DELETE FROM mined_items WHERE user_id = $1`, userID)
	_, _ = tx.Exec(`DELETE FROM word_mining_jobs WHERE user_id = $1`, userID)
	_, _ = tx.Exec(`DELETE FROM tutor_trial_credits WHERE user_id = $1`, userID)
	_, _ = tx.Exec(`DELETE FROM invitations WHERE inviter_user_id = $1`, userID)
	_, _ = tx.Exec(`DELETE FROM email_outbox WHERE recipient = (SELECT email FROM users WHERE id = $1)`, userID)
	_, _ = tx.Exec(`DELETE FROM chat_preferences WHERE user_id = $1`, userID)

	_, _ = tx.Exec(`UPDATE messages SET text = '[deleted]', translations = '{}' WHERE sender_id = $1`, userID)
	_, _ = tx.Exec(`DELETE FROM message_receipts WHERE user_id = $1`, userID)
	_, _ = tx.Exec(`DELETE FROM chat_participants WHERE user_id = $1`, userID)

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}
