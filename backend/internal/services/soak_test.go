package services

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/chorus/messenger/internal/models"
)

func TestSoakZeroLoss_PersistBeforeAck(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()
	svc := NewMessageService(db, nil)
	msg := &models.Message{ID: "soak-m1", ChatID: "c1", SenderID: "u1", Text: "hello soak"}
	mock.ExpectExec(`INSERT INTO message_receipts`).WithArgs("soak-m1", "u2", "c1").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO message_receipts`).WithArgs("soak-m1", "u3", "c1").WillReturnResult(sqlmock.NewResult(1, 1))
	if err := svc.InitializeReceipts(context.Background(), msg, []string{"u1", "u2", "u3"}); err != nil {
		t.Fatalf("InitializeReceipts: %v", err)
	}
	if len(msg.Receipts) != 2 {
		t.Fatalf("expected 2 receipts, got %d", len(msg.Receipts))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("receipts expectations: %v", err)
	}
}

func TestSoakZeroLoss_ConcurrentHubNoDrop(t *testing.T) {
	rdb, _ := newTestRedis(t)
	hub := NewWebSocketHub(rdb, "soak-s1")
	go hub.Run()
	const clients = 50
	const msgs = 100
	for i := 0; i < clients; i++ {
		ch := make(chan []byte, 512)
		c := &Client{ID: "soak-c-" + string(rune('a'+i%26)) + "-" + string(rune('0'+i%10)), UserID: "user-soak", Hub: hub, Send: ch}
		hub.Register <- c
	}
	time.Sleep(50 * time.Millisecond)
	for i := 0; i < msgs; i++ {
		hub.SendToUser("user-soak", "new_message", map[string]string{"seq": "x"})
	}
	time.Sleep(200 * time.Millisecond)
	hub.mu.RLock()
	dropped := 0
	for _, conns := range hub.userConns {
		for _, c := range conns {
			if len(c.Send) == cap(c.Send) {
				dropped++
			}
		}
	}
	hub.mu.RUnlock()
	if dropped > 0 {
		t.Fatalf("soak: %d clients had full buffers (loss > 0)", dropped)
	}
}

func TestSoakZeroLoss_InboxDrainIsLossless(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()
	mock.ExpectQuery(`SELECT m\.id.*FROM message_receipts r`).WithArgs("u-soak", 100).WillReturnRows(
		sqlmock.NewRows([]string{"id", "chat_id", "sender_id", "text", "original_language", "translations", "delivery_status", "reply_to_id", "created_at", "deleted_at", "forwarded_from_message_id", "forwarded_from_chat_id", "forwarded_from_sender_id"}).
			AddRow("m1", "c1", "u2", "hello", "", []byte(`{}`), "sent", nil, time.Now(), nil, nil, nil, nil).
			AddRow("m2", "c1", "u2", "world", "", []byte(`{}`), "sent", nil, time.Now(), nil, nil, nil, nil),
	)
	svc := NewInboxService(db, nil)
	msgs, err := svc.GetPendingMessagesForUser(context.Background(), "u-soak", 100)
	if err != nil {
		t.Fatalf("GetPending: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 pending, got %d", len(msgs))
	}
	mock.ExpectExec(`UPDATE message_receipts SET received_at`).WithArgs("u-soak", sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 2))
	if err := svc.MarkPendingDelivered(context.Background(), "u-soak", []string{"m1", "m2"}); err != nil {
		t.Fatalf("MarkPendingDelivered: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestSoakZeroLoss_CrossServerParallel(t *testing.T) {
	rdb, _ := newTestRedis(t)
	ctx := context.Background()
	hubA := NewWebSocketHub(rdb, "soak-A")
	go hubA.Run()
	hubB := NewWebSocketHub(rdb, "soak-B")
	go hubB.Run()
	regB := NewConnectionRegistry(rdb, "soak-B")
	_ = regB.Register(ctx, "u-target", "conn-t")
	db, mock, _ := sqlmock.New()
	defer db.Close()
	msgSvc := NewMessageService(db, nil)
	routerA := NewDeliveryRouter(rdb, hubA, NewConnectionRegistry(rdb, "soak-A"), msgSvc, "soak-A")
	routerB := NewDeliveryRouter(rdb, hubB, regB, msgSvc, "soak-B")
	delivered := make(chan []byte, 32)
	hubB.Register <- &Client{ID: "conn-t", UserID: "u-target", Hub: hubB, Send: delivered}
	time.Sleep(30 * time.Millisecond)
	routerB.Start(ctx)
	time.Sleep(80 * time.Millisecond)
	defer routerB.Stop()
	const n = 20
	for i := 0; i < n; i++ {
		mock.ExpectExec(`UPDATE message_receipts SET received_at`).WithArgs(sqlmock.AnyArg(), "u-target", "c-soak").WillReturnResult(sqlmock.NewResult(0, 1))
	}
	for i := 0; i < n; i++ {
		msg := &models.Message{ID: "cm-" + string(rune('0'+i%10)) + "-" + string(rune('a'+i%26)), ChatID: "c-soak", SenderID: "u-sender", Text: "soak"}
		routerA.RouteMessage(ctx, msg, []string{"u-sender", "u-target"})
	}
	time.Sleep(400 * time.Millisecond)
	got := 0
	for {
		select {
		case <-delivered:
			got++
		default:
			goto done
		}
	}
done:
	if got != n {
		t.Fatalf("zero-loss: expected %d deliveries on B, got %d", n, got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("mock: %v", err)
	}
}
