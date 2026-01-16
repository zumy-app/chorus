package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/chorus/messenger/internal/models"
	"github.com/redis/go-redis/v9"
)

// InboxService handles offline message delivery and client management
type InboxService struct {
	db    *sql.DB
	redis *redis.Client
}

// NewInboxService creates a new Inbox service
func NewInboxService(db *sql.DB, redis *redis.Client) *InboxService {
	return &InboxService{
		db:    db,
		redis: redis,
	}
}

// RegisterClient registers a new client device
func (s *InboxService) RegisterClient(userID, deviceType string, deviceInfo models.DeviceInfo) (*models.Client, error) {
	ctx := context.Background()

	// Check if user already has 3 clients
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM clients WHERE user_id = $1`, userID).Scan(&count)
	if err != nil {
		return nil, fmt.Errorf("failed to check client count: %w", err)
	}

	if count >= 3 {
		// Remove oldest client
		_, err = s.db.Exec(`
			DELETE FROM clients 
			WHERE id = (
				SELECT id FROM clients WHERE user_id = $1 
				ORDER BY last_active ASC LIMIT 1
			)
		`, userID)
		if err != nil {
			return nil, fmt.Errorf("failed to remove old client: %w", err)
		}
	}

	// Create new client
	deviceInfoJSON, _ := json.Marshal(deviceInfo)
	var client models.Client
	err = s.db.QueryRow(`
		INSERT INTO clients (user_id, device_type, device_info, connection_status)
		VALUES ($1, $2, $3, 'online')
		RETURNING id, user_id, device_type, connection_status, last_active, created_at
	`, userID, deviceType, deviceInfoJSON).Scan(
		&client.ID, &client.UserID, &client.DeviceType,
		&client.ConnectionStatus, &client.LastActive, &client.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	client.DeviceInfo = deviceInfo

	// Update Redis with online status
	s.redis.HSet(ctx, fmt.Sprintf("user:%s:clients", userID), client.ID, "online")

	return &client, nil
}

// UpdateClientStatus updates a client's connection status
func (s *InboxService) UpdateClientStatus(clientID, status string) error {
	ctx := context.Background()

	var userID string
	err := s.db.QueryRow(`
		UPDATE clients SET connection_status = $1, last_active = CURRENT_TIMESTAMP
		WHERE id = $2 RETURNING user_id
	`, status, clientID).Scan(&userID)
	if err != nil {
		return fmt.Errorf("failed to update client status: %w", err)
	}

	// Update Redis
	s.redis.HSet(ctx, fmt.Sprintf("user:%s:clients", userID), clientID, status)

	return nil
}

// GetUserClients returns all clients for a user
func (s *InboxService) GetUserClients(userID string) ([]models.Client, error) {
	rows, err := s.db.Query(`
		SELECT id, user_id, device_type, device_info, connection_status, last_active, created_at
		FROM clients WHERE user_id = $1
		ORDER BY last_active DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user clients: %w", err)
	}
	defer rows.Close()

	var clients []models.Client
	for rows.Next() {
		var client models.Client
		var deviceInfoJSON []byte
		if err := rows.Scan(
			&client.ID, &client.UserID, &client.DeviceType,
			&deviceInfoJSON, &client.ConnectionStatus,
			&client.LastActive, &client.CreatedAt,
		); err != nil {
			return nil, err
		}

		json.Unmarshal(deviceInfoJSON, &client.DeviceInfo)
		clients = append(clients, client)
	}

	return clients, nil
}

