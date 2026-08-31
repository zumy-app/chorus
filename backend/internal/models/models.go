package models

import "time"

type User struct {
	ID           string `json:"id" db:"id"`
	Username     string `json:"username" db:"username"`
	Email        string `json:"email" db:"email"`
	PasswordHash string `json:"-" db:"password_hash"`
	// FirstName and LastName are the onboarding-provided structured name;
	// DisplayName is the composed (and still overridable) display name.
	FirstName   string `json:"firstName" db:"first_name"`
	LastName    string `json:"lastName" db:"last_name"`
	DisplayName string `json:"displayName" db:"display_name"`
	// AvatarColor is the deterministic hex color backing the initials avatar
	// (REQ 2.2 / FR-20). Derived from the user's stable ID so it is identical
	// on every client and session. Not persisted — computed on read.
	AvatarColor string `json:"avatarColor" db:"-"`
	// AvatarURL is the reserved upload path for a custom avatar image. It stays
	// NULL until attachment infrastructure exists; clients fall back to the
	// initials + AvatarColor rendering when it is unset.
	AvatarURL       *string   `json:"avatarUrl,omitempty" db:"avatar_url"`
	NativeLanguage  string    `json:"nativeLanguage" db:"native_language"`
	TargetLanguages []string  `json:"targetLanguages" db:"target_languages"`
	Role            string    `json:"role" db:"role"` // member, moderator, admin
	CreatedAt       time.Time `json:"createdAt" db:"created_at"`
	LastActiveAt    time.Time `json:"lastActiveAt" db:"last_active_at"`
	// Plan is the stored billing plan ("free" or "premium"). Entitlements are
	// resolved through the entitlement service (see internal/services).
	Plan string `json:"plan" db:"plan"`
	// PlanGraceUntil, when set, is the explicit grace-upgrade deadline for
	// accounts created before a paid cutover: entitlements behave as premium
	// until this time, then fall back to the stored plan.
	PlanGraceUntil *time.Time `json:"planGraceUntil,omitempty" db:"plan_grace_until"`
	// Subscription fields (Phase 1.5). premium_since marks the first activation;
	// subscription_id is the provider-side subscription id; subscription_status
	// mirrors the provider lifecycle (ACTIVE, CANCELLED, SUSPENDED, ...).
	PremiumSince         *time.Time `json:"premiumSince,omitempty" db:"premium_since"`
	SubscriptionID       *string    `json:"subscriptionId,omitempty" db:"subscription_id"`
	SubscriptionProvider string     `json:"subscriptionProvider,omitempty" db:"subscription_provider"`
	SubscriptionPlanID   *string    `json:"subscriptionPlanId,omitempty" db:"subscription_plan_id"`
	SubscriptionStatus   *string    `json:"subscriptionStatus,omitempty" db:"subscription_status"`
	NextBillingDate      *time.Time `json:"nextBillingDate,omitempty" db:"next_billing_date"`
	LastPaymentAt        *time.Time `json:"lastPaymentAt,omitempty" db:"last_payment_at"`
	// SuspendedAt is a reversible soft ban; the account cannot authenticate
	// while non-nil.
	SuspendedAt *time.Time `json:"suspendedAt,omitempty" db:"suspended_at"`
	// DeletedAt is a permanent soft delete; the account is blocked and hidden
	// from the user directory.
	DeletedAt *time.Time `json:"deletedAt,omitempty" db:"deleted_at"`
	// Phase 2: Learning settings
	LearningSettings *LearningSettings `json:"learningSettings,omitempty" db:"-"`
	// Phase 2: Privacy settings
	PrivacySettings *PrivacySettings `json:"privacySettings,omitempty" db:"-"`
}

// Phase 2: Learning settings for grammar and vocabulary features
type LearningSettings struct {
	GrammarEnabled    bool   `json:"grammarEnabled" db:"grammar_enabled"`
	VocabularyEnabled bool   `json:"vocabularyEnabled" db:"vocabulary_enabled"`
	DifficultyLevel   string `json:"difficultyLevel" db:"difficulty_level"` // beginner, intermediate, advanced
}

// Phase 2: Privacy settings
type PrivacySettings struct {
	TranscriptRecording  bool `json:"transcriptRecording" db:"transcript_recording"`
	MessageRetentionDays int  `json:"messageRetentionDays" db:"message_retention_days"`
}

