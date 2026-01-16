package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/chorus/messenger/internal/models"
)

type InboxService struct {
	db *sql.DB
}

func NewInboxService(db *sql.DB) *InboxService {
	return &InboxService{
		db: db,
	}
}

// QueueMessageForClient queues a message for delivery to an offline client
func (s *InboxService) QueueMessageForClient(ctx context.Context, clientID string, messageID string, chatID string) error {
	query := `
		INSERT INTO inbox (client_id, message_id, chat_id, delivery_attempts, created_at, ttl)
		VALUES ($1, $2, $3, 0, $4, $5)
		ON CONFLICT (client_id, message_id) DO NOTHING
	`
	
	now := time.Now()
	ttl := now.Add(30 * 24 * time.Hour) // 30 days
	
	_, err := s.db.ExecContext(ctx, query, clientID, messageID, chatID, now, ttl)
	if err != nil {
		return fmt.Errorf("failed to queue message: %w", err)
	}
	
	return nil
}

// QueueMessageForUser queues a message for all offline clients of a user
func (s *InboxService) QueueMessageForUser(ctx context.Context, userID string, messageID string, chatID string) error {
	// Get all offline clients for this user
	query := `
		SELECT id FROM clients 
		WHERE user_id = $1 AND connection_status = 'offline'
	`
	
	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to query offline clients: %w", err)
	}
	defer rows.Close()
	
	for rows.Next() {
		var clientID string
		if err := rows.Scan(&clientID); err != nil {
			continue
		}
		
		if err := s.QueueMessageForClient(ctx, clientID, messageID, chatID); err != nil {
			// Log error but continue with other clients
			continue
		}
	}
	
	return nil
}

// GetPendingMessages retrieves all pending messages for a client
func (s *InboxService) GetPendingMessages(ctx context.Context, clientID string) ([]models.InboxEntry, error) {
	query := `
		SELECT client_id, message_id, chat_id, delivery_attempts, created_at, ttl
		FROM inbox
		WHERE client_id = $1 AND ttl > $2
		ORDER BY created_at ASC
		LIMIT 1000
	`
	
	rows, err := s.db.QueryContext(ctx, query, clientID, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to query pending messages: %w", err)
	}
	defer rows.Close()
	
	entries := []models.InboxEntry{}
	for rows.Next() {
		var entry models.InboxEntry
		
		err := rows.Scan(
			&entry.ClientID, &entry.MessageID, &entry.ChatID,
			&entry.DeliveryAttempts, &entry.CreatedAt, &entry.TTL,
		)
		if err != nil {
			continue
		}
		
		entries = append(entries, entry)
	}
	
	return entries, nil
}

// MarkMessageDelivered marks a message as delivered and removes from inbox
func (s *InboxService) MarkMessageDelivered(ctx context.Context, clientID string, messageID string) error {
	query := `DELETE FROM inbox WHERE client_id = $1 AND message_id = $2`
	
	result, err := s.db.ExecContext(ctx, query, clientID, messageID)
	if err != nil {
		return fmt.Errorf("failed to mark message delivered: %w", err)
	}
	
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		// Message not in inbox (might have been already delivered)
		return nil
	}
	
	return nil
}

// IncrementDeliveryAttempt increments the delivery attempt counter
func (s *InboxService) IncrementDeliveryAttempt(ctx context.Context, clientID string, messageID string) error {
	query := `
		UPDATE inbox
		SET delivery_attempts = delivery_attempts + 1
		WHERE client_id = $1 AND message_id = $2
	`
	
	_, err := s.db.ExecContext(ctx, query, clientID, messageID)
	if err != nil {
		return fmt.Errorf("failed to increment delivery attempt: %w", err)
	}
	
	return nil
}

// CleanupExpiredMessages removes expired messages from inbox (TTL passed)
func (s *InboxService) CleanupExpiredMessages(ctx context.Context) (int, error) {
	query := `DELETE FROM inbox WHERE ttl <= $1`
	
	result, err := s.db.ExecContext(ctx, query, time.Now())
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired messages: %w", err)
	}
	
	rowsAffected, _ := result.RowsAffected()
	return int(rowsAffected), nil
}

// GetPendingMessageCount returns the count of pending messages for a client
func (s *InboxService) GetPendingMessageCount(ctx context.Context, clientID string) (int, error) {
	query := `
		SELECT COUNT(*) 
		FROM inbox 
		WHERE client_id = $1 AND ttl > $2
	`
	
	var count int
	err := s.db.QueryRowContext(ctx, query, clientID, time.Now()).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count pending messages: %w", err)
	}
	
	return count, nil
}

// GetOldestPendingMessage retrieves the oldest pending message for a client
func (s *InboxService) GetOldestPendingMessage(ctx context.Context, clientID string) (*models.InboxEntry, error) {
	query := `
		SELECT client_id, message_id, chat_id, delivery_attempts, created_at, ttl
		FROM inbox
		WHERE client_id = $1 AND ttl > $2
		ORDER BY created_at ASC
		LIMIT 1
	`
	
	var entry models.InboxEntry
	err := s.db.QueryRowContext(ctx, query, clientID, time.Now()).Scan(
		&entry.ClientID, &entry.MessageID, &entry.ChatID,
		&entry.DeliveryAttempts, &entry.CreatedAt, &entry.TTL,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get oldest pending message: %w", err)
	}
	
	return &entry, nil
}

