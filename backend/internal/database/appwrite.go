package database

import (
	"log"
)

type AppwriteClient struct {
	Endpoint   string
	ProjectID  string
	DatabaseID string
	APIKey     string
}

func ConnectAppwrite(endpoint, projectID, apiKey, databaseID string) (*AppwriteClient, error) {
	log.Println("Appwrite configuration loaded (SDK integration pending)")

	return &AppwriteClient{
		Endpoint:   endpoint,
		ProjectID:  projectID,
		DatabaseID: databaseID,
		APIKey:     apiKey,
	}, nil
}

// Collection IDs - these should match your Appwrite database collections
const (
	CollectionUsers            = "users"
	CollectionChats            = "chats"
	CollectionChatParticipants = "chat_participants"
	CollectionMessages         = "messages"
	CollectionRefreshTokens    = "refresh_tokens"
)