// UserSettings is the full, persisted per-account settings row (the
// user_settings table). It carries both the Phase 2 settings and the FR-25
// feature toggles. Set is returned by GET /users/me/settings and updated via
// PUT /users/me/settings.
type UserSettings struct {
	UserID               string    `json:"userId" db:"user_id"`
	GrammarEnabled       bool      `json:"grammarEnabled" db:"grammar_enabled"`
	VocabularyEnabled    bool      `json:"vocabularyEnabled" db:"vocabulary_enabled"`
	DifficultyLevel      string    `json:"difficultyLevel" db:"difficulty_level"`
	TranscriptRecording  bool      `json:"transcriptRecording" db:"transcript_recording"`
	MessageRetentionDays int       `json:"messageRetentionDays" db:"message_retention_days"`
	TranslationEnabled   bool      `json:"translationEnabled" db:"translation_enabled"`
	GrammarAuto          bool      `json:"grammarAuto" db:"grammar_auto"`
	HighlightsEnabled    bool      `json:"highlightsEnabled" db:"highlights_enabled"`
	UpdatedAt            time.Time `json:"updatedAt" db:"updated_at"`
}

// FeatureSettings are the FR-25 per-account feature toggles. They gate
// server-side behaviour: auto-translation, auto-grammar, and learning
// highlights. Toggles default to enabled; turning one off prevents the
// corresponding server-side job from being enqueued (translation jobs are
// never enqueued when translation_enabled is off).
type FeatureSettings struct {
	TranslationEnabled bool `json:"translationEnabled"`
	GrammarAuto        bool `json:"grammarAuto"`
	HighlightsEnabled  bool `json:"highlightsEnabled"`
}

// UpdateFeatureSettingsRequest is the partial-update payload for the FR-25
// toggles. Omitted (nil) pointers are left unchanged, so a client can toggle a
// single feature without re-sending the others.
type UpdateFeatureSettingsRequest struct {
	TranslationEnabled *bool `json:"translationEnabled"`
	GrammarAuto        *bool `json:"grammarAuto"`
	HighlightsEnabled  *bool `json:"highlightsEnabled"`
}

// Phase 2: Client model for multi-device support
type Client struct {
	ID               string     `json:"id" db:"id"`
	UserID           string     `json:"userId" db:"user_id"`
	DeviceType       string     `json:"deviceType" db:"device_type"` // mobile, web, desktop
	DeviceInfo       DeviceInfo `json:"deviceInfo" db:"-"`
	ConnectionStatus string     `json:"connectionStatus" db:"connection_status"` // online, offline
	LastActive       time.Time  `json:"lastActive" db:"last_active"`
	CreatedAt        time.Time  `json:"createdAt" db:"created_at"`
}

type DeviceInfo struct {
	Platform  string `json:"platform"`
	Version   string `json:"version"`
	UserAgent string `json:"userAgent,omitempty"`
}

// Phase 2: Inbox for offline message delivery
type InboxEntry struct {
	ClientID         string    `json:"clientId" db:"client_id"`
	MessageID        string    `json:"messageId" db:"message_id"`
	ChatID           string    `json:"chatId" db:"chat_id"`
	DeliveryAttempts int       `json:"deliveryAttempts" db:"delivery_attempts"`
	CreatedAt        time.Time `json:"createdAt" db:"created_at"`
	TTL              time.Time `json:"ttl" db:"ttl"` // 30 days from creation
}

// Phase 3: Vocabulary entry for language learning
type VocabularyEntry struct {
	ID           string        `json:"id" db:"id"`
	UserID       string        `json:"userId" db:"user_id"`
	Term         string        `json:"term" db:"term"`
	Language     string        `json:"language" db:"language"`
	Translation  string        `json:"translation" db:"translation"`
	Definition   string        `json:"definition" db:"definition"`
	Context      VocabContext  `json:"context" db:"-"`
	LearningData *LearningData `json:"learningData,omitempty" db:"-"`
	CreatedAt    time.Time     `json:"createdAt" db:"created_at"`
}

type VocabContext struct {
	MessageID string `json:"messageId"`
	Sentence  string `json:"sentence"`
	ChatID    string `json:"chatId"`
}

type LearningData struct {
	ReviewCount  int       `json:"reviewCount" db:"review_count"`
	CorrectCount int       `json:"correctCount" db:"correct_count"`
	NextReview   time.Time `json:"nextReview" db:"next_review"`
	Interval     int       `json:"interval" db:"interval"` // days
}

// Phase 3: Grammar analysis
type GrammarAnalysis struct {
	Difficulty   string   `json:"difficulty"` // CEFR level (A1-C2)
	Patterns     []string `json:"patterns"`
	Explanations []string `json:"explanations"`
}

