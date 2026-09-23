package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/chorus/messenger/internal/models"
	"github.com/redis/go-redis/v9"
)

const (
	gradingChannel     = "grading:jobs"
	gradingMaxAttempts = 5
	gradingBaseDelay   = 10 * time.Second
	gradingMaxDelay    = 5 * time.Minute
	// gradingStaleTTL resets processing jobs whose worker died mid-flight
	// (network reset, crash) so upstream assignment flows re-evaluate cleanly
	// (plan V3 Phase 6: 60 seconds).
	gradingStaleTTL   = 60 * time.Second
	gradingSweepEvery = 15 * time.Second
	gradingPoolSize   = 3
)

// ProductionGrader is the subset of LearningAIService used by the queue.
// LearningAIService satisfies it; tests inject a fake.
type ProductionGrader interface {
	GradeProduction(ctx context.Context, p models.GradingJobPayload) (*models.GradingJobResult, string, error)
}

// GradingNotifier pushes a completed/failed grade to the requesting user over
// the WebSocket hub ("grade_result"); wired in main.go.
type GradingNotifier func(userID string, payload *models.GradingJobResult)

// GradingQueueService is the async, non-blocking grading pipeline (plan V3
// section D): API paths insert a durable grading_jobs row and immediately
// return 202 + job_id; a background pool claims rows with FOR UPDATE
// SKIP LOCKED (zero contention across replicas), grades via the provider
// chain, writes the result, and pushes "grade_result" to the learner. The
// client polls GET /learning/grading-jobs/:id as the socket fallback.
type GradingQueueService struct {
	db     *sql.DB
	redis  *redis.Client
	grader ProductionGrader
	notify GradingNotifier

	ctx    context.Context
	cancel context.CancelFunc
	sema   chan struct{}
	wg     sync.WaitGroup
	stopCh chan struct{}
}

func NewGradingQueueService(db *sql.DB, redisClient *redis.Client, grader ProductionGrader, notify GradingNotifier) *GradingQueueService {
	ctx, cancel := context.WithCancel(context.Background())
	return &GradingQueueService{
		db:     db,
		redis:  redisClient,
		grader: grader,
		notify: notify,
		ctx:    ctx,
		cancel: cancel,
		sema:   make(chan struct{}, gradingPoolSize),
		stopCh: make(chan struct{}),
	}
}

// Start launches the pub/sub consumer, the poller (FOR UPDATE SKIP LOCKED
// drain), and the stale sweeper.
func (q *GradingQueueService) Start() {
	q.recover()
	if q.redis != nil {
		q.wg.Add(1)
		go q.subscribeLoop()
	}
	q.wg.Add(2)
	go q.pollLoop()
	go q.sweepLoop()
	log.Println("[Grading] queue started (async workers + SKIP LOCKED poller + sweeper)")
}

func (q *GradingQueueService) Stop() {
	close(q.stopCh)
	q.cancel()
	q.wg.Wait()
}

// Enqueue registers a grading job and returns its id for the 202 response.
// jobType: production | writing | placement_open.
func (q *GradingQueueService) Enqueue(ctx context.Context, userID string, jobType string, payload models.GradingJobPayload, submissionID string) (string, error) {
	payloadJSON, _ := json.Marshal(payload)
	var id string
	err := q.db.QueryRowContext(ctx, `
		INSERT INTO grading_jobs (user_id, job_type, payload, submission_id, status)
		VALUES ($1, $2, $3, $4, 'pending')
		RETURNING id::text`,
		userID, jobType, string(payloadJSON), nullStr(submissionID)).Scan(&id)
	if err != nil {
		return "", err
	}
	q.trigger(id)
	return id, nil
}

