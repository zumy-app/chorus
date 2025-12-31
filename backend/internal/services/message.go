package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/chorus/messenger/internal/models"
	"github.com/redis/go-redis/v9"
)

type MessageService struct {
	db    *sql.DB
	redis *redis.Client
}

func NewMessageService(db *sql.DB, redis *redis.Client) *MessageService {
	return &MessageService{
		db:    db,
		redis: redis,
	}
}

func (s *MessageService) Create(chatID, senderID, text string, replyToID *string) (*models.Message, error) {
	message := &models.Message{}
	query := `
		INSERT INTO messages (chat_id, sender_id, text, delivery_status, reply_to_id)
		VALUES ($1, $2, $3, 'sent', $4)
		RETURNING id, chat_id, sender_id, text, COALESCE(original_language, ''), COALESCE(translations, '{}'::jsonb), delivery_status, reply_to_id, created_at
	`

	var translationsBytes []byte
	err := s.db.QueryRow(query, chatID, senderID, text, replyToID).Scan(
		&message.ID,
		&message.ChatID,
		&message.SenderID,
		&message.Text,
		&message.OriginalLanguage,
		&translationsBytes,
		&message.DeliveryStatus,
		&message.ReplyToID,
		&message.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	if len(translationsBytes) > 0 {
		json.Unmarshal(translationsBytes, &message.Translations)
	}

	return message, nil
}

func (s *MessageService) GetMessages(chatID string, limit int, before *string) ([]models.Message, error) {
	var query string
	var rows *sql.Rows
	var err error

	if before != nil {
		query = `
			SELECT id, chat_id, sender_id, text, COALESCE(original_language, ''), COALESCE(translations, '{}'::jsonb), delivery_status, reply_to_id, created_at
			FROM messages
			WHERE chat_id = $1 AND created_at < (SELECT created_at FROM messages WHERE id = $2)
			ORDER BY created_at DESC
			LIMIT $3
		`
		rows, err = s.db.Query(query, chatID, *before, limit)
	} else {
		query = `
			SELECT id, chat_id, sender_id, text, COALESCE(original_language, ''), COALESCE(translations, '{}'::jsonb), delivery_status, reply_to_id, created_at
			FROM messages
			WHERE chat_id = $1
			ORDER BY created_at DESC
			LIMIT $2
		`
		rows, err = s.db.Query(query, chatID, limit)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := []models.Message{}
	for rows.Next() {
		msg := models.Message{}
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

		if len(translationsBytes) > 0 {
			json.Unmarshal(translationsBytes, &msg.Translations)
		}

		messages = append(messages, msg)
	}

	return messages, nil
}

func (s *MessageService) UpdateTranslations(messageID string, translations map[string]string) error {
	translationsJSON, err := json.Marshal(translations)
	if err != nil {
		return err
	}

	query := `UPDATE messages SET translations = $1 WHERE id = $2`
	_, err = s.db.Exec(query, translationsJSON, messageID)
	return err
}

func (s *MessageService) MarkAsRead(chatID, userID, messageID string) error {
	query := `
		UPDATE chat_participants
		SET last_read_message_id = $1
		WHERE chat_id = $2 AND user_id = $3
	`

	_, err := s.db.Exec(query, messageID, chatID, userID)
	return err
}

func (s *MessageService) Search(query string, chatID *string, limit int) ([]models.Message, error) {
	var sqlQuery string
	var rows *sql.Rows
	var err error

	if chatID != nil {
		sqlQuery = `
			SELECT id, chat_id, sender_id, text, original_language, translations, delivery_status, reply_to_id, created_at
			FROM messages
			WHERE chat_id = $1 AND to_tsvector('english', text) @@ plainto_tsquery('english', $2)
			ORDER BY created_at DESC
			LIMIT $3
		`
		rows, err = s.db.Query(sqlQuery, *chatID, query, limit)
	} else {
		sqlQuery = `
			SELECT id, chat_id, sender_id, text, original_language, translations, delivery_status, reply_to_id, created_at
			FROM messages
			WHERE to_tsvector('english', text) @@ plainto_tsquery('english', $1)
			ORDER BY created_at DESC
			LIMIT $2
		`
		rows, err = s.db.Query(sqlQuery, query, limit)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := []models.Message{}
	for rows.Next() {
		msg := models.Message{}
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

		if len(translationsBytes) > 0 {
			json.Unmarshal(translationsBytes, &msg.Translations)
		}

		messages = append(messages, msg)
	}

	return messages, nil
}

func (s *MessageService) GetLastMessage(chatID string) (*models.Message, error) {
	message := &models.Message{}
	query := `
		SELECT id, chat_id, sender_id, text, original_language, translations, delivery_status, reply_to_id, created_at
		FROM messages
		WHERE chat_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	var translationsBytes []byte
	err := s.db.QueryRow(query, chatID).Scan(
		&message.ID,
		&message.ChatID,
		&message.SenderID,
		&message.Text,
		&message.OriginalLanguage,
		&translationsBytes,
		&message.DeliveryStatus,
		&message.ReplyToID,
		&message.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	if len(translationsBytes) > 0 {
		json.Unmarshal(translationsBytes, &message.Translations)
	}

	return message, nil
}

func (s *MessageService) GetUnreadCount(chatID, userID string) (int, error) {
	var count int
	query := `
		SELECT COUNT(*)
		FROM messages m
		LEFT JOIN chat_participants cp ON m.chat_id = cp.chat_id AND cp.user_id = $2
		WHERE m.chat_id = $1
		  AND m.sender_id != $2
		  AND (cp.last_read_message_id IS NULL OR m.created_at > (
			SELECT created_at FROM messages WHERE id = cp.last_read_message_id
		  ))
	`

	err := s.db.QueryRow(query, chatID, userID).Scan(&count)
	return count, err
}

// Cache helper methods
func (s *MessageService) CacheMessage(message *models.Message) error {
	if s.redis == nil {
		return nil
	}

	ctx := context.Background()
	key := "message:" + message.ID
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	return s.redis.Set(ctx, key, data, 1*time.Hour).Err()
}

func (s *MessageService) GetCachedMessage(messageID string) (*models.Message, error) {
	if s.redis == nil {
		return nil, nil
	}

	ctx := context.Background()
	key := "message:" + messageID
	data, err := s.redis.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}

	var message models.Message
	err = json.Unmarshal(data, &message)
	return &message, err
}