// AI-powered grammar analysis (enriched via AI API)
type AIGrammarAnalysis struct {
	Difficulty        string          `json:"difficulty"`
	Summary           string          `json:"summary"`
	SentenceStructure string          `json:"sentenceStructure,omitempty"`
	KeyPhrases        []KeyPhrase     `json:"keyPhrases,omitempty"`
	DetailedBreakdown []BreakdownItem `json:"detailedBreakdown,omitempty"`
	GrammarNotes      []GrammarNote   `json:"grammarNotes,omitempty"`
}

type KeyPhrase struct {
	Phrase      string `json:"phrase"`
	Translation string `json:"translation"`
	Context     string `json:"context,omitempty"`
}

type GrammarNote struct {
	Title       string   `json:"title"`
	Explanation string   `json:"explanation"`
	Examples    []string `json:"examples,omitempty"`
}

type GrammarPattern struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Example     string `json:"example"`
}

type BreakdownItem struct {
	Text        string `json:"text"`
	Translation string `json:"translation,omitempty"`
	Role        string `json:"role,omitempty"`
	Type        string `json:"type"`
	Note        string `json:"note,omitempty"`
	Explanation string `json:"explanation,omitempty"`
}

type LearningContent struct {
	Action           string   `json:"action"`
	Content          string   `json:"content"`
	Details          []string `json:"details"`
	SuggestedActions []string `json:"suggestedActions"`
}

type LearnRequest struct {
	Text           string `json:"text" binding:"required"`
	Language       string `json:"language" binding:"required"`
	NativeLanguage string `json:"nativeLanguage" binding:"required"`
	Action         string `json:"action" binding:"required"` // breakdown, examples, flashcards, custom
	CustomQuery    string `json:"customQuery,omitempty"`
}

// Phase 3: Call session for voice/video
type CallSession struct {
	ID           string     `json:"id" db:"id"`
	ChatID       string     `json:"chatId" db:"chat_id"`
	Participants []string   `json:"participants" db:"-"`
	Type         string     `json:"type" db:"type"`     // audio, video
	Status       string     `json:"status" db:"status"` // active, ended
	StartedAt    time.Time  `json:"startedAt" db:"started_at"`
	EndedAt      *time.Time `json:"endedAt,omitempty" db:"ended_at"`
}

// Phase 3: Call transcript
type CallTranscript struct {
	ID        string              `json:"id" db:"id"`
	CallID    string              `json:"callId" db:"call_id"`
	Segments  []TranscriptSegment `json:"segments" db:"-"`
	CreatedAt time.Time           `json:"createdAt" db:"created_at"`
}

type TranscriptSegment struct {
	SpeakerID        string            `json:"speakerId"`
	StartTime        float64           `json:"startTime"`
	EndTime          float64           `json:"endTime"`
	OriginalText     string            `json:"originalText"`
	OriginalLanguage string            `json:"originalLanguage"`
	Translations     map[string]string `json:"translations"`
	Confidence       float64           `json:"confidence"`
}

type Chat struct {
	ID           string                 `json:"id" db:"id"`
	Type         string                 `json:"type" db:"type"` // 'direct' or 'group'
	Name         string                 `json:"name,omitempty" db:"name"`
	CreatedBy    string                 `json:"createdBy" db:"created_by"`
	Settings     map[string]interface{} `json:"settings,omitempty" db:"settings"`
	CreatedAt    time.Time              `json:"createdAt" db:"created_at"`
	Participants []ChatParticipant      `json:"participants,omitempty" db:"-"`
	LastMessage  *Message               `json:"lastMessage,omitempty" db:"-"`
	UnreadCount  int                    `json:"unreadCount,omitempty" db:"-"`
}

type ChatParticipant struct {
	ID                string    `json:"-" db:"id"`
	ChatID            string    `json:"chatId" db:"chat_id"`
	UserID            string    `json:"userId" db:"user_id"`
	Role              string    `json:"role" db:"role"` // 'member' or 'admin'
	JoinedAt          time.Time `json:"joinedAt" db:"joined_at"`
	LastReadMessageID *string   `json:"lastReadMessageId,omitempty" db:"last_read_message_id"`
	User              *User     `json:"user,omitempty" db:"-"`
}

type Message struct {
	ID                  string            `json:"id" db:"id"`
	ChatID              string            `json:"chatId" db:"chat_id"`
	SenderID            string            `json:"senderId" db:"sender_id"`
	Text                string            `json:"text" db:"text"`
	OriginalLanguage    string            `json:"originalLanguage" db:"original_language"`
	Translations        map[string]string `json:"translations,omitempty" db:"translations"`
	TranslationEnhanced bool              `json:"translationEnhanced"`
	DeliveryStatus      string            `json:"deliveryStatus" db:"delivery_status"`
	ReplyToID           *string           `json:"replyToId,omitempty" db:"reply_to_id"`
	CreatedAt           time.Time         `json:"timestamp" db:"created_at"`
	Sender              *User             `json:"sender,omitempty" db:"-"`
}

