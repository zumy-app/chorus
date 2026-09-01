package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/chorus/messenger/internal/models"
	"github.com/lib/pq"
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
			SELECT id, chat_id, sender_id, text, COALESCE(original_language, ''), COALESCE(translations, '{}'::jsonb), delivery_status, reply_to_id, created_at, deleted_at, forwarded_from_message_id, forwarded_from_chat_id, forwarded_from_sender_id
			FROM messages
			WHERE chat_id = $1 AND deleted_at IS NULL AND created_at < (SELECT created_at FROM messages WHERE id = $2)
			ORDER BY created_at DESC
			LIMIT $3
		`
		rows, err = s.db.Query(query, chatID, *before, limit)
	} else {
		query = `
			SELECT id, chat_id, sender_id, text, COALESCE(original_language, ''), COALESCE(translations, '{}'::jsonb), delivery_status, reply_to_id, created_at, deleted_at, forwarded_from_message_id, forwarded_from_chat_id, forwarded_from_sender_id
			FROM messages
			WHERE chat_id = $1 AND deleted_at IS NULL
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
			&msg.DeletedAt,
			&msg.ForwardedFromMessageID,
			&msg.ForwardedFromChatID,
			&msg.ForwardedFromSenderID,
		)

		if err != nil {
			continue
		}

		msg.Forwarded = msg.ForwardedFromMessageID != nil

		if len(translationsBytes) > 0 {
			json.Unmarshal(translationsBytes, &msg.Translations)
		}

		messages = append(messages, msg)
	}

	return messages, nil
}