// GetJob returns a job owned by the requesting user (polling fallback).
func (q *GradingQueueService) GetJob(ctx context.Context, jobID, userID string) (*models.GradingJob, error) {
	var j models.GradingJob
	var payloadJSON, resultJSON string
	var lastErr sql.NullString
	var submissionID sql.NullString
	var completedAt sql.NullTime
	err := q.db.QueryRowContext(ctx, `
		SELECT id::text, user_id, COALESCE(submission_id::text,''), job_type,
		       COALESCE(payload::text,'{}'), status, COALESCE(result::text,''),
		       attempts, last_error, created_at, next_attempt_at, completed_at
		FROM grading_jobs WHERE id = $1 AND user_id = $2`, jobID, userID).
		Scan(&j.ID, &j.UserID, &submissionID, &j.JobType, &payloadJSON, &j.Status, &resultJSON,
			&j.Attempts, &lastErr, &j.CreatedAt, &j.NextAttemptAt, &completedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("grading job not found")
	}
	if err != nil {
		return nil, err
	}
	if submissionID.Valid {
		s := submissionID.String
		j.SubmissionID = &s
	}
	if lastErr.Valid {
		e := lastErr.String
		j.LastError = &e
	}
	if completedAt.Valid {
		t := completedAt.Time
		j.CompletedAt = &t
	}
	_ = json.Unmarshal([]byte(payloadJSON), &j.Payload)
	if resultJSON != "" {
		var r models.GradingJobResult
		if json.Unmarshal([]byte(resultJSON), &r) == nil {
			res := r
			j.Result = &res
		}
	}
	return &j, nil
}