type AuthTokens struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int    `json:"expiresIn"`
}

type RegisterRequest struct {
	Username        string   `json:"username" binding:"omitempty,min=3,max=255"`
	Email           string   `json:"email" binding:"required,email"`
	Password        string   `json:"password" binding:"required,min=8"`
	FirstName       string   `json:"firstName" binding:"omitempty,min=1,max=100"`
	LastName        string   `json:"lastName" binding:"omitempty,min=1,max=100"`
	DisplayName     string   `json:"displayName" binding:"omitempty,min=1,max=100"`
	NativeLanguage  string   `json:"nativeLanguage" binding:"omitempty"`
	TargetLanguages []string `json:"targetLanguages"`
	InviteToken     string   `json:"inviteToken"`
}

type WaitlistRequest struct {
	Email           string   `json:"email" binding:"required,email"`
	SpokenLanguages []string `json:"spokenLanguages" binding:"required,min=1"`
	TargetLanguages []string `json:"targetLanguages"`
	Reasons         []string `json:"reasons" binding:"required,min=1"`
	Comments        string   `json:"comments"`
}

type WaitlistEntry struct {
	ID              string     `json:"id"`
	Email           string     `json:"email"`
	SpokenLanguages []string   `json:"spokenLanguages"`
	TargetLanguages []string   `json:"targetLanguages"`
	Reasons         []string   `json:"reasons"`
	Comments        string     `json:"comments"`
	Status          string     `json:"status"`
	QueuePosition   int        `json:"queuePosition"`
	CreatedAt       time.Time  `json:"createdAt"`
	ApprovedAt      *time.Time `json:"approvedAt,omitempty"`
}

// ContactMatch is a discovered on-platform user from a permission-gated contact
// scan. Raw contacts never leave the client: the client uploads only SHA-256
// hashes of normalized identifiers, and the server echoes back the matching
// hash so the client can correlate the result with its own address-book entry.
type ContactMatch struct {
	UserID          string   `json:"userId"`
	Username        string   `json:"username"`
	DisplayName     string   `json:"displayName"`
	Email           string   `json:"email"`
	EmailHash       string   `json:"emailHash"`
	NativeLanguage  string   `json:"nativeLanguage"`
	TargetLanguages []string `json:"targetLanguages"`
}

// ContactScanRequest is the privacy-preserving contact upload. Only hashed
// identifiers are transmitted; the server never sees raw contact data.
type ContactScanRequest struct {
	// Hashes are SHA-256 (lower) of each contact's normalized email, matching
	// the server's HashIdentifier scheme. Max 1000 per call to bound work.
	Hashes []string `json:"hashes" binding:"required"`
}

// ContactInviteRequest asks the backend to create a single-use invite for an
// off-platform contact and dispatch it (email) or return a shareable link
// (SMS/WhatsApp).
type ContactInviteRequest struct {
	Channel string `json:"channel" binding:"required,oneof=email sms whatsapp"`
	Contact struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Phone string `json:"phone"`
	} `json:"contact"`
}

// ContactInvite is a single-use, expiring invite created by an existing user
// for a contact who is not yet on Chorus. Status is one of pending, sent,
// redeemed, or expired. Token and Link are only populated on creation (listed
// invites are status-only; tokens are stored hashed and never re-exposed).
type ContactInvite struct {
	ID         string     `json:"id"`
	InviterID  string     `json:"inviterId"`
	Channel    string     `json:"channel"`
	Recipient  string     `json:"recipient"`
	Name       string     `json:"name,omitempty"`
	Token      string     `json:"token,omitempty"`
	Link       string     `json:"link,omitempty"`
	Status     string     `json:"status"`
	ExpiresAt  time.Time  `json:"expiresAt"`
	CreatedAt  time.Time  `json:"createdAt"`
	RedeemedAt *time.Time `json:"redeemedAt,omitempty"`
}

