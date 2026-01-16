package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type PresenceService struct {
	db      *sql.DB
	redis   *redis.Client
	pubsub  *PubSubService
	ctx     context.Context
}

func NewPresenceService(db *sql.DB, redis *redis.Client, pubsub *PubSubService) *PresenceService {
	return &PresenceService{
		db:     db,
		redis:  redis,
		pubsub: pubsub,
		ctx:    context.Background(),
	}
}

func (s *PresenceService) SetUserOnline(userID string) error {
	// Set user online in Redis with TTL
	key := fmt.Sprintf("presence:user:%s", userID)
	return s.redis.Set(s.ctx, key, "online", 5*time.Minute).Err()
}

func (s *PresenceService) SetUserOffline(userID string) error {
	key := fmt.Sprintf("presence:user:%s", userID)
	return s.redis.Del(s.ctx, key).Err()
}

func (s *PresenceService) IsUserOnline(userID string) (bool, error) {
	key := fmt.Sprintf("presence:user:%s", userID)
	exists, err := s.redis.Exists(s.ctx, key).Result()
	return exists > 0, err
}

func (s *PresenceService) UpdateUserActivity(userID string) error {
	// Update user activity timestamp
	query := `UPDATE users SET last_seen = NOW() WHERE id = $1`
	_, err := s.db.Exec(query, userID)
	if err != nil {
		return err
	}

	// Refresh online status in Redis
	return s.SetUserOnline(userID)
}

func (s *PresenceService) GetUserLastSeen(userID string) (*time.Time, error) {
	var lastSeen time.Time
	query := `SELECT last_seen FROM users WHERE id = $1`
	err := s.db.QueryRow(query, userID).Scan(&lastSeen)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &lastSeen, nil
}

func (s *PresenceService) StartPresenceCleanup() {
	// Periodically clean up stale presence data
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			// Cleanup logic already handled by Redis TTL
			// This is just a placeholder for any additional cleanup
		}
	}()
}
