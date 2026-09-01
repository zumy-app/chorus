package services

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/chorus/messenger/internal/models"
	"github.com/chorus/messenger/internal/observability"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

// WebSocket message throttle bounds (NFR-24): at most wsMessageLimit inbound
// messages per wsMessageWindow per connection. A client that floods the hub
// with typing/join events is disconnected with a policy-violation close.
const (
	wsMessageLimit  = 60
	wsMessageWindow = 10 * time.Second
)

type Client struct {
	ID       string
	UserID   string
	Conn     *websocket.Conn
	Send     chan []byte
	Hub      *WebSocketHub
	Receipts *ReceiptService
	Calls    *CallService
	msgLimit *FixedWindowLimiter
}

// MessageLimiter returns the per-connection throttle, creating it lazily so the
// handler does not need to construct it. It is only touched from the client's
// single read goroutine.
func (c *Client) MessageLimiter() *FixedWindowLimiter {
	if c.msgLimit == nil {
		c.msgLimit = NewFixedWindowLimiter(wsMessageLimit, wsMessageWindow, 1)
	}
	return c.msgLimit
}

type WebSocketHub struct {
	clients    map[string]*Client            // clientID -> Client
	userConns  map[string]map[string]*Client // userID -> clientID -> Client
	Register   chan *Client
	Unregister chan *Client
	Broadcast  chan *BroadcastMessage
	redis      *redis.Client
	mu         sync.RWMutex
}

type BroadcastMessage struct {
	Type       string
	Data       interface{}
	TargetUser string // Empty means broadcast to all
	ChatID     string // For chat-specific broadcasts
}

func NewWebSocketHub(redis *redis.Client) *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[string]*Client),
		userConns:  make(map[string]map[string]*Client),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Broadcast:  make(chan *BroadcastMessage, 256),
		redis:      redis,
	}
}

func (h *WebSocketHub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.registerClient(client)

		case client := <-h.Unregister:
			h.unregisterClient(client)

		case message := <-h.Broadcast:
			h.broadcastMessage(message)
		}
	}
}

func (h *WebSocketHub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.clients[client.ID] = client

	if h.userConns[client.UserID] == nil {
		h.userConns[client.UserID] = make(map[string]*Client)
	}
	h.userConns[client.UserID][client.ID] = client

	// NFR-18: connection metrics — gauge the current sockets + distinct online
	// users and count total accepted connections.
	observability.IncWSConnections()
	observability.SetWSConnections(len(h.clients), len(h.userConns))

	log.Printf("Client registered: %s (user: %s)", client.ID, client.UserID)
}

func (h *WebSocketHub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client.ID]; ok {
		delete(h.clients, client.ID)
		close(client.Send)

		if userConns, ok := h.userConns[client.UserID]; ok {
			delete(userConns, client.ID)
			if len(userConns) == 0 {
				delete(h.userConns, client.UserID)
			}
		}

		// NFR-18: keep the connection gauges in sync after a disconnect.
		observability.SetWSConnections(len(h.clients), len(h.userConns))

		log.Printf("Client unregistered: %s (user: %s)", client.ID, client.UserID)
	}
}

func (h *WebSocketHub) broadcastMessage(msg *BroadcastMessage) {
	data, err := json.Marshal(models.WebSocketMessage{
		Type: msg.Type,
		Data: msg.Data,
	})

	if err != nil {
		log.Printf("Error marshaling message: %v", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	if msg.TargetUser != "" {
		// Send to specific user's all connections
		if userConns, ok := h.userConns[msg.TargetUser]; ok {
			for _, client := range userConns {
				select {
				case client.Send <- data:
				default:
					observability.IncWSFastDropped()
					close(client.Send)
					delete(h.clients, client.ID)
					delete(userConns, client.ID)
				}
			}
		}
	} else {
		// Broadcast to all clients
		for _, client := range h.clients {
			select {
			case client.Send <- data:
			default:
				observability.IncWSFastDropped()
				close(client.Send)
				delete(h.clients, client.ID)
			}
		}
	}
}

func (h *WebSocketHub) SendToUser(userID string, msgType string, data interface{}) {
	h.Broadcast <- &BroadcastMessage{
		Type:       msgType,
		Data:       data,
		TargetUser: userID,
	}
}

func (h *WebSocketHub) SendToChat(chatID string, userIDs []string, msgType string, data interface{}) {
	for _, userID := range userIDs {
		h.SendToUser(userID, msgType, data)
	}
}

func (h *WebSocketHub) BroadcastToAll(msgType string, data interface{}) {
	h.Broadcast <- &BroadcastMessage{
		Type: msgType,
		Data: data,
	}
}

func (h *WebSocketHub) IsUserOnline(userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	conns, ok := h.userConns[userID]
	return ok && len(conns) > 0
}

func (h *WebSocketHub) GetOnlineUsers() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	users := make([]string, 0, len(h.userConns))
	for userID := range h.userConns {
		users = append(users, userID)
	}
	return users
}

// Client read/write methods
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// NFR-24: throttle inbound messages per connection so a single client
		// cannot flood the hub and burn CPU/bandwidth. Over-the-limit clients
		// are closed with a policy-violation frame.
		if !c.MessageLimiter().Allow(c.ID) {
			log.Printf("WebSocket client %s (user %s) exceeded message rate limit", c.ID, c.UserID)
			_ = c.Conn.WriteControl(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "message rate limit exceeded"),
				time.Now().Add(time.Second))
			return
		}

		// Handle incoming messages (typing indicators, etc.)
		var wsMsg models.WebSocketMessage
		if err := json.Unmarshal(message, &wsMsg); err != nil {
			log.Printf("Error parsing WebSocket message: %v", err)
			continue
		}

		// Process message based on type
		switch wsMsg.Type {
		case "typing_start", "typing_stop":
			c.handleTypingEvent(wsMsg)
		case "message_ack":
			c.handleMessageAck(wsMsg)
		case "webrtc_signal", "call_signal":
			c.handleCallSignal(wsMsg)
		case "live_caption", "caption":
			c.handleLiveCaption(wsMsg)
		case "join_chat":
			// Handle join chat event
		}
	}
}

