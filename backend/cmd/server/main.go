package main

import (
	"context"
	"log"
	"os"
	"strconv"
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

	// Startup configuration checks: warn about missing/invalid provider keys
	// so misconfigured cloud providers are obvious before traffic arrives.
	// Missing keys do NOT stop the server — the chain skips those providers
	// at runtime and falls through to the next (ultimately the local fallback).
	logStartupConfigWarnings(cfg)

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
	waitlistService := services.NewWaitlistService(db)
	invitationService := services.NewInvitationService(db, time.Duration(cfg.InviteTTLHours)*time.Hour)
	chatService := services.NewChatService(db)
	messageService := services.NewMessageService(db, redisClient)

	// Create translation provider chain.
	translationProvider := buildTranslationProviderChain(cfg)
	log.Printf("Using translation provider: %s", translationProvider.Name())
	probeLocalProviders(translationProvider)
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
	log.Printf("[Startup] GRAMMAR_ANALYSIS_PROVIDER_ORDER=%q from env", os.Getenv("GRAMMAR_ANALYSIS_PROVIDER_ORDER"))
	log.Printf("[Startup] TRANSLATION_PROVIDER_ORDER=%q from env", os.Getenv("TRANSLATION_PROVIDER_ORDER"))
	log.Printf("[Startup] PROVIDER_OLLAMA_LOCAL_TYPE=%q", os.Getenv("PROVIDER_OLLAMA_LOCAL_TYPE"))
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
	authHandler := handlers.NewAuthHandler(authService, userService, invitationService)
	emailSender := services.NewSMTPEmailSender(
		cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUsername, cfg.SMTPPassword, cfg.SMTPFromEmail,
	)
	waitlistHandler := handlers.NewWaitlistHandler(waitlistService, emailSender)
	adminWaitlistHandler := handlers.NewAdminWaitlistHandler(
		db, userService, invitationService,
		emailSender,
		cfg.AdminEmails, cfg.InviteBaseURL,
	)
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
		public.POST("/waitlist", middleware.IPRateLimiter(10, time.Hour), waitlistHandler.Submit)
		public.POST("/auth/register", middleware.IPRateLimiter(10, time.Hour), authHandler.Register)
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
		protected.GET("/admin/waitlist", adminWaitlistHandler.List)
		protected.POST("/admin/waitlist/:id/approve", adminWaitlistHandler.Approve)

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
// If TRANSLATION_PROVIDER_ORDER is set, it builds a ChainProvider from the
// ordered list of aliases. Providers that are missing required config (e.g. a
// cloud provider with an empty API key) are skipped so the chain falls through
// to the next one at runtime. Otherwise, it falls back to the legacy
// single-provider config.
func buildTranslationProviderChain(cfg *config.Config) translation.Provider {
	if len(cfg.TranslationProviderOrder) > 0 {
		var providers []translation.Provider
		for _, alias := range cfg.TranslationProviderOrder {
			def, ok := cfg.Providers[alias]
			if !ok {
				log.Printf("Warning: provider %q in TRANSLATION_PROVIDER_ORDER not configured, skipping", alias)
				continue
			}
			provCfg := translation.Config{
				Provider: translation.ProviderType(def.Type),
				APIURL:   def.APIURL,
				APIKey:   def.APIKey,
				Model:    def.Model,
				Timeout:  def.Timeout,
			}
			if !provCfg.Configured() {
				log.Printf("Warning: provider %q (%s) is not configured (missing required env keys), skipping",
					alias, def.Type)
				continue
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
			log.Fatal("No translation providers could be created from TRANSLATION_PROVIDER_ORDER — " +
				"set at least one API key for a cloud provider or ensure a local provider " +
				"(libretranslate, ollama) is configured")
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
// If GRAMMAR_PROVIDER_ORDER is set, it follows that order. Otherwise it falls back
// to the legacy GRAMMAR_API_* env vars.
func buildGrammarEndpoints(cfg *config.Config) []services.GrammarEndpoint {
	log.Printf("[Startup] GrammarProviderOrder=%v, Providers keys:", cfg.GrammarProviderOrder)
	for k, v := range cfg.Providers {
		log.Printf("  Provider %q: type=%q url=%q model=%q", k, v.Type, v.APIURL, v.Model)
	}
	if len(cfg.GrammarProviderOrder) > 0 {
		var endpoints []services.GrammarEndpoint
		for _, alias := range cfg.GrammarProviderOrder {
			def, ok := cfg.Providers[alias]
			if !ok {
				log.Printf("Warning: provider %q in GRAMMAR_PROVIDER_ORDER not configured, skipping", alias)
				continue
			}
			provCfg := translation.Config{
				Provider: translation.ProviderType(def.Type),
				APIURL:   def.APIURL,
				APIKey:   def.APIKey,
				Model:    def.Model,
				Timeout:  def.Timeout,
			}
			if !provCfg.Configured() {
				log.Printf("Warning: provider %q (%s) is not configured (missing required env keys), skipping",
					alias, def.Type)
				continue
			}
			ep := services.NewGrammarEndpoint(alias, def.Type, def.APIURL, def.APIKey, def.Model, def.Timeout)
			endpoints = append(endpoints, ep)
			log.Printf("  Grammar endpoint %d: %s (%s model=%s)", len(endpoints), alias, def.APIURL, def.Model)
		}
		if len(endpoints) == 0 {
			log.Printf("Warning: no grammar AI endpoints could be created from GRAMMAR_PROVIDER_ORDER — " +
				"grammar analysis will use the built-in regex fallback")
			return nil
		}
		return endpoints
	}

	// Legacy fallback: single endpoint from GRAMMAR_API_* env vars.
	ep := services.NewGrammarEndpoint("legacy", "", cfg.GrammarAPIURL, cfg.GrammarAPIKey, cfg.GrammarModel, 0)
	log.Printf("  Grammar endpoint: %s model=%s", cfg.GrammarAPIURL, cfg.GrammarModel)
	return []services.GrammarEndpoint{ep}
}

// logStartupConfigWarnings prints the provider config validation results.
// Missing keys are warnings, not fatal errors — the chain skips those providers
// at runtime and falls through to the next one (ultimately the local fallback).
func logStartupConfigWarnings(cfg *config.Config) {
	warnings := cfg.Validate()
	if len(warnings) == 0 {
		log.Printf("[Startup] Provider configuration OK: all providers have required keys")
		return
	}
	log.Printf("[Startup] Provider configuration warnings (%d):", len(warnings))
	for _, w := range warnings {
		if w.EnvKey != "" {
			log.Printf("  [Startup] MISSING/INVALID %s (provider %q): %s", w.EnvKey, w.Provider, w.Message)
		} else {
			log.Printf("  [Startup] %s: %s", w.Provider, w.Message)
		}
	}
	log.Printf("[Startup] Cloud providers with missing keys are skipped automatically; " +
		"the local fallback (libretranslate/ollama) is always tried last.")
}

// probeLocalProviders performs lightweight readiness checks against the
// offline/local providers in the chain. If an Ollama model is missing, it
// auto-pulls it (instead of only warning) so the offline grammar/translation
// provider works on first boot. Cloud providers are skipped (no probe).
func probeLocalProviders(p translation.Provider) {
	var providers []translation.Provider
	if chain, ok := p.(*translation.ChainProvider); ok {
		providers = chain.Providers()
	} else {
		providers = []translation.Provider{p}
	}

	for _, prov := range providers {
		pingable, ok := prov.(translation.Pingable)
		if !ok {
			continue // cloud provider — no probe
		}

		probeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := pingable.Ping(probeCtx)
		cancel()
		if err == nil {
			log.Printf("[Startup] Local provider %s: reachable", prov.Name())
			continue
		}

		// Self-heal: a missing Ollama model is pulled automatically instead of
		// just warning. This blocks startup until the model is ready (bounded by
		// OLLAMA_STARTUP_PULL_TIMEOUT). In Docker the ollama container pulls the
		// model itself and the healthcheck gates backend start, so this is a no-op.
		if ensure, ok := prov.(translation.ModelEnsurer); ok {
			log.Printf("[Startup] Local provider %s: %v — auto-pulling model...", prov.Name(), err)
			pullCtx, cancel := context.WithTimeout(context.Background(), startupPullTimeout())
			pullErr := ensure.EnsureModel(pullCtx)
			cancel()
			if pullErr != nil {
				log.Printf("[Startup] WARNING: auto-pull for %s failed: %v — will retry on demand", prov.Name(), pullErr)
			} else {
				log.Printf("[Startup] Local provider %s: model installed, ready", prov.Name())
			}
			continue
		}

		log.Printf("[Startup] WARNING: local provider %s is unreachable (%v). "+
			"Translation will fall through to the next provider until it is up.",
			prov.Name(), err)
	}
}

// startupPullTimeout returns how long the backend blocks startup while
// auto-pulling a missing Ollama model. Configurable via
// OLLAMA_STARTUP_PULL_TIMEOUT (seconds); default 600s (10 minutes).
func startupPullTimeout() time.Duration {
	secs := 600
	if v := os.Getenv("OLLAMA_STARTUP_PULL_TIMEOUT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			secs = n
		}
	}
	return time.Duration(secs) * time.Second
}
