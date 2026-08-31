package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	wordMiningChannel     = "wordmining:jobs"
	wordMiningMaxAttempts = 5
	wordMiningBaseDelay   = 15 * time.Second
	wordMiningMaxDelay    = 5 * time.Minute
	wordMiningStaleTTL    = 5 * time.Minute
	wordMiningSweepEvery  = 30 * time.Second
	wordMiningPoolSize    = 2
)

// WordMiningNotifier is invoked after a mining job completes; the server wires it
// to push a websocket "word_mining_completed" (and "learning_dashboard_updated")
// event to the learner.
type WordMiningNotifier func(userID string, payload *WordMiningJobResult)

// WordMiningJobResult is the per-user notification/poll payload for a mining job.
type WordMiningJobResult struct {
	JobID     string `json:"jobId"`
	MessageID string `json:"messageId,omitempty"`
	ChatID    string `json:"chatId,omitempty"`
	Status    string `json:"status"` // pending, processing, done, failed
	Accepted  int    `json:"accepted,omitempty"`
	Error     string `json:"error,omitempty"`
}

// WordMiningQueueService is a durable, asynchronous vocabulary miner. Design
// mirrors GrammarQueueService (durable outbox + pub/sub trigger + retry sweeper
// + startup recovery) so mined words never silently disappear when a worker
// crashes or the process restarts mid-job.
type WordMiningQueueService struct {
	db     *sql.DB
	redis  *redis.Client
	mining *WordMiningService
	notify WordMiningNotifier

	ctx    context.Context
	cancel context.CancelFunc
	sema   chan struct{}
	wg     sync.WaitGroup
	stopCh chan struct{}
}

func NewWordMiningQueueService(db *sql.DB, redisClient *redis.Client, mining *WordMiningService, notify WordMiningNotifier) *WordMiningQueueService {
	ctx, cancel := context.WithCancel(context.Background())
	return &WordMiningQueueService{
		db:     db,
		redis:  redisClient,
		mining: mining,
		notify: notify,
		ctx:    ctx,
		cancel: cancel,
		sema:   make(chan struct{}, wordMiningPoolSize),
		stopCh: make(chan struct{}),
	}
}

func (q *WordMiningQueueService) Start() {
	q.recover()
	if q.redis != nil {
		q.wg.Add(1)
		go q.subscribeLoop()
	}
	q.wg.Add(1)
	go q.sweepLoop()
	log.Println("[WordMining] queue started (durable outbox + pub/sub + sweeper)")
}

func (q *WordMiningQueueService) Stop() {
	close(q.stopCh)
	q.cancel()
	q.wg.Wait()
}

// EnqueueForMessage inserts a durable mining job for a message and triggers
// processing. Duplicate jobs for the same (user, message) are skipped.
func (q *WordMiningQueueService) EnqueueForMessage(userID, chatID, messageID, sourceType, text, sourceLang, nativeLang string) (string, error) {
	if text == "" || len(text) > 750 {
		return "", fmt.Errorf("message too long or empty to mine")
	}
	var id string
	err := q.db.QueryRow(`
		INSERT INTO word_mining_jobs (
			user_id, chat_id, message_id, source_type, source_text,
			source_language, native_language
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id, message_id) DO NOTHING
		RETURNING id`, userID, nullStr(chatID), nullStr(messageID), sourceType, text, sourceLang, nativeLang).Scan(&id)
	if err == sql.ErrNoRows {
		return "", nil // already queued
	}
	if err != nil {
		return "", err
	}
	q.trigger(id)
	return id, nil
}

func (q *WordMiningQueueService) GetJob(jobID, userID string) (*WordMiningJobResult, error) {
	var messageID, chatID, status, lastErr, result string
	var accepted int
	err := q.db.QueryRow(`
		SELECT COALESCE(message_id::text,''), COALESCE(chat_id::text,''), status,
		       COALESCE(last_error,''), COALESCE(result,'')
		FROM word_mining_jobs WHERE id = $1 AND user_id = $2`, jobID, userID).
		Scan(&messageID, &chatID, &status, &lastErr, &result)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("mining job not found")
		}
		return nil, err
	}
	if result != "" {
		var r struct {
			Accepted int `json:"accepted"`
		}
		if json.Unmarshal([]byte(result), &r) == nil {
			accepted = r.Accepted
		}
	}
	return &WordMiningJobResult{
		JobID: jobID, MessageID: messageID, ChatID: chatID, Status: status, Accepted: accepted, Error: lastErr,
	}, nil
}

type wordMiningJob struct {
	ID             string
	UserID         string
	MessageID      string
	ChatID         string
	SourceType     string
	SourceText     string
	SourceLang     string
	NativeLang     string
	Attempts       int
}