func (s *MessageService) GetMessageByID(ctx context.Context, messageID string) (*models.Message, error) {
	message := &models.Message{}
	query := `
		SELECT id, chat_id, sender_id, text, COALESCE(original_language, ''), COALESCE(translations, '{}'::jsonb), delivery_status, reply_to_id, created_at, deleted_at, forwarded_from_message_id, forwarded_from_chat_id, forwarded_from_sender_id
		FROM messages
		WHERE id = $1
	`

	var translationsBytes []byte
	err := s.db.QueryRowContext(ctx, query, messageID).Scan(
		&message.ID,
		&message.ChatID,
		&message.SenderID,
		&message.Text,
		&message.OriginalLanguage,
		&translationsBytes,
		&message.DeliveryStatus,
		&message.ReplyToID,
		&message.CreatedAt,
		&message.DeletedAt,
		&message.ForwardedFromMessageID,
		&message.ForwardedFromChatID,
		&message.ForwardedFromSenderID,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	message.Forwarded = message.ForwardedFromMessageID != nil

	if len(translationsBytes) > 0 {
		json.Unmarshal(translationsBytes, &message.Translations)
	}

	return message, nil
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
	_, err := s.MarkRead(chatID, messageID, userID)
	return err
}

func (s *MessageService) Search(query string, chatID *string, limit int) ([]models.Message, error) {
	var sqlQuery string
	var rows *sql.Rows
	var err error

	if chatID != nil {
		sqlQuery = `
			SELECT id, chat_id, sender_id, text, original_language, translations, delivery_status, reply_to_id, created_at, deleted_at, forwarded_from_message_id, forwarded_from_chat_id, forwarded_from_sender_id
			FROM messages
			WHERE chat_id = $1 AND deleted_at IS NULL AND to_tsvector('english', text) @@ plainto_tsquery('english', $2)
			ORDER BY created_at DESC
			LIMIT $3
		`
		rows, err = s.db.Query(sqlQuery, *chatID, query, limit)
	} else {
		sqlQuery = `
			SELECT id, chat_id, sender_id, text, original_language, translations, delivery_status, reply_to_id, created_at, deleted_at, forwarded_from_message_id, forwarded_from_chat_id, forwarded_from_sender_id
			FROM messages
			WHERE deleted_at IS NULL AND to_tsvector('english', text) @@ plainto_tsquery('english', $1)
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
			&msg.DeletedAt,
			&msg.ForwardedFromMessageID,
			&msg.ForwardedFromChatID,
			&msg.ForwardedFromSenderID,
		)

		if err != nil {
			continue
		}

		msg.Forwarded = msg.ForwardedFromMessageID != nil

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
		SELECT id, chat_id, sender_id, text, original_language, translations, delivery_status, reply_to_id, created_at, deleted_at, forwarded_from_message_id, forwarded_from_chat_id, forwarded_from_sender_id
		FROM messages
		WHERE chat_id = $1 AND deleted_at IS NULL
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
		&message.DeletedAt,
		&message.ForwardedFromMessageID,
		&message.ForwardedFromChatID,
		&message.ForwardedFromSenderID,
	)

	if err != nil {
		return nil, err
	}

	message.Forwarded = message.ForwardedFromMessageID != nil

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
		  AND m.deleted_at IS NULL
		  AND (cp.last_read_message_id IS NULL OR m.created_at > (
			SELECT created_at FROM messages WHERE id = cp.last_read_message_id
		  ))
	`

	err := s.db.QueryRow(query, chatID, userID).Scan(&count)
	return count, err
}

// InitializeReceipts creates a 'sent' receipt row for every participant other
// than the sender and attaches the resulting tick state to the message.
func (s *MessageService) InitializeReceipts(ctx context.Context, message *models.Message, participantIDs []string) error {
	if message == nil {
		return nil
	}

	receipts := make([]models.MessageReceipt, 0, len(participantIDs))
	for _, userID := range participantIDs {
		if userID == "" || userID == message.SenderID {
			continue
		}

		_, err := s.db.ExecContext(ctx, `
			INSERT INTO message_receipts (message_id, user_id, chat_id)
			VALUES ($1, $2, $3)
			ON CONFLICT (message_id, user_id) DO NOTHING
		`, message.ID, userID, message.ChatID)
		if err != nil {
			return err
		}

		receipts = append(receipts, models.MessageReceipt{
			MessageID: message.ID,
			UserID:    userID,
			ChatID:    message.ChatID,
			Status:    "sent",
		})
	}

	message.Receipts = receipts
	return nil
}

// MarkDelivered advances a recipient's tick to 'delivered' and reports whether
// the state actually changed (so callers only notify once).
func (s *MessageService) MarkDelivered(chatID, messageID, userID string) (bool, error) {
	result, err := s.db.Exec(`
		UPDATE message_receipts
		SET received_at = COALESCE(received_at, CURRENT_TIMESTAMP)
		WHERE message_id = $1 AND user_id = $2 AND chat_id = $3
		  AND received_at IS NULL
	`, messageID, userID, chatID)
	if err != nil {
		return false, err
	}

	n, _ := result.RowsAffected()
	return n > 0, nil
}

// MarkRead advances a recipient's tick to 'read', advances the chat's read
// cursor for that user, and reports whether the state actually changed.
func (s *MessageService) MarkRead(chatID, messageID, userID string) (bool, error) {
	result, err := s.db.Exec(`
		UPDATE message_receipts
		SET read_at = COALESCE(read_at, CURRENT_TIMESTAMP),
		    received_at = COALESCE(received_at, CURRENT_TIMESTAMP)
		WHERE message_id = $1 AND user_id = $2 AND chat_id = $3
		  AND read_at IS NULL
	`, messageID, userID, chatID)
	if err != nil {
		return false, err
	}

	n, _ := result.RowsAffected()
	changed := n > 0

	_, err = s.db.Exec(`
		UPDATE chat_participants
		SET last_read_message_id = $1
		WHERE chat_id = $2 AND user_id = $3
	`, messageID, chatID, userID)
	if err != nil {
		return false, err
	}

	return changed, nil
}

// AttachReceipts populates the per-recipient tick state on each message in a
// chat so REST and WebSocket consumers can render sent/delivered/read.
func (s *MessageService) AttachReceipts(ctx context.Context, chatID string, messages []models.Message) error {
	if len(messages) == 0 {
		return nil
	}

	ids := make([]string, 0, len(messages))
	index := make(map[string]int, len(messages))
	for i := range messages {
		if messages[i].ID == "" {
			continue
		}
		index[messages[i].ID] = i
		ids = append(ids, messages[i].ID)
	}
	if len(ids) == 0 {
		return nil
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT message_id, user_id, chat_id, received_at, read_at
		FROM message_receipts
		WHERE chat_id = $1 AND message_id = ANY($2)
	`, chatID, pq.Array(ids))
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var rec models.MessageReceipt
		if err := rows.Scan(&rec.MessageID, &rec.UserID, &rec.ChatID, &rec.ReceivedAt, &rec.ReadAt); err != nil {
			continue
		}
		rec.Status = receiptStatus(rec.ReadAt, rec.ReceivedAt)
		if i, ok := index[rec.MessageID]; ok {
			messages[i].Receipts = append(messages[i].Receipts, rec)
		}
	}

	return rows.Err()
}

// GetMessageReceipts returns the per-recipient tick state for a single message.
func (s *MessageService) GetMessageReceipts(ctx context.Context, messageID, chatID string) ([]models.MessageReceipt, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT message_id, user_id, chat_id, received_at, read_at
		FROM message_receipts
		WHERE message_id = $1 AND chat_id = $2
	`, messageID, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	receipts := []models.MessageReceipt{}
	for rows.Next() {
		var rec models.MessageReceipt
		if err := rows.Scan(&rec.MessageID, &rec.UserID, &rec.ChatID, &rec.ReceivedAt, &rec.ReadAt); err != nil {
			continue
		}
		rec.Status = receiptStatus(rec.ReadAt, rec.ReceivedAt)
		receipts = append(receipts, rec)
	}

	return receipts, rows.Err()
}

func receiptStatus(readAt, receivedAt *time.Time) string {
	if readAt != nil {
		return "read"
	}
	if receivedAt != nil {
		return "delivered"
	}
	return "sent"
}

// AttachMedia populates the Media slice on each message in a chat page (task
// 6.6) so a loaded history carries the file/document attachments alongside the
// message rows. It batch-loads media_attachments for the supplied message IDs
// so N messages cost one query.
func (s *MessageService) AttachMedia(ctx context.Context, chatID string, messages []models.Message) error {
	if len(messages) == 0 {
		return nil
	}

	ids := make([]string, 0, len(messages))
	index := make(map[string]int, len(messages))
	for i := range messages {
		if messages[i].ID == "" {
			continue
		}
		index[messages[i].ID] = i
		ids = append(ids, messages[i].ID)
	}
	if len(ids) == 0 {
		return nil
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT ma.id, ma.message_id, m.chat_id, ma.type, ma.file_name, ma.file_size,
		       ma.mime_type, ma.url, COALESCE(ma.thumbnail_url, ''), ma.created_at,
		       ma.latitude, ma.longitude, COALESCE(ma.location_name, '')
		FROM media_attachments ma
		JOIN messages m ON m.id = ma.message_id
		WHERE ma.message_id = ANY($1)
		ORDER BY ma.created_at ASC
	`, pq.Array(ids))
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var att models.MediaAttachment
		var thumbnail sql.NullString
		var latitude, longitude sql.NullFloat64
		if err := rows.Scan(
			&att.ID, &att.MessageID, &att.ChatID, &att.Type, &att.FileName, &att.FileSize,
			&att.MimeType, &att.URL, &thumbnail, &att.CreatedAt,
			&latitude, &longitude, &att.LocationName,
		); err != nil {
			continue
		}
		if thumbnail.Valid && thumbnail.String != "" {
			att.ThumbnailURL = &thumbnail.String
		}
		if latitude.Valid {
			att.Latitude = &latitude.Float64
		}
		if longitude.Valid {
			att.Longitude = &longitude.Float64
		}
		if i, ok := index[att.MessageID]; ok {
			messages[i].Media = append(messages[i].Media, att)
		}
	}

	return rows.Err()
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

// UpdateOriginalLanguage sets the original language on an existing message.
func (s *MessageService) UpdateOriginalLanguage(messageID, language string) error {
	query := `UPDATE messages SET original_language = $1 WHERE id = $2`
	_, err := s.db.Exec(query, language, messageID)
	return err
}

// ---------------------------------------------------------------------------
// Message actions (task 6.2): delete, forward, pin, unpin, pinned list.
// ---------------------------------------------------------------------------

// DeleteMessage soft-deletes a message (stamps deleted_at). The row is kept so
// replies/forwards referencing it keep working and the data is recoverable, but
// it is excluded from chat history, search, and unread counts. It reports
// whether a message in this chat was actually targeted (not already-deleted no-ops
// still report true so callers can 204 idempotently; a nonexistent message is false).
func (s *MessageService) DeleteMessage(ctx context.Context, chatID, messageID string) (bool, error) {
	result, err := s.db.ExecContext(ctx, `
		UPDATE messages
		SET deleted_at = COALESCE(deleted_at, CURRENT_TIMESTAMP)
		WHERE id = $1 AND chat_id = $2
	`, messageID, chatID)
	if err != nil {
		return false, err
	}
	n, _ := result.RowsAffected()
	return n > 0, nil
}

// ForwardMessage copies an existing message into a target chat as a new message
// authored by forwarderID, carrying a trail to the original author / message /
// chat so clients can render the "Forwarded from ..." label. It resolves the
// source message text and author, returns an error when the source is missing,
// deleted, or belongs to a different chat.
func (s *MessageService) ForwardMessage(ctx context.Context, sourceChatID, messageID, targetChatID, forwarderID string) (*models.Message, error) {
	var text, originalLang string
	var origSenderID string

	err := s.db.QueryRowContext(ctx, `
		SELECT text, COALESCE(original_language, ''), sender_id
		FROM messages
		WHERE id = $1 AND chat_id = $2 AND deleted_at IS NULL
	`, messageID, sourceChatID).Scan(&text, &originalLang, &origSenderID)
	if err != nil {
		return nil, err
	}

	message := &models.Message{}
	var translationsBytes []byte
	query := `
		INSERT INTO messages (chat_id, sender_id, text, delivery_status, original_language, forwarded_from_message_id, forwarded_from_chat_id, forwarded_from_sender_id)
		VALUES ($1, $2, $3, 'sent', $4, $5, $6, $7)
		RETURNING id, chat_id, sender_id, text, COALESCE(original_language, ''), COALESCE(translations, '{}'::jsonb), delivery_status, reply_to_id, created_at, deleted_at, forwarded_from_message_id, forwarded_from_chat_id, forwarded_from_sender_id
	`
	err = s.db.QueryRowContext(ctx, query, targetChatID, forwarderID, text, originalLang, messageID, sourceChatID, origSenderID).Scan(
		&message.ID,
		&message.ChatID,
		&message.SenderID,
		&message.Text,
		&message.OriginalLanguage,
		&translationsBytes,
		&message.DeliveryStatus,
		&message.ReplyToID,
		&message.CreatedAt,
		&message.DeletedAt,
		&message.ForwardedFromMessageID,
		&message.ForwardedFromChatID,
		&message.ForwardedFromSenderID,
	)
	if err != nil {
		return nil, err
	}

	if len(translationsBytes) > 0 {
		json.Unmarshal(translationsBytes, &message.Translations)
	}
	message.Forwarded = message.ForwardedFromMessageID != nil

	return message, nil
}

// PinMessage pins a message to a chat. Re-pinning is a no-op (ON CONFLICT).
func (s *MessageService) PinMessage(ctx context.Context, chatID, messageID, userID string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO pinned_messages (chat_id, message_id, pinned_by)
		VALUES ($1, $2, $3)
		ON CONFLICT (chat_id, message_id) DO NOTHING
	`, chatID, messageID, userID)
	return err
}

// UnpinMessage removes a message from a chat's pin list. It is a no-op when the
// message was not pinned.
func (s *MessageService) UnpinMessage(ctx context.Context, chatID, messageID string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM pinned_messages
		WHERE chat_id = $1 AND message_id = $2
	`, chatID, messageID)
	return err
}

// GetPinnedMessages returns the chat's pinned messages, newest-pin first, with
// the "pinned by" attribution. Deleted messages are excluded from the list.
func (s *MessageService) GetPinnedMessages(ctx context.Context, chatID string) ([]models.PinnedMessage, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT pm.pinned_by, pm.created_at,
		       m.id, m.chat_id, m.sender_id, m.text, COALESCE(m.original_language, ''), COALESCE(m.translations, '{}'::jsonb), m.delivery_status, m.reply_to_id, m.created_at,
		       m.deleted_at, m.forwarded_from_message_id, m.forwarded_from_chat_id, m.forwarded_from_sender_id
		FROM pinned_messages pm
		JOIN messages m ON m.id = pm.message_id
		WHERE pm.chat_id = $1 AND m.deleted_at IS NULL
		ORDER BY pm.created_at DESC
	`, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pins := []models.PinnedMessage{}
	for rows.Next() {
		var pin models.PinnedMessage
		var translationsBytes []byte

		if err := rows.Scan(
			&pin.PinnedBy,
			&pin.PinnedAt,
			&pin.Message.ID,
			&pin.Message.ChatID,
			&pin.Message.SenderID,
			&pin.Message.Text,
			&pin.Message.OriginalLanguage,
			&translationsBytes,
			&pin.Message.DeliveryStatus,
			&pin.Message.ReplyToID,
			&pin.Message.CreatedAt,
			&pin.Message.DeletedAt,
			&pin.Message.ForwardedFromMessageID,
			&pin.Message.ForwardedFromChatID,
			&pin.Message.ForwardedFromSenderID,
		); err != nil {
			continue
		}

		if len(translationsBytes) > 0 {
			json.Unmarshal(translationsBytes, &pin.Message.Translations)
		}
		pin.Message.Forwarded = pin.Message.ForwardedFromMessageID != nil
		pins = append(pins, pin)
	}

	return pins, rows.Err()
}
