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
	translationChannel     = "translation:jobs"
	translationMaxAttempts = 6
	translationBaseDelay   = 30 * time.Second
	translationMaxDelay    = 10 * time.Minute
	translationMaxPending  = 24 * time.Hour // failed jobs wait before admin manual retry
	staleProcessingTTL     = 5 * time.Minute
	translationSweepEvery  = 30 * time.Second
	translationPoolSize    = 6
)

// TextTranslator is the subset of TranslationService used by the queue.
type TextTranslator interface {
	TranslateQuick(text, targetLang, sourceLang string) (string, error)
}

// TranslationLoadMessage loads a fresh message row by id.
type TranslationLoadMessage func(messageID string) (*models.Message, error)

// TranslationNotifier is invoked after a translation completes; the server wires
// it to fan out message_updated over the WebSocket hub and Redis pub/sub.
type TranslationNotifier func(chatID string, message *models.Message)

// TranslationQueueService provides guaranteed, near-real-time translations.
//
// Design (durable outbox + pub/sub trigger):
//   - EnqueueForMessage inserts one durable row per (message, target language)
//     into translation_jobs — the DB is the source of truth.
//   - The row is then published to a Redis channel; a consumer translates it
//     immediately (near-real-time) using the existing provider chain.
//   - A periodic sweeper is the safety net: it reclaims stale processing rows,
//     re-publishes due pending/failed jobs, and (in lean mode without Redis)
//     processes them inline. Nothing depends on pub/sub being reliable.
//   - On Start() it recovers after a restart: stale processing rows are reset
//     and every incomplete translation is re-queued. Completed translations are
//     persisted in messages.translations, so reconnecting clients refetch them.
type TranslationQueueService struct {
	db         *sql.DB
	redis      *redis.Client
	translator TextTranslator
	load       TranslationLoadMessage
	notify     TranslationNotifier

	ctx    context.Context
	cancel context.CancelFunc
	sema   chan struct{}
	wg     sync.WaitGroup
	stopCh chan struct{}
}

func NewTranslationQueueService(
	db *sql.DB,
	redisClient *redis.Client,
	translator TextTranslator,
	load TranslationLoadMessage,
	notify TranslationNotifier,
) *TranslationQueueService {
	ctx, cancel := context.WithCancel(context.Background())
	return &TranslationQueueService{
		db:         db,
		redis:      redisClient,
		translator: translator,
		load:       load,
		notify:     notify,
		ctx:        ctx,
		cancel:     cancel,
		sema:       make(chan struct{}, translationPoolSize),
		stopCh:     make(chan struct{}),
	}
}

// Start recovers incomplete work from a previous run and launches the
// pub/sub consumer and the retry sweeper.
func (q *TranslationQueueService) Start() {
	q.recover()
	if q.redis != nil {
		q.wg.Add(1)
		go q.subscribeLoop()
	}
	q.wg.Add(1)
	go q.sweepLoop()
	log.Println("[Translate] queue started (durable outbox + pub/sub + sweeper)")
}

// Stop halts the sweeper and consumer. In-flight translations are allowed to
// finish; durable rows ensure nothing is lost across restarts.
func (q *TranslationQueueService) Stop() {
	close(q.stopCh)
	q.cancel()
	q.wg.Wait()
}

// EnqueueForMessage creates one durable job per target language and triggers
// processing. Existing jobs for the same (message, language) are left alone.
// Priority (0 = free, 1 = premium) gives faster responses to premium senders.
func (q *TranslationQueueService) EnqueueForMessage(message *models.Message, sourceLang string, targetLangs []string, priority int) {
	var created []string
	for _, lang := range targetLangs {
		lang = strings.TrimSpace(lang)
		if lang == "" || lang == sourceLang {
			continue
		}
		var id string
		err := q.db.QueryRow(`
			INSERT INTO translation_jobs (message_id, chat_id, text, source_lang, target_lang, priority)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (message_id, target_lang) DO NOTHING
			RETURNING id`, message.ID, message.ChatID, message.Text, sourceLang, lang, priority).Scan(&id)
		if err != nil {
			if err == sql.ErrNoRows {
				continue // already queued
			}
			log.Printf("[Translate] failed to enqueue %s -> %s: %v", message.ID, lang, err)
			continue
		}
		created = append(created, lang)
	}
	if len(created) == 0 {
		return
	}
	log.Printf("[Translate] enqueued %d jobs for message %s (%v)", len(created), message.ID, created)
	if q.redis != nil {
		q.publish(message.ID)
	} else {
		// Lean mode (no Redis): process inline so translations stay near-real-time.
		q.spawn(message.ID)
	}
}

