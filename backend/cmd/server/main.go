package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/chorus/messenger/internal/config"
	"github.com/chorus/messenger/internal/database"
	"github.com/chorus/messenger/internal/handlers"
	"github.com/chorus/messenger/internal/middleware"
	"github.com/chorus/messenger/internal/services"
	"github.com/chorus/messenger/pkg/logutil"
	"github.com/chorus/messenger/pkg/translation"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables (.env always wins over system env)
	if err := godotenv.Overload(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Load configuration
	cfg := config.Load()

	// Initialize log level
	logutil.SetLevelFromString(cfg.LogLevel)
	log.Printf("[Startup] Log level set to %q", cfg.LogLevel)

	// Initialize Appwrite (if configured)
	if cfg.AppwriteEndpoint != "" && cfg.AppwriteProjectID != "" {
		_, err := database.ConnectAppwrite(
			cfg.AppwriteEndpoint,
			cfg.AppwriteProjectID,
			cfg.AppwriteAPIKey,
			cfg.AppwriteDatabaseID,
		)
		if err != nil {
			log.Printf("Warning: Failed to connect to Appwrite: %v", err)
		}
	}

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

	// Initialize Redis (optional in lean local mode)
	redisClient := database.ConnectRedis(cfg.RedisURL)
	if redisClient != nil {
		defer redisClient.Close()
	}

	// Initialize core services
	authService := services.NewAuthService(db, cfg.JWTSecret)
	userService := services.NewUserService(db)
	chatService := services.NewChatService(db)
	messageService := services.NewMessageService(db, redisClient)

	// Create translation provider chain.
	translationProvider := buildTranslationProviderChain(cfg)
	log.Printf("Using translation provider: %s", translationProvider.Name())
	translationService := services.NewTranslationService(
		translationProvider,
		redisClient,
		time.Duration(cfg.TranslationChainTimeout)*time.Second,
	)
	wsHub := services.NewWebSocketHub(redisClient)

	var pubsubService *services.PubSubService
	if redisClient != nil {
		pubsubService = services.NewPubSubService(redisClient, wsHub)
		pubsubService.Start()
		defer pubsubService.Stop()
	}

	// Phase 2: Initialize Inbox service for offline message delivery
	_ = services.NewInboxService(db, redisClient)

	var presenceService *services.PresenceService
	if redisClient != nil {
		presenceService = services.NewPresenceService(db, redisClient, pubsubService)
		presenceService.StartPresenceCleanup()
	}

	// Phase 2: Initialize Search service
	searchService := services.NewSearchService(db, redisClient)

	// Phase 3: Initialize Grammar service with endpoint chain
	log.Printf("[Startup] EXTERNAL_LLM_PROVIDER_ORDER=%q from env", os.Getenv("EXTERNAL_LLM_PROVIDER_ORDER"))
	grammarService := services.NewGrammarService(redisClient, buildGrammarEndpoints(cfg))

	// Phase 3: Initialize Vocabulary service
	vocabularyService := services.NewVocabularyService(db, redisClient)

	// Phase 3: Initialize Speech-to-Text service
	ctx := context.Background()
	sttService, err := services.NewSpeechToTextService(ctx)
	if err != nil {
		log.Printf("Warning: Speech-to-Text service initialization failed: %v", err)
	}

	// Phase 3: Initialize Call service
	callService := services.NewCallService(db, translationService, sttService)

	// Start WebSocket hub
	go wsHub.Run()

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService, userService)
	chatHandler := handlers.NewChatHandler(chatService, userService, wsHub)
	messageHandler := handlers.NewMessageHandler(messageService, chatService, translationService, wsHub)
	wsHandler := handlers.NewWebSocketHandler(wsHub, authService)

	// Phase 2 & 3 handlers
	searchHandler := handlers.NewSearchHandler(searchService)
	presenceHandler := handlers.NewPresenceHandler(presenceService)
	grammarHandler := handlers.NewGrammarHandler(grammarService, messageService)
	vocabularyHandler := handlers.NewVocabularyHandler(vocabularyService, messageService, translationService)
	callHandler := handlers.NewCallHandler(callService)

	// Setup Gin router
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	// CORS configuration
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:5173", "http://10.0.2.2:5173", "https://chorus.talk"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy", "version": "2.0.0"})
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

		// Phase 2: Search routes
		protected.GET("/messages/search", searchHandler.SearchMessages)
		protected.GET("/chats/search", searchHandler.SearchChats)
		protected.GET("/contacts/search", searchHandler.SearchContacts)

		// Phase 2: Presence routes
		protected.GET("/presence/:userId", presenceHandler.GetPresence)
		protected.PUT("/presence", presenceHandler.UpdatePresence)
		protected.POST("/presence/activity", presenceHandler.UpdateActivity)

		// Phase 3: Grammar analysis routes
		protected.POST("/grammar/analyze", grammarHandler.AnalyzeMessageGrammar)
		protected.POST("/grammar/analyze-text", grammarHandler.AnalyzeText)
		protected.POST("/grammar/analyze-ai", grammarHandler.AnalyzeTextWithAI)
		protected.POST("/grammar/learn", grammarHandler.LearnGrammar)
		protected.GET("/grammar/suggestions", grammarHandler.GetGrammarSuggestions)
		protected.GET("/grammar/report", grammarHandler.GetGrammarReport)

		// Phase 3: Vocabulary routes
		protected.POST("/vocabulary", vocabularyHandler.SaveVocabulary)
		protected.GET("/vocabulary", vocabularyHandler.GetVocabulary)
		protected.GET("/vocabulary/due", vocabularyHandler.GetDueVocabulary)
		protected.GET("/vocabulary/:id", vocabularyHandler.GetVocabularyByID)
		protected.POST("/vocabulary/practice", vocabularyHandler.UpdatePracticeResult)
		protected.GET("/vocabulary/progress", vocabularyHandler.GetProgress)
		protected.DELETE("/vocabulary/:id", vocabularyHandler.DeleteVocabulary)
		protected.GET("/vocabulary/search", vocabularyHandler.SearchVocabulary)

		// Phase 3: Call routes
		protected.POST("/calls/initiate", callHandler.InitiateCall)
		protected.POST("/calls/:callId/end", callHandler.EndCall)
		protected.GET("/calls/:callId", callHandler.GetCallSession)
		protected.GET("/calls/:callId/transcript", callHandler.GetCallTranscript)
		protected.GET("/calls/history", callHandler.GetCallHistory)
		protected.DELETE("/calls/:callId/transcript", callHandler.DeleteCallTranscript)
		protected.GET("/calls/transcripts/search", callHandler.SearchTranscripts)
		protected.POST("/calls/:callId/signal", callHandler.HandleWebRTCSignaling)
	}

	// WebSocket endpoint (auth handled inside handler via query param or header)
	r.GET("/ws", wsHandler.HandleWebSocket)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s (Phase 2 & 3 features enabled)", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// buildTranslationProviderChain constructs a provider chain from config.
