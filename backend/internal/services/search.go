package services

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/chorus/messenger/internal/models"
	"github.com/redis/go-redis/v9"
)

type SearchService struct {
	db    *sql.DB
	redis *redis.Client
	ctx   context.Context
}

func NewSearchService(db *sql.DB, redis *redis.Client) *SearchService {
	return &SearchService{
		db:    db,
		redis: redis,
		ctx:   context.Background(),
	}
}

func (s *SearchService) SearchMessages(userID string, chatID string, query string, limit int) ([]models.Message, error) {
	sqlQuery := `
		SELECT m.id, m.chat_id, m.sender_id, m.text, 
		       COALESCE(m.original_language, ''), 
		       COALESCE(m.translations, '{}'::jsonb),
		       m.delivery_status, m.reply_to_id, m.created_at
		FROM messages m
		INNER JOIN chat_participants cp ON m.chat_id = cp.chat_id
		WHERE cp.user_id = $1
		  AND ($2 = '' OR m.chat_id = $2)
		  AND m.text ILIKE $3
		ORDER BY m.created_at DESC
		LIMIT $4
	`

	rows, err := s.db.Query(sqlQuery, userID, chatID, "%"+query+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := []models.Message{}
	for rows.Next() {
		var msg models.Message
		var translationsBytes []byte

		err := rows.Scan(
			&msg.ID,
			&msg.ChatID,
			&msg.SenderID,
			&msg.Text,
			&msg.OriginalLanguage,
			&translationsBytes,
			&msg.DeliveryStatus,
			&msg.ReplyToID,
			&msg.CreatedAt,
		)
		if err != nil {
			continue
		}

		messages = append(messages, msg)
	}

	return messages, nil
}

func (s *SearchService) SearchChats(userID string, query string) ([]models.Chat, error) {
	sqlQuery := `
		SELECT DISTINCT c.id, c.name, c.type, c.created_at
		FROM chats c
		INNER JOIN chat_participants cp ON c.id = cp.chat_id
		WHERE cp.user_id = $1
		  AND c.name ILIKE $2
		ORDER BY c.created_at DESC
		LIMIT 20
	`

	rows, err := s.db.Query(sqlQuery, userID, "%"+query+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	chats := []models.Chat{}
	for rows.Next() {
		var chat models.Chat
		err := rows.Scan(
			&chat.ID,
			&chat.Name,
			&chat.Type,
			&chat.CreatedAt,
		)
		if err != nil {
			continue
		}
		chats = append(chats, chat)
	}

	return chats, nil
}

func (s *SearchService) SearchContacts(userID string, query string) ([]models.User, error) {
	sqlQuery := `
		SELECT u.id, u.username, u.email, u.display_name, 
		       u.native_language, u.target_languages,
		       u.created_at
		FROM users u
		WHERE u.id != $1
		  AND (u.username ILIKE $2 OR u.display_name ILIKE $2 OR u.email ILIKE $2)
		ORDER BY u.username
		LIMIT 20
	`

	rows, err := s.db.Query(sqlQuery, userID, "%"+query+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := []models.User{}
	for rows.Next() {
		var user models.User
		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&user.DisplayName,
			&user.NativeLanguage,
			&user.TargetLanguages,
			&user.CreatedAt,
		)
		if err != nil {
			continue
		}
		users = append(users, user)
	}

	return users, nil
}

func (s *SearchService) SearchVocabulary(userID string, query string) ([]models.VocabularyEntry, error) {
	sqlQuery := `
		SELECT id, user_id, language, term, translation, definition, created_at
		FROM vocabulary
		WHERE user_id = $1
		  AND (term ILIKE $2 OR translation ILIKE $2 OR definition ILIKE $2)
		ORDER BY created_at DESC
		LIMIT 50
	`

	rows, err := s.db.Query(sqlQuery, userID, "%"+query+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := []models.VocabularyEntry{}
	for rows.Next() {
		var entry models.VocabularyEntry
		err := rows.Scan(
			&entry.ID,
			&entry.UserID,
			&entry.Language,
			&entry.Term,
			&entry.Translation,
			&entry.Definition,
			&entry.CreatedAt,
		)
		if err != nil {
			fmt.Printf("Scan error: %v\n", err)
			continue
		}
		entries = append(entries, entry)
	}

	return entries, nil
}
