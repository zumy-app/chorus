package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const RegistryKeyPrefix = "ws:registry:"
const RegistryTTL = 45 * time.Second
const RegistryHeartbeatInterval = 15 * time.Second

type RegistryEntry struct {
	ServerID  string    `json:"serverId"`
	ConnID    string    `json:"connId"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func RegistryKey(userID string) string {
	return RegistryKeyPrefix + userID
}

type ConnectionRegistry struct {
	redis    *redis.Client
	serverID string
	ttl      time.Duration
}

func NewConnectionRegistry(redisClient *redis.Client, serverID string) *ConnectionRegistry {
	return &ConnectionRegistry{
		redis:    redisClient,
		serverID: serverID,
		ttl:      RegistryTTL,
	}
}

func NewConnectionRegistryWithTTL(redisClient *redis.Client, serverID string, ttl time.Duration) *ConnectionRegistry {
	if ttl <= 0 {
		ttl = RegistryTTL
	}
	return &ConnectionRegistry{
		redis:    redisClient,
		serverID: serverID,
		ttl:      ttl,
	}
}

func (r *ConnectionRegistry) TTL() time.Duration {
	return r.ttl
}

func (r *ConnectionRegistry) ServerID() string {
	return r.serverID
}

func (r *ConnectionRegistry) Register(ctx context.Context, userID, connID string) error {
	if r == nil || r.redis == nil || userID == "" || connID == "" {
		return nil
	}
	entry := RegistryEntry{
		ServerID:  r.serverID,
		ConnID:    connID,
		UpdatedAt: time.Now().UTC(),
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	return r.redis.Set(ctx, RegistryKey(userID), data, r.ttl).Err()
}

func (r *ConnectionRegistry) Refresh(ctx context.Context, userID, connID string) error {
	if r == nil || r.redis == nil || userID == "" || connID == "" {
		return nil
	}
	script := redis.NewScript(`
local v = redis.call('GET', KEYS[1])
if not v then return 0 end
local ok, data = pcall(cjson.decode, v)
if ok and data and data.connId == ARGV[1] then
  redis.call('EXPIRE', KEYS[1], ARGV[2])
  return 1
end
if string.find(v, ARGV[1], 1, true) then
  redis.call('EXPIRE', KEYS[1], ARGV[2])
  return 1
end
return 0
`)
	_, err := script.Run(ctx, r.redis, []string{RegistryKey(userID)}, connID, fmt.Sprintf("%d", int(r.ttl.Seconds()))).Result()
	return err
}

func (r *ConnectionRegistry) Unregister(ctx context.Context, userID, connID string) error {
	if r == nil || r.redis == nil || userID == "" || connID == "" {
		return nil
	}
	script := redis.NewScript(`
local v = redis.call('GET', KEYS[1])
if not v then return 0 end
local ok, data = pcall(cjson.decode, v)
if ok and data and data.connId == ARGV[1] then
  redis.call('DEL', KEYS[1])
  return 1
end
if string.find(v, ARGV[1], 1, true) then
  redis.call('DEL', KEYS[1])
  return 1
end
return 0
`)
	_, err := script.Run(ctx, r.redis, []string{RegistryKey(userID)}, connID).Result()
	return err
}

func (r *ConnectionRegistry) Lookup(ctx context.Context, userID string) (*RegistryEntry, error) {
	if r == nil || r.redis == nil || userID == "" {
		return nil, nil
	}
	data, err := r.redis.Get(ctx, RegistryKey(userID)).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var entry RegistryEntry
	if err := json.Unmarshal([]byte(data), &entry); err != nil {
		return nil, err
	}
	return &entry, nil
}

func (r *ConnectionRegistry) IsOnline(ctx context.Context, userID string) (bool, error) {
	if r == nil || r.redis == nil {
		return false, nil
	}
	n, err := r.redis.Exists(ctx, RegistryKey(userID)).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}