// EmailOutboxEntry is a row in the durable email outbox. Status is one of
// pending, sent, or failed. Pending rows are retried by the background worker
// with exponential backoff until they succeed or exhaust max attempts.
type EmailOutboxEntry struct {
	ID            string     `json:"id"`
	Recipient     string     `json:"recipient"`
	Subject       string     `json:"subject"`
	Status        string     `json:"status"`
	Attempts      int        `json:"attempts"`
	LastError     string     `json:"lastError,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	NextAttemptAt time.Time  `json:"nextAttemptAt"`
	SentAt        *time.Time `json:"sentAt,omitempty"`
}

// AdminStats aggregates high-level numbers for the admin dashboard.
type AdminStats struct {
	TotalUsers            int `json:"totalUsers"`
	WaitlistPending       int `json:"waitlistPending"`
	WaitlistApproved      int `json:"waitlistApproved"`
	WaitlistDeclined      int `json:"waitlistDeclined"`
	EmailsPending         int `json:"emailsPending"`
	EmailsSent            int `json:"emailsSent"`
	EmailsFailed          int `json:"emailsFailed"`
	TranslationsPending   int `json:"translationsPending"`
	TranslationsFailed    int `json:"translationsFailed"`
	TranslationsCompleted int `json:"translationsCompleted"`
	Moderators            int `json:"moderators"`
	Admins                int `json:"admins"`
	SuspendedUsers        int `json:"suspendedUsers"`
}

// TranslationJob is a durable record of one (message, target language)
// translation request. Status is one of pending, processing, done, failed.
type TranslationJob struct {
	ID          string     `json:"id"`
	MessageID   string     `json:"messageId"`
	ChatID      string     `json:"chatId"`
	Text        string     `json:"text"`
	SourceLang  string     `json:"sourceLang,omitempty"`
	TargetLang  string     `json:"targetLang"`
	Priority    int        `json:"priority"`
	Status      string     `json:"status"`
	Result      string     `json:"result,omitempty"`
	Attempts    int        `json:"attempts"`
	LastError   string     `json:"lastError,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	NextAttempt time.Time  `json:"nextAttempt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
	// FR-30 lineage: which provider/model produced the result, how long it took,
	// how many tokens were used, whether it hit the Redis cache, and which
	// prompt version was active. These feed the quality KPIs.
	Provider      string `json:"provider,omitempty"`
	Model         string `json:"model,omitempty"`
	PromptVersion string `json:"promptVersion,omitempty"`
	LatencyMS     int    `json:"latencyMs,omitempty"`
	Tokens        int    `json:"tokens,omitempty"`
	CacheHit      bool   `json:"cacheHit,omitempty"`
}

// TranslationEval is one cross-model evaluation of a produced translation
// (FR-30). The evaluator is a *different* model than the producer; it scores
// accuracy and fluency and emits a critique and CEFR estimate.
type TranslationEval struct {
	ID                string     `json:"id"`
	TranslationJobID  string     `json:"translationJobId"`
	MessageID         string     `json:"messageId"`
	ChatID            string     `json:"chatId"`
	SourceLang        string     `json:"sourceLang,omitempty"`
	TargetLang        string     `json:"targetLang"`
	SourceText        string     `json:"sourceText"`
	TranslatedText    string     `json:"translatedText"`
	ProducerProvider  string     `json:"producerProvider,omitempty"`
	EvaluatorProvider string     `json:"evaluatorProvider,omitempty"`
	AccuracyScore     float64    `json:"accuracyScore,omitempty"`
	FluencyScore      float64    `json:"fluencyScore,omitempty"`
	CEFRLevel         string     `json:"cefrLevel,omitempty"`
	Critique          string     `json:"critique,omitempty"`
	Status            string     `json:"status"`
	Attempts          int        `json:"attempts"`
	LastError         string     `json:"lastError,omitempty"`
	CreatedAt         time.Time  `json:"createdAt"`
	CompletedAt       *time.Time `json:"completedAt,omitempty"`
}

// GrammarEval is one cross-model evaluation of a produced grammar analysis
// (FR-30). criterion_scores holds per-pattern scores (grammar accuracy,
// structure clarity, helpfulness).
type GrammarEval struct {
	ID                string         `json:"id"`
	GrammarJobID      string         `json:"grammarJobId"`
	UserID            string         `json:"userId"`
	Text              string         `json:"text"`
	Language          string         `json:"language"`
	NativeLanguage    string         `json:"nativeLanguage"`
	ProducerProvider  string         `json:"producerProvider,omitempty"`
	EvaluatorProvider string         `json:"evaluatorProvider,omitempty"`
	AccuracyScore     float64        `json:"accuracyScore,omitempty"`
	CEFRLevel         string         `json:"cefrLevel,omitempty"`
	CriterionScores   map[string]any `json:"criterionScores,omitempty"`
	Critique          string         `json:"critique,omitempty"`
	Status            string         `json:"status"`
	Attempts          int            `json:"attempts"`
	LastError         string         `json:"lastError,omitempty"`
	CreatedAt         time.Time      `json:"createdAt"`
	CompletedAt       *time.Time     `json:"completedAt,omitempty"`
}

