package services

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/chorus/messenger/internal/models"
)

func TestMessageCreate(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := NewMessageService(db, nil)

	translationsJSON, _ := json.Marshal(map[string]string{})

	mock.ExpectQuery(`INSERT INTO messages \(chat_id, sender_id, text, delivery_status, reply_to_id\) VALUES \(\$1, \$2, \$3, 'sent', \$4\) RETURNING id, chat_id, sender_id, text, COALESCE\(original_language, ''\), COALESCE\(translations, '{}'::jsonb\), delivery_status, reply_to_id, created_at`).
		WithArgs("chat-1", "user-1", "Hello, World!", nil).
		WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "sender_id", "text", "original_language", "translations", "delivery_status", "reply_to_id", "created_at"}).
			AddRow("msg-1", "chat-1", "user-1", "Hello, World!", "en", translationsJSON, "sent", nil, time.Now()))

	msg, err := s.Create("chat-1", "user-1", "Hello, World!", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if msg.ID != "msg-1" {
		t.Fatalf("expected msg-1, got %s", msg.ID)
	}
	if msg.Text != "Hello, World!" {
		t.Fatalf("expected 'Hello, World!', got '%s'", msg.Text)
	}
	if msg.DeliveryStatus != "sent" {
		t.Fatalf("expected 'sent', got '%s'", msg.DeliveryStatus)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMessageCreate_WithReply(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := NewMessageService(db, nil)
	replyToID := "original-msg-1"
	translationsJSON, _ := json.Marshal(map[string]string{})

	mock.ExpectQuery(`INSERT INTO messages \(chat_id, sender_id, text, delivery_status, reply_to_id\) VALUES \(\$1, \$2, \$3, 'sent', \$4\) RETURNING id, chat_id, sender_id, text, COALESCE\(original_language, ''\), COALESCE\(translations, '{}'::jsonb\), delivery_status, reply_to_id, created_at`).
		WithArgs("chat-1", "user-1", "Reply message", &replyToID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "sender_id", "text", "original_language", "translations", "delivery_status", "reply_to_id", "created_at"}).
			AddRow("msg-2", "chat-1", "user-1", "Reply message", "en", translationsJSON, "sent", &replyToID, time.Now()))

	msg, err := s.Create("chat-1", "user-1", "Reply message", &replyToID)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if msg.ReplyToID == nil || *msg.ReplyToID != "original-msg-1" {
		t.Fatalf("expected reply to original-msg-1")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMessageGetMessages(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := NewMessageService(db, nil)
	translationsJSON, _ := json.Marshal(map[string]string{})

	mock.ExpectQuery(`SELECT id, chat_id, sender_id, text, COALESCE\(original_language, ''\), COALESCE\(translations, '{}'::jsonb\), delivery_status, reply_to_id, created_at, deleted_at, forwarded_from_message_id, forwarded_from_chat_id, forwarded_from_sender_id FROM messages WHERE chat_id = \$1 AND deleted_at IS NULL ORDER BY created_at DESC LIMIT \$2`).
		WithArgs("chat-1", 50).
		WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "sender_id", "text", "original_language", "translations", "delivery_status", "reply_to_id", "created_at", "deleted_at", "forwarded_from_message_id", "forwarded_from_chat_id", "forwarded_from_sender_id"}).
			AddRow("msg-1", "chat-1", "user-1", "Hello", "en", translationsJSON, "sent", nil, time.Now(), nil, nil, nil, nil).
			AddRow("msg-2", "chat-1", "user-2", "Hi back", "es", translationsJSON, "sent", nil, time.Now(), nil, nil, nil, nil))

	messages, err := s.GetMessages("chat-1", 50, nil)
	if err != nil {
		t.Fatalf("GetMessages failed: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}
	if messages[0].Text != "Hello" {
		t.Fatalf("expected 'Hello', got '%s'", messages[0].Text)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMessageGetMessages_BeforeCursor(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := NewMessageService(db, nil)
	translationsJSON, _ := json.Marshal(map[string]string{})
	before := "msg-10"

	mock.ExpectQuery(`SELECT id, chat_id, sender_id, text, COALESCE\(original_language, ''\), COALESCE\(translations, '{}'::jsonb\), delivery_status, reply_to_id, created_at, deleted_at, forwarded_from_message_id, forwarded_from_chat_id, forwarded_from_sender_id FROM messages WHERE chat_id = \$1 AND deleted_at IS NULL AND created_at < \(SELECT created_at FROM messages WHERE id = \$2\) ORDER BY created_at DESC LIMIT \$3`).
		WithArgs("chat-1", before, 20).
		WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "sender_id", "text", "original_language", "translations", "delivery_status", "reply_to_id", "created_at", "deleted_at", "forwarded_from_message_id", "forwarded_from_chat_id", "forwarded_from_sender_id"}).
			AddRow("msg-1", "chat-1", "user-1", "Older message", "en", translationsJSON, "sent", nil, time.Now(), nil, nil, nil, nil))

	messages, err := s.GetMessages("chat-1", 20, &before)
	if err != nil {
		t.Fatalf("GetMessages failed: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMessageGetMessageByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := NewMessageService(db, nil)
	translationsJSON, _ := json.Marshal(map[string]string{})

	mock.ExpectQuery(`SELECT id, chat_id, sender_id, text, COALESCE\(original_language, ''\), COALESCE\(translations, '{}'::jsonb\), delivery_status, reply_to_id, created_at, deleted_at, forwarded_from_message_id, forwarded_from_chat_id, forwarded_from_sender_id FROM messages WHERE id = \$1`).
		WithArgs("msg-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "sender_id", "text", "original_language", "translations", "delivery_status", "reply_to_id", "created_at", "deleted_at", "forwarded_from_message_id", "forwarded_from_chat_id", "forwarded_from_sender_id"}).
			AddRow("msg-1", "chat-1", "user-1", "Hello", "en", translationsJSON, "sent", nil, time.Now(), nil, nil, nil, nil))

	msg, err := s.GetMessageByID(context.Background(), "msg-1")
	if err != nil {
		t.Fatalf("GetMessageByID failed: %v", err)
	}
	if msg == nil {
		t.Fatal("expected non-nil message")
	}
	if msg.ID != "msg-1" {
		t.Fatalf("expected msg-1, got %s", msg.ID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMessageGetMessageByID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := NewMessageService(db, nil)

	mock.ExpectQuery(`SELECT id, chat_id, sender_id, text, COALESCE\(original_language, ''\), COALESCE\(translations, '{}'::jsonb\), delivery_status, reply_to_id, created_at, deleted_at, forwarded_from_message_id, forwarded_from_chat_id, forwarded_from_sender_id FROM messages WHERE id = \$1`).
		WithArgs("nonexistent").
		WillReturnError(sqlmock.ErrCancelled)

	msg, err := s.GetMessageByID(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent message")
	}
	if msg != nil {
		t.Fatal("expected nil message for error")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMessageUpdateTranslations(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := NewMessageService(db, nil)

	translations := map[string]string{"es": "Hola", "fr": "Bonjour"}

	mock.ExpectExec(`UPDATE messages SET translations = \$1 WHERE id = \$2`).
		WithArgs(sqlmock.AnyArg(), "msg-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = s.UpdateTranslations("msg-1", translations)
	if err != nil {
		t.Fatalf("UpdateTranslations failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMessageMarkAsRead(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := NewMessageService(db, nil)

	mock.ExpectExec(`UPDATE message_receipts SET read_at = COALESCE\(read_at, CURRENT_TIMESTAMP\), received_at = COALESCE\(received_at, CURRENT_TIMESTAMP\) WHERE message_id = \$1 AND user_id = \$2 AND chat_id = \$3 AND read_at IS NULL`).
		WithArgs("msg-10", "user-1", "chat-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE chat_participants SET last_read_message_id = \$1 WHERE chat_id = \$2 AND user_id = \$3`).
		WithArgs("msg-10", "chat-1", "user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = s.MarkAsRead("chat-1", "user-1", "msg-10")
	if err != nil {
		t.Fatalf("MarkAsRead failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMessageInitializeReceipts(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := NewMessageService(db, nil)
	msg := &models.Message{ID: "msg-1", ChatID: "chat-1", SenderID: "sender-1"}

	for _, userID := range []string{"user-2", "user-3"} {
		mock.ExpectExec(`INSERT INTO message_receipts \(message_id, user_id, chat_id\) VALUES \(\$1, \$2, \$3\) ON CONFLICT \(message_id, user_id\) DO NOTHING`).
			WithArgs("msg-1", userID, "chat-1").
			WillReturnResult(sqlmock.NewResult(0, 1))
	}

	err = s.InitializeReceipts(context.Background(), msg, []string{"sender-1", "user-2", "user-3"})
	if err != nil {
		t.Fatalf("InitializeReceipts failed: %v", err)
	}
	if len(msg.Receipts) != 2 {
		t.Fatalf("expected 2 receipts, got %d", len(msg.Receipts))
	}
	if msg.Receipts[0].UserID != "user-2" || msg.Receipts[0].Status != "sent" {
		t.Fatalf("unexpected receipt: %+v", msg.Receipts[0])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMessageMarkDelivered(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := NewMessageService(db, nil)

	mock.ExpectExec(`UPDATE message_receipts SET received_at = COALESCE\(received_at, CURRENT_TIMESTAMP\) WHERE message_id = \$1 AND user_id = \$2 AND chat_id = \$3 AND received_at IS NULL`).
		WithArgs("msg-1", "user-2", "chat-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	changed, err := s.MarkDelivered("chat-1", "msg-1", "user-2")
	if err != nil {
		t.Fatalf("MarkDelivered failed: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true when receipt had no delivered_at yet")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMessageMarkDelivered_NoChange(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := NewMessageService(db, nil)

	mock.ExpectExec(`UPDATE message_receipts SET received_at = COALESCE\(received_at, CURRENT_TIMESTAMP\) WHERE message_id = \$1 AND user_id = \$2 AND chat_id = \$3 AND received_at IS NULL`).
		WithArgs("msg-1", "user-2", "chat-1").
		WillReturnResult(sqlmock.NewResult(0, 0))

	changed, err := s.MarkDelivered("chat-1", "msg-1", "user-2")
	if err != nil {
		t.Fatalf("MarkDelivered failed: %v", err)
	}
	if changed {
		t.Fatal("expected changed=false when receipt was already delivered")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMessageMarkRead(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := NewMessageService(db, nil)

	mock.ExpectExec(`UPDATE message_receipts SET read_at = COALESCE\(read_at, CURRENT_TIMESTAMP\), received_at = COALESCE\(received_at, CURRENT_TIMESTAMP\) WHERE message_id = \$1 AND user_id = \$2 AND chat_id = \$3 AND read_at IS NULL`).
		WithArgs("msg-1", "user-2", "chat-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE chat_participants SET last_read_message_id = \$1 WHERE chat_id = \$2 AND user_id = \$3`).
		WithArgs("msg-1", "chat-1", "user-2").
		WillReturnResult(sqlmock.NewResult(0, 1))

	changed, err := s.MarkRead("chat-1", "msg-1", "user-2")
	if err != nil {
		t.Fatalf("MarkRead failed: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true when receipt had no read_at yet")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMessageDeleteMessage(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := NewMessageService(db, nil)

	mock.ExpectExec(`UPDATE messages SET deleted_at = COALESCE\(deleted_at, CURRENT_TIMESTAMP\) WHERE id = \$1 AND chat_id = \$2`).
		WithArgs("msg-10", "chat-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	deleted, err := s.DeleteMessage(context.Background(), "chat-1", "msg-10")
	if err != nil {
		t.Fatalf("DeleteMessage failed: %v", err)
	}
	if !deleted {
		t.Fatal("expected deleted=true when the message exists")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMessageDeleteMessage_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := NewMessageService(db, nil)

	mock.ExpectExec(`UPDATE messages SET deleted_at = COALESCE\(deleted_at, CURRENT_TIMESTAMP\) WHERE id = \$1 AND chat_id = \$2`).
		WithArgs("nope", "chat-1").
		WillReturnResult(sqlmock.NewResult(0, 0))

	deleted, err := s.DeleteMessage(context.Background(), "chat-1", "nope")
	if err != nil {
		t.Fatalf("DeleteMessage failed: %v", err)
	}
	if deleted {
		t.Fatal("expected deleted=false for a nonexistent message")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMessageForwardMessage(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := NewMessageService(db, nil)
	translationsJSON, _ := json.Marshal(map[string]string{})

	sourceMessageID := "msg-1"
	sourceChatID := "chat-1"
	targetChatID := "chat-2"
	forwarderID := "user-9"
	originalSenderID := "user-1"

	// Source lookup.
	mock.ExpectQuery(`SELECT text, COALESCE\(original_language, ''\), sender_id FROM messages WHERE id = \$1 AND chat_id = \$2 AND deleted_at IS NULL`).
		WithArgs(sourceMessageID, sourceChatID).
		WillReturnRows(sqlmock.NewRows([]string{"text", "original_language", "sender_id"}).
			AddRow("Hola mundo", "es", originalSenderID))

	// Forwarded copy insert.
	mock.ExpectQuery(`INSERT INTO messages \(chat_id, sender_id, text, delivery_status, original_language, forwarded_from_message_id, forwarded_from_chat_id, forwarded_from_sender_id\) VALUES \(\$1, \$2, \$3, 'sent', \$4, \$5, \$6, \$7\) RETURNING id, chat_id, sender_id, text, COALESCE\(original_language, ''\), COALESCE\(translations, '{}'::jsonb\), delivery_status, reply_to_id, created_at, deleted_at, forwarded_from_message_id, forwarded_from_chat_id, forwarded_from_sender_id`).
		WithArgs(targetChatID, forwarderID, "Hola mundo", "es", sourceMessageID, sourceChatID, originalSenderID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "sender_id", "text", "original_language", "translations", "delivery_status", "reply_to_id", "created_at", "deleted_at", "forwarded_from_message_id", "forwarded_from_chat_id", "forwarded_from_sender_id"}).
			AddRow("fwd-1", targetChatID, forwarderID, "Hola mundo", "es", translationsJSON, "sent", nil, time.Now(), nil, sourceMessageID, sourceChatID, originalSenderID))

	fwd, err := s.ForwardMessage(context.Background(), sourceChatID, sourceMessageID, targetChatID, forwarderID)
	if err != nil {
		t.Fatalf("ForwardMessage failed: %v", err)
	}
	if fwd.ID != "fwd-1" {
		t.Fatalf("expected fwd-1, got %s", fwd.ID)
	}
	if !fwd.Forwarded {
		t.Fatal("expected forwarded=true on the forwarded copy")
	}
	if fwd.SenderID != forwarderID {
		t.Fatalf("expected the forwarder as sender, got %s", fwd.SenderID)
	}
	if fwd.ForwardedFromMessageID == nil || *fwd.ForwardedFromMessageID != sourceMessageID {
		t.Fatalf("expected forwardedFromMessageId=%s", sourceMessageID)
	}
	if fwd.ForwardedFromChatID == nil || *fwd.ForwardedFromChatID != sourceChatID {
		t.Fatalf("expected forwardedFromChatId=%s", sourceChatID)
	}
	if fwd.ForwardedFromSenderID == nil || *fwd.ForwardedFromSenderID != originalSenderID {
		t.Fatalf("expected forwardedFromSenderId=%s", originalSenderID)
	}
	if fwd.Text != "Hola mundo" {
		t.Fatalf("expected the original text, got %q", fwd.Text)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMessagePinMessage(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := NewMessageService(db, nil)

	mock.ExpectExec(`INSERT INTO pinned_messages \(chat_id, message_id, pinned_by\) VALUES \(\$1, \$2, \$3\) ON CONFLICT \(chat_id, message_id\) DO NOTHING`).
		WithArgs("chat-1", "msg-1", "user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = s.PinMessage(context.Background(), "chat-1", "msg-1", "user-1")
	if err != nil {
		t.Fatalf("PinMessage failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMessageUnpinMessage(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := NewMessageService(db, nil)

	mock.ExpectExec(`DELETE FROM pinned_messages WHERE chat_id = \$1 AND message_id = \$2`).
		WithArgs("chat-1", "msg-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = s.UnpinMessage(context.Background(), "chat-1", "msg-1")
	if err != nil {
		t.Fatalf("UnpinMessage failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMessageGetPinnedMessages(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := NewMessageService(db, nil)
	translationsJSON, _ := json.Marshal(map[string]string{})

	mock.ExpectQuery(`SELECT pm.pinned_by, pm.created_at, m.id, m.chat_id, m.sender_id, m.text, COALESCE\(m.original_language, ''\), COALESCE\(m.translations, '{}'::jsonb\), m.delivery_status, m.reply_to_id, m.created_at, m.deleted_at, m.forwarded_from_message_id, m.forwarded_from_chat_id, m.forwarded_from_sender_id FROM pinned_messages pm JOIN messages m ON m.id = pm.message_id WHERE pm.chat_id = \$1 AND m.deleted_at IS NULL ORDER BY pm.created_at DESC`).
		WithArgs("chat-1").
		WillReturnRows(sqlmock.NewRows([]string{"pinned_by", "pinned_at", "id", "chat_id", "sender_id", "text", "original_language", "translations", "delivery_status", "reply_to_id", "created_at", "deleted_at", "forwarded_from_message_id", "forwarded_from_chat_id", "forwarded_from_sender_id"}).
			AddRow("user-1", time.Now(), "msg-1", "chat-1", "sender-1", "Hola", "es", translationsJSON, "sent", nil, time.Now(), nil, nil, nil, nil))

	pins, err := s.GetPinnedMessages(context.Background(), "chat-1")
	if err != nil {
		t.Fatalf("GetPinnedMessages failed: %v", err)
	}
	if len(pins) != 1 {
		t.Fatalf("expected 1 pin, got %d", len(pins))
	}
	if pins[0].PinnedBy != "user-1" {
		t.Fatalf("expected pinnedBy user-1, got %s", pins[0].PinnedBy)
	}
	if pins[0].Message.ID != "msg-1" {
		t.Fatalf("expected pinned message msg-1, got %s", pins[0].Message.ID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMessageAttachReceipts(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := NewMessageService(db, nil)
	messages := []models.Message{
		{ID: "msg-1", ChatID: "chat-1"},
		{ID: "msg-2", ChatID: "chat-1"},
	}

	mock.ExpectQuery(`SELECT message_id, user_id, chat_id, received_at, read_at FROM message_receipts WHERE chat_id = \$1 AND message_id = ANY\(\$2\)`).
		WithArgs("chat-1", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"message_id", "user_id", "chat_id", "received_at", "read_at"}).
			AddRow("msg-1", "user-2", "chat-1", time.Now(), nil).
			AddRow("msg-1", "user-3", "chat-1", time.Now(), time.Now()))

	err = s.AttachReceipts(context.Background(), "chat-1", messages)
	if err != nil {
		t.Fatalf("AttachReceipts failed: %v", err)
	}
	if len(messages[0].Receipts) != 2 {
		t.Fatalf("expected 2 receipts on msg-1, got %d", len(messages[0].Receipts))
	}
	if messages[0].Receipts[0].Status != "delivered" {
		t.Fatalf("expected status delivered, got %s", messages[0].Receipts[0].Status)
	}
	if messages[0].Receipts[1].Status != "read" {
		t.Fatalf("expected status read, got %s", messages[0].Receipts[1].Status)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
