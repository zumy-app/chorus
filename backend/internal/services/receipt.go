package services

import (
	"context"

	"github.com/chorus/messenger/internal/models"
)

// ReceiptService bridges per-recipient receipt persistence (MessageService) with
// real-time fan-out: when a recipient's client acks a message as delivered or
// read, the sender (and other participants) are notified so their tick updates.
type ReceiptService struct {
	hub     *WebSocketHub
	pubsub  *PubSubService
	message *MessageService
	chat    *ChatService
}

func NewReceiptService(hub *WebSocketHub, pubsub *PubSubService, message *MessageService, chat *ChatService) *ReceiptService {
	return &ReceiptService{
		hub:     hub,
		pubsub:  pubsub,
		message: message,
		chat:    chat,
	}
}

// AcknowledgeReceived marks a recipient's tick 'delivered' and, if the state
// changed, notifies the chat participants.
func (r *ReceiptService) AcknowledgeReceived(ctx context.Context, chatID, messageID, userID string) error {
	if r.message == nil {
		return nil
	}

	changed, err := r.message.MarkDelivered(chatID, messageID, userID)
	if err != nil {
		return err
	}
	if changed {
		r.FanOutReceipt(ctx, chatID, messageID, userID, "delivered")
	}
	return nil
}

// AcknowledgeRead marks a recipient's tick 'read' and, if the state changed,
// notifies the chat participants.
func (r *ReceiptService) AcknowledgeRead(ctx context.Context, chatID, messageID, userID string) error {
	if r.message == nil {
		return nil
	}

	changed, err := r.message.MarkRead(chatID, messageID, userID)
	if err != nil {
		return err
	}
	if changed {
		r.FanOutReceipt(ctx, chatID, messageID, userID, "read")
	}
	return nil
}

// FanOutReceipt pushes a delivered/read tick to every participant of the chat,
// locally through the hub and across instances through Redis pub/sub.
func (r *ReceiptService) FanOutReceipt(ctx context.Context, chatID, messageID, recipientUserID, status string) {
	if r.hub == nil || r.chat == nil {
		return
	}

	participants, err := r.chat.GetParticipants(chatID)
	if err != nil {
		return
	}

	userIDs := make([]string, 0, len(participants))
	for _, p := range participants {
		userIDs = append(userIDs, p.UserID)
	}

	eventType := "message_delivered"
	if status == "read" {
		eventType = "message_read"
	}

	event := models.ReceiptEvent{
		ChatID:    chatID,
		MessageID: messageID,
		UserID:    recipientUserID,
		Status:    status,
	}

	r.hub.SendToChat(chatID, userIDs, eventType, event)
	if r.pubsub != nil {
		_ = r.pubsub.PublishToChat(chatID, userIDs, eventType, event)
	}
}