// QualityKPIs aggregates the FR-30 quality metrics so the admin console can
// surface accuracy, p95 latency, cost/1k tokens, and cache hit rate at a glance.
type QualityKPIs struct {
	Evaluations           float64 `json:"evaluations"`
	AvgAccuracy           float64 `json:"avgAccuracy"`
	AvgFluency            float64 `json:"avgFluency"`
	P95LatencyMS          float64 `json:"p95LatencyMs"`
	CacheHitRate          float64 `json:"cacheHitRate"`
	TotalTranslations     int     `json:"totalTranslations"`
	EvaluatedTranslations int     `json:"evaluatedTranslations"`
	AvgTokens             float64 `json:"avgTokens"`
	CostPer1KTokens       float64 `json:"costPer1kTokens"`
	EstCostUSD            float64 `json:"estCostUsd"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ResetPasswordRequest struct {
	Token    string `json:"token" binding:"required"`
	Password string `json:"password" binding:"required,min=8"`
}

type CreateChatRequest struct {
	Type         string   `json:"type" binding:"required,oneof=direct group"`
	Participants []string `json:"participants" binding:"required,min=1,max=100"`
	Name         string   `json:"name"`
}

type SendMessageRequest struct {
	Text      string  `json:"text" binding:"required,min=1,max=10000"`
	ReplyToID *string `json:"replyToId"`
}

type UpdateUserRequest struct {
	FirstName       string   `json:"firstName"`
	LastName        string   `json:"lastName"`
	DisplayName     string   `json:"displayName"`
	NativeLanguage  string   `json:"nativeLanguage"`
	TargetLanguages []string `json:"targetLanguages"`
}

// OnboardRequest captures the post-registration name capture step: the user
// provides a first and last name, and the backend composes the displayName
// (first + last) unless an explicit override is supplied.
type OnboardRequest struct {
	FirstName   string `json:"firstName" binding:"required,min=1,max=100"`
	LastName    string `json:"lastName" binding:"required,min=1,max=100"`
	DisplayName string `json:"displayName" binding:"omitempty,min=1,max=100"`
}

type WebSocketMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type TypingEvent struct {
	ChatID   string `json:"chatId"`
	UserID   string `json:"userId"`
	IsTyping bool   `json:"isTyping"`
}

// Phase 2: Presence status
type PresenceStatus struct {
	UserID     string    `json:"userId"`
	Status     string    `json:"status"` // online, offline, away
	LastSeen   time.Time `json:"lastSeen"`
	DeviceType string    `json:"deviceType,omitempty"`
}

// Phase 2: Message acknowledgment
type MessageAck struct {
	MessageID string `json:"messageId"`
	ChatID    string `json:"chatId"`
	Status    string `json:"status"` // received, read
}

// Phase 2: Search request/response
type SearchRequest struct {
	Query    string   `json:"query" binding:"required,min=1"`
	ChatIDs  []string `json:"chatIds,omitempty"`
	Language string   `json:"language,omitempty"`
	Limit    int      `json:"limit,omitempty"`
	Offset   int      `json:"offset,omitempty"`
}

type SearchResult struct {
	Messages []Message `json:"messages"`
	Total    int       `json:"total"`
	HasMore  bool      `json:"hasMore"`
}

// Phase 3: Grammar analysis request
type GrammarAnalysisRequest struct {
	MessageID      string `json:"messageId" binding:"required"`
	TargetLanguage string `json:"targetLanguage" binding:"required"`
	NativeLanguage string `json:"nativeLanguage"`
}

// Phase 3: Vocabulary requests
type SaveVocabularyRequest struct {
	MessageID string `json:"messageId" binding:"required"`
	Term      string `json:"term" binding:"required"`
	Language  string `json:"language" binding:"required"`
}

type PracticeResultRequest struct {
	VocabularyID string `json:"vocabularyId" binding:"required"`
	Correct      bool   `json:"correct"`
}

// Phase 3: Call requests
type InitiateCallRequest struct {
	ChatID string `json:"chatId" binding:"required"`
	Type   string `json:"type" binding:"required,oneof=audio video"`
}

// Phase 2: Media attachment
type MediaAttachment struct {
	ID           string    `json:"id" db:"id"`
	MessageID    string    `json:"messageId" db:"message_id"`
	Type         string    `json:"type" db:"type"` // image, video, audio, document
	FileName     string    `json:"fileName" db:"file_name"`
	FileSize     int64     `json:"fileSize" db:"file_size"`
	MimeType     string    `json:"mimeType" db:"mime_type"`
	URL          string    `json:"url" db:"url"`
	ThumbnailURL *string   `json:"thumbnailUrl,omitempty" db:"thumbnail_url"`
	CreatedAt    time.Time `json:"createdAt" db:"created_at"`
}

// Phase 2: Metrics for monitoring
type ServerMetrics struct {
	ActiveConnections int     `json:"activeConnections"`
	MessagesPerSecond float64 `json:"messagesPerSecond"`
	AverageLatency    float64 `json:"averageLatency"`
	ErrorRate         float64 `json:"errorRate"`
	MemoryUsage       uint64  `json:"memoryUsage"`
	CPUUsage          float64 `json:"cpuUsage"`
}

// Redis Pub/Sub message types
type PubSubMessage struct {
	Type       string      `json:"type"`
	Data       interface{} `json:"data"`
	TargetUser string      `json:"targetUser,omitempty"`
	ChatID     string      `json:"chatId,omitempty"`
	Timestamp  time.Time   `json:"timestamp"`
}

// Inbox delivery status
type DeliveryStatus struct {
	MessageID string `json:"messageId"`
	ClientID  string `json:"clientId"`
	Status    string `json:"status"` // pending, delivered, failed
	Attempts  int    `json:"attempts"`
}

// ---------------------------------------------------------------------------
// Monetization (Phase 1.5)
// ---------------------------------------------------------------------------

// PlanChange is one row in the plan audit trail. actor_id is null for
// system/provider-driven (webhook) changes.
type PlanChange struct {
	ID         string     `json:"id"`
	UserID     string     `json:"userId"`
	ActorID    *string    `json:"actorId,omitempty"`
	FromPlan   string     `json:"fromPlan"`
	ToPlan     string     `json:"toPlan"`
	GraceUntil *time.Time `json:"graceUntil,omitempty"`
	Source     string     `json:"source"` // webhook, admin
	Reason     string     `json:"reason"`
	CreatedAt  time.Time  `json:"createdAt"`
}

// SubscriptionEvent is one ingested provider event. provider_event_id is the
// idempotency key.
type SubscriptionEvent struct {
	ID              string     `json:"id"`
	UserID          *string    `json:"userId,omitempty"`
	EventType       string     `json:"eventType"`
	Provider        string     `json:"provider"`
	ProviderEventID string     `json:"providerEventId"`
	Status          string     `json:"status"` // received, processed, failed, ignored
	Payload         any        `json:"payload"`
	Error           string     `json:"error,omitempty"`
	ReceivedAt      time.Time  `json:"receivedAt"`
	HandledAt       *time.Time `json:"handledAt,omitempty"`
}

// SubscriptionInfo is the user-facing subscription view returned by
// GET /users/me/subscription. EffectivePlan is the plan after applying any
// grace window (resolved through the entitlement service).
type SubscriptionInfo struct {
	Plan            string     `json:"plan"`
	EffectivePlan   string     `json:"effectivePlan"`
	InGrace         bool       `json:"inGrace"`
	PremiumSince    *time.Time `json:"premiumSince,omitempty"`
	Status          string     `json:"status,omitempty"` // provider lifecycle: ACTIVE, CANCELLED, ...
	Provider        string     `json:"provider,omitempty"`
	SubscriptionID  string     `json:"subscriptionId,omitempty"`
	NextBillingDate *time.Time `json:"nextBillingDate,omitempty"`
	GraceUntil      *time.Time `json:"graceUntil,omitempty"`
	ManageURL       string     `json:"manageUrl,omitempty"`
	WordLimit       *int       `json:"wordLimit,omitempty"`
	AutoGrammar     bool       `json:"autoGrammar"`
}

// PremiumUserRow is a user row for the admin premium list endpoint.
type PremiumUserRow struct {
	ID                 string     `json:"id" db:"id"`
	Username           string     `json:"username" db:"username"`
	DisplayName        string     `json:"displayName" db:"display_name"`
	Email              string     `json:"email" db:"email"`
	Plan               string     `json:"plan" db:"plan"`
	EffectivePlan      string     `json:"effectivePlan"`
	InGrace            bool       `json:"inGrace"`
	SubscriptionID     *string    `json:"subscriptionId,omitempty" db:"subscription_id"`
	SubscriptionStatus *string    `json:"subscriptionStatus,omitempty" db:"subscription_status"`
	PremiumSince       *time.Time `json:"premiumSince,omitempty" db:"premium_since"`
	NextBillingDate    *time.Time `json:"nextBillingDate,omitempty" db:"next_billing_date"`
	GraceUntil         *time.Time `json:"graceUntil,omitempty" db:"plan_grace_until"`
	MessagesSent       int        `json:"messagesSent" db:"messages_sent"`
	CreatedAt          time.Time  `json:"createdAt" db:"created_at"`
}

// PremiumAnalytics aggregates premium figures for the admin console.
type PremiumAnalytics struct {
	TotalPremiumUsers    int              `json:"totalPremiumUsers"` // users current premium (stored plan = premium)
	StoredPremium        int              `json:"storedPremium"`     // plan = premium
	InGrace              int              `json:"inGrace"`           // plan = free but within grace window
	MonthlySubscriptions int              `json:"monthlySubscriptions"`
	YearlySubscriptions  int              `json:"yearlySubscriptions"`
	NewThisMonth         int              `json:"newThisMonth"`
	ChurnedThisMonth     int              `json:"churnedThisMonth"`
	ProjectedMRR         float64          `json:"projectedMRR"`
	RevenueLastYear      float64          `json:"revenueLastYear"` // simple 12-month projection
	TopUsersByUsage      []PremiumUserRow `json:"topUsersByUsage"`
}

// GrantPlanRequest is the admin grant/extend/revoke payload.
//   - For grant: Plan = "premium", Mode = "indefinite" | "days" | "until".
//   - For revoke: Plan = "free".
type GrantPlanRequest struct {
	Plan   string `json:"plan" binding:"required,oneof=free premium"`
	Mode   string `json:"mode" binding:"omitempty,oneof=indefinite days until"`
	Days   int    `json:"days"`
	Until  string `json:"until"` // RFC3339 timestamp used with mode=until
	Reason string `json:"reason"`
	// ClearGraceInDays, when revoking, keeps the user premium for this many
	// extra days before switching to free (0 = immediate downgrade).
	ClearGraceInDays int `json:"clearGraceInDays"`
}

// CheckoutRequest creates a provider subscription for the user. Plan is
// "monthly" or "annual".
type CheckoutRequest struct {
	Plan string `json:"plan" binding:"required,oneof=monthly annual"`
}

// CheckoutResponse returns the provider approval URL the client redirects to.
type CheckoutResponse struct {
	ApprovalURL string `json:"approvalUrl"`
	PlanKey     string `json:"plan"`
}

// PlannedChange describes the change a webhook event would apply, used for
// tests and for mapping provider events to actions.
type PlannedChange struct {
	Action  string // grant, extend, grace, revoke
	Plan    string
	Grace   *time.Time
	Status  string
	PlanKey string // "" or "monthly"/"annual"
}

// Block is one directed block record: blocker has hidden/blocked blocked.
// Enforcement treats the relationship as mutual — either direction stops
// direct-chat creation and messaging.
type Block struct {
	ID        string    `json:"id" db:"id"`
	BlockerID string    `json:"blockerId" db:"blocker_id"`
	BlockedID string    `json:"blockedId" db:"blocked_id"`
	Reason    string    `json:"reason" db:"reason"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	Blocked   *User     `json:"blocked,omitempty" db:"-"`
}