// EnqueueManual queues a single translation job for a specific target language,
// explicitly requested by a participant (e.g. the "Translate" button on a
// message). Unlike EnqueueForMessage it always uses sourceLang "auto" so the
// provider detects the real source, and it forcibly resets any prior job for
// (message, target) — even one that previously failed permanently — because an
// explicit request from the viewer overrides the earlier state.
func (q *TranslationQueueService) EnqueueManual(message *models.Message, targetLang string, priority int) {
	targetLang = strings.TrimSpace(targetLang)
	if targetLang == "" {
		return
	}
	_, err := q.db.Exec(`
		INSERT INTO translation_jobs (message_id, chat_id, text, source_lang, target_lang, priority, status, attempts)
		VALUES ($1, $2, $3, 'auto', $4, $5, 'pending', 0)
		ON CONFLICT (message_id, target_lang) DO UPDATE
		SET text = EXCLUDED.text, source_lang = 'auto', priority = EXCLUDED.priority,
		    status = 'pending', attempts = 0, last_error = NULL, result = NULL,
		    processing_at = NULL, completed_at = NULL, next_attempt_at = CURRENT_TIMESTAMP`,
		message.ID, message.ChatID, message.Text, targetLang, priority)
	if err != nil {
		log.Printf("[Translate] manual enqueue %s -> %s: %v", message.ID, targetLang, err)
		return
	}
	log.Printf("[Translate] manual enqueue %s -> %s", message.ID, targetLang)
	if q.redis != nil {
		q.publish(message.ID)
	} else {
		q.spawn(message.ID)
	}
}

// processMessage claims and translates every due job for a message.
func (q *TranslationQueueService) processMessage(messageID string) {
	rows, err := q.db.Query(`
		SELECT id, text, source_lang, target_lang FROM translation_jobs
		WHERE message_id = $1 AND status IN ('pending', 'failed')
		  AND next_attempt_at <= CURRENT_TIMESTAMP
		ORDER BY priority DESC, created_at`, messageID)
	if err != nil {
		log.Printf("[Translate] load jobs for %s: %v", messageID, err)
		return
	}
	var jobs []queuedJob
	for rows.Next() {
		var j queuedJob
		if err := rows.Scan(&j.ID, &j.Text, &j.SourceLang, &j.TargetLang); err != nil {
			continue
		}
		jobs = append(jobs, j)
	}
	rows.Close()
	if rows.Err() != nil {
		log.Printf("[Translate] scan jobs for %s: %v", messageID, rows.Err())
		return
	}
	for _, j := range jobs {
		q.translateJob(j)
	}
}

// claimJob atomically reserves a job for translation (prevents double work
// between the pub/sub consumer, the sweeper, and across instances).
type queuedJob struct {
	ID         string
	MessageID  string
	ChatID     string
	Text       string
	SourceLang string
	TargetLang string
	Attempts   int
}

func (q *TranslationQueueService) claimJob(id string) (*queuedJob, bool) {
	j := &queuedJob{}
	err := q.db.QueryRow(`
		UPDATE translation_jobs
		SET status = 'processing', processing_at = CURRENT_TIMESTAMP, attempts = attempts + 1
		WHERE id = $1 AND status IN ('pending', 'failed') AND next_attempt_at <= CURRENT_TIMESTAMP
		RETURNING id, message_id, chat_id, text, source_lang, target_lang, attempts`,
		id).Scan(&j.ID, &j.MessageID, &j.ChatID, &j.Text, &j.SourceLang, &j.TargetLang, &j.Attempts)
	if err != nil {
		return nil, false
	}
	return j, true
}

func (q *TranslationQueueService) translateJob(job queuedJob) {
	source := job.SourceLang
	if source == "" {
		source = "auto"
	}
	start := time.Now()
	translated, err := q.translator.TranslateQuick(job.Text, job.TargetLang, source)
	if err != nil {
		log.Printf("[Translate] job %s (%s): %v", job.ID, job.TargetLang, err)
		q.markFailed(job.ID, job.Attempts, err)
		return
	}
	log.Printf("[Translate] job %s (%s) done in %s", job.ID, job.TargetLang, time.Since(start).Round(time.Millisecond))
	q.markDone(job.ID, job.TargetLang, translated)
}