// claimNext selects the oldest due pending job while bypassing rows locked by
// adjacent workers (FOR UPDATE SKIP LOCKED), marks it processing, and commits
// the lease before grading so no other node can double-process it.
func (q *GradingQueueService) claimNext(ctx context.Context) (string, bool) {
	tx, err := q.db.BeginTx(ctx, nil)
	if err != nil {
		return "", false
	}
	defer tx.Rollback()

	var jobID string
	err = tx.QueryRowContext(ctx, `
		SELECT id::text FROM grading_jobs
		WHERE status = 'pending' AND next_attempt_at <= NOW()
		ORDER BY created_at ASC
		LIMIT 1
		FOR UPDATE SKIP LOCKED`).Scan(&jobID)
	if err == sql.ErrNoRows {
		return "", false
	}
	if err != nil {
		return "", false
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE grading_jobs SET status = 'processing', processing_at = NOW(), attempts = attempts + 1
		WHERE id = $1`, jobID); err != nil {
		return "", false
	}
	if err := tx.Commit(); err != nil {
		return "", false
	}
	return jobID, true
}

type gradingQueuedJob struct {
	ID           string
	UserID       string
	SubmissionID string
	Attempts     int
	Payload      models.GradingJobPayload
	JobType      string
}

func (q *GradingQueueService) loadJob(id string) (*gradingQueuedJob, bool) {
	j := &gradingQueuedJob{}
	var payloadJSON string
	var submissionID sql.NullString
	err := q.db.QueryRow(`
		SELECT id::text, user_id, COALESCE(submission_id::text,''), job_type,
		       COALESCE(payload::text,'{}'), attempts
		FROM grading_jobs WHERE id = $1 AND status = 'processing'`, id).
		Scan(&j.ID, &j.UserID, &submissionID, &j.JobType, &payloadJSON, &j.Attempts)
	if err != nil {
		return nil, false
	}
	if submissionID.Valid {
		j.SubmissionID = submissionID.String
	}
	_ = json.Unmarshal([]byte(payloadJSON), &j.Payload)
	return j, true
}

func (q *GradingQueueService) processJob(id string) {
	j, ok := q.loadJob(id)
	if !ok {
		return
	}
	result, provider, err := q.grader.GradeProduction(q.ctx, j.Payload)
	if err != nil {
		log.Printf("[Grading] job %s attempt %d via %q: %v", j.ID, j.Attempts, provider, err)
		// Heuristic fallback already resolves the job when providers are down;
		// a hard error (grader nil, etc.) retries with backoff.
		if result == nil {
			q.markFailed(j, err)
			return
		}
	}
	q.finishJob(j, result)
}

func (q *GradingQueueService) finishJob(j *gradingQueuedJob, result *models.GradingJobResult) {
	if result == nil {
		return
	}
	result.JobID = j.ID
	resultJSON, _ := json.Marshal(result)
	if _, err := q.db.Exec(`
		UPDATE grading_jobs
		SET status = 'done', result = $1, last_error = NULL, processing_at = NULL, completed_at = NOW()
		WHERE id = $2`, string(resultJSON), j.ID); err != nil {
		log.Printf("[Grading] mark done %s: %v", j.ID, err)
		return
	}
	// Fan the AI feedback into the assignment submission when one is linked.
	if j.SubmissionID != "" {
		if _, err := q.db.Exec(`
			UPDATE assignment_submissions SET ai_feedback = $1 WHERE id = $2`,
			string(resultJSON), j.SubmissionID); err != nil {
			log.Printf("[Grading] link submission %s: %v", j.SubmissionID, err)
		}
	}
	if q.notify != nil {
		q.notify(j.UserID, result)
	}
}

func (q *GradingQueueService) markFailed(j *gradingQueuedJob, cause error) {
	// Retryable failures back off exponentially; exhausted jobs are parked so
	// the sweeper does not spin on them forever.
	parked := j.Attempts >= gradingMaxAttempts
	delay := gradingBaseDelay
	if parked {
		delay = 24 * time.Hour
	} else {
		for i := 1; i < j.Attempts && delay < gradingMaxDelay; i++ {
			delay *= 2
		}
	}
	if _, err := q.db.Exec(`UPDATE grading_jobs
		SET status = 'failed', last_error = $1, processing_at = NULL,
		    next_attempt_at = NOW() + ($2 * INTERVAL '1 second')
		WHERE id = $3`, cause.Error(), int(delay.Seconds()), j.ID); err != nil {
		log.Printf("[Grading] mark failed %s: %v", j.ID, err)
	}
	if q.notify != nil {
		q.notify(j.UserID, &models.GradingJobResult{JobID: j.ID, Feedback: "Grading is taking longer than expected. Your submission is safe and will be retried."})
	}
}

// trigger publishes a job id for near-real-time processing (Redis pub/sub) or
// spawns inline in lean mode.
func (q *GradingQueueService) trigger(id string) {
	if q.redis != nil {
		payload, _ := json.Marshal(map[string]string{"jobId": id})
		if err := q.redis.Publish(q.ctx, gradingChannel, payload).Err(); err != nil {
			log.Printf("[Grading] publish %s: %v (poller will drain)", id, err)
		}
		return
	}
	q.spawn(id)
}

func (q *GradingQueueService) spawn(id string) {
	q.sema <- struct{}{}
	q.wg.Add(1)
	go func() {
		defer q.wg.Done()
		defer func() { <-q.sema }()
		q.processJob(id)
	}()
}

// subscribeLoop consumes published job triggers.
func (q *GradingQueueService) subscribeLoop() {
	defer q.wg.Done()
	sub := q.redis.Subscribe(q.ctx, gradingChannel)
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

// pollLoop drains due pending jobs with FOR UPDATE SKIP LOCKED — the safety
// net when a Redis trigger is lost and the primary drain in multi-node setups.
func (q *GradingQueueService) pollLoop() {
	defer q.wg.Done()
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if id, ok := q.claimNext(q.ctx); ok {
				q.spawn(id)
			}
		case <-q.stopCh:
			return
		}
	}
}

// sweepLoop resets stale processing leases (> 60s) back to pending and
// re-publishes due jobs.
func (q *GradingQueueService) sweepLoop() {
	defer q.wg.Done()
	ticker := time.NewTicker(gradingSweepEvery)
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

func (q *GradingQueueService) sweep() {
	if _, err := q.db.Exec(`UPDATE grading_jobs SET status = 'pending', processing_at = NULL
		WHERE status = 'processing' AND processing_at < NOW() - ($1 * INTERVAL '1 second')`,
		int(gradingStaleTTL.Seconds())); err != nil {
		log.Printf("[Grading] reclaim stale processing: %v", err)
	}
}

// recover re-queues incomplete work after a restart.
func (q *GradingQueueService) recover() {
	if _, err := q.db.Exec(`UPDATE grading_jobs SET status = 'pending', processing_at = NULL
		WHERE status = 'processing'`); err != nil {
		log.Printf("[Grading] recover reset processing: %v", err)
	}
}