// handleMessageAck processes an inbound delivered/read acknowledgment (task
// 6.1). The recipient's client acks each message it receives ("received") and,
// when opened, "read"; the server records the per-recipient tick and notifies
// the participants through the ReceiptService.
func (c *Client) handleMessageAck(msg models.WebSocketMessage) {
	if c.Receipts == nil {
		return
	}

	var ack models.MessageAck
	data, err := json.Marshal(msg.Data)
	if err != nil {
		return
	}
	if err := json.Unmarshal(data, &ack); err != nil {
		return
	}
	if ack.MessageID == "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	switch ack.Status {
	case "received":
		_ = c.Receipts.AcknowledgeReceived(ctx, ack.ChatID, ack.MessageID, c.UserID)
	case "read":
		_ = c.Receipts.AcknowledgeRead(ctx, ack.ChatID, ack.MessageID, c.UserID)
	}
}

func (c *Client) WritePump() {
	defer c.Conn.Close()

	for message := range c.Send {
		if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
			log.Printf("WebSocket write error: %v", err)
			return
		}
	}
}

func (c *Client) handleLiveCaption(msg models.WebSocketMessage) {
	if c.Calls == nil {
		return
	}
	data, ok := msg.Data.(map[string]interface{})
	if !ok {
		return
	}
	callID, _ := data["callId"].(string)
	if callID == "" {
		callID, _ = data["call_id"].(string)
	}
	text, _ := data["text"].(string)
	if text == "" {
		text, _ = data["originalText"].(string)
	}
	lang, _ := data["language"].(string)
	if lang == "" {
		lang, _ = data["originalLanguage"].(string)
	}
	if callID == "" || text == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _ = c.Calls.PublishLiveCaption(ctx, callID, c.UserID, text, lang)
}

func (c *Client) handleCallSignal(msg models.WebSocketMessage) {
	if c.Calls == nil {
		return
	}
	data, ok := msg.Data.(map[string]interface{})
	if !ok {
		return
	}
	callID, _ := data["callId"].(string)
	if callID == "" {
		callID, _ = data["call_id"].(string)
	}
	sigType, _ := data["type"].(string)
	sdp, _ := data["sdp"].(string)
	candidate, _ := data["candidate"].(string)
	if candidate == "" {
		if v, ok := data["candidate"]; ok && v != nil {
			if b, err := json.Marshal(v); err == nil {
				candidate = string(b)
			}
		}
	}
	if callID == "" || sigType == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = c.Calls.HandleSignal(ctx, callID, c.UserID, sigType, sdp, candidate, data)
}

func (c *Client) handleTypingEvent(msg models.WebSocketMessage) {
	data, ok := msg.Data.(map[string]interface{})
	if !ok {
		return
	}

	chatID, _ := data["chatId"].(string)
	isTyping := msg.Type == "typing_start"

	// Broadcast typing event to other users in the chat
	c.Hub.Broadcast <- &BroadcastMessage{
		Type: "user_typing",
		Data: models.TypingEvent{
			ChatID:   chatID,
			UserID:   c.UserID,
			IsTyping: isTyping,
		},
		ChatID: chatID,
	}
}
