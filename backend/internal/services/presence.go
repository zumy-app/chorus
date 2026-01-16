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

// PresenceService handles user online/offline status
type PresenceService struct {
	db     *sql.DB
	redis  *redis.Client
	pubsub *PubSubService
}

// NewPresenceService creates a new Presence service
func NewPresenceService(db *sql.DB, redis *redis.Client, pubsub *PubSubService) *PresenceService {
	return &PresenceService{
		db:     db,
		redis:  redis,
		pubsub: pubsub,
	}
}

// SetUserOnline is a compatibility wrapper for legacy handlers.
func (s *PresenceService) SetUserOnline(userID string) error {
	return s.SetOnline(userID, "")
}

// SetUserOffline is a compatibility wrapper for legacy handlers.
func (s *PresenceService) SetUserOffline(userID string) error {
	return s.SetOffline(userID)
}

// SetOnline marks a user as online
func (s *PresenceService) SetOnline(userID, deviceType string) error {
	ctx := context.Background()

	presence := models.PresenceStatus{
		UserID:     userID,
		Status:     "online",
		LastSeen:   time.Now(),
		DeviceType: deviceType,
	}

	// Store in Redis with TTL
	key := fmt.Sprintf("presence:%s", userID)
	data, _ := json.Marshal(presence)
	s.redis.Set(ctx, key, data, 5*time.Minute)

	// Add to online users set
	s.redis.SAdd(ctx, "online_users", userID)

	// Update database
	s.db.Exec(`
		INSERT INTO presence_log (user_id, status, device_type)
		VALUES ($1, 'online', $2)
	`, userID, deviceType)

	// Broadcast presence update
	if s.pubsub != nil {
		s.pubsub.PublishPresenceUpdate(userID, "online", deviceType)
	}

	return nil
}

// SetOffline marks a user as offline
func (s *PresenceService) SetOffline(userID string) error {
	ctx := context.Background()

	// Update Redis
	key := fmt.Sprintf("presence:%s", userID)

	presence := models.PresenceStatus{
		UserID:   userID,
		Status:   "offline",
		LastSeen: time.Now(),
	}
	data, _ := json.Marshal(presence)
	s.redis.Set(ctx, key, data, 24*time.Hour)

	// Remove from online users set
	s.redis.SRem(ctx, "online_users", userID)

	// Update database
	s.db.Exec(`
		INSERT INTO presence_log (user_id, status)
		VALUES ($1, 'offline')
	`, userID)

	// Broadcast presence update
	if s.pubsub != nil {
		s.pubsub.PublishPresenceUpdate(userID, "offline", "")
	}

	return nil
}

// UpdateUserActivity refreshes last_seen in the DB and presence TTL in Redis.
func (s *PresenceService) UpdateUserActivity(userID string) error {
	if _, err := s.db.Exec(`UPDATE users SET last_seen = CURRENT_TIMESTAMP WHERE id = $1`, userID); err != nil {
		return err
	}
	return s.SetOnline(userID, "")
}

// SetAway marks a user as away
func (s *PresenceService) SetAway(userID string) error {
	ctx := context.Background()

	key := fmt.Sprintf("presence:%s", userID)

	presence := models.PresenceStatus{
		UserID:   userID,
		Status:   "away",
		LastSeen: time.Now(),
	}
	data, _ := json.Marshal(presence)
	s.redis.Set(ctx, key, data, 5*time.Minute)

	// Broadcast presence update
	if s.pubsub != nil {
		s.pubsub.PublishPresenceUpdate(userID, "away", "")
	}

	return nil
}

// GetPresence returns the presence status for a user
func (s *PresenceService) GetPresence(userID string) (*models.PresenceStatus, error) {
	ctx := context.Background()
	key := fmt.Sprintf("presence:%s", userID)

	data, err := s.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		// User not in Redis, check database
		var status string
		var timestamp time.Time
		var deviceType sql.NullString

		err = s.db.QueryRow(`
			SELECT status, device_type, timestamp
			FROM presence_log
			WHERE user_id = $1
			ORDER BY timestamp DESC
			LIMIT 1
		`, userID).Scan(&status, &deviceType, &timestamp)

		if err == sql.ErrNoRows {
			return &models.PresenceStatus{
				UserID:   userID,
				Status:   "offline",
				LastSeen: time.Time{},
			}, nil
		}
		if err != nil {
			return nil, err
		}

		presence := &models.PresenceStatus{
			UserID:   userID,
			Status:   status,
			LastSeen: timestamp,
		}
		if deviceType.Valid {
			presence.DeviceType = deviceType.String
		}

		return presence, nil
	}
	if err != nil {
		return nil, err
	}

	var presence models.PresenceStatus
	if err := json.Unmarshal([]byte(data), &presence); err != nil {
		return nil, err
	}

	return &presence, nil
}

