package services

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestReceiptServiceAcknowledgeReceived(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	chatSvc := NewChatService(db)
	msgSvc := NewMessageService(db, nil)
	// hub is nil so FanOutReceipt short-circuits without further DB queries.
	rs := NewReceiptService(nil, nil, msgSvc, chatSvc)

	mock.ExpectExec(`UPDATE message_receipts SET received_at = COALESCE\(received_at, CURRENT_TIMESTAMP\) WHERE message_id = \$1 AND user_id = \$2 AND chat_id = \$3 AND received_at IS NULL`).
		WithArgs("msg-1", "user-2", "chat-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := rs.AcknowledgeReceived(context.Background(), "chat-1", "msg-1", "user-2"); err != nil {
		t.Fatalf("AcknowledgeReceived failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestReceiptServiceAcknowledgeRead(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	chatSvc := NewChatService(db)
	msgSvc := NewMessageService(db, nil)
	rs := NewReceiptService(nil, nil, msgSvc, chatSvc)

	mock.ExpectExec(`UPDATE message_receipts SET read_at = COALESCE\(read_at, CURRENT_TIMESTAMP\), received_at = COALESCE\(received_at, CURRENT_TIMESTAMP\) WHERE message_id = \$1 AND user_id = \$2 AND chat_id = \$3 AND read_at IS NULL`).
		WithArgs("msg-1", "user-2", "chat-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE chat_participants SET last_read_message_id = \$1 WHERE chat_id = \$2 AND user_id = \$3`).
		WithArgs("msg-1", "chat-1", "user-2").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := rs.AcknowledgeRead(context.Background(), "chat-1", "msg-1", "user-2"); err != nil {
		t.Fatalf("AcknowledgeRead failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestReceiptServiceFanOutToParticipants(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	chatSvc := NewChatService(db)
	msgSvc := NewMessageService(db, nil)
	hub := NewWebSocketHub(nil)
	rs := NewReceiptService(hub, nil, msgSvc, chatSvc)

	mock.ExpectQuery(`SELECT id, chat_id, user_id, role, joined_at, last_read_message_id FROM chat_participants WHERE chat_id = \$1`).
		WithArgs("chat-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "user_id", "role", "joined_at", "last_read_message_id"}).
			AddRow("p1", "chat-1", "sender-1", "member", time.Now(), nil).
			AddRow("p2", "chat-1", "user-2", "member", time.Now(), nil))

	rs.FanOutReceipt(context.Background(), "chat-1", "msg-1", "user-2", "read")

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