// GetOnlineClients returns all online clients for a user
func (s *InboxService) GetOnlineClients(userID string) ([]models.Client, error) {
	rows, err := s.db.Query(`
		SELECT id, user_id, device_type, device_info, connection_status, last_active, created_at
		FROM clients WHERE user_id = $1 AND connection_status = 'online'
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get online clients: %w", err)
	}
	defer rows.Close()

	var clients []models.Client
	for rows.Next() {
		var client models.Client
		var deviceInfoJSON []byte
		if err := rows.Scan(
			&client.ID, &client.UserID, &client.DeviceType,
			&deviceInfoJSON, &client.ConnectionStatus,
			&client.LastActive, &client.CreatedAt,
		); err != nil {
			return nil, err
		}

		json.Unmarshal(deviceInfoJSON, &client.DeviceInfo)
		clients = append(clients, client)
	}

	return clients, nil
}

// QueueMessageForOfflineClients queues a message for offline clients
func (s *InboxService) QueueMessageForOfflineClients(message *models.Message, participantUserIDs []string) error {
	for _, userID := range participantUserIDs {
		// Skip the sender
		if userID == message.SenderID {
			continue
		}

		// Get offline clients for this user
		clients, err := s.GetUserClients(userID)
		if err != nil {
			log.Printf("Error getting clients for user %s: %v", userID, err)
			continue
		}

		for _, client := range clients {
			if client.ConnectionStatus == "offline" {
				err = s.AddToInbox(client.ID, message.ID, message.ChatID)
				if err != nil {
					log.Printf("Error adding to inbox for client %s: %v", client.ID, err)
				}
			}
		}
	}

	return nil
}

// AddToInbox adds a message to a client's inbox
func (s *InboxService) AddToInbox(clientID, messageID, chatID string) error {
	_, err := s.db.Exec(`
		INSERT INTO inbox (client_id, message_id, chat_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (client_id, message_id) DO NOTHING
	`, clientID, messageID, chatID)

	if err != nil {
		return fmt.Errorf("failed to add to inbox: %w", err)
	}

	return nil
}

// GetPendingMessages returns pending messages for a client
func (s *InboxService) GetPendingMessages(clientID string) ([]models.InboxEntry, error) {
	rows, err := s.db.Query(`
		SELECT client_id, message_id, chat_id, delivery_attempts, created_at, ttl
		FROM inbox
		WHERE client_id = $1 AND ttl > CURRENT_TIMESTAMP
		ORDER BY created_at ASC
	`, clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending messages: %w", err)
	}
	defer rows.Close()

	var entries []models.InboxEntry
	for rows.Next() {
		var entry models.InboxEntry
		if err := rows.Scan(
			&entry.ClientID, &entry.MessageID, &entry.ChatID,
			&entry.DeliveryAttempts, &entry.CreatedAt, &entry.TTL,
		); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// MarkAsDelivered removes a message from a client's inbox
func (s *InboxService) MarkAsDelivered(clientID, messageID string) error {
	_, err := s.db.Exec(`
		DELETE FROM inbox WHERE client_id = $1 AND message_id = $2
	`, clientID, messageID)

	if err != nil {
		return fmt.Errorf("failed to mark as delivered: %w", err)
	}

	return nil
}

// IncrementDeliveryAttempts increments the delivery attempt counter
func (s *InboxService) IncrementDeliveryAttempts(clientID, messageID string) error {
	_, err := s.db.Exec(`
		UPDATE inbox SET delivery_attempts = delivery_attempts + 1
		WHERE client_id = $1 AND message_id = $2
	`, clientID, messageID)

	if err != nil {
		return fmt.Errorf("failed to increment delivery attempts: %w", err)
	}

	return nil
}

// CleanupExpiredEntries removes expired inbox entries
func (s *InboxService) CleanupExpiredEntries() (int64, error) {
	result, err := s.db.Exec(`DELETE FROM inbox WHERE ttl < CURRENT_TIMESTAMP`)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired entries: %w", err)
	}

	return result.RowsAffected()
}

// SyncMessagesOnReconnect delivers pending messages when a client reconnects
func (s *InboxService) SyncMessagesOnReconnect(clientID string, messageService *MessageService) ([]models.Message, error) {
	entries, err := s.GetPendingMessages(clientID)
	if err != nil {
		return nil, err
	}

	var messages []models.Message
	for _, entry := range entries {
		message, err := messageService.GetMessageByID(context.Background(), entry.MessageID)
		if err != nil {
			log.Printf("Error getting message %s: %v", entry.MessageID, err)
			continue
		}
		messages = append(messages, *message)
	}

	return messages, nil
}

// StartCleanupScheduler starts a background job to cleanup expired entries
func (s *InboxService) StartCleanupScheduler() {
	go func() {
		ticker := time.NewTicker(6 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			deleted, err := s.CleanupExpiredEntries()
			if err != nil {
				log.Printf("Error cleaning up inbox: %v", err)
			} else if deleted > 0 {
				log.Printf("Cleaned up %d expired inbox entries", deleted)
			}
		}
	}()
}

// GetInboxStats returns statistics about the inbox
func (s *InboxService) GetInboxStats() (map[string]interface{}, error) {
	var totalEntries, expiredEntries int

	err := s.db.QueryRow(`SELECT COUNT(*) FROM inbox`).Scan(&totalEntries)
	if err != nil {
		return nil, err
	}

	err = s.db.QueryRow(`SELECT COUNT(*) FROM inbox WHERE ttl < CURRENT_TIMESTAMP`).Scan(&expiredEntries)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"totalEntries":   totalEntries,
		"expiredEntries": expiredEntries,
		"pendingEntries": totalEntries - expiredEntries,
	}, nil
}
