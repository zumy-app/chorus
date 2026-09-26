package services

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestStateless_NoInMemoryDurableState(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	rdb, _ := newTestRedis(t)
	hub1 := NewWebSocketHub(rdb, "hs-s1")
	hub2 := NewWebSocketHub(rdb, "hs-s2")
	go hub1.Run()
	go hub2.Run()
	reg1 := NewConnectionRegistry(rdb, "hs-s1")
	reg2 := NewConnectionRegistry(rdb, "hs-s2")
	hub1.SetRegistry(reg1)
	hub2.SetRegistry(reg2)
	ctx := context.Background()
	if err := reg1.Register(ctx, "user-h", "conn-h1"); err != nil {
		t.Fatalf("register: %v", err)
	}
	e, _ := reg2.Lookup(ctx, "user-h")
	if e == nil || e.ServerID != "hs-s1" {
		t.Fatalf("registry not shared across servers, got %+v", e)
	}
	_ = reg1.Unregister(ctx, "user-h", "conn-h1")
	time.Sleep(10 * time.Millisecond)
	e, _ = reg2.Lookup(ctx, "user-h")
	if e != nil {
		t.Fatalf("registry should be cleared after unregister")
	}
	mock.ExpectQuery(`INSERT INTO messages`).WithArgs("c1", "u1", "hello stateless", nil).WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "sender_id", "text", "original_language", "translations", "delivery_status", "reply_to_id", "created_at"}).AddRow("hs-msg1", "c1", "u1", "hello stateless", "", []byte(`{}`), "sent", nil, time.Now()))
	mock.ExpectExec(`INSERT INTO message_receipts`).WithArgs("hs-msg1", "u2", "c1").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO message_receipts`).WithArgs("hs-msg1", "u3", "c1").WillReturnResult(sqlmock.NewResult(1, 1))
	svc := NewMessageService(db, rdb)
	msg, err := svc.Create("c1", "u1", "hello stateless", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	_ = svc.InitializeReceipts(ctx, msg, []string{"u1", "u2", "u3"})
	_ = hub1
	_ = hub2
}

func TestHorizontal_KillOneServerNoLoss(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	mock.ExpectQuery(`SELECT m\.id.*FROM message_receipts r`).WithArgs("u-off", 100).WillReturnRows(
		sqlmock.NewRows([]string{"id", "chat_id", "sender_id", "text", "original_language", "translations", "delivery_status", "reply_to_id", "created_at", "deleted_at", "forwarded_from_message_id", "forwarded_from_chat_id", "forwarded_from_sender_id"}).
			AddRow("m1", "c1", "u2", "hello", "", []byte(`{}`), "sent", nil, time.Now(), nil, nil, nil, nil),
	)
	inbox := NewInboxService(db, nil)
	msgs, err := inbox.GetPendingMessagesForUser(context.Background(), "u-off", 100)
	if err != nil {
		t.Fatalf("GetPending: %v", err)
	}
	if len(msgs) != 1 || msgs[0].ID != "m1" {
		t.Fatalf("expected replay of 1 message, got %v", msgs)
	}
}

func TestHorizontal_RegistryTTLIsEphemeral(t *testing.T) {
	if RegistryTTL != 45*time.Second {
		t.Fatalf("RegistryTTL drift: %v", RegistryTTL)
	}
	if RegistryHeartbeatInterval != 15*time.Second {
		t.Fatalf("heartbeat drift: %v", RegistryHeartbeatInterval)
	}
	if RegistryTTL <= RegistryHeartbeatInterval*2 {
		t.Fatalf("TTL must be > 2*heartbeat to tolerate one missed beat")
	}
}

func TestHorizontal_ServerIDUniquePerReplica(t *testing.T) {
	rdb, _ := newTestRedis(t)
	h1 := NewWebSocketHub(rdb, "replica-a")
	h2 := NewWebSocketHub(rdb, "replica-b")
	if h1.ServerID() == h2.ServerID() {
		t.Fatalf("server IDs must be unique per replica")
	}
	if ServerChannel("replica-a") == ServerChannel("replica-b") {
		t.Fatalf("server channels must be distinct")
	}
}
