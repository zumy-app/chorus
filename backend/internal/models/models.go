package models

import "time"

type User struct {
	ID              string    `json:"id" db:"id"`
	Username        string    `json:"username" db:"username"`
	Email           string    `json:"email" db:"email"`
	PasswordHash    string    `json:"-" db:"password_hash"`
	DisplayName     string    `json:"displayName" db:"display_name"`
	NativeLanguage  string    `json:"nativeLanguage" db:"native_language"`
	TargetLanguages []string  `json:"targetLanguages" db:"target_languages"`
	CreatedAt       time.Time `json:"createdAt" db:"created_at"`
	LastActiveAt    time.Time `json:"lastActiveAt" db:"last_active_at"`
	// Phase 2: Learning settings
	LearningSettings *LearningSettings `json:"learningSettings,omitempty" db:"-"`
	// Phase 2: Privacy settings
	PrivacySettings *PrivacySettings `json:"privacySettings,omitempty" db:"-"`
}

// Phase 2: Learning settings for grammar and vocabulary features
type LearningSettings struct {
	GrammarEnabled   bool   `json:"grammarEnabled" db:"grammar_enabled"`
	VocabularyEnabled bool  `json:"vocabularyEnabled" db:"vocabulary_enabled"`
	DifficultyLevel  string `json:"difficultyLevel" db:"difficulty_level"` // beginner, intermediate, advanced
}

// Phase 2: Privacy settings
type PrivacySettings struct {
	TranscriptRecording  bool `json:"transcriptRecording" db:"transcript_recording"`
	MessageRetentionDays int  `json:"messageRetentionDays" db:"message_retention_days"`
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

// Phase 3: Call session for voice/video
type CallSession struct {
	ID           string    `json:"id" db:"id"`
	ChatID       string    `json:"chatId" db:"chat_id"`
	Participants []string  `json:"participants" db:"-"`
	Type         string    `json:"type" db:"type"` // audio, video
	Status       string    `json:"status" db:"status"` // active, ended
	StartedAt    time.Time `json:"startedAt" db:"started_at"`
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
	ID               string                 `json:"id" db:"id"`
	ChatID           string                 `json:"chatId" db:"chat_id"`
	SenderID         string                 `json:"senderId" db:"sender_id"`
	Text             string                 `json:"text" db:"text"`
	OriginalLanguage string                 `json:"originalLanguage" db:"original_language"`
	Translations     map[string]string      `json:"translations,omitempty" db:"translations"`
	DeliveryStatus   string                 `json:"deliveryStatus" db:"delivery_status"` // 'sent', 'delivered', 'failed'
	ReplyToID        *string                `json:"replyToId,omitempty" db:"reply_to_id"`
	CreatedAt        time.Time              `json:"timestamp" db:"created_at"`
	Sender           *User                  `json:"sender,omitempty" db:"-"`
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
	DisplayName     string   `json:"displayName" binding:"omitempty,min=1,max=100"`
	NativeLanguage  string   `json:"nativeLanguage" binding:"omitempty"`
	TargetLanguages []string `json:"targetLanguages"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
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
	DisplayName     string   `json:"displayName"`
	NativeLanguage  string   `json:"nativeLanguage"`
	TargetLanguages []string `json:"targetLanguages"`
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
	ID          string    `json:"id" db:"id"`
	MessageID   string    `json:"messageId" db:"message_id"`
	Type        string    `json:"type" db:"type"` // image, video, audio, document
	FileName    string    `json:"fileName" db:"file_name"`
	FileSize    int64     `json:"fileSize" db:"file_size"`
	MimeType    string    `json:"mimeType" db:"mime_type"`
	URL         string    `json:"url" db:"url"`
	ThumbnailURL *string  `json:"thumbnailUrl,omitempty" db:"thumbnail_url"`
	CreatedAt   time.Time `json:"createdAt" db:"created_at"`
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