func (q *TranslationQueueService) markDone(id, targetLang, result string) {
	if _, err := q.db.Exec(`UPDATE translation_jobs
		SET status = 'done', result = $1, last_error = NULL, processing_at = NULL, completed_at = CURRENT_TIMESTAMP
		WHERE id = $2`, result, id); err != nil {
		log.Printf("[Translate] mark done %s: %v", id, err)
		return
	}
	// Merge into the message's translations JSON.
	pair, _ := json.Marshal(map[string]string{targetLang: result})
	if _, err := q.db.Exec(`UPDATE messages
		SET translations = COALESCE(translations, '{}'::jsonb) || $1::jsonb
		WHERE id = (SELECT message_id FROM translation_jobs WHERE id = $2)`, pair, id); err != nil {
		log.Printf("[Translate] merge into message: %v", err)
		return
	}
	// Notify connected participants with the freshly merged message.
	message, err := q.loadByMessageID(id)
	if err != nil {
		log.Printf("[Translate] notify load %s: %v", id, err)
		return
	}
	message.TranslationEnhanced = true
	if q.notify != nil {
		q.notify(message.ChatID, message)
	}
}

func (q *TranslationQueueService) loadByMessageID(jobID string) (*models.Message, error) {
	var messageID string
	if err := q.db.QueryRow(`SELECT message_id FROM translation_jobs WHERE id = $1`, jobID).Scan(&messageID); err != nil {
		return nil, err
	}
	return q.load(messageID)
}

func (q *TranslationQueueService) markFailed(id string, attempts int, cause error) {
	status := "failed"
	delay := translationMaxPending
	if attempts < translationMaxAttempts {
		status = "failed" // still retryable
		delay = translationBaseDelay
		for i := 1; i < attempts; i++ {
			if delay >= translationMaxDelay {
				delay = translationMaxDelay
				break
			}
			delay *= 2
		}
	}
	if _, err := q.db.Exec(`UPDATE translation_jobs
		SET status = $1, last_error = $2, processing_at = NULL,
		    next_attempt_at = CURRENT_TIMESTAMP + ($3 * INTERVAL '1 second')
		WHERE id = $4`, status, cause.Error(), int(delay.Seconds()), id); err != nil {
		log.Printf("[Translate] mark failed %s: %v", id, err)
	}
}

// publish triggers the near-real-time consumer.
func (q *TranslationQueueService) publish(messageID string) {
	payload, _ := json.Marshal(map[string]string{"messageId": messageID})
	if err := q.redis.Publish(q.ctx, translationChannel, payload).Err(); err != nil {
		log.Printf("[Translate] publish %s: %v (sweeper will re-queue)", messageID, err)
	}
}

// spawn processes a message on a bounded worker pool.
func (q *TranslationQueueService) spawn(messageID string) {
	q.sema <- struct{}{}
	q.wg.Add(1)
	go func() {
		defer q.wg.Done()
		defer func() { <-q.sema }()
		q.processMessage(messageID)
	}()
}

// subscribeLoop consumes published job triggers.
func (q *TranslationQueueService) subscribeLoop() {
	defer q.wg.Done()
	sub := q.redis.Subscribe(q.ctx, translationChannel)
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
				MessageID string `json:"messageId"`
			}
			if err := json.Unmarshal([]byte(msg.Payload), &envelope); err != nil || envelope.MessageID == "" {
				continue
			}
			q.spawn(envelope.MessageID)
		}
	}
}