func (q *WordMiningQueueService) claimJob(id string) (*wordMiningJob, bool) {
	j := &wordMiningJob{}
	err := q.db.QueryRow(`
		UPDATE word_mining_jobs
		SET status = 'processing', processing_at = CURRENT_TIMESTAMP, attempts = attempts + 1
		WHERE id = $1 AND status IN ('pending','failed') AND next_attempt_at <= CURRENT_TIMESTAMP
		RETURNING id, user_id, COALESCE(message_id::text,''), COALESCE(chat_id::text,''),
		          source_type, source_text, source_language, native_language, attempts`,
		id).Scan(&j.ID, &j.UserID, &j.MessageID, &j.ChatID, &j.SourceType, &j.SourceText, &j.SourceLang, &j.NativeLang, &j.Attempts)
	if err != nil {
		return nil, false
	}
	return j, true
}

func (q *WordMiningQueueService) processJob(id string) {
	j, ok := q.claimJob(id)
	if !ok {
		return
	}
	if q.notify != nil {
		q.notify(j.UserID, &WordMiningJobResult{JobID: j.ID, MessageID: j.MessageID, ChatID: j.ChatID, Status: "processing"})
	}

	accepted, err := q.mining.ProcessJob(q.ctx, j.ID)
	if err != nil {
		q.markFailed(j.ID, j.Attempts, err)
		q.notify(j.UserID, &WordMiningJobResult{JobID: j.ID, MessageID: j.MessageID, ChatID: j.ChatID, Status: "failed", Error: err.Error()})
		return
	}
	q.notify(j.UserID, &WordMiningJobResult{JobID: j.ID, MessageID: j.MessageID, ChatID: j.ChatID, Status: "done", Accepted: accepted})
}

func (q *WordMiningQueueService) markFailed(id string, attempts int, cause error) {
	status := "failed"
	delay := wordMiningBaseDelay
	for i := 1; i < attempts && delay < wordMiningMaxDelay; i++ {
		delay *= 2
	}
	if delay > wordMiningMaxDelay {
		delay = wordMiningMaxDelay
	}
	if attempts >= wordMiningMaxAttempts {
		delay = 24 * time.Hour
	}
	_, _ = q.db.Exec(`UPDATE word_mining_jobs
		SET status = $1, last_error = $2, processing_at = NULL,
		    next_attempt_at = CURRENT_TIMESTAMP + ($3 * INTERVAL '1 second')
		WHERE id = $4`, status, cause.Error(), int(delay.Seconds()), id)
}

func (q *WordMiningQueueService) trigger(id string) {
	if q.redis != nil {
		payload, _ := json.Marshal(map[string]string{"jobId": id})
		if err := q.redis.Publish(q.ctx, wordMiningChannel, payload).Err(); err != nil {
			log.Printf("[WordMining] publish %s: %v (sweeper will re-queue)", id, err)
		}
		return
	}
	q.spawn(id)
}

func (q *WordMiningQueueService) spawn(id string) {
	q.sema <- struct{}{}
	q.wg.Add(1)
	go func() {
		defer q.wg.Done()
		defer func() { <-q.sema }()
		q.processJob(id)
	}()
}

func (q *WordMiningQueueService) subscribeLoop() {
	defer q.wg.Done()
	sub := q.redis.Subscribe(q.ctx, wordMiningChannel)
	defer sub.Close()
	ch := sub.Channel()
	for {
		select {
		case <-q.ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			var envelope struct {
				JobID string `json:"jobId"`
			}
			if err := json.Unmarshal([]byte(msg.Payload), &envelope); err != nil || envelope.JobID == "" {
				continue
			}
			q.spawn(envelope.JobID)
		}
	}
}

func (q *WordMiningQueueService) sweepLoop() {
	defer q.wg.Done()
	ticker := time.NewTicker(wordMiningSweepEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			q.sweep()
		case <-q.stopCh:
			return
		}
	}
}

func (q *WordMiningQueueService) sweep() {
	if _, err := q.db.Exec(`UPDATE word_mining_jobs SET status = 'pending', processing_at = NULL
		WHERE status = 'processing' AND processing_at < CURRENT_TIMESTAMP - ($1 * INTERVAL '1 minute')`,
		int(wordMiningStaleTTL.Minutes())); err != nil {
		log.Printf("[WordMining] reclaim stale processing: %v", err)
	}
	rows, err := q.db.Query(`
		SELECT id FROM word_mining_jobs
		WHERE status IN ('pending','failed') AND next_attempt_at <= CURRENT_TIMESTAMP
		ORDER BY created_at LIMIT 50`)
	if err != nil {
		log.Printf("[WordMining] sweep due: %v", err)
		return
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()
	for _, id := range ids {
		q.trigger(id)
	}
}

func (q *WordMiningQueueService) recover() {
	if _, err := q.db.Exec(`UPDATE word_mining_jobs SET status = 'pending', processing_at = NULL
		WHERE status = 'processing'`); err != nil {
		log.Printf("[WordMining] recover reset processing: %v", err)
	}
	rows, err := q.db.Query(`SELECT id FROM word_mining_jobs WHERE status IN ('pending','failed') ORDER BY created_at`)
	if err != nil {
		log.Printf("[WordMining] recover load incomplete: %v", err)
		return
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()
	log.Printf("[WordMining] recover: %d job(s) with incomplete mining re-queued", len(ids))
	for _, id := range ids {
		q.trigger(id)
	}
}