// ReportRequest is the payload for submitting a moderation report. Type is
// "user" or "message". For message reports the messageId is required; the
// reportedUserId is resolved server-side from the message when omitted.
type ReportRequest struct {
	Type           string `json:"type" binding:"required,oneof=user message"`
	ReportedUserID string `json:"reportedUserId,omitempty"`
	MessageID      string `json:"messageId,omitempty"`
	ChatID         string `json:"chatId,omitempty"`
	Reason         string `json:"reason" binding:"required,min=2,max=500"`
}

// Report is a moderation report row. Status is open, resolved, or dismissed.
type Report struct {
	ID             string     `json:"id" db:"id"`
	ReporterID     string     `json:"reporterId" db:"reporter_id"`
	Type           string     `json:"type" db:"type"` // user | message
	ReportedUserID string     `json:"reportedUserId" db:"reported_user_id"`
	MessageID      *string    `json:"messageId,omitempty" db:"message_id"`
	ChatID         *string    `json:"chatId,omitempty" db:"chat_id"`
	Reason         string     `json:"reason" db:"reason"`
	Status         string     `json:"status" db:"status"` // open | resolved | dismissed
	ResolverID     *string    `json:"resolverId,omitempty" db:"resolver_id"`
	ResolutionNote string     `json:"resolutionNote,omitempty" db:"resolution_note"`
	CreatedAt      time.Time  `json:"createdAt" db:"created_at"`
	ResolvedAt     *time.Time `json:"resolvedAt,omitempty" db:"resolved_at"`
	Reporter       *User      `json:"reporter,omitempty" db:"-"`
	ReportedUser   *User      `json:"reportedUser,omitempty" db:"-"`
}

// ReportStats aggregates open-report figures for the moderation console.
type ReportStats struct {
	OpenReports    int `json:"openReports"`
	UserReports    int `json:"userReports"`
	MessageReports int `json:"messageReports"`
	ResolvedToday  int `json:"resolvedToday"`
}