// sweepLoop is the safety net: reclaim stale leases and re-publish due jobs.
func (q *TranslationQueueService) sweepLoop() {
	defer q.wg.Done()
	ticker := time.NewTicker(translationSweepEvery)
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

func (q *TranslationQueueService) sweep() {
	// Reclaim jobs left in 'processing' by a crash (or a slow-but-dead call).
	if _, err := q.db.Exec(`UPDATE translation_jobs SET status = 'pending', processing_at = NULL
		WHERE status = 'processing' AND processing_at < CURRENT_TIMESTAMP - ($1 * INTERVAL '1 minute')`,
		int(staleProcessingTTL.Minutes())); err != nil {
		log.Printf("[Translate] reclaim stale processing: %v", err)
	}
	// Re-queue due jobs. Premium (higher priority) messages are processed
	// first so premium users get faster responses under load. The DISTINCT
	// set is ordered by the highest priority among each message's jobs.
	rows, err := q.db.Query(`
		SELECT message_id, MAX(priority) AS prio FROM translation_jobs
		WHERE status IN ('pending', 'failed') AND next_attempt_at <= CURRENT_TIMESTAMP
		GROUP BY message_id
		ORDER BY prio DESC, message_id
		LIMIT 100`)
	if err != nil {
		log.Printf("[Translate] sweep due: %v", err)
		return
	}
	var messageIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			messageIDs = append(messageIDs, id)
		}
	}
	rows.Close()
	for _, id := range messageIDs {
		if q.redis != nil {
			q.publish(id)
		} else {
			q.spawn(id)
		}
	}
}

// recover re-queues incomplete work after a restart.
func (q *TranslationQueueService) recover() {
	if _, err := q.db.Exec(`UPDATE translation_jobs SET status = 'pending', processing_at = NULL
		WHERE status = 'processing'`); err != nil {
		log.Printf("[Translate] recover reset processing: %v", err)
	}
	rows, err := q.db.Query(`
		SELECT message_id, MAX(priority) AS prio FROM translation_jobs
		WHERE status IN ('pending', 'failed')
		GROUP BY message_id
		ORDER BY prio DESC, message_id`)
	if err != nil {
		log.Printf("[Translate] recover load incomplete: %v", err)
		return
	}
	var messageIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			messageIDs = append(messageIDs, id)
		}
	}
	rows.Close()
	log.Printf("[Translate] recover: %d message(s) with incomplete translations re-queued", len(messageIDs))
	for _, id := range messageIDs {
		if q.redis != nil {
			q.publish(id)
		} else {
			q.spawn(id)
		}
	}
}

// List returns translation jobs, optionally filtered by status and text query.
func (q *TranslationQueueService) List(status, qq string, limit int) ([]models.TranslationJob, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	query := `SELECT id, message_id, chat_id, text, COALESCE(source_lang,''), target_lang, priority, status,
		COALESCE(result,''), attempts, COALESCE(last_error,''), created_at, next_attempt_at, completed_at
		FROM translation_jobs WHERE 1=1`
	args := []interface{}{}
	if status != "" && status != "all" {
		args = append(args, status)
		query += fmt.Sprintf(` AND status = $%d`, len(args))
	}
	if qq != "" {
		args = append(args, "%"+qq+"%")
		query += fmt.Sprintf(` AND (text ILIKE $%d OR target_lang ILIKE $%d)`, len(args), len(args))
	}
	query += fmt.Sprintf(` ORDER BY created_at DESC LIMIT $%d`, len(args)+1)
	args = append(args, limit)

	rows, err := q.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	jobs := []models.TranslationJob{}
	for rows.Next() {
		var j models.TranslationJob
		var source, result, lastErr string
		if err := rows.Scan(&j.ID, &j.MessageID, &j.ChatID, &j.Text, &source, &j.TargetLang,
			&j.Priority, &j.Status, &result, &j.Attempts, &lastErr, &j.CreatedAt, &j.NextAttempt, &j.CompletedAt); err != nil {
			return nil, err
		}
		j.SourceLang = source
		j.Result = result
		j.LastError = lastErr
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

// RetryByID forces an immediate re-attempt of a pending or failed job.
func (q *TranslationQueueService) RetryByID(id string) error {
	var messageID string
	err := q.db.QueryRow(`UPDATE translation_jobs
		SET status = 'pending', processing_at = NULL, last_error = NULL, next_attempt_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND status <> 'done'
		RETURNING message_id`, id).Scan(&messageID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("translation job not found or already completed")
		}
		return err
	}
	if q.redis != nil {
		q.publish(messageID)
	} else {
		q.spawn(messageID)
	}
	return nil
}

// Stats returns counts for the admin dashboard.
func (q *TranslationQueueService) Stats() (pending, processing, done, failed int) {
	rows, err := q.db.Query(`SELECT status, COUNT(*) FROM translation_jobs GROUP BY status`)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err == nil {
			switch status {
			case "pending":
				pending = count
			case "processing":
				processing = count
			case "done":
				done = count
			case "failed":
				failed = count
			}
		}
	}
	return
}