// GetMultiplePresence returns presence status for multiple users
func (s *PresenceService) GetMultiplePresence(userIDs []string) (map[string]*models.PresenceStatus, error) {
	ctx := context.Background()
	result := make(map[string]*models.PresenceStatus)

	// Build keys
	keys := make([]string, len(userIDs))
	for i, id := range userIDs {
		keys[i] = fmt.Sprintf("presence:%s", id)
	}

	// Get from Redis
	values, err := s.redis.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}

	for i, val := range values {
		userID := userIDs[i]
		if val == nil {
			result[userID] = &models.PresenceStatus{
				UserID: userID,
				Status: "offline",
			}
			continue
		}

		var presence models.PresenceStatus
		if err := json.Unmarshal([]byte(val.(string)), &presence); err != nil {
			result[userID] = &models.PresenceStatus{
				UserID: userID,
				Status: "offline",
			}
			continue
		}
		result[userID] = &presence
	}

	return result, nil
}

// IsOnline checks if a user is currently online
func (s *PresenceService) IsOnline(userID string) bool {
	ctx := context.Background()
	return s.redis.SIsMember(ctx, "online_users", userID).Val()
}

// GetOnlineUsers returns all online users
func (s *PresenceService) GetOnlineUsers() ([]string, error) {
	ctx := context.Background()
	return s.redis.SMembers(ctx, "online_users").Result()
}

// GetOnlineUserCount returns the count of online users
func (s *PresenceService) GetOnlineUserCount() (int64, error) {
	ctx := context.Background()
	return s.redis.SCard(ctx, "online_users").Result()
}

// GetUserLastSeen returns last seen timestamp if available.
func (s *PresenceService) GetUserLastSeen(userID string) (*time.Time, error) {
	presence, err := s.GetPresence(userID)
	if err != nil {
		return nil, err
	}
	if presence.LastSeen.IsZero() {
		return nil, nil
	}
	return &presence.LastSeen, nil
}

// Heartbeat updates the user's last activity time
func (s *PresenceService) Heartbeat(userID, deviceType string) error {
	ctx := context.Background()
	key := fmt.Sprintf("presence:%s", userID)

	presence := models.PresenceStatus{
		UserID:     userID,
		Status:     "online",
		LastSeen:   time.Now(),
		DeviceType: deviceType,
	}
	data, _ := json.Marshal(presence)

	return s.redis.Set(ctx, key, data, 5*time.Minute).Err()
}

// StartPresenceCleanup starts a background job to cleanup stale presence data
func (s *PresenceService) StartPresenceCleanup() {
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			s.cleanupStalePresence()
		}
	}()
}

// cleanupStalePresence removes stale presence entries
func (s *PresenceService) cleanupStalePresence() {
	ctx := context.Background()

	// Get all online users
	onlineUsers, err := s.redis.SMembers(ctx, "online_users").Result()
	if err != nil {
		log.Printf("Error getting online users: %v", err)
		return
	}

	for _, userID := range onlineUsers {
		key := fmt.Sprintf("presence:%s", userID)
		exists, err := s.redis.Exists(ctx, key).Result()
		if err != nil || exists == 0 {
			// Presence key expired, mark user as offline
			s.redis.SRem(ctx, "online_users", userID)
			s.SetOffline(userID)
		}
	}
}

// GetLastSeen returns the last seen time for a user
func (s *PresenceService) GetLastSeen(userID string) (time.Time, error) {
	presence, err := s.GetPresence(userID)
	if err != nil {
		return time.Time{}, err
	}
	return presence.LastSeen, nil
}

// GetPresenceStats returns presence statistics
func (s *PresenceService) GetPresenceStats() (map[string]interface{}, error) {
	ctx := context.Background()

	onlineCount, err := s.redis.SCard(ctx, "online_users").Result()
	if err != nil {
		return nil, err
	}

	// Get recent activity count from database
	var recentActive int
	s.db.QueryRow(`
		SELECT COUNT(DISTINCT user_id)
		FROM presence_log
		WHERE timestamp > CURRENT_TIMESTAMP - INTERVAL '24 hours'
	`).Scan(&recentActive)

	return map[string]interface{}{
		"onlineUsers":    onlineCount,
		"activeUsers24h": recentActive,
	}, nil
}
