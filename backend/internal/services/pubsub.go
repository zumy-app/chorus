package services

import (
	"context"
	"encoding/json"
	"log"

	"github.com/redis/go-redis/v9"
)

type PubSubService struct {
	redis *redis.Client
	wsHub *WebSocketHub
	ctx   context.Context
	cancel context.CancelFunc
}

func NewPubSubService(redis *redis.Client, wsHub *WebSocketHub) *PubSubService {
	ctx, cancel := context.WithCancel(context.Background())
	return &PubSubService{
		redis:  redis,
		wsHub:  wsHub,
		ctx:    ctx,
		cancel: cancel,
	}
}

func (s *PubSubService) Start() {
	go func() {
		pubsub := s.redis.Subscribe(s.ctx, "chat:messages", "chat:events")
		defer pubsub.Close()

		ch := pubsub.Channel()

		for {
			select {
			case msg, ok := <-ch:
				if !ok {
					log.Println("PubSub channel closed")
					return
				}
				if msg == nil {
					continue
				}
				// Forward Redis pub/sub messages to WebSocket clients
				var data map[string]interface{}
				if err := json.Unmarshal([]byte(msg.Payload), &data); err != nil {
					log.Printf("Failed to unmarshal pub/sub message: %v", err)
					continue
				}

				// Broadcast to appropriate WebSocket clients
				// based on the chat_id in the message
				log.Printf("Received pub/sub message on %s: %v", msg.Channel, data)
			case <-s.ctx.Done():
				return
			}
		}
	}()
}

func (s *PubSubService) Stop() {
	s.cancel()
}

func (s *PubSubService) PublishMessage(channel string, message interface{}) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}
	return s.redis.Publish(s.ctx, channel, data).Err()
}
