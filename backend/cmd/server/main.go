package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/chorus/messenger/internal/config"
	"github.com/chorus/messenger/internal/database"
	"github.com/chorus/messenger/internal/handlers"
	"github.com/chorus/messenger/internal/middleware"
	"github.com/chorus/messenger/internal/models"
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

	// Seed admin roles from the legacy admin-emails allowlist.
	if err := services.EnsureAdminRoles(db, cfg.AdminEmails); err != nil {
		log.Fatalf("Failed to seed admin roles: %v", err)
	}

	// Initialize Redis (optional in lean local mode)
	redisClient := database.ConnectRedis(cfg.RedisURL)
	if redisClient != nil {
		defer redisClient.Close()
	}

	// Initialize core services
	authService := services.NewAuthService(db, cfg.JWTSecret)
	userService := services.NewUserService(db)
	entitlementService := services.NewEntitlementService(cfg.SelfHost)
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

	// Durable, near-real-time translation queue: translation_jobs rows are the
	// source of truth, Redis pub/sub is the trigger, a sweeper retries failures,
	// and startup recovery re-queues incomplete work. Completions fan out to chat
	// participants over the WebSocket hub and (for multi-instance) Redis pub/sub.
	translationQueue := services.NewTranslationQueueService(
		db,
		redisClient,
		translationService,
		func(messageID string) (*models.Message, error) {
			return messageService.GetMessageByID(context.Background(), messageID)
		},
		func(chatID string, message *models.Message) {
			participants, _ := chatService.GetParticipants(chatID)
			userIDs := make([]string, 0, len(participants))
			for _, p := range participants {
				userIDs = append(userIDs, p.UserID)
			}
			wsHub.SendToChat(chatID, userIDs, "message_updated", message)
			if pubsubService != nil {
				pubsubService.PublishToChat(chatID, userIDs, "message_updated", message)
			}
		},
	)
	translationQueue.Start()
	defer translationQueue.Stop()

	var presenceService *services.PresenceService
	if redisClient != nil {
		presenceService = services.NewPresenceService(db, redisClient, pubsubService)
		presenceService.StartPresenceCleanup()
	}

	// Phase 2: Initialize Search service
	searchService := services.NewSearchService(db, redisClient)

	// Phase 3: Initialize Grammar service with endpoint chain
	log.Printf("[Startup] TRANSLATION_FALLBACK_ORDER=%q from env", os.Getenv("TRANSLATION_FALLBACK_ORDER"))
	grammarService := services.NewGrammarService(redisClient, buildGrammarEndpoints(cfg))

	// Asynchronous, durable AI grammar analysis. Jobs are stored in
	// grammar_jobs (DB outbox), triggered via Redis pub/sub, and results are
	// pushed to the requesting user only over the WebSocket hub (+ pub/sub for
	// multi-instance). The HTTP handler returns a job id immediately instead of
	// holding a connection open for the provider chain.
	grammarQueue := services.NewGrammarQueueService(
		db,
		redisClient,
		grammarService,
		grammarService.CachedAIAnalysis,
		func(userID string, payload *services.GrammarJobResult) {
			wsHub.SendToUser(userID, "grammar_analysis", payload)
			if pubsubService != nil {
				pubsubService.PublishToUser(userID, "grammar_analysis", payload)
			}
		},
	)
	grammarQueue.Start()
	defer grammarQueue.Stop()

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
	emailSender := services.NewSMTPEmailSender(
		cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUsername, cfg.SMTPPassword, cfg.SMTPFromEmail, cfg.SMTPFromName,
	)
	// Durable notification layer: every email is persisted to the outbox and
	// retried on failure, giving delivery guarantees for all notifications.
	notificationService := services.NewNotificationService(db, emailSender)
	notificationService.Start()
	defer notificationService.Stop()

	authHandler := handlers.NewAuthHandler(authService, userService, invitationService, notificationService, entitlementService, cfg.PasswordResetBaseURL)
	waitlistHandler := handlers.NewWaitlistHandler(waitlistService, notificationService)
	adminWaitlistHandler := handlers.NewAdminWaitlistHandler(
		db, userService, invitationService,
		notificationService,
		translationQueue,
		cfg.AdminEmails, cfg.InviteBaseURL,
	)
	adminUsersHandler := handlers.NewAdminUsersHandler(userService, authService)
	adminTranslationsHandler := handlers.NewAdminTranslationsHandler(translationQueue, translationService)

	// Report & Block (REQ §8.2): moderation service enforces blocks on chat
	// creation and messaging; reports feed the moderator console.
	moderationService := services.NewModerationService(db)
	chatHandler := handlers.NewChatHandler(chatService, userService, moderationService, wsHub)
	messageHandler := handlers.NewMessageHandler(messageService, chatService, userService, entitlementService, translationQueue, moderationService, wsHub)
	wsHandler := handlers.NewWebSocketHandler(wsHub, authService)

	// Monetization (Phase 1.5): PayPal client + billing service + handler.
	paypalClient := services.NewPayPalClient(cfg)
	billingService := services.NewBillingService(db, paypalClient, entitlementService, cfg.PayPalPlanMonthlyID, cfg.PayPalPlanYearlyID)
	// Phase 4 (P15): premium lifecycle emails, sent durably through the outbox.
	billingService.SetNotifier(func(ctx context.Context, user *models.User, kind string, graceUntil *time.Time) {
		var manageLink string
		if user.SubscriptionID != nil {
			manageLink = paypalClient.ManageURL(*user.SubscriptionID)
		}
		var subject, html string
		switch kind {
		case services.NotifyActivated:
			subject, html = services.PremiumActivatedEmail(user.DisplayName, manageLink)
		case services.NotifyEnterGrace:
			subject, html = services.PremiumGraceEmail(user.DisplayName, *graceUntil, manageLink)
		case services.NotifyDowngrade:
			subject, html = services.PremiumDowngradedEmail(user.DisplayName)
		}
		if subject != "" {
			notificationService.Send(user.Email, subject, html)
		}
	})
	billingService.StartGraceSweeper()
	defer billingService.StopGraceSweeper()
	billingHandler := handlers.NewBillingHandler(billingService, paypalClient, entitlementService)

	// Phase 2 & 3 handlers
	searchHandler := handlers.NewSearchHandler(searchService)
	presenceHandler := handlers.NewPresenceHandler(presenceService)
	grammarHandler := handlers.NewGrammarHandler(grammarService, grammarQueue, messageService)
	vocabularyHandler := handlers.NewVocabularyHandler(vocabularyService, messageService, translationService)
	callHandler := handlers.NewCallHandler(callService)

	moderationHandler := handlers.NewModerationHandler(moderationService)

	// Setup Gin router
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	// CORS configuration
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:4173", "http://localhost:5173", "http://10.0.2.2:5173", "https://chorus.talk"},
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
		public.POST("/auth/forgot-password", middleware.IPRateLimiter(5, time.Hour), authHandler.ForgotPassword)
		public.POST("/auth/reset-password", authHandler.ResetPassword)
		// PayPal webhooks must be reachable without auth; signature verification
		// happens inside the handler.
		public.POST("/webhooks/paypal", billingHandler.Webhook)
	}

	// Protected routes
	protected := r.Group("/api/v1")
	protected.Use(middleware.AuthMiddleware(authService, userService))
	{
		// User routes
		protected.GET("/users/me", authHandler.GetMe)
		protected.GET("/users/me/entitlements", authHandler.GetMyEntitlements)
		protected.PUT("/users/me", authHandler.UpdateMe)
		protected.GET("/users/search", authHandler.SearchUsers)

		// Subscription routes (Phase 1.5).
		protected.GET("/users/me/subscription", billingHandler.GetMySubscription)
		protected.POST("/users/me/subscription/checkout", billingHandler.Checkout)

		// Report & Block (REQ §8.2)
		protected.POST("/blocks", moderationHandler.Block)
		protected.DELETE("/blocks/:userId", moderationHandler.Unblock)
		protected.GET("/blocks", moderationHandler.ListBlocked)
		protected.POST("/reports", moderationHandler.Report)

		// Non-gated status probe used by the client to learn the caller's role.
		protected.GET("/admin/status", adminWaitlistHandler.Status)

		// Admin routes (admin role + legacy email allowlist fallback).
		admin := protected.Group("/admin")
		admin.Use(middleware.RequireRole(services.RoleAdmin))
		{
			admin.GET("/waitlist", adminWaitlistHandler.List)
			admin.POST("/waitlist/:id/approve", adminWaitlistHandler.Approve)
			admin.POST("/waitlist/:id/decline", adminWaitlistHandler.Decline)
			admin.POST("/waitlist/:id/resend-invite", adminWaitlistHandler.ResendInvite)
			admin.GET("/stats", adminWaitlistHandler.Stats)
			admin.GET("/emails", adminWaitlistHandler.Emails)
			admin.POST("/emails/:id/retry", adminWaitlistHandler.RetryEmail)
			admin.PUT("/users/:id/role", adminUsersHandler.SetRole)
			admin.DELETE("/users/:id", adminUsersHandler.Delete)
			// Premium management (Phase 1.5).
			admin.PUT("/users/:id/plan", billingHandler.GrantPlan)
			admin.POST("/users/:id/plan/revoke", billingHandler.RevokePlan)
			admin.GET("/users/:id/plan-history", billingHandler.PlanHistory)
			admin.GET("/premium/users", billingHandler.ListPremiumUsers)
			admin.GET("/premium/analytics", billingHandler.PremiumAnalytics)
		}

		// Moderator routes (moderator or admin).
		moderator := protected.Group("/admin")
		moderator.Use(middleware.RequireRole(services.RoleModerator))
		{
			moderator.GET("/users", adminUsersHandler.List)
			moderator.POST("/users/:id/suspend", adminUsersHandler.Suspend)
			moderator.POST("/users/:id/unsuspend", adminUsersHandler.Unsuspend)
			moderator.GET("/translations", adminTranslationsHandler.List)
			moderator.POST("/translations/:id/retry", adminTranslationsHandler.Retry)
			moderator.GET("/translations/health", adminTranslationsHandler.Health)

			// Moderation console: reports
			moderator.GET("/reports", moderationHandler.ListReports)
			moderator.GET("/reports/stats", moderationHandler.ReportStats)
			moderator.POST("/reports/:id/resolve", moderationHandler.ResolveReport)
			moderator.POST("/reports/:id/dismiss", moderationHandler.DismissReport)
		}

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
		protected.GET("/grammar/analyze/:jobId", grammarHandler.GetGrammarJob)
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
// It follows TRANSLATION_FALLBACK_ORDER. Providers that are missing required
// config (e.g. a cloud provider with an empty API key) are skipped so the chain
// falls through to the next one at runtime.
func buildTranslationProviderChain(cfg *config.Config) translation.Provider {
	if len(cfg.TranslationProviderOrder) == 0 {
		log.Fatal("TRANSLATION_FALLBACK_ORDER is not set — no translation providers are configured. " +
			"Set TRANSLATION_FALLBACK_ORDER plus PROVIDER_<NAME>_URL / _KEY and MODEL_TRANSLATION_<NAME> " +
			"(and MODEL_GRAMMAR_<NAME> for grammar) per provider.")
	}
	var providers []translation.Provider
	for _, name := range cfg.TranslationProviderOrder {
		def, ok := cfg.Providers[name]
		if !ok {
			log.Printf("Warning: provider %q in TRANSLATION_FALLBACK_ORDER not configured, skipping", name)
			continue
		}
		provCfg := translation.Config{
			Provider: translation.ProviderType(def.EffectiveType()),
			APIURL:   def.URL,
			APIKey:   def.Key,
			Model:    def.TranslationModel(),
			Timeout:  def.Timeout,
		}
		if !provCfg.Configured() {
			log.Printf("Warning: provider %q (%s) is not configured (missing: %s), skipping",
				name, def.EffectiveType(), strings.Join(missingProviderEnvKeys(name, def), ", "))
			continue
		}
		prov, err := translation.NewProvider(provCfg)
		if err != nil {
			log.Printf("Warning: failed to create provider %q: %v", name, err)
			continue
		}
		providers = append(providers, prov)
		log.Printf("  Translation provider %d: %s (%s)", len(providers), name, prov.Name())
	}
	if len(providers) == 0 {
		log.Fatal("No translation providers could be created from TRANSLATION_FALLBACK_ORDER — " +
			"set at least one API key for a cloud provider or ensure a local provider " +
			"(libretranslate, ollama) is configured")
	}
	if len(providers) == 1 {
		return providers[0]
	}
	return translation.NewChainProvider(providers)
}

// missingProviderEnvKeys returns the canonical env var names that must be set
// for a provider to be created from a ProviderDef. When a provider is skipped
// because it isn't configured, the exact keys are printed so the operator knows
// what to set instead of seeing a generic "missing required env keys".
func missingProviderEnvKeys(name string, def config.ProviderDef) []string {
	var missing []string
	pType := translation.ProviderType(def.EffectiveType())
	if translation.NeedsAPIKey(pType) && def.Key == "" {
		missing = append(missing, config.EnvKeyFor(name, "KEY"))
	}
	if translation.NeedsAPIKey(pType) && def.URL == "" {
		missing = append(missing, config.EnvKeyFor(name, "URL"))
	}
	return missing
}

// buildGrammarEndpoints constructs an ordered list of GrammarEndpoints from
// config. It follows GRAMMAR_FALLBACK_ORDER; when no order is set, grammar
// analysis falls back to the built-in regex.
func buildGrammarEndpoints(cfg *config.Config) []services.GrammarEndpoint {
	log.Printf("[Startup] GrammarFallbackOrder=%v, Providers:", cfg.GrammarProviderOrder)
	for k, v := range cfg.Providers {
		log.Printf("  Provider %q: type=%q url=%q gmodel=%q tmodel=%q", k, v.EffectiveType(), v.URL, v.GrammarModel(), v.TranslationModel())
	}
	if len(cfg.GrammarProviderOrder) == 0 {
		log.Printf("Warning: GRAMMAR_FALLBACK_ORDER is not set — " +
			"grammar analysis will use the built-in regex fallback")
		return nil
	}
	var endpoints []services.GrammarEndpoint
	for _, name := range cfg.GrammarProviderOrder {
		def, ok := cfg.Providers[name]
		if !ok {
			log.Printf("Warning: provider %q in GRAMMAR_FALLBACK_ORDER not configured, skipping", name)
			continue
		}
		provCfg := translation.Config{
			Provider: translation.ProviderType(def.EffectiveType()),
			APIURL:   def.URL,
			APIKey:   def.Key,
			Model:    def.GrammarModel(),
			Timeout:  def.Timeout,
		}
		if !provCfg.Configured() {
			log.Printf("Warning: provider %q (%s) is not configured (missing: %s), skipping",
				name, def.EffectiveType(), strings.Join(missingProviderEnvKeys(name, def), ", "))
			continue
		}
		ep := services.NewGrammarEndpoint(name, def.EffectiveType(), def.URL, def.Key, def.GrammarModel(), def.Timeout)
		endpoints = append(endpoints, ep)
		log.Printf("  Grammar endpoint %d: %s (%s model=%s)", len(endpoints), name, def.URL, def.GrammarModel())
	}
	if len(endpoints) == 0 {
		log.Printf("Warning: no grammar AI endpoints could be created from GRAMMAR_FALLBACK_ORDER — " +
			"grammar analysis will use the built-in regex fallback")
		return nil
	}
	return endpoints
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
