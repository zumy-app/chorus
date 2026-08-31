package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/chorus/messenger/internal/models"
	"github.com/redis/go-redis/v9"
)

const (
	grammarChannel     = "grammar:jobs"
	grammarMaxAttempts = 5
	grammarBaseDelay   = 20 * time.Second
	grammarMaxDelay    = 10 * time.Minute
	grammarMaxPending  = 24 * time.Hour // failed jobs wait before admin manual retry
	grammarStaleTTL    = 5 * time.Minute
	grammarSweepEvery  = 30 * time.Second
	grammarPoolSize    = 3
)

// GrammarPromptVersion mirrors grammar.go's aiAnalysisCacheVersion and is
// persisted as grammar_jobs.prompt_version so evals (FR-30) can be attributed
// to the exact prompt that produced the analysis.
const GrammarPromptVersion = "v2"

// GrammarAnalyzer is the subset of GrammarService used by the queue. GrammarService
// satisfies it, and tests inject a fake.
type GrammarAnalyzer interface {
	GenerateAIAnalysis(text, language, nativeLanguage string) (*models.AIGrammarAnalysis, string, error)
}

// GrammarNotifier is invoked after a grammar job completes or fails; the server
// wires it to send a "grammar_analysis" event to the requesting user over the
// WebSocket hub and Redis pub/sub.
type GrammarNotifier func(userID string, payload *GrammarJobResult)

// GrammarJobResult is the per-user notification/poll payload for a grammar job.
type GrammarJobResult struct {
	JobID        string                    `json:"jobId"`
	MessageID    string                    `json:"messageId,omitempty"`
	ChatID       string                    `json:"chatId,omitempty"`
	Status       string                    `json:"status"` // pending, processing, done, failed
	Analysis     *models.AIGrammarAnalysis `json:"analysis,omitempty"`
	ProviderUsed string                    `json:"providerUsed,omitempty"`
	Error        string                    `json:"error,omitempty"`
}

// GrammarQueueService provides guaranteed, asynchronous AI grammar analysis.
//
// Design mirrors TranslationQueueService (durable outbox + pub/sub trigger):
//   - EnqueueForAnalysis inserts a durable row per request into grammar_jobs.
//   - The row is published to a Redis channel; a worker analyzes it using the
//     existing provider chain and pushes the result to the requesting user.
//   - A periodic sweeper reclaims stale processing rows and re-publishes due
//     pending/failed jobs; in lean mode (no Redis) it processes inline.
//   - Start() recovers incomplete work after a restart.
//
// Because analysis is user-initiated and private, results are fanned out to the
// requesting user's connections only, never to the whole chat.
type GrammarQueueService struct {
	db        *sql.DB
	redis     *redis.Client
	analyzer  GrammarAnalyzer
	loadCache func(text, language, nativeLanguage string) (*models.AIGrammarAnalysis, string, bool)
	notify    GrammarNotifier

	ctx    context.Context
	cancel context.CancelFunc
	sema   chan struct{}
	wg     sync.WaitGroup
	stopCh chan struct{}

	onDone func(jobID string)
}

// NewGrammarQueueService creates a grammar job queue.
// loadCache (optional) returns a cached analysis when one exists.
func NewGrammarQueueService(
	db *sql.DB,
	redisClient *redis.Client,
	analyzer GrammarAnalyzer,
	loadCache func(text, language, nativeLanguage string) (*models.AIGrammarAnalysis, string, bool),
	notify GrammarNotifier,
) *GrammarQueueService {
	ctx, cancel := context.WithCancel(context.Background())
	return &GrammarQueueService{
		db:        db,
		redis:     redisClient,
		analyzer:  analyzer,
		loadCache: loadCache,
		notify:    notify,
		ctx:       ctx,
		cancel:    cancel,
		sema:      make(chan struct{}, grammarPoolSize),
		stopCh:    make(chan struct{}),
	}
}

// Start launches the pub/sub consumer and the retry sweeper.
func (q *GrammarQueueService) Start() {
	q.recover()
	if q.redis != nil {
		q.wg.Add(1)
		go q.subscribeLoop()
	}
	q.wg.Add(1)
	go q.sweepLoop()
	log.Println("[Grammar] queue started (durable outbox + pub/sub + sweeper)")
}

// Stop halts the sweeper and consumer. In-flight jobs are allowed to finish;
// durable rows ensure nothing is lost across restarts.
func (q *GrammarQueueService) Stop() {
	close(q.stopCh)
	q.cancel()
	q.wg.Wait()
}

// SetOnDone wires a callback fired after a grammar job completes. Used by the
// quality pipeline (FR-30) to enqueue a cross-model evaluation.
func (q *GrammarQueueService) SetOnDone(fn func(jobID string)) {
	q.onDone = fn
}

// EnqueueForAnalysis inserts a job and triggers near-real-time processing.
// Returns the job id. Duplicate prevention is handled client-side (the bubble
// won't submit again while a job for that message is in flight) and by the
// Redis cache hot-path in processJob, so identical requests stay cheap.
func (q *GrammarQueueService) EnqueueForAnalysis(userID, chatID, messageID, text, language, nativeLanguage string) (string, error) {
	lang := strings.TrimSpace(language)
	if lang == "" {
		lang = "en"
	}
	native := strings.TrimSpace(nativeLanguage)
	if native == "" {
		native = "en"
	}

	var id string
	err := q.db.QueryRow(`
		INSERT INTO grammar_jobs (user_id, chat_id, message_id, text, language, native_language)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`, userID, nullStr(chatID), nullStr(messageID), text, lang, native).Scan(&id)
	if err != nil {
		log.Printf("[Grammar] enqueue for user %s: %v", userID, err)
		return "", err
	}

	log.Printf("[Grammar] enqueued job %s for user %s", id, userID)
	q.trigger(id)
	return id, nil
}

// GetJob returns a job owned by the requesting user (privacy: cross-user reads
// are forbidden). Results include the analysis so clients can resync after a
// dropped WebSocket or page reload.
func (q *GrammarQueueService) GetJob(jobID, userID string) (*GrammarJobResult, error) {
	var (
		messageID, chatID, status, providerUsed, lastErr, resultJSON string
		attempts                                                     int
	)
	err := q.db.QueryRow(`
		SELECT COALESCE(message_id::text,''), COALESCE(chat_id::text,''), status,
		       COALESCE(provider_used,''), COALESCE(result::text,''), COALESCE(last_error,''), attempts
		FROM grammar_jobs WHERE id = $1 AND user_id = $2`, jobID, userID).
		Scan(&messageID, &chatID, &status, &providerUsed, &resultJSON, &lastErr, &attempts)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("grammar job not found")
		}
		return nil, err
	}

	res := &GrammarJobResult{
		JobID:        jobID,
		MessageID:    messageID,
		ChatID:       chatID,
		Status:       status,
		ProviderUsed: providerUsed,
		Error:        lastErr,
	}
	if status == "done" && resultJSON != "" {
		var analysis models.AIGrammarAnalysis
		if err := json.Unmarshal([]byte(resultJSON), &analysis); err == nil {
			res.Analysis = &analysis
		}
	}
	return res, nil
}

// retryDueForces re-publishes a specific job (admin reuse / sweeper).
func (q *GrammarQueueService) retryJob(id string) error {
	var messageID string
	err := q.db.QueryRow(`UPDATE grammar_jobs
		SET status = 'pending', processing_at = NULL, last_error = NULL, next_attempt_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND status <> 'done'
		RETURNING id`, id).Scan(&messageID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("grammar job not found or already completed")
		}
		return err
	}
	q.trigger(id)
	return nil
}

// claimJob atomically reserves a job for analysis.
type grammarQueuedJob struct {
	ID             string
	UserID         string
	MessageID      string
	ChatID         string
	Text           string
	Language       string
	NativeLanguage string
	Attempts       int
}

func (q *GrammarQueueService) claimJob(id string) (*grammarQueuedJob, bool) {
	j := &grammarQueuedJob{}
	err := q.db.QueryRow(`
		UPDATE grammar_jobs
		SET status = 'processing', processing_at = CURRENT_TIMESTAMP, attempts = attempts + 1
		WHERE id = $1 AND status IN ('pending', 'failed') AND next_attempt_at <= CURRENT_TIMESTAMP
		RETURNING id, user_id, COALESCE(message_id::text,''), COALESCE(chat_id::text,''), text, language, native_language, attempts`,
		id).Scan(&j.ID, &j.UserID, &j.MessageID, &j.ChatID, &j.Text, &j.Language, &j.NativeLanguage, &j.Attempts)
	if err != nil {
		return nil, false
	}
	return j, true
}

func (q *GrammarQueueService) processJob(id string) {
	j, ok := q.claimJob(id)
	if !ok {
		return
	}

	// Inform the client that a worker has picked the job up.
	q.notifyStatus(j.UserID, &GrammarJobResult{
		JobID:     j.ID,
		MessageID: j.MessageID,
		ChatID:    j.ChatID,
		Status:    "processing",
	})

	// Cheap hot-path: reuse the existing Redis cache before attempting a model.
	if q.loadCache != nil {
		if analysis, provider, found := q.loadCache(j.Text, j.Language, j.NativeLanguage); found {
			q.finishJob(j, analysis, provider, GrammarPromptVersion, 0, "")
			return
		}
	}

	start := time.Now()
	analysis, provider, err := q.analyzer.GenerateAIAnalysis(j.Text, j.Language, j.NativeLanguage)
	if err != nil {
		log.Printf("[Grammar] job %s: %v", j.ID, err)
		q.markFailed(j.ID, j.Attempts, err)
		return
	}
	latencyMS := int(time.Since(start).Milliseconds())
	log.Printf("[Grammar] job %s done in %dms via %q", j.ID, latencyMS, provider)
	q.finishJob(j, analysis, provider, GrammarPromptVersion, latencyMS, "")
	if q.onDone != nil {
		q.onDone(j.ID)
	}
}

func (q *GrammarQueueService) finishJob(j *grammarQueuedJob, analysis *models.AIGrammarAnalysis, provider, promptVersion string, latencyMS int, errMsg string) {
	if errMsg != "" {
		q.markFailed(j.ID, j.Attempts, fmt.Errorf("%s", errMsg))
		return
	}
	resultJSON, _ := json.Marshal(analysis)
	if _, err := q.db.Exec(`
		UPDATE grammar_jobs
		SET status = 'done', result = $1, provider_used = $2, last_error = NULL,
		    processing_at = NULL, completed_at = CURRENT_TIMESTAMP,
		    provider_used_lineage = $2, prompt_version = $3, latency_ms = $4
		WHERE id = $5`, string(resultJSON), provider, promptVersion, latencyMS, j.ID); err != nil {
		log.Printf("[Grammar] mark done %s: %v", j.ID, err)
		return
	}
	q.notifyStatus(j.UserID, &GrammarJobResult{
		JobID:        j.ID,
		MessageID:    j.MessageID,
		ChatID:       j.ChatID,
		Status:       "done",
		Analysis:     analysis,
		ProviderUsed: provider,
	})
}

func (q *GrammarQueueService) markFailed(id string, attempts int, cause error) {
	status := "failed"
	delay := grammarMaxPending
	if attempts < grammarMaxAttempts {
		status = "failed" // still retryable
		delay = grammarBaseDelay
		for i := 1; i < attempts; i++ {
			if delay >= grammarMaxDelay {
				delay = grammarMaxDelay
				break
			}
			delay *= 2
		}
	}
	if _, err := q.db.Exec(`UPDATE grammar_jobs
		SET status = $1, last_error = $2, processing_at = NULL,
		    next_attempt_at = CURRENT_TIMESTAMP + ($3 * INTERVAL '1 second')
		WHERE id = $4`, status, cause.Error(), int(delay.Seconds()), id); err != nil {
		log.Printf("[Grammar] mark failed %s: %v", id, err)
	}

	// Notify the requesting user so the UI can stop spinning and fall back.
	// Load remaining fields for the payload.
	var userID string
	_ = q.db.QueryRow(`SELECT user_id FROM grammar_jobs WHERE id = $1`, id).Scan(&userID)
	if userID != "" {
		q.notifyStatus(userID, &GrammarJobResult{
			JobID:  id,
			Status: "failed",
			Error:  cause.Error(),
		})
	}
}

func (q *GrammarQueueService) notifyStatus(userID string, payload *GrammarJobResult) {
	if payload == nil || payload.Status == "" {
		return
	}
	if q.notify != nil {
		q.notify(userID, payload)
	}
}

// trigger publishes a job id to the consumer channel (or spawns inline in lean mode).
func (q *GrammarQueueService) trigger(id string) {
	if q.redis != nil {
		payload, _ := json.Marshal(map[string]string{"jobId": id})
		if err := q.redis.Publish(q.ctx, grammarChannel, payload).Err(); err != nil {
			log.Printf("[Grammar] publish %s: %v (sweeper will re-queue)", id, err)
		}
		return
	}
	q.spawn(id)
}

// spawn processes a job on a bounded worker pool.
func (q *GrammarQueueService) spawn(id string) {
	q.sema <- struct{}{}
	q.wg.Add(1)
	go func() {
		defer q.wg.Done()
		defer func() { <-q.sema }()
		q.processJob(id)
	}()
}

// subscribeLoop consumes published job triggers.
func (q *GrammarQueueService) subscribeLoop() {
	defer q.wg.Done()
	sub := q.redis.Subscribe(q.ctx, grammarChannel)
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

// sweepLoop is the safety net: reclaim stale leases and re-publish due jobs.
func (q *GrammarQueueService) sweepLoop() {
	defer q.wg.Done()
	ticker := time.NewTicker(grammarSweepEvery)
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

func (q *GrammarQueueService) sweep() {
	if _, err := q.db.Exec(`UPDATE grammar_jobs SET status = 'pending', processing_at = NULL
		WHERE status = 'processing' AND processing_at < CURRENT_TIMESTAMP - ($1 * INTERVAL '1 minute')`,
		int(grammarStaleTTL.Minutes())); err != nil {
		log.Printf("[Grammar] reclaim stale processing: %v", err)
	}
	rows, err := q.db.Query(`
		SELECT id FROM grammar_jobs
		WHERE status IN ('pending', 'failed') AND next_attempt_at <= CURRENT_TIMESTAMP
		ORDER BY created_at
		LIMIT 50`)
	if err != nil {
		log.Printf("[Grammar] sweep due: %v", err)
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

// recover re-queues incomplete work after a restart.
func (q *GrammarQueueService) recover() {
	if _, err := q.db.Exec(`UPDATE grammar_jobs SET status = 'pending', processing_at = NULL
		WHERE status = 'processing'`); err != nil {
		log.Printf("[Grammar] recover reset processing: %v", err)
	}
	rows, err := q.db.Query(`
		SELECT id FROM grammar_jobs
		WHERE status IN ('pending', 'failed')
		ORDER BY created_at`)
	if err != nil {
		log.Printf("[Grammar] recover load incomplete: %v", err)
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
	log.Printf("[Grammar] recover: %d job(s) with incomplete analysis re-queued", len(ids))
	for _, id := range ids {
		q.trigger(id)
	}
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
