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
	Username        string   `json:"username" binding:"required,min=3,max=30"`
	Email           string   `json:"email" binding:"required,email"`
	Password        string   `json:"password" binding:"required,min=8"`
	DisplayName     string   `json:"displayName" binding:"required,min=1,max=100"`
	NativeLanguage  string   `json:"nativeLanguage" binding:"required"`
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
