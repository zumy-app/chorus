package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/chorus/messenger/internal/models"
	"github.com/google/uuid"
)

type ClientService struct {
	db *sql.DB
}

func NewClientService(db *sql.DB) *ClientService {
	return &ClientService{
		db: db,
	}
}

// RegisterClient registers a new client device for a user
func (s *ClientService) RegisterClient(ctx context.Context, userID string, deviceType string, deviceInfo models.DeviceInfo) (*models.Client, error) {
	// Check if client already exists for this device
	existingID, err := s.findExistingClient(ctx, userID, deviceInfo)
	if err == nil && existingID != "" {
		// Update existing client
		return s.updateClientStatus(ctx, existingID, "online")
	}
	
	// Check client limit (max 3 devices per user)
	count, err := s.getActiveClientCount(ctx, userID)
	if err != nil {
		return nil, err
	}
	
	if count >= 3 {
		return nil, fmt.Errorf("maximum 3 devices allowed per user")
	}
	
	clientID := uuid.New().String()
	deviceInfoJSON, _ := json.Marshal(deviceInfo)
	
	query := `
		INSERT INTO clients (
			id, user_id, device_type, device_info, 
			connection_status, last_active, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	
	now := time.Now()
	_, err = s.db.ExecContext(ctx, query,
		clientID, userID, deviceType, deviceInfoJSON,
		"online", now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to register client: %w", err)
	}
	
	return &models.Client{
		ID:               clientID,
		UserID:           userID,
		DeviceType:       deviceType,
		DeviceInfo:       deviceInfo,
		ConnectionStatus: "online",
		LastActive:       now,
		CreatedAt:        now,
	}, nil
}

// UpdateClientStatus updates the connection status of a client
func (s *ClientService) updateClientStatus(ctx context.Context, clientID string, status string) (*models.Client, error) {
	query := `
		UPDATE clients
		SET connection_status = $1, last_active = $2
		WHERE id = $3
		RETURNING id, user_id, device_type, device_info, connection_status, last_active, created_at
	`
	
	var client models.Client
	var deviceInfoJSON []byte
	
	err := s.db.QueryRowContext(ctx, query, status, time.Now(), clientID).Scan(
		&client.ID, &client.UserID, &client.DeviceType, &deviceInfoJSON,
		&client.ConnectionStatus, &client.LastActive, &client.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update client status: %w", err)
	}
	
	json.Unmarshal(deviceInfoJSON, &client.DeviceInfo)
	
	return &client, nil
}

// SetClientOnline marks a client as online
func (s *ClientService) SetClientOnline(ctx context.Context, clientID string) error {
	_, err := s.updateClientStatus(ctx, clientID, "online")
	return err
}

// SetClientOffline marks a client as offline
func (s *ClientService) SetClientOffline(ctx context.Context, clientID string) error {
	_, err := s.updateClientStatus(ctx, clientID, "offline")
	return err
}

// GetUserClients retrieves all clients for a user
func (s *ClientService) GetUserClients(ctx context.Context, userID string) ([]models.Client, error) {
	query := `
		SELECT id, user_id, device_type, device_info, 
		       connection_status, last_active, created_at
		FROM clients
		WHERE user_id = $1
		ORDER BY last_active DESC
	`
	
	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query clients: %w", err)
	}
	defer rows.Close()
	
	clients := []models.Client{}
	for rows.Next() {
		var client models.Client
		var deviceInfoJSON []byte
		
		err := rows.Scan(
			&client.ID, &client.UserID, &client.DeviceType, &deviceInfoJSON,
			&client.ConnectionStatus, &client.LastActive, &client.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		
		json.Unmarshal(deviceInfoJSON, &client.DeviceInfo)
		clients = append(clients, client)
	}
	
	return clients, nil
}

// GetOnlineClients retrieves all online clients for a user
func (s *ClientService) GetOnlineClients(ctx context.Context, userID string) ([]models.Client, error) {
	query := `
		SELECT id, user_id, device_type, device_info, 
		       connection_status, last_active, created_at
		FROM clients
		WHERE user_id = $1 AND connection_status = 'online'
		ORDER BY last_active DESC
	`
	
	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query online clients: %w", err)
	}
	defer rows.Close()
	
	clients := []models.Client{}
	for rows.Next() {
		var client models.Client
		var deviceInfoJSON []byte
		
		err := rows.Scan(
			&client.ID, &client.UserID, &client.DeviceType, &deviceInfoJSON,
			&client.ConnectionStatus, &client.LastActive, &client.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		
		json.Unmarshal(deviceInfoJSON, &client.DeviceInfo)
		clients = append(clients, client)
	}
	
	return clients, nil
}

// GetClient retrieves a specific client
func (s *ClientService) GetClient(ctx context.Context, clientID string) (*models.Client, error) {
	query := `
		SELECT id, user_id, device_type, device_info, 
		       connection_status, last_active, created_at
		FROM clients
		WHERE id = $1
	`
	
	var client models.Client
	var deviceInfoJSON []byte
	
	err := s.db.QueryRowContext(ctx, query, clientID).Scan(
		&client.ID, &client.UserID, &client.DeviceType, &deviceInfoJSON,
		&client.ConnectionStatus, &client.LastActive, &client.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("client not found")
		}
		return nil, err
	}
	
	json.Unmarshal(deviceInfoJSON, &client.DeviceInfo)
	
	return &client, nil
}

// DeleteClient removes a client device
func (s *ClientService) DeleteClient(ctx context.Context, clientID string, userID string) error {
	query := `DELETE FROM clients WHERE id = $1 AND user_id = $2`
	result, err := s.db.ExecContext(ctx, query, clientID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete client: %w", err)
	}
	
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("client not found")
	}
	
	// Clean up inbox entries for this client
	_, err = s.db.ExecContext(ctx, `DELETE FROM inbox WHERE client_id = $1`, clientID)
	
	return err
}

// CleanupInactiveClients marks clients as offline if inactive for > 5 minutes
func (s *ClientService) CleanupInactiveClients(ctx context.Context) error {
	query := `
		UPDATE clients
		SET connection_status = 'offline'
		WHERE connection_status = 'online' 
		  AND last_active < $1
	`
	
	fiveMinutesAgo := time.Now().Add(-5 * time.Minute)
	_, err := s.db.ExecContext(ctx, query, fiveMinutesAgo)
	if err != nil {
		return fmt.Errorf("failed to cleanup inactive clients: %w", err)
	}
	
	return nil
}

// UpdateLastActive updates the last active timestamp
func (s *ClientService) UpdateLastActive(ctx context.Context, clientID string) error {
	query := `UPDATE clients SET last_active = $1 WHERE id = $2`
	_, err := s.db.ExecContext(ctx, query, time.Now(), clientID)
	if err != nil {
		return fmt.Errorf("failed to update last active: %w", err)
	}
	
	return nil
}

// Helper functions

func (s *ClientService) findExistingClient(ctx context.Context, userID string, deviceInfo models.DeviceInfo) (string, error) {
	query := `
		SELECT id FROM clients
		WHERE user_id = $1 
		  AND device_info->>'platform' = $2
		  AND device_info->>'version' = $3
		LIMIT 1
	`
	
	var clientID string
	err := s.db.QueryRowContext(ctx, query, userID, deviceInfo.Platform, deviceInfo.Version).Scan(&clientID)
	if err != nil {
		return "", err
	}
	
	return clientID, nil
}

func (s *ClientService) getActiveClientCount(ctx context.Context, userID string) (int, error) {
	query := `SELECT COUNT(*) FROM clients WHERE user_id = $1`
	
	var count int
	err := s.db.QueryRowContext(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, err
	}
	
	return count, nil
}

// GetClientByUserAndDevice finds a client by user ID and device info
func (s *ClientService) GetClientByUserAndDevice(ctx context.Context, userID string, deviceType string) (*models.Client, error) {
	query := `
		SELECT id, user_id, device_type, device_info, 
		       connection_status, last_active, created_at
		FROM clients
		WHERE user_id = $1 AND device_type = $2
		ORDER BY last_active DESC
		LIMIT 1
	`
	
	var client models.Client
	var deviceInfoJSON []byte
	
	err := s.db.QueryRowContext(ctx, query, userID, deviceType).Scan(
		&client.ID, &client.UserID, &client.DeviceType, &deviceInfoJSON,
		&client.ConnectionStatus, &client.LastActive, &client.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("client not found")
		}
		return nil, err
	}
	
	json.Unmarshal(deviceInfoJSON, &client.DeviceInfo)
	
	return &client, nil
}

// GetAllOnlineClientsForChat retrieves all online clients for all participants in a chat
func (s *ClientService) GetAllOnlineClientsForChat(ctx context.Context, chatID string) (map[string][]models.Client, error) {
	query := `
		SELECT c.id, c.user_id, c.device_type, c.device_info, 
		       c.connection_status, c.last_active, c.created_at
		FROM clients c
		INNER JOIN chat_participants cp ON c.user_id = cp.user_id
		WHERE cp.chat_id = $1 AND c.connection_status = 'online'
		ORDER BY c.user_id, c.last_active DESC
	`
	
	rows, err := s.db.QueryContext(ctx, query, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to query chat clients: %w", err)
	}
	defer rows.Close()
	
	clientsByUser := make(map[string][]models.Client)
	
	for rows.Next() {
		var client models.Client
		var deviceInfoJSON []byte
		
		err := rows.Scan(
			&client.ID, &client.UserID, &client.DeviceType, &deviceInfoJSON,
			&client.ConnectionStatus, &client.LastActive, &client.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		
		json.Unmarshal(deviceInfoJSON, &client.DeviceInfo)
		
		clientsByUser[client.UserID] = append(clientsByUser[client.UserID], client)
	}
	
	return clientsByUser, nil
}
