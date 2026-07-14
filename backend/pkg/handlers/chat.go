package handlers

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/chorus/messenger/internal/services"
	"github.com/chorus/messenger/pkg/ai"
)

type ChatRouter struct {
	Engine     *ai.Engine
	MessageSvc *services.MessageService
	ChatSvc    *services.ChatService
	Hub        *services.WebSocketHub
	DB         interface{}
	mu         sync.Mutex
}

type IncomingChatFrame struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type ChatMessageEnvelope struct {
	ChatID      string `json:"chatId"`
	MessageID   string `json:"messageId"`
	SenderID    string `json:"senderId"`
	RecipientID string `json:"recipientId"`
	Text        string `json:"text"`
	SourceLang  string `json:"sourceLang,omitempty"`
	TargetLang  string `json:"targetLang,omitempty"`
}

func NewChatRouter(engine *ai.Engine, messageSvc *services.MessageService, chatSvc *services.ChatService, hub *services.WebSocketHub) *ChatRouter {
	return &ChatRouter{Engine: engine, MessageSvc: messageSvc, ChatSvc: chatSvc, Hub: hub}
}

func (r *ChatRouter) HandleFrame(raw []byte) error {
	var frame IncomingChatFrame
	if err := json.Unmarshal(raw, &frame); err != nil {
		return err
	}
	if frame.Type != "message" {
		return nil
	}

	var env ChatMessageEnvelope
	if err := json.Unmarshal(frame.Payload, &env); err != nil {
		return err
	}
	if strings.TrimSpace(env.Text) == "" {
		return nil
	}

	translated, err := r.Engine.ExecuteFastTranslation(env.Text, env.SourceLang, env.TargetLang)
	if err != nil {
		log.Printf("fast translation failed: %v", err)
		translated = env.Text
	}

	payload := map[string]any{
		"chatId":      env.ChatID,
		"messageId":   env.MessageID,
		"senderId":    env.SenderID,
		"recipientId": env.RecipientID,
		"text":        env.Text,
		"translated":  translated,
		"timestamp":   time.Now().UTC().Format(time.RFC3339Nano),
	}
	if r.Hub != nil && env.RecipientID != "" {
		r.Hub.SendToUser(env.RecipientID, "new_message", payload)
	}

	go func(msgPayload map[string]any, chatID, messageID, senderID, recipientID, text, srcLang, tgtLang string) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("tutor analysis panicked: %v", rec)
			}
		}()

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		report, err := r.Engine.ProcessTutorAnalysis(text, srcLang, tgtLang)
		if err != nil {
			log.Printf("tutor analysis failed: %v", err)
			return
		}

		if r.MessageSvc != nil {
			if err := r.MessageSvc.UpdateTranslations(messageID, map[string]string{tgtLang: translatedFromReport(report)}); err != nil {
				log.Printf("hydrate tutor result failed: %v", err)
			}
		}

		if r.Hub != nil {
			payload := map[string]any{
				"chatId":      chatID,
				"messageId":   messageID,
				"senderId":    senderID,
				"recipientId": recipientID,
				"text":        text,
				"analysis":    report,
				"timestamp":   time.Now().UTC().Format(time.RFC3339Nano),
			}
			if recipientID != "" {
				r.Hub.SendToUser(recipientID, "message_updated", payload)
			}
		}

		_ = ctx
	}(payload, env.ChatID, env.MessageID, env.SenderID, env.RecipientID, env.Text, env.SourceLang, env.TargetLang)
	return nil
}

func translatedFromReport(report ai.TutorJSONReport) string {
	if len(report.Corrections) == 0 && len(report.Vocabulary) == 0 {
		return ""
	}
	var parts []string
	for _, correction := range report.Corrections {
		if correction.Corrected != "" {
			parts = append(parts, correction.Corrected)
		}
	}
	if len(parts) == 0 && len(report.Vocabulary) > 0 {
		parts = append(parts, report.Vocabulary[0].Word)
	}
	return strings.Join(parts, " | ")
}