// GetPendingMessagesByChatID retrieves pending messages for a specific chat
func (s *InboxService) GetPendingMessagesByChatID(ctx context.Context, clientID string, chatID string) ([]models.InboxEntry, error) {
	query := `
		SELECT client_id, message_id, chat_id, delivery_attempts, created_at, ttl
		FROM inbox
		WHERE client_id = $1 AND chat_id = $2 AND ttl > $3
		ORDER BY created_at ASC
	`
	
	rows, err := s.db.QueryContext(ctx, query, clientID, chatID, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to query pending messages by chat: %w", err)
	}
	defer rows.Close()
	
	entries := []models.InboxEntry{}
	for rows.Next() {
		var entry models.InboxEntry
		
		err := rows.Scan(
			&entry.ClientID, &entry.MessageID, &entry.ChatID,
			&entry.DeliveryAttempts, &entry.CreatedAt, &entry.TTL,
		)
		if err != nil {
			continue
		}
		
		entries = append(entries, entry)
	}
	
	return entries, nil
}

// RetryFailedDeliveries attempts to redeliver messages that have failed < 3 times
func (s *InboxService) RetryFailedDeliveries(ctx context.Context, maxAttempts int) ([]models.InboxEntry, error) {
	query := `
		SELECT client_id, message_id, chat_id, delivery_attempts, created_at, ttl
		FROM inbox
		WHERE delivery_attempts < $1 AND ttl > $2
		ORDER BY created_at ASC
		LIMIT 100
	`
	
	rows, err := s.db.QueryContext(ctx, query, maxAttempts, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to query failed deliveries: %w", err)
	}
	defer rows.Close()
	
	entries := []models.InboxEntry{}
	for rows.Next() {
		var entry models.InboxEntry
		
		err := rows.Scan(
			&entry.ClientID, &entry.MessageID, &entry.ChatID,
			&entry.DeliveryAttempts, &entry.CreatedAt, &entry.TTL,
		)
		if err != nil {
			continue
		}
		
		entries = append(entries, entry)
	}
	
	return entries, nil
}

// GetInboxStats returns statistics about the inbox
type InboxStats struct {
	TotalPending       int            `json:"totalPending"`
	ExpiringSoon       int            `json:"expiringSoon"`      // < 24 hours
	ByClient           map[string]int `json:"byClient"`
	OldestMessageAge   int64          `json:"oldestMessageAge"` // seconds
	AverageAttempts    float64        `json:"averageAttempts"`
}

func (s *InboxService) GetInboxStats(ctx context.Context) (*InboxStats, error) {
	stats := &InboxStats{
		ByClient: make(map[string]int),
	}
	
	// Total pending
	query := `SELECT COUNT(*) FROM inbox WHERE ttl > $1`
	err := s.db.QueryRowContext(ctx, query, time.Now()).Scan(&stats.TotalPending)
	if err != nil {
		return nil, err
	}
	
	// Expiring soon (< 24 hours)
	query = `SELECT COUNT(*) FROM inbox WHERE ttl > $1 AND ttl < $2`
	tomorrow := time.Now().Add(24 * time.Hour)
	err = s.db.QueryRowContext(ctx, query, time.Now(), tomorrow).Scan(&stats.ExpiringSoon)
	if err != nil {
		return nil, err
	}
	
	// By client
	query = `SELECT client_id, COUNT(*) FROM inbox WHERE ttl > $1 GROUP BY client_id`
	rows, err := s.db.QueryContext(ctx, query, time.Now())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	for rows.Next() {
		var clientID string
		var count int
		rows.Scan(&clientID, &count)
		stats.ByClient[clientID] = count
	}
	
	// Oldest message age
	query = `SELECT MIN(created_at) FROM inbox WHERE ttl > $1`
	var oldestTime sql.NullTime
	err = s.db.QueryRowContext(ctx, query, time.Now()).Scan(&oldestTime)
	if err == nil && oldestTime.Valid {
		stats.OldestMessageAge = int64(time.Since(oldestTime.Time).Seconds())
	}
	
	// Average attempts
	query = `SELECT AVG(delivery_attempts) FROM inbox WHERE ttl > $1`
	var avgAttempts sql.NullFloat64
	err = s.db.QueryRowContext(ctx, query, time.Now()).Scan(&avgAttempts)
	if err == nil && avgAttempts.Valid {
		stats.AverageAttempts = avgAttempts.Float64
	}
	
	return stats, nil
}

// ClearClientInbox removes all messages from a client's inbox
func (s *InboxService) ClearClientInbox(ctx context.Context, clientID string) error {
	query := `DELETE FROM inbox WHERE client_id = $1`
	
	_, err := s.db.ExecContext(ctx, query, clientID)
	if err != nil {
		return fmt.Errorf("failed to clear client inbox: %w", err)
	}
	
	return nil
}

// BatchMarkDelivered marks multiple messages as delivered
func (s *InboxService) BatchMarkDelivered(ctx context.Context, clientID string, messageIDs []string) error {
	if len(messageIDs) == 0 {
		return nil
	}
	
	// Use IN clause for batch deletion
	query := `DELETE FROM inbox WHERE client_id = $1 AND message_id = ANY($2)`
	
	_, err := s.db.ExecContext(ctx, query, clientID, messageIDs)
	if err != nil {
		return fmt.Errorf("failed to batch mark delivered: %w", err)
	}
	
	return nil
}
