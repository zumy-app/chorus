package services

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/chorus/messenger/internal/models"
)

func p95(durs []time.Duration) time.Duration {
	if len(durs) == 0 {
		return 0
	}
	sort.Slice(durs, func(i, j int) bool { return durs[i] < durs[j] })
	idx := int(float64(len(durs)) * 0.95)
	if idx >= len(durs) {
		idx = len(durs) - 1
	}
	return durs[idx]
}

func TestPerf_NFR1_TranslationCacheHit(t *testing.T) {
	rdb, _ := newTestRedis(t)
	provider := &fakeProvider{}
	svc := NewTranslationService(provider, rdb, 0)
	warm := "hello perf"
	_, _ = svc.TranslateQuick(warm, "es", "en")
	provider.calls = 0
	samples := 200
	durs := make([]time.Duration, 0, samples)
	for i := 0; i < samples; i++ {
		start := time.Now()
		res, err := svc.TranslateQuickResult(warm, "es", "en")
		if err != nil || res == nil || !res.CacheHit {
			t.Fatalf("expected cache hit, err=%v res=%+v", err, res)
		}
		durs = append(durs, time.Since(start))
	}
	got := p95(durs)
	t.Logf("NFR-1 cache-hit p95=%v samples=%d budget=500ms", got, samples)
	if got > 500*time.Millisecond {
		t.Fatalf("NFR-1 FAIL p95 %v > 500ms", got)
	}
	if provider.calls != 0 {
		t.Fatalf("cache should prevent LLM, provider calls=%d", provider.calls)
	}
}

func TestPerf_NFR1_TranslationLearnedWordSkip(t *testing.T) {
	provider := &fakeProvider{}
	svc := NewTranslationService(provider, nil, 0)
	svc.SetKnownWordsResolver(knownResolver("hola", "mundo"))
	samples := 200
	durs := make([]time.Duration, 0, samples)
	for i := 0; i < samples; i++ {
		start := time.Now()
		res, _ := svc.TranslateWithLearnedFilter("Hola mundo", "es", "en", "u-perf")
		_ = res
		durs = append(durs, time.Since(start))
	}
	got := p95(durs)
	t.Logf("FR-26 learned-word skip p95=%v budget=50ms", got)
	if got > 50*time.Millisecond {
		t.Fatalf("learned-word skip p95 %v > 50ms", got)
	}
	if provider.calls != 0 {
		t.Fatalf("expected 0 LLM calls for all-known, got %d", provider.calls)
	}
}

func TestPerf_NFR2_MessagePersistBeforeAck(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()
	svc := NewMessageService(db, nil)
	samples := 200
	durs := make([]time.Duration, 0, samples)
	for i := 0; i < samples; i++ {
		mock.ExpectExec(`INSERT INTO message_receipts`).WithArgs("m-perf", "u2", "c1").WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec(`INSERT INTO message_receipts`).WithArgs("m-perf", "u3", "c1").WillReturnResult(sqlmock.NewResult(1, 1))
		msg := &models.Message{ID: "m-perf", ChatID: "c1", SenderID: "u1", Text: "hello"}
		start := time.Now()
		_ = svc.InitializeReceipts(context.Background(), msg, []string{"u1", "u2", "u3"})
		durs = append(durs, time.Since(start))
	}
	got := p95(durs)
	t.Logf("NFR-2 persist-before-ack p95=%v budget=50ms", got)
	if got > 50*time.Millisecond {
		t.Fatalf("NFR-2 persist p95 %v > 50ms", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("mock: %v", err)
	}
}

func TestPerf_NFR2_WebSocketFanout(t *testing.T) {
	rdb, _ := newTestRedis(t)
	hub := NewWebSocketHub(rdb, "perf-hub")
	go hub.Run()
	const clients = 100
	for i := 0; i < clients; i++ {
		ch := make(chan []byte, 512)
		c := &Client{ID: "perf-c-" + time.Now().Format("150405.000") + string(rune('a'+i%26)), UserID: "u-fanout", Hub: hub, Send: ch}
		hub.Register <- c
	}
	time.Sleep(50 * time.Millisecond)
	samples := 200
	durs := make([]time.Duration, 0, samples)
	for i := 0; i < samples; i++ {
		start := time.Now()
		hub.SendToUser("u-fanout", "new_message", map[string]string{"seq": "x"})
		durs = append(durs, time.Since(start))
	}
	got := p95(durs)
	t.Logf("NFR-2 WS fanout 100 clients p95=%v budget=10ms", got)
	if got > 10*time.Millisecond {
		t.Fatalf("WS fanout p95 %v > 10ms", got)
	}
}

func TestPerf_NFR3_HistoryPagination(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()
	svc := NewMessageService(db, nil)
	rows := sqlmock.NewRows([]string{"id", "chat_id", "sender_id", "text", "original_language", "translations", "delivery_status", "reply_to_id", "created_at", "deleted_at", "forwarded_from_message_id", "forwarded_from_chat_id", "forwarded_from_sender_id"}).
		AddRow("m1", "c1", "u1", "hello", "", []byte(`{}`), "sent", nil, time.Now(), nil, nil, nil, nil)
	samples := 200
	durs := make([]time.Duration, 0, samples)
	for i := 0; i < samples; i++ {
		mock.ExpectQuery(`SELECT id, chat_id.*FROM messages`).WithArgs("c1", 50).WillReturnRows(rows)
		start := time.Now()
		_, _ = svc.GetMessages("c1", 50, nil)
		durs = append(durs, time.Since(start))
	}
	got := p95(durs)
	t.Logf("NFR-3 history pagination p95=%v budget=50ms", got)
	if got > 50*time.Millisecond {
		t.Fatalf("NFR-3 pagination p95 %v > 50ms", got)
	}
}

func TestPerf_SearchLatency(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()
	svc := NewMessageService(db, nil)
	rows := sqlmock.NewRows([]string{"id", "chat_id", "sender_id", "text", "original_language", "translations", "delivery_status", "reply_to_id", "created_at", "deleted_at", "forwarded_from_message_id", "forwarded_from_chat_id", "forwarded_from_sender_id"})
	samples := 100
	durs := make([]time.Duration, 0, samples)
	for i := 0; i < samples; i++ {
		mock.ExpectQuery(`SELECT id, chat_id`).WithArgs("hello", 20).WillReturnRows(rows)
		start := time.Now()
		_, _ = svc.Search("hello", nil, 20)
		durs = append(durs, time.Since(start))
	}
	got := p95(durs)
	t.Logf("search p95=%v budget=50ms", got)
	if got > 50*time.Millisecond {
		t.Fatalf("search p95 %v > 50ms", got)
	}
}

func BenchmarkTranslationCacheHit(b *testing.B) {
	rdb, _ := newTestRedis(&testing.T{})
	provider := &fakeProvider{}
	svc := NewTranslationService(provider, rdb, 0)
	ctx := context.Background()
	rdb.Set(ctx, "translation:v1:hello:es", "hola", 0)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.TranslateWithLearnedFilter("hello", "es", "en", "")
	}
}

func BenchmarkWSFanout(b *testing.B) {
	rdb, _ := newTestRedis(&testing.T{})
	hub := NewWebSocketHub(rdb, "bench-hub")
	go hub.Run()
	for i := 0; i < 50; i++ {
		ch := make(chan []byte, 512)
		hub.Register <- &Client{ID: "bench-c", UserID: "u-bench", Hub: hub, Send: ch}
	}
	time.Sleep(20 * time.Millisecond)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hub.SendToUser("u-bench", "new_message", map[string]string{"x": "y"})
	}
}
