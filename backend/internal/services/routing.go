package services

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/chorus/messenger/internal/models"
	"github.com/redis/go-redis/v9"
)

func ServerChannel(serverID string) string {
	return "server:" + serverID
}

type DeliveryRouter struct {
	redis          *redis.Client
	hub            *WebSocketHub
	registry       *ConnectionRegistry
	messageService *MessageService
	serverID       string
	cancel         context.CancelFunc
}

func NewDeliveryRouter(redisClient *redis.Client, hub *WebSocketHub, registry *ConnectionRegistry, msgSvc *MessageService, serverID string) *DeliveryRouter {
	return &DeliveryRouter{
		redis:          redisClient,
		hub:            hub,
		registry:       registry,
		messageService: msgSvc,
		serverID:       serverID,
	}
}

func (r *DeliveryRouter) Start(ctx context.Context) {
	if r.redis == nil || r.serverID == "" {
		return
	}
	cctx, cancel := context.WithCancel(ctx)
	r.cancel = cancel
	go r.subscribe(cctx)
}

func (r *DeliveryRouter) Stop() {
	if r.cancel != nil {
		r.cancel()
	}
}

func (r *DeliveryRouter) RouteMessage(ctx context.Context, message *models.Message, participantIDs []string) {
	if message == nil {
		return
	}
	for _, uid := range participantIDs {
		if uid == "" || uid == message.SenderID {
			continue
		}
		if r.registry == nil || r.redis == nil {
			if r.hub != nil {
				r.hub.SendToUser(uid, "new_message", message)
			}
			continue
		}
		entry, _ := r.registry.Lookup(ctx, uid)
		if entry == nil || entry.ServerID == "" {
			if r.hub != nil {
				r.hub.SendToUser(uid, "new_message", message)
			}
			continue
		}
		if entry.ServerID == r.serverID {
			if r.hub != nil {
				r.hub.SendToUser(uid, "new_message", message)
			}
			if r.hub != nil && r.hub.IsUserOnline(uid) && r.messageService != nil {
				_, _ = r.messageService.MarkDelivered(message.ChatID, message.ID, uid)
			}
			continue
		}
		if err := r.publishToServer(ctx, entry.ServerID, message, uid); err != nil {
			log.Printf("[Router] publish to server %s for user %s failed: %v", entry.ServerID, uid, err)
			if r.hub != nil {
				r.hub.SendToUser(uid, "new_message", message)
			}
		}
	}
}

func (r *DeliveryRouter) publishToServer(ctx context.Context, targetServer string, message *models.Message, targetUser string) error {
	if r.redis == nil || targetServer == "" {
		return nil
	}
	msg := models.PubSubMessage{
		Type:       "new_message",
		Data:       message,
		TargetUser: targetUser,
		ChatID:     message.ChatID,
		Timestamp:  time.Now(),
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	cctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return r.redis.Publish(cctx, ServerChannel(targetServer), data).Err()
}

func (r *DeliveryRouter) subscribe(ctx context.Context) {
	channel := ServerChannel(r.serverID)
	sub := r.redis.Subscribe(ctx, channel)
	defer sub.Close()
	ch := sub.Channel()
	log.Printf("[Router] subscribed to %s", channel)
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			r.handleServerMessage(msg.Payload)
		}
	}
}

func (r *DeliveryRouter) handleServerMessage(payload string) {
	var pm models.PubSubMessage
	if err := json.Unmarshal([]byte(payload), &pm); err != nil {
		log.Printf("[Router] unmarshal server message: %v", err)
		return
	}
	if pm.TargetUser == "" {
		return
	}
	var message *models.Message
	if pm.Data != nil {
		raw, err := json.Marshal(pm.Data)
		if err == nil {
			var m models.Message
			if err := json.Unmarshal(raw, &m); err == nil && m.ID != "" {
				message = &m
			}
		}
	}
	if r.hub != nil {
		data := pm.Data
		if message != nil {
			data = message
		}
		r.hub.SendToUser(pm.TargetUser, pm.Type, data)
	}
	if message != nil && r.messageService != nil {
		_, _ = r.messageService.MarkDelivered(pm.ChatID, message.ID, pm.TargetUser)
	}
}

func (r *DeliveryRouter) HandlePayloadForTest(payload string) {
	r.handleServerMessage(payload)
}
