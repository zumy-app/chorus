package main

import (
	"log"
	"os"

	"github.com/chorus/messenger/internal/config"
	"github.com/chorus/messenger/internal/database"
	"github.com/chorus/messenger/internal/handlers"
	"github.com/chorus/messenger/internal/middleware"
	"github.com/chorus/messenger/internal/services"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Load configuration
	cfg := config.Load()

	// Initialize database
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := database.Migrate(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize Redis
	redisClient := database.ConnectRedis(cfg.RedisURL)
	defer redisClient.Close()

	// Initialize services
	authService := services.NewAuthService(db, cfg.JWTSecret)
	userService := services.NewUserService(db)
	chatService := services.NewChatService(db)
	messageService := services.NewMessageService(db, redisClient)
	translationService := services.NewTranslationService(cfg.GoogleTranslateAPIKey, redisClient)
	wsHub := services.NewWebSocketHub(redisClient)

	// Start WebSocket hub
	go wsHub.Run()

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService, userService)
	chatHandler := handlers.NewChatHandler(chatService, userService)
	messageHandler := handlers.NewMessageHandler(messageService, chatService, translationService, wsHub)
	wsHandler := handlers.NewWebSocketHandler(wsHub, authService)

	// Setup Gin router
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	// CORS configuration
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy", "version": "1.0.0"})
	})

	// Public routes
	public := r.Group("/api/v1")
	{
		public.POST("/auth/register", authHandler.Register)
		public.POST("/auth/login", authHandler.Login)
		public.POST("/auth/refresh", authHandler.RefreshToken)
	}

	// Protected routes
	protected := r.Group("/api/v1")
	protected.Use(middleware.AuthMiddleware(authService))
	{
		// User routes
		protected.GET("/users/me", authHandler.GetMe)
		protected.PUT("/users/me", authHandler.UpdateMe)
		protected.GET("/users/search", authHandler.SearchUsers)

		// Chat routes
		protected.GET("/chats", chatHandler.GetUserChats)
		protected.POST("/chats", chatHandler.CreateChat)
		protected.GET("/chats/:chatId", chatHandler.GetChat)
		protected.PUT("/chats/:chatId", chatHandler.UpdateChat)
		protected.POST("/chats/:chatId/participants", chatHandler.AddParticipant)
		protected.DELETE("/chats/:chatId/participants/:userId", chatHandler.RemoveParticipant)
		protected.DELETE("/chats/:chatId/leave", chatHandler.LeaveChat)

		// Message routes
		protected.GET("/chats/:chatId/messages", messageHandler.GetMessages)
		protected.POST("/chats/:chatId/messages", messageHandler.SendMessage)
		protected.PUT("/chats/:chatId/read", messageHandler.MarkAsRead)
		protected.GET("/messages/search", messageHandler.SearchMessages)
	}

	// WebSocket endpoint
	r.GET("/ws", middleware.AuthMiddleware(authService), wsHandler.HandleWebSocket)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
