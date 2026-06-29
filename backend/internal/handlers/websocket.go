package handlers

import (
	"log"
	"net/http"

	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

type WebSocketHandler struct {
	hub         *services.WebSocketHub
	authService *services.AuthService
}

func NewWebSocketHandler(hub *services.WebSocketHub, authService *services.AuthService) *WebSocketHandler {
	return &WebSocketHandler{
		hub:         hub,
		authService: authService,
	}
}

func (h *WebSocketHandler) HandleWebSocket(c *gin.Context) {
	// Try to get userID from middleware first (Authorization header)
	userID := c.GetString("userID")

	// If not found, try query parameter token (for WebSocket connections)
	if userID == "" {
		token := c.Query("token")
		if token != "" {
			var err error
			userID, err = h.authService.ValidateAccessToken(token)
			if err != nil {
				c.JSON(401, gin.H{"error": "Invalid token"})
				return
			}
		}
	}

	if userID == "" {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := &services.Client{
		ID:     uuid.New().String(),
		UserID: userID,
		Conn:   conn,
		Send:   make(chan []byte, 256),
		Hub:    h.hub,
	}

	h.hub.Register <- client

	// Start client read/write pumps
	go client.WritePump()
	go client.ReadPump()
}
