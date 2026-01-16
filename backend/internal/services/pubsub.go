package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/chorus/messenger/internal/models"
	"github.com/redis/go-redis/v9"
)

// PubSubService handles Redis Pub/Sub for scalable real-time messaging
type PubSubService struct {
	redis       *redis.Client
	hub         *WebSocketHub
	ctx         context.Context
	cancel      context.CancelFunc
	subscribers map[string]*redis.PubSub
	mu          sync.RWMutex
}

// NewPubSubService creates a new Pub/Sub service
func NewPubSubService(redisClient *redis.Client, hub *WebSocketHub) *PubSubService {
	ctx, cancel := context.WithCancel(context.Background())
	return &PubSubService{
		redis:       redisClient,
		hub:         hub,
		ctx:         ctx,
		cancel:      cancel,
		subscribers: make(map[string]*redis.PubSub),
	}
}

// Start begins listening to Pub/Sub channels
func (p *PubSubService) Start() {
	// Subscribe to general broadcast channel
	go p.subscribeToChannel("broadcast")

	log.Println("PubSub service started")
}

// Stop gracefully shuts down the Pub/Sub service
func (p *PubSubService) Stop() {
	p.cancel()

	p.mu.Lock()
	defer p.mu.Unlock()

	for channel, sub := range p.subscribers {
		sub.Close()
		delete(p.subscribers, channel)
	}

	log.Println("PubSub service stopped")
}

// SubscribeToUser subscribes to a user's message channel
func (p *PubSubService) SubscribeToUser(userID string) {
	channel := fmt.Sprintf("user:%s", userID)
	go p.subscribeToChannel(channel)
}

// UnsubscribeFromUser unsubscribes from a user's message channel
func (p *PubSubService) UnsubscribeFromUser(userID string) {
	channel := fmt.Sprintf("user:%s", userID)

	p.mu.Lock()
	defer p.mu.Unlock()

	if sub, ok := p.subscribers[channel]; ok {
		sub.Close()
		delete(p.subscribers, channel)
	}
}

// SubscribeToChat subscribes to a chat channel for typing indicators, etc.
func (p *PubSubService) SubscribeToChat(chatID string) {
	channel := fmt.Sprintf("chat:%s", chatID)
	go p.subscribeToChannel(channel)
}

// PublishToUser publishes a message to a specific user's channel
func (p *PubSubService) PublishToUser(userID string, msgType string, data interface{}) error {
	channel := fmt.Sprintf("user:%s", userID)
	return p.publish(channel, msgType, data, userID, "")
}

// PublishToChat publishes a message to all users in a chat
func (p *PubSubService) PublishToChat(chatID string, userIDs []string, msgType string, data interface{}) error {
	// Publish to each user's channel
	for _, userID := range userIDs {
		if err := p.PublishToUser(userID, msgType, data); err != nil {
			log.Printf("Error publishing to user %s: %v", userID, err)
		}
	}

	// Also publish to chat channel for chat-wide events
	channel := fmt.Sprintf("chat:%s", chatID)
	return p.publish(channel, msgType, data, "", chatID)
}

// PublishBroadcast publishes a message to all connected users
func (p *PubSubService) PublishBroadcast(msgType string, data interface{}) error {
	return p.publish("broadcast", msgType, data, "", "")
}

// PublishTypingEvent publishes a typing indicator
func (p *PubSubService) PublishTypingEvent(chatID, userID string, isTyping bool) error {
	event := models.TypingEvent{
		ChatID:   chatID,
		UserID:   userID,
		IsTyping: isTyping,
	}

	channel := fmt.Sprintf("chat:%s", chatID)
	return p.publish(channel, "user_typing", event, "", chatID)
}

// PublishPresenceUpdate publishes a user's presence status
func (p *PubSubService) PublishPresenceUpdate(userID, status, deviceType string) error {
	presence := models.PresenceStatus{
		UserID:     userID,
		Status:     status,
		LastSeen:   time.Now(),
		DeviceType: deviceType,
	}

	return p.publish("broadcast", "user_presence", presence, "", "")
}

// PublishMessage provides a simple channel publish helper (compatibility with remote API).
func (p *PubSubService) PublishMessage(channel string, message interface{}) error {
	jsonData, err := json.Marshal(message)
	if err != nil {
		return err
	}
	return p.redis.Publish(p.ctx, channel, jsonData).Err()
}

// PublishNewMessage publishes a new message to chat participants
func (p *PubSubService) PublishNewMessage(chatID string, userIDs []string, message *models.Message) error {
	return p.PublishToChat(chatID, userIDs, "new_message", message)
}

// PublishMessageDelivered publishes message delivery confirmation
func (p *PubSubService) PublishMessageDelivered(chatID, messageID, userID string) error {
	data := map[string]string{
		"chatId":    chatID,
		"messageId": messageID,
		"status":    "delivered",
	}
	return p.PublishToUser(userID, "message_delivered", data)
}

// PublishMessageRead publishes message read status
func (p *PubSubService) PublishMessageRead(chatID, messageID, userID string) error {
	data := map[string]string{
		"chatId":    chatID,
		"messageId": messageID,
		"status":    "read",
	}
	return p.PublishToUser(userID, "message_read", data)
}

// publish sends a message to a Redis channel
func (p *PubSubService) publish(channel, msgType string, data interface{}, targetUser, chatID string) error {
	msg := models.PubSubMessage{
		Type:       msgType,
		Data:       data,
		TargetUser: targetUser,
		ChatID:     chatID,
		Timestamp:  time.Now(),
	}

	jsonData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	return p.redis.Publish(p.ctx, channel, jsonData).Err()
}

// subscribeToChannel listens to a specific Redis channel
func (p *PubSubService) subscribeToChannel(channel string) {
	p.mu.Lock()
	if _, exists := p.subscribers[channel]; exists {
		p.mu.Unlock()
		return
	}

	sub := p.redis.Subscribe(p.ctx, channel)
	p.subscribers[channel] = sub
	p.mu.Unlock()

	ch := sub.Channel()

	for {
		select {
		case <-p.ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			p.handleMessage(msg)
		}
	}
}

// handleMessage processes incoming Pub/Sub messages
func (p *PubSubService) handleMessage(msg *redis.Message) {
	var pubSubMsg models.PubSubMessage
	if err := json.Unmarshal([]byte(msg.Payload), &pubSubMsg); err != nil {
		log.Printf("Error unmarshaling PubSub message: %v", err)
		return
	}

	// Route message to appropriate WebSocket connections
	if pubSubMsg.TargetUser != "" {
		// Send to specific user
		p.hub.SendToUser(pubSubMsg.TargetUser, pubSubMsg.Type, pubSubMsg.Data)
	} else if pubSubMsg.ChatID != "" {
		// Handle chat-specific events (typing, etc.)
		// These are handled by individual user subscriptions
	} else {
		// Broadcast to all
		p.hub.BroadcastToAll(pubSubMsg.Type, pubSubMsg.Data)
	}
}

// GetChannelStats returns statistics about active channels
func (p *PubSubService) GetChannelStats() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return map[string]interface{}{
		"activeChannels": len(p.subscribers),
	}
}