// If EXTERNAL_LLM_PROVIDER_ORDER is set, it builds a ChainProvider from the
// ordered list of aliases. Otherwise, it falls back to the legacy single-provider config.
func buildTranslationProviderChain(cfg *config.Config) translation.Provider {
	if len(cfg.ExternalLLMProviderOrder) > 0 {
		var providers []translation.Provider
		for _, alias := range cfg.ExternalLLMProviderOrder {
			def, ok := cfg.Providers[alias]
			if !ok {
				log.Printf("Warning: provider %q in EXTERNAL_LLM_PROVIDER_ORDER not configured, skipping", alias)
				continue
			}
			provCfg := translation.Config{
				Provider: translation.ProviderType(def.Type),
				APIURL:   def.APIURL,
				APIKey:   def.APIKey,
				Model:    def.Model,
				Timeout:  def.Timeout,
			}
			prov, err := translation.NewProvider(provCfg)
			if err != nil {
				log.Printf("Warning: failed to create provider %q: %v", alias, err)
				continue
			}
			providers = append(providers, prov)
			log.Printf("  Translation provider %d: %s (%s)", len(providers), alias, prov.Name())
		}
		if len(providers) == 0 {
			log.Fatal("No translation providers could be created from EXTERNAL_LLM_PROVIDER_ORDER")
		}
		if len(providers) == 1 {
			return providers[0]
		}
		return translation.NewChainProvider(providers)
	}

	// Legacy fallback: single provider from TRANSLATION_PROVIDER_NAME etc.
	provCfg := translation.Config{
		Provider: translation.ProviderType(cfg.TranslationProviderName),
		APIURL:   cfg.TranslationProviderURL,
		APIKey:   cfg.TranslationProviderKey,
		Model:    cfg.TranslationProviderModel,
	}
	prov, err := translation.NewProvider(provCfg)
	if err != nil {
		log.Fatalf("Failed to create translation provider: %v", err)
	}
	return prov
}

// buildGrammarEndpoints constructs an ordered list of GrammarEndpoints from config.
// If EXTERNAL_LLM_PROVIDER_ORDER is set, it follows that order. Otherwise it falls back
// to the legacy GRAMMAR_API_* env vars (maintained for backward compatibility).
func buildGrammarEndpoints(cfg *config.Config) []services.GrammarEndpoint {
	log.Printf("[Startup] Using EXTERNAL_LLM_PROVIDER_ORDER=%v", cfg.ExternalLLMProviderOrder)
	for k, v := range cfg.Providers {
		log.Printf("  Provider %q: type=%q url=%q model=%q", k, v.Type, v.APIURL, v.Model)
	}
	if len(cfg.ExternalLLMProviderOrder) > 0 {
		var endpoints []services.GrammarEndpoint
		for _, alias := range cfg.ExternalLLMProviderOrder {
			def, ok := cfg.Providers[alias]
			if !ok {
				log.Printf("Warning: provider %q in EXTERNAL_LLM_PROVIDER_ORDER not configured, skipping", alias)
				continue
			}
			ep := services.NewGrammarEndpoint(alias, def.Type, def.APIURL, def.APIKey, def.Model, def.Timeout)
			endpoints = append(endpoints, ep)
			log.Printf("  Grammar endpoint %d: %s (%s model=%s)", len(endpoints), alias, def.APIURL, def.Model)
		}
		if len(endpoints) == 0 {
			log.Fatal("No grammar endpoints could be created from EXTERNAL_LLM_PROVIDER_ORDER")
		}
		return endpoints
	}

	// Legacy fallback: single endpoint from GRAMMAR_API_* env vars.
	ep := services.NewGrammarEndpoint("legacy", "", "", "", "", 0)
	log.Printf("  Grammar endpoint: legacy (no EXTERNAL_LLM_PROVIDER_ORDER configured)")
	return []services.GrammarEndpoint{ep}
}