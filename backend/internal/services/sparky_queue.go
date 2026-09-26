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

	"github.com/redis/go-redis/v9"
)

const (
	sparkyChannel     = "sparky:jobs"
	sparkyMaxAttempts = 5
	sparkyBaseDelay   = 20 * time.Second
	sparkyMaxDelay    = 10 * time.Minute
	sparkyMaxPending  = 24 * time.Hour // failed jobs wait before admin manual retry
	sparkyStaleTTL    = 5 * time.Minute
	sparkySweepEvery  = 30 * time.Second
	sparkyPoolSize    = 3
	// sparkyJobTimeout bounds a single worker attempt. The non-stream path is
	// already bounded by per-endpoint client timeouts; the stream path uses a
	// timeout-less client (streams stay open while tokens flow), so without
	// this a stalled upstream could hold a pool worker forever.
	sparkyJobTimeout = 90 * time.Second
)

// SparkyPromptVersion attributes async Sparky answers to the prompt that
// produced them (mirrors GrammarPromptVersion for grammar jobs).
const SparkyPromptVersion = "v1"

// SparkyAnswerer is the subset of GrammarService used by the queue. It is a
// thin wrapper over the "custom" learning-content path so Sparky and the web
// AI Tutor share the exact same prompt and fallback behavior.
type SparkyAnswerer interface {
	GenerateSparkyAnswer(contextText, language, nativeLanguage, query string) (string, string, error)
}

// SparkyAnswerStreamer is optionally implemented by the answerer (Phase 2).
// When present, the worker streams answer deltas via onToken as the provider
// produces them; each delta fans out as a "streaming" status event before the
// final "done". Answerers that don't implement it use the one-shot path.
type SparkyAnswerStreamer interface {
	GenerateSparkyAnswerStream(ctx context.Context, contextText, language, nativeLanguage, query string, onToken func(delta string)) (fullContent, provider string, err error)
}

// SparkyNotifier is invoked after a job changes state; the server wires it to
// send events to the requesting user over the WebSocket hub and Redis pub/sub.
type SparkyNotifier func(userID string, payload *SparkyJobResult)

// SparkyJobResult is the per-user notification/poll payload for a Sparky job.
type SparkyJobResult struct {
	JobID        string `json:"jobId"`
	ChatID       string `json:"chatId,omitempty"`
	Status       string `json:"status"` // pending, processing, streaming, done, failed
	Content      string `json:"content,omitempty"`
	ProviderUsed string `json:"providerUsed,omitempty"`
	Error        string `json:"error,omitempty"`
}

// SparkyQueueService provides guaranteed, asynchronous Sparky answers.
//
// Design mirrors GrammarQueueService (durable outbox + pub/sub trigger):
//   - EnqueueForAsk inserts a durable row per request into sparky_jobs and
//     returns the id immediately (the HTTP handler answers 202).
//   - The row is published to a Redis channel; a worker answers it using the
//     provider chain and pushes the result to the requesting user only.
//   - A periodic sweeper reclaims stale processing rows and re-publishes due
//     pending/failed jobs; in lean mode (no Redis) it processes inline.
//   - Start() recovers incomplete work after a restart.
//
// Phase 2 adds token streaming on top: the worker detects a SparkyStreamer
// answerer and forwards deltas as "streaming" status events before the final
// "done". Phase 3 adds a Redis answer cache via loadCache/storeCache.
type SparkyQueueService struct {
	db        *sql.DB
	redis     *redis.Client
	answerer  SparkyAnswerer
	loadCache func(contextText, language, nativeLanguage, query string) (content, provider string, found bool)
	storeCache func(contextText, language, nativeLanguage, query, content, provider string)
	notify    SparkyNotifier

	ctx    context.Context
	cancel context.CancelFunc
	sema   chan struct{}
	wg     sync.WaitGroup
	stopCh chan struct{}

	onDone func(jobID string)
}

// NewSparkyQueueService creates a Sparky job queue. loadCache/storeCache are
// optional (Phase 3); pass nil until the Redis answer cache lands.
func NewSparkyQueueService(
	db *sql.DB,
	redisClient *redis.Client,
	answerer SparkyAnswerer,
	loadCache func(contextText, language, nativeLanguage, query string) (string, string, bool),
	storeCache func(contextText, language, nativeLanguage, query, content, provider string),
	notify SparkyNotifier,
) *SparkyQueueService {
	ctx, cancel := context.WithCancel(context.Background())
	return &SparkyQueueService{
		db:         db,
		redis:      redisClient,
		answerer:   answerer,
		loadCache:  loadCache,
		storeCache: storeCache,
		notify:     notify,
		ctx:        ctx,
		cancel:     cancel,
		sema:       make(chan struct{}, sparkyPoolSize),
		stopCh:     make(chan struct{}),
	}
}

// Start recovers incomplete work and launches the subscriber + sweeper.
func (q *SparkyQueueService) Start() {
	q.recover()
	if q.redis != nil {
		q.wg.Add(1)
		go q.subscribeLoop()
	}
	q.wg.Add(1)
	go q.sweepLoop()
}

// Stop shuts down gracefully.
func (q *SparkyQueueService) Stop() {
	close(q.stopCh)
	q.cancel()
	q.wg.Wait()
}

// SetOnDone registers a completion hook (evals, metrics).
func (q *SparkyQueueService) SetOnDone(fn func(jobID string)) {
	q.onDone = fn
}

// EnqueueForAsk persists a Sparky question and triggers async processing.
// Returns the job id immediately; the answer arrives via notify + GetJob.
func (q *SparkyQueueService) EnqueueForAsk(userID, chatID, contextText, language, nativeLanguage, query string) (string, error) {
	lang := trimOrDefault(language, "en")
	native := trimOrDefault(nativeLanguage, "en")

	var id string
	err := q.db.QueryRow(`
		INSERT INTO sparky_jobs (user_id, chat_id, context, query, language, native_language)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`, userID, nullStr(chatID), contextText, query, lang, native).Scan(&id)
	if err != nil {
		log.Printf("[Sparky] enqueue for user %s: %v", userID, err)
		return "", err
	}

	log.Printf("[Sparky] enqueued job %s for user %s", id, userID)
	q.trigger(id)
	return id, nil
}

// GetJob returns a job owned by the requesting user (privacy: cross-user reads
// are forbidden). Results include the answer so clients can resync after a
// dropped WebSocket or app backgrounding.
func (q *SparkyQueueService) GetJob(jobID, userID string) (*SparkyJobResult, error) {
	var (
		chatID, status, providerUsed, lastErr, result string
		attempts                                     int
	)
	err := q.db.QueryRow(`
		SELECT COALESCE(chat_id::text,''), status,
		       COALESCE(provider_used,''), COALESCE(last_error,''), COALESCE(result,'') ,
		       attempts
		FROM sparky_jobs WHERE id = $1 AND user_id = $2`, jobID, userID).
		Scan(&chatID, &status, &providerUsed, &lastErr, &result, &attempts)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("sparky job not found")
		}
		return nil, err
	}

	return &SparkyJobResult{
		JobID:        jobID,
		ChatID:       chatID,
		Status:       status,
		Content:      result,
		ProviderUsed: providerUsed,
		Error:        lastErr,
	}, nil
}

// retryDueForces re-publishes a specific job (admin reuse / sweeper).
func (q *SparkyQueueService) retryJob(id string) error {
	var messageID string
	err := q.db.QueryRow(`UPDATE sparky_jobs
		SET status = 'pending', processing_at = NULL, last_error = NULL, next_attempt_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND status <> 'done'
		RETURNING id`, id).Scan(&messageID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("sparky job not found or already completed")
		}
		return err
	}
	q.trigger(id)
	return nil
}

// claimJob atomically reserves a job for answering.
type sparkyQueuedJob struct {
	ID             string
	UserID         string
	ChatID         string
	Context        string
	Query          string
	Language       string
	NativeLanguage string
	Attempts       int
}

func (q *SparkyQueueService) claimJob(id string) (*sparkyQueuedJob, bool) {
	j := &sparkyQueuedJob{}
	err := q.db.QueryRow(`
		UPDATE sparky_jobs
		SET status = 'processing', processing_at = CURRENT_TIMESTAMP, attempts = attempts + 1
		WHERE id = $1 AND status IN ('pending', 'failed') AND next_attempt_at <= CURRENT_TIMESTAMP
		RETURNING id, user_id, COALESCE(chat_id::text,''), context, query, language, native_language, attempts`,
		id).Scan(&j.ID, &j.UserID, &j.ChatID, &j.Context, &j.Query, &j.Language, &j.NativeLanguage, &j.Attempts)
	if err != nil {
		return nil, false
	}
	return j, true
}

func (q *SparkyQueueService) processJob(id string) {
	j, ok := q.claimJob(id)
	if !ok {
		return
	}

	// Inform the client that a worker has picked the job up.
	q.notifyStatus(j.UserID, &SparkyJobResult{
		JobID:  j.ID,
		ChatID: j.ChatID,
		Status: "processing",
	})

	// Cheap hot-path: reuse a cached answer when one exists (main.go wires the
	// Redis cache; nil in lean mode).
	if q.loadCache != nil {
		if content, provider, found := q.loadCache(j.Context, j.Language, j.NativeLanguage, j.Query); found {
			q.finishJob(j, content, provider, SparkyPromptVersion, 0, "")
			return
		}
	}

	// Streaming path: if the answerer implements SparkyAnswerStreamer, deltas
	// fan out as "streaming" events. The attempt runs under a per-job deadline
	// so a stalled stream can't hold a pool worker (the stream client itself
	// has no overall timeout by design — streams stay open while tokens flow).
	if streamer, ok := q.answerer.(SparkyAnswerStreamer); ok && streamer != nil {
		ctx, cancel := context.WithTimeout(q.ctx, sparkyJobTimeout)
		defer cancel()
		start := time.Now()
		content, provider, err := streamer.GenerateSparkyAnswerStream(ctx, j.Context, j.Language, j.NativeLanguage, j.Query, func(delta string) {
			q.notifyStatus(j.UserID, &SparkyJobResult{
				JobID:   j.ID,
				ChatID:  j.ChatID,
				Status:  "streaming",
				Content: delta,
			})
		})
		if err != nil {
			log.Printf("[Sparky] job %s (stream): %v", j.ID, err)
			q.markFailed(j.ID, j.Attempts, err)
			return
		}
		latencyMS := int(time.Since(start).Milliseconds())
		log.Printf("[Sparky] job %s done in %dms via %q (streamed)", j.ID, latencyMS, provider)
		q.finishJob(j, content, provider, SparkyPromptVersion, latencyMS, "")
		if q.onDone != nil {
			q.onDone(j.ID)
		}
		return
	}

	start := time.Now()
	content, provider, err := q.answerer.GenerateSparkyAnswer(j.Context, j.Language, j.NativeLanguage, j.Query)
	if err != nil {
		log.Printf("[Sparky] job %s: %v", j.ID, err)
		q.markFailed(j.ID, j.Attempts, err)
		return
	}
	latencyMS := int(time.Since(start).Milliseconds())
	log.Printf("[Sparky] job %s done in %dms via %q", j.ID, latencyMS, provider)
	q.finishJob(j, content, provider, SparkyPromptVersion, latencyMS, "")
	if q.onDone != nil {
		q.onDone(j.ID)
	}
}

func (q *SparkyQueueService) finishJob(j *sparkyQueuedJob, content, provider, promptVersion string, latencyMS int, errMsg string) {
	if errMsg != "" {
		q.markFailed(j.ID, j.Attempts, fmt.Errorf("%s", errMsg))
		return
	}
	if _, err := q.db.Exec(`
		UPDATE sparky_jobs
		SET status = 'done', result = $1, provider_used = $2, last_error = NULL,
		    processing_at = NULL, completed_at = CURRENT_TIMESTAMP
		WHERE id = $3`, content, provider, j.ID); err != nil {
		log.Printf("[Sparky] mark done %s: %v", j.ID, err)
		return
	}
	if q.storeCache != nil {
		q.storeCache(j.Context, j.Language, j.NativeLanguage, j.Query, content, provider)
	}
	q.notifyStatus(j.UserID, &SparkyJobResult{
		JobID:        j.ID,
		ChatID:       j.ChatID,
		Status:       "done",
		Content:      content,
		ProviderUsed: provider,
	})
}

func (q *SparkyQueueService) markFailed(id string, attempts int, cause error) {
	status := "failed"
	delay := sparkyMaxPending
	if attempts < sparkyMaxAttempts {
		status = "failed" // still retryable
		delay = sparkyBaseDelay
		for i := 1; i < attempts; i++ {
			if delay >= sparkyMaxDelay {
				delay = sparkyMaxDelay
				break
			}
			delay *= 2
		}
	}
	if _, err := q.db.Exec(`UPDATE sparky_jobs
		SET status = $1, last_error = $2, processing_at = NULL,
		    next_attempt_at = CURRENT_TIMESTAMP + ($3 * INTERVAL '1 second')
		WHERE id = $4`, status, cause.Error(), int(delay.Seconds()), id); err != nil {
		log.Printf("[Sparky] mark failed %s: %v", id, err)
	}

	// Notify the requesting user so the UI can stop spinning and fall back.
	// Load remaining fields for the payload.
	var userID string
	_ = q.db.QueryRow(`SELECT user_id FROM sparky_jobs WHERE id = $1`, id).Scan(&userID)
	if userID != "" {
		q.notifyStatus(userID, &SparkyJobResult{
			JobID:  id,
			Status: "failed",
			Error:  cause.Error(),
		})
	}
}

func (q *SparkyQueueService) notifyStatus(userID string, payload *SparkyJobResult) {
	if payload == nil || payload.Status == "" {
		return
	}
	if q.notify != nil {
		q.notify(userID, payload)
	}
}

// trigger publishes a job id to the consumer channel (or spawns inline in lean mode).
func (q *SparkyQueueService) trigger(id string) {
	if q.redis != nil {
		payload, _ := json.Marshal(map[string]string{"jobId": id})
		if err := q.redis.Publish(q.ctx, sparkyChannel, payload).Err(); err != nil {
			log.Printf("[Sparky] publish %s: %v (sweeper will re-queue)", id, err)
		}
		return
	}
	q.spawn(id)
}

// spawn processes a job on a bounded worker pool.
func (q *SparkyQueueService) spawn(id string) {
	q.sema <- struct{}{}
	q.wg.Add(1)
	go func() {
		defer q.wg.Done()
		defer func() { <-q.sema }()
		q.processJob(id)
	}()
}

// subscribeLoop consumes published job triggers.
func (q *SparkyQueueService) subscribeLoop() {
	defer q.wg.Done()
	sub := q.redis.Subscribe(q.ctx, sparkyChannel)
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
func (q *SparkyQueueService) sweepLoop() {
	defer q.wg.Done()
	ticker := time.NewTicker(sparkySweepEvery)
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

func (q *SparkyQueueService) sweep() {
	if _, err := q.db.Exec(`UPDATE sparky_jobs SET status = 'pending', processing_at = NULL
		WHERE status = 'processing' AND processing_at < CURRENT_TIMESTAMP - ($1 * INTERVAL '1 minute')`,
		int(sparkyStaleTTL.Minutes())); err != nil {
		log.Printf("[Sparky] reclaim stale processing: %v", err)
	}
	rows, err := q.db.Query(`
		SELECT id FROM sparky_jobs
		WHERE status IN ('pending', 'failed') AND next_attempt_at <= CURRENT_TIMESTAMP
		ORDER BY created_at
		LIMIT 50`)
	if err != nil {
		log.Printf("[Sparky] sweep due: %v", err)
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
func (q *SparkyQueueService) recover() {
	if _, err := q.db.Exec(`UPDATE sparky_jobs SET status = 'pending', processing_at = NULL
		WHERE status = 'processing'`); err != nil {
		log.Printf("[Sparky] recover reset processing: %v", err)
	}
	rows, err := q.db.Query(`
		SELECT id FROM sparky_jobs
		WHERE status IN ('pending', 'failed')
		ORDER BY created_at`)
	if err != nil {
		log.Printf("[Sparky] recover load incomplete: %v", err)
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
	log.Printf("[Sparky] recover: %d job(s) with incomplete answers re-queued", len(ids))
	for _, id := range ids {
		q.trigger(id)
	}
}

func trimOrDefault(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return strings.TrimSpace(s)
}
