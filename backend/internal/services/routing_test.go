package services

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/chorus/messenger/internal/models"
	"github.com/redis/go-redis/v9"
)

func newTestRedis(t *testing.T) (*redis.Client, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	t.Cleanup(func() { mr.Close() })
	return redis.NewClient(&redis.Options{Addr: mr.Addr()}), mr
}

func TestCrossServerRoutingRoundTrip(t *testing.T) {
	mr, _ := newTestRedis(t)
	_ = mr
	rdb, _ := newTestRedis(t)
	ctx := context.Background()

	hubS1 := NewWebSocketHub(rdb, "server-s1")
	go hubS1.Run()
	hubS2 := NewWebSocketHub(rdb, "server-s2")
	go hubS2.Run()

	regS1 := NewConnectionRegistry(rdb, "server-s1")
	regS2 := NewConnectionRegistry(rdb, "server-s2")

	if err := regS2.Register(ctx, "user-b", "conn-b1"); err != nil {
		t.Fatalf("register S2: %v", err)
	}

	entry, err := regS1.Lookup(ctx, "user-b")
	if err != nil || entry == nil || entry.ServerID != "server-s2" {
		t.Fatalf("lookup from S1 should see user-b on S2, got %+v err %v", entry, err)
	}

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()
	msgSvc := NewMessageService(db, nil)
	routerS1 := NewDeliveryRouter(rdb, hubS1, regS1, msgSvc, "server-s1")
	routerS2 := NewDeliveryRouter(rdb, hubS2, regS2, msgSvc, "server-s2")

	delivered := make(chan []byte, 1)
	clientB := &Client{ID: "conn-b1", UserID: "user-b", Hub: hubS2, Send: delivered}
	hubS2.Register <- clientB
	time.Sleep(50 * time.Millisecond)

	mock.ExpectExec(`UPDATE message_receipts SET received_at`).WithArgs("msg-1", "user-b", "chat-1").WillReturnResult(sqlmock.NewResult(0, 1))

	routerS2.Start(ctx)
	time.Sleep(100 * time.Millisecond)
	defer routerS2.Stop()

	msg := &models.Message{ID: "msg-1", ChatID: "chat-1", SenderID: "user-a", Text: "hello cross-server"}
	routerS1.RouteMessage(ctx, msg, []string{"user-a", "user-b"})

	select {
	case payload := <-rdb.Subscribe(ctx, ServerChannel("server-s2")).Channel():
		_ = payload
	case <-time.After(2 * time.Second):
	}

	time.Sleep(300 * time.Millisecond)

	select {
	case raw := <-delivered:
		var ws models.WebSocketMessage
		if err := json.Unmarshal(raw, &ws); err != nil {
			t.Fatalf("unmarshal ws: %v", err)
		}
		if ws.Type != "new_message" {
			t.Fatalf("expected new_message, got %s", ws.Type)
		}
		data, _ := json.Marshal(ws.Data)
		var got models.Message
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("unmarshal message data: %v", err)
		}
		if got.ID != "msg-1" || got.Text != "hello cross-server" {
			t.Fatalf("unexpected message %+v", got)
		}
	default:
		t.Fatal("user-b did not receive cross-server message on S2")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("msgSvc expectations: %v", err)
	}
}

func TestRoutingFallsBackToLocalWhenRegistryMissing(t *testing.T) {
	rdb, _ := newTestRedis(t)
	ctx := context.Background()
	hub := NewWebSocketHub(rdb, "server-s1")
	go hub.Run()
	reg := NewConnectionRegistry(rdb, "server-s1")
	msg := &models.Message{ID: "m1", ChatID: "c1", SenderID: "u1", Text: "hi"}
	router := NewDeliveryRouter(rdb, hub, reg, nil, "server-s1")
	delivered := make(chan []byte, 1)
	client := &Client{ID: "c1", UserID: "u2", Hub: hub, Send: delivered}
	hub.Register <- client
	time.Sleep(20 * time.Millisecond)
	router.RouteMessage(ctx, msg, []string{"u1", "u2"})
	select {
	case raw := <-delivered:
		var ws models.WebSocketMessage
		json.Unmarshal(raw, &ws)
		if ws.Type != "new_message" {
			t.Fatalf("expected new_message got %s", ws.Type)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected local delivery when registry missing")
	}
}

func TestHandleServerMessageMarksDelivered(t *testing.T) {
	rdb, _ := newTestRedis(t)
	hub := NewWebSocketHub(rdb, "server-s2")
	go hub.Run()
	db, mock, _ := sqlmock.New()
	defer db.Close()
	msgSvc := NewMessageService(db, nil)
	router := NewDeliveryRouter(rdb, hub, NewConnectionRegistry(rdb, "server-s2"), msgSvc, "server-s2")
	delivered := make(chan []byte, 1)
	hub.Register <- &Client{ID: "conn-x", UserID: "user-x", Hub: hub, Send: delivered}
	time.Sleep(20 * time.Millisecond)
	mock.ExpectExec(`UPDATE message_receipts SET received_at`).WithArgs("mid", "user-x", "cid").WillReturnResult(sqlmock.NewResult(0, 1))
	msg := models.Message{ID: "mid", ChatID: "cid", SenderID: "sender", Text: "payload"}
	pm := models.PubSubMessage{Type: "new_message", Data: msg, TargetUser: "user-x", ChatID: "cid"}
	raw, _ := json.Marshal(pm)
	router.HandlePayloadForTest(string(raw))
	time.Sleep(50 * time.Millisecond)
	select {
	case <-delivered:
	default:
		t.Fatal("expected delivery via handleServerMessage")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}
