package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/chorus/messenger/internal/models"
	"github.com/redis/go-redis/v9"
)

// Quality evaluation (FR-30) — cross-model critique + KPIs.
//
// Every produced translation / grammar analysis should be scored by a *model
// different from the one that produced it* so the quality signal is not
// self-referential. The evaluator rows live in translation_evals / grammar_evals
// (durable outbox, source of truth) and are processed by this service's worker:
// a 'pending' row is claimed, scored by a (preferably different) endpoint from
// the evaluator chain, and marked 'done'/'failed'. A sweeper retries failures
// and a startup recovery re-queues rows left in 'processing' by a crash.
//
// KPIs (accuracy, p95 latency, cost/1k tokens, cache hit rate) are exposed for
// the admin console via KPIs(). Phoenix traces are emitted from the producer
// services directly (translation.go / grammar.go); this service only turns
// stored outputs into scored feedback.
const (
	evalChannel      = "quality:evals"
	evalMaxAttempts  = 5
	evalBaseDelay    = 20 * time.Second
	evalMaxDelay     = 10 * time.Minute
	evalMaxPending   = 24 * time.Hour // failed evals wait before admin manual retry
	evalStaleTTL     = 5 * time.Minute
	evalSweepEvery   = 30 * time.Second
	evalPoolSize     = 4
	evalCostPer1KEnv = "EVAL_COST_PER_1K_TOKENS" // USD per 1k tokens (for cost KPIs)
	defaultCostPer1K = 0.0
)

// evalKind identifies which eval table a worker should act on.
type evalKind string

const (
	evalKindTranslation evalKind = "translation"
	evalKindGrammar     evalKind = "grammar"
)

// evalEnvelope is the pub/sub trigger payload.
type evalEnvelope struct {
	Kind evalKind `json:"kind"`
	ID   string   `json:"id"`
}

// QualityEvaluatorService scores stored translations/grammar analyses with a
// cross-model critique and aggregates FR-30 KPIs.
type QualityEvaluatorService struct {
	db        *sql.DB
	redis     *redis.Client
	endpoints []GrammarEndpoint
	notifier  func(kind evalKind, evalID string) // optional (admin KPIs live-update)

	ctx    context.Context
	cancel context.CancelFunc
	sema   chan struct{}
	wg     sync.WaitGroup
	stopCh chan struct{}

	costPer1K float64
}

// NewQualityEvaluatorService creates a quality evaluator. Redis is optional;
// when nil the sweeper processes pending evals inline. endpoints is the
// evaluator model chain; when empty the evaluator is effectively disabled
// (Enqueue* become no-ops).
func NewQualityEvaluatorService(db *sql.DB, redisClient *redis.Client, endpoints []GrammarEndpoint) *QualityEvaluatorService {
	ctx, cancel := context.WithCancel(context.Background())
	return &QualityEvaluatorService{
		db:        db,
		redis:     redisClient,
		endpoints: endpoints,
		sema:      make(chan struct{}, evalPoolSize),
		ctx:       ctx,
		cancel:    cancel,
		stopCh:    make(chan struct{}),
		costPer1K: evalCostPer1K(),
	}
}

// SetNotifier wires a callback fired after an eval completes (used by the admin
// console to refresh KPIs).
func (e *QualityEvaluatorService) SetNotifier(n func(kind evalKind, evalID string)) {
	e.notifier = n
}

// Enabled reports whether an evaluator model chain is configured.
func (e *QualityEvaluatorService) Enabled() bool {
	return len(e.endpoints) > 0
}

// Start launches the worker. With no endpoints configured it still runs the
// sweeper so pending rows are not silently stuck, but the sweeper will not find
// any (Enqueue* are no-ops when disabled).
func (e *QualityEvaluatorService) Start() {
	e.recover()
	if e.redis != nil {
		e.wg.Add(1)
		go e.subscribeLoop()
	}
	e.wg.Add(1)
	go e.sweepLoop()
	log.Println("[Eval] quality evaluator started (durable outbox + pub/sub + sweeper)")
}

// Stop halts the sweeper and consumer.
func (e *QualityEvaluatorService) Stop() {
	close(e.stopCh)
	e.cancel()
	e.wg.Wait()
}

// EnqueueTranslation creates a pending cross-model eval for a completed
// translation job, copying its lineage (source/target, producer provider,
// source + translated text) from translation_jobs. No-op when evaluator is
// disabled or the job is not done / already evaluated.
func (e *QualityEvaluatorService) EnqueueTranslation(jobID string) error {
	if !e.Enabled() {
		return nil
	}
	var id string
	err := e.db.QueryRow(`
		INSERT INTO translation_evals
			(translation_job_id, message_id, chat_id, source_lang, target_lang,
			 source_text, translated_text, producer_provider)
		SELECT j.id, j.message_id, j.chat_id, j.source_lang, j.target_lang,
		       j.text, j.result, j.provider
		FROM translation_jobs j
		WHERE j.id = $1 AND j.status = 'done'
		  AND NOT EXISTS (SELECT 1 FROM translation_evals ev WHERE ev.translation_job_id = j.id)
		RETURNING id`, jobID).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil // not done or already evaluated
		}
		log.Printf("[Eval] enqueue translation %s: %v", jobID, err)
		return err
	}
	log.Printf("[Eval] enqueued translation eval %s (job %s)", id, jobID)
	e.trigger(evalEnvelope{Kind: evalKindTranslation, ID: id})
	return nil
}

// EnqueueGrammar creates a pending cross-model eval for a completed grammar job.
func (e *QualityEvaluatorService) EnqueueGrammar(grammarJobID string) error {
	if !e.Enabled() {
		return nil
	}
	var id string
	err := e.db.QueryRow(`
		INSERT INTO grammar_evals
			(grammar_job_id, user_id, text, language, native_language, producer_provider)
		SELECT j.id, j.user_id, j.text, j.language, j.native_language, j.provider_used
		FROM grammar_jobs j
		WHERE j.id = $1 AND j.status = 'done'
		  AND NOT EXISTS (SELECT 1 FROM grammar_evals ev WHERE ev.grammar_job_id = j.id)
		RETURNING id`, grammarJobID).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		log.Printf("[Eval] enqueue grammar %s: %v", grammarJobID, err)
		return err
	}
	log.Printf("[Eval] enqueued grammar eval %s (job %s)", id, grammarJobID)
	e.trigger(evalEnvelope{Kind: evalKindGrammar, ID: id})
	return nil
}

// pickEndpoint selects an evaluator endpoint. Preference is a model different
// from the producer's (so the critique is cross-model); if none differ it falls
// back to the first configured endpoint. Producer is e.g. "openai(model)".
func (e *QualityEvaluatorService) pickEndpoint(producer string) GrammarEndpoint {
	if len(e.endpoints) == 0 {
		return GrammarEndpoint{}
	}
	for _, ep := range e.endpoints {
		if ep.Name != "" && producer != "" && !strings.Contains(producer, ep.Name) {
			return ep
		}
	}
	return e.endpoints[0]
}

// ---------------------------------------------------------------------------
// Worker: claim -> score -> persist
// ---------------------------------------------------------------------------

func (e *QualityEvaluatorService) process(k evalEnvelope) {
	if e.kindTranslation(k.Kind) {
		e.processTranslationEval(k.ID)
	} else if e.kindGrammar(k.Kind) {
		e.processGrammarEval(k.ID)
	}
}

func (e *QualityEvaluatorService) kindTranslation(k evalKind) bool { return k == evalKindTranslation }
func (e *QualityEvaluatorService) kindGrammar(k evalKind) bool     { return k == evalKindGrammar }

// processTranslationEval claims and scores one translation eval row.
func (e *QualityEvaluatorService) processTranslationEval(id string) {
	row, ok := e.claimTranslationEval(id)
	if !ok {
		return
	}
	start := time.Now()
	ep := e.pickEndpoint(row.producerProvider)
	if ep.Name == "" {
		e.failTranslationEval(id, row.attempts, fmt.Errorf("no evaluator endpoint configured"))
		return
	}
	score, err := e.scoreTranslation(ep, row)
	latency := time.Since(start)
	if err != nil {
		log.Printf("[Eval] translation eval %s via %s failed: %v", id, ep.Name, err)
		e.failTranslationEval(id, row.attempts, err)
		return
	}
	log.Printf("[Eval] translation eval %s via %s scored in %s (accuracy=%.1f)", id, ep.Name, latency.Round(time.Millisecond), score.AccuracyScore)
	e.doneTranslationEval(id, ep.Name, score)
}

// scoreTranslation calls the evaluator endpoint to rate a translation.
func (e *QualityEvaluatorService) scoreTranslation(ep GrammarEndpoint, row *translationEvalRow) (*models.TranslationEval, error) {
	prompt := fmt.Sprintf(`<role>You are a rigorous translation quality evaluator. The target audience is language learners of %[2]s.
Rate BOTH fidelity to the source and naturalness in the target language.</role>

<task>
Source text (%[2]s): "%[1]s"
Translation (%[3]s): "%[4]s"
Rate this translation on these criteria.
</task>

<output_format>
Return ONLY valid JSON using this EXACT structure:
{
  "accuracy_score": <integer 0-100>,
  "fluency_score": <integer 0-100>,
  "cefr_level": "<A1|A2|B1|B2|C1|C2>",
  "critique": "2-3 sentences: what's accurate, what's awkward, any meaning shifts"
}
</output_format>

<rules>
- accuracy_score: how faithfully the meaning is preserved (0-100).
- fluency_score: how natural/idiomatic the target reading is (0-100).
- cefr_level: the CEFR complexity level of the SOURCE text, not the translation.
- critique: concise, specific, actionable. Never mention scores.
</rules>`, row.sourceText, row.sourceLang, row.targetLang, row.translatedText)

	raw, providerName, err := e.callEvaluator(ep, prompt)
	if err != nil {
		return nil, err
	}
	var score models.TranslationEval
	parsed := false
	for _, obj := range extractJSONObjects(raw) {
		var m map[string]interface{}
		if json.Unmarshal([]byte(obj), &m) != nil {
			continue
		}
		if acc, ok := num(m["accuracy_score"]); ok {
			score.AccuracyScore = acc
			parsed = true
		}
		if flu, ok := num(m["fluency_score"]); ok {
			score.FluencyScore = flu
		}
		if lvl, ok := m["cefr_level"].(string); ok {
			score.CEFRLevel = lvl
		}
		if crit, ok := m["critique"].(string); ok {
			score.Critique = crit
		}
		if parsed {
			break
		}
	}
	if !parsed {
		return nil, fmt.Errorf("evaluator returned unparseable JSON: %.200s", raw)
	}
	score.EvaluatorProvider = providerName
	return &score, nil
}

// processGrammarEval claims and scores one grammar eval row.
func (e *QualityEvaluatorService) processGrammarEval(id string) {
	row, ok := e.claimGrammarEval(id)
	if !ok {
		return
	}
	ep := e.pickEndpoint(row.producerProvider)
	if ep.Name == "" {
		e.failGrammarEval(id, row.attempts, fmt.Errorf("no evaluator endpoint configured"))
		return
	}
	score, err := e.scoreGrammar(ep, row)
	if err != nil {
		log.Printf("[Eval] grammar eval %s via %s failed: %v", id, ep.Name, err)
		e.failGrammarEval(id, row.attempts, err)
		return
	}
	e.doneGrammarEval(id, ep.Name, score)
}

// scoreGrammar calls the evaluator endpoint to rate a grammar analysis.
func (e *QualityEvaluatorService) scoreGrammar(ep GrammarEndpoint, row *grammarEvalRow) (*models.GrammarEval, error) {
	prompt := fmt.Sprintf(`<role>You are a language-teaching expert reviewing an AI grammar analysis that will be shown to a learner of %[1]s.</role>

<task>
Target sentence: "%[2]s"
AI analysis (JSON): %[3]s
Review the correctness, clarity, and teaching helpfulness of this analysis.
</task>

<output_format>
Return ONLY valid JSON using this EXACT structure:
{
  "accuracy_score": <integer 0-100>,
  "cefr_level": "<A1|A2|B1|B2|C1|C2>",
  "criterion_scores": {
    "grammar_accuracy": <integer 0-100>,
    "clarity": <integer 0-100>,
    "helpfulness": <integer 0-100>
  },
  "critique": "2-3 sentences: what's correct, misleading, or missing"
}
</output_format>

<rules>
- accuracy_score: how grammatically correct the analysis is (0-100).
- cefr_level: the estimated learner level this analysis targets.
- criterion_scores: sub-scores for grammar accuracy, clarity, helpfulness.
- critique: concise, specific, actionable.
</rules>`, row.language, row.text, row.resultJSON)

	raw, providerName, err := e.callEvaluator(ep, prompt)
	if err != nil {
		return nil, err
	}
	var score models.GrammarEval
	parsed := false
	for _, obj := range extractJSONObjects(raw) {
		var m map[string]interface{}
		if json.Unmarshal([]byte(obj), &m) != nil {
			continue
		}
		if acc, ok := num(m["accuracy_score"]); ok {
			score.AccuracyScore = acc
			parsed = true
		}
		if lvl, ok := m["cefr_level"].(string); ok {
			score.CEFRLevel = lvl
		}
		if crit, ok := m["critique"].(string); ok {
			score.Critique = crit
		}
		if cs, ok := m["criterion_scores"].(map[string]interface{}); ok {
			score.CriterionScores = cs
		}
		if parsed {
			break
		}
	}
	if !parsed {
		return nil, fmt.Errorf("evaluator returned unparseable JSON: %.200s", raw)
	}
	score.EvaluatorProvider = providerName
	return &score, nil
}

// callEvaluator invokes the evaluator chain, preferring a DIFFERENT endpoint
// than the producer (cross-model). It returns the raw text and the endpoint that
// answered.
func (e *QualityEvaluatorService) callEvaluator(ep GrammarEndpoint, prompt string) (string, string, error) {
	// Keep the producer-endpoint preference inside one request: use ep, but if
	// it happens to be the only available model, that's acceptable (a single
	// endpoint still produces a critique).
	raw, err := ep.call(context.Background(), prompt, "en", true)
	if err != nil {
		// Try the rest of the chain if the preferred endpoint failed.
		var lastErr error
		for _, other := range e.endpoints {
			if other.Name == ep.Name {
				continue
			}
			res, err := other.call(context.Background(), prompt, "en", true)
			if err == nil {
				return res, other.Name, nil
			}
			lastErr = err
		}
		if lastErr != nil {
			return "", "", fmt.Errorf("all evaluator endpoints failed: %w", lastErr)
		}
		return "", "", err
	}
	return raw, ep.Name, nil
}

// ---------------------------------------------------------------------------
// Claim + persist row helpers
// ---------------------------------------------------------------------------

type translationEvalRow struct {
	id               string
	sourceLang       string
	targetLang       string
	sourceText       string
	translatedText   string
	producerProvider string
	attempts         int
}

type grammarEvalRow struct {
	id               string
	text             string
	language         string
	nativeLanguage   string
	resultJSON       string
	producerProvider string
	attempts         int
}

func (e *QualityEvaluatorService) claimTranslationEval(id string) (*translationEvalRow, bool) {
	r := &translationEvalRow{}
	err := e.db.QueryRow(`
		UPDATE translation_evals
		SET status = 'processing', processing_at = CURRENT_TIMESTAMP, attempts = attempts + 1
		WHERE id = $1 AND status IN ('pending', 'failed')
		RETURNING id, COALESCE(source_lang,''), target_lang, source_text, translated_text,
		          COALESCE(producer_provider,''), attempts`,
		id).Scan(&r.id, &r.sourceLang, &r.targetLang, &r.sourceText, &r.translatedText, &r.producerProvider, &r.attempts)
	if err != nil {
		return nil, false
	}
	return r, true
}

func (e *QualityEvaluatorService) claimGrammarEval(id string) (*grammarEvalRow, bool) {
	r := &grammarEvalRow{}
	err := e.db.QueryRow(`
		UPDATE grammar_evals g
		SET status = 'processing', processing_at = CURRENT_TIMESTAMP, attempts = attempts + 1
		FROM grammar_jobs j
		WHERE g.id = $1 AND g.grammar_job_id = j.id AND g.status IN ('pending', 'failed')
		RETURNING g.id, COALESCE(g.text,''), g.language, g.native_language,
		          COALESCE(j.result::text,''), COALESCE(g.producer_provider,''), g.attempts`,
		id).Scan(&r.id, &r.text, &r.language, &r.nativeLanguage, &r.resultJSON, &r.producerProvider, &r.attempts)
	if err != nil {
		return nil, false
	}
	return r, true
}

func (e *QualityEvaluatorService) doneTranslationEval(id, evaluatorProvider string, s *models.TranslationEval) {
	if _, err := e.db.Exec(`UPDATE translation_evals
		SET status = 'done', evaluator_provider = $1, accuracy_score = $2, fluency_score = $3,
		    cefr_level = $4, critique = $5, processing_at = NULL, completed_at = CURRENT_TIMESTAMP,
		    last_error = NULL
		WHERE id = $6`,
		nullStr(evaluatorProvider), s.AccuracyScore, s.FluencyScore, nullStr(s.CEFRLevel), s.Critique, id); err != nil {
		log.Printf("[Eval] mark translation eval done %s: %v", id, err)
		return
	}
	if e.notifier != nil {
		e.notifier(evalKindTranslation, id)
	}
}

func (e *QualityEvaluatorService) doneGrammarEval(id, evaluatorProvider string, s *models.GrammarEval) {
	criterionJSON, _ := json.Marshal(s.CriterionScores)
	if _, err := e.db.Exec(`UPDATE grammar_evals
		SET status = 'done', evaluator_provider = $1, accuracy_score = $2, cefr_level = $3,
		    criterion_scores = $4, critique = $5, processing_at = NULL, completed_at = CURRENT_TIMESTAMP,
		    last_error = NULL
		WHERE id = $6`,
		nullStr(evaluatorProvider), s.AccuracyScore, nullStr(s.CEFRLevel), string(criterionJSON), s.Critique, id); err != nil {
		log.Printf("[Eval] mark grammar eval done %s: %v", id, err)
		return
	}
	if e.notifier != nil {
		e.notifier(evalKindGrammar, id)
	}
}

func (e *QualityEvaluatorService) failTranslationEval(id string, attempts int, cause error) {
	status, delay := evalBackoff(attempts)
	if _, err := e.db.Exec(`UPDATE translation_evals
		SET status = $1, last_error = $2, processing_at = NULL,
		    next_attempt_at = CURRENT_TIMESTAMP + ($3 * INTERVAL '1 second')
		WHERE id = $4`, status, cause.Error(), int(delay.Seconds()), id); err != nil {
		log.Printf("[Eval] mark translation eval failed %s: %v", id, err)
	}
}

func (e *QualityEvaluatorService) failGrammarEval(id string, attempts int, cause error) {
	status, delay := evalBackoff(attempts)
	if _, err := e.db.Exec(`UPDATE grammar_evals
		SET status = $1, last_error = $2, processing_at = NULL,
		    next_attempt_at = CURRENT_TIMESTAMP + ($3 * INTERVAL '1 second')
		WHERE id = $4`, status, cause.Error(), int(delay.Seconds()), id); err != nil {
		log.Printf("[Eval] mark grammar eval failed %s: %v", id, err)
	}
}

// evalBackoff computes the failure status + retry delay for an eval attempt.
func evalBackoff(attempts int) (string, time.Duration) {
	delay := evalMaxPending
	if attempts < evalMaxAttempts {
		delay = evalBaseDelay
		for i := 1; i < attempts; i++ {
			if delay >= evalMaxDelay {
				delay = evalMaxDelay
				break
			}
			delay *= 2
		}
	}
	return "failed", delay
}

// ---------------------------------------------------------------------------
// Pub/sub trigger + sweeper + recovery
// ---------------------------------------------------------------------------

func (e *QualityEvaluatorService) trigger(env evalEnvelope) {
	if e.redis != nil {
		payload, _ := json.Marshal(env)
		if err := e.redis.Publish(e.ctx, evalChannel, payload).Err(); err != nil {
			log.Printf("[Eval] publish %s: %v (sweeper will re-queue)", env.ID, err)
		}
		return
	}
	e.spawn(env)
}

func (e *QualityEvaluatorService) spawn(env evalEnvelope) {
	e.sema <- struct{}{}
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		defer func() { <-e.sema }()
		e.process(env)
	}()
}

// subscribeLoop consumes published eval triggers.
func (e *QualityEvaluatorService) subscribeLoop() {
	defer e.wg.Done()
	sub := e.redis.Subscribe(e.ctx, evalChannel)
	defer sub.Close()
	ch := sub.Channel()
	for {
		select {
		case <-e.ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			var env evalEnvelope
			if err := json.Unmarshal([]byte(msg.Payload), &env); err != nil || env.ID == "" {
				continue
			}
			e.spawn(env)
		}
	}
}

// sweepLoop retries due pending/failed evals and reclaims stale processing rows.
func (e *QualityEvaluatorService) sweepLoop() {
	defer e.wg.Done()
	ticker := time.NewTicker(evalSweepEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			e.sweep()
		case <-e.stopCh:
			return
		}
	}
}

func (e *QualityEvaluatorService) sweep() {
	if _, err := e.db.Exec(`UPDATE translation_evals SET status = 'pending', processing_at = NULL
		WHERE status = 'processing' AND processing_at < CURRENT_TIMESTAMP - ($1 * INTERVAL '1 minute')`,
		int(evalStaleTTL.Minutes())); err != nil {
		log.Printf("[Eval] reclaim stale translation evals: %v", err)
	}
	if _, err := e.db.Exec(`UPDATE grammar_evals SET status = 'pending', processing_at = NULL
		WHERE status = 'processing' AND processing_at < CURRENT_TIMESTAMP - ($1 * INTERVAL '1 minute')`,
		int(evalStaleTTL.Minutes())); err != nil {
		log.Printf("[Eval] reclaim stale grammar evals: %v", err)
	}
	rows, err := e.db.Query(`
		SELECT 'translation' AS kind, id FROM translation_evals
		WHERE status IN ('pending', 'failed') AND next_attempt_at <= CURRENT_TIMESTAMP
		UNION ALL
		SELECT 'grammar', id FROM grammar_evals
		WHERE status IN ('pending', 'failed') AND next_attempt_at <= CURRENT_TIMESTAMP
		ORDER BY kind
		LIMIT 50`)
	if err != nil {
		log.Printf("[Eval] sweep due: %v", err)
		return
	}
	var envs []evalEnvelope
	for rows.Next() {
		var kind, id string
		if err := rows.Scan(&kind, &id); err == nil {
			envs = append(envs, evalEnvelope{Kind: evalKind(kind), ID: id})
		}
	}
	rows.Close()
	for _, env := range envs {
		e.trigger(env)
	}
}

// recover re-queues eval work left in 'processing' after a restart and publishes
// any due pending/failed evals.
func (e *QualityEvaluatorService) recover() {
	if _, err := e.db.Exec(`UPDATE translation_evals SET status = 'pending', processing_at = NULL
		WHERE status = 'processing'`); err != nil {
		log.Printf("[Eval] recover translation evals: %v", err)
	}
	if _, err := e.db.Exec(`UPDATE grammar_evals SET status = 'pending', processing_at = NULL
		WHERE status = 'processing'`); err != nil {
		log.Printf("[Eval] recover grammar evals: %v", err)
	}
	rows, err := e.db.Query(`
		SELECT 'translation', id FROM translation_evals
		WHERE status IN ('pending', 'failed')
		UNION ALL
		SELECT 'grammar', id FROM grammar_evals
		WHERE status IN ('pending', 'failed')`)
	if err != nil {
		log.Printf("[Eval] recover load incomplete: %v", err)
		return
	}
	var envs []evalEnvelope
	for rows.Next() {
		var kind, id string
		if err := rows.Scan(&kind, &id); err == nil {
			envs = append(envs, evalEnvelope{Kind: evalKind(kind), ID: id})
		}
	}
	rows.Close()
	log.Printf("[Eval] recover: %d incomplete eval(s) re-queued", len(envs))
	for _, env := range envs {
		e.trigger(env)
	}
}

// ---------------------------------------------------------------------------
// KPIs (FR-30: accuracy, latency, cost, cache hit rate)
// ---------------------------------------------------------------------------

// KPIs aggregates the quality metrics for the admin console.
func (e *QualityEvaluatorService) KPIs() (*models.QualityKPIs, error) {
	k := &models.QualityKPIs{}
	k.CostPer1KTokens = e.costPer1K

	// Accuracy + fluency + evaluated count from translation evals.
	err := e.db.QueryRow(`SELECT COUNT(*), COALESCE(AVG(accuracy_score),0), COALESCE(AVG(fluency_score),0)
		FROM translation_evals WHERE status = 'done'`).
		Scan(&k.Evaluations, &k.AvgAccuracy, &k.AvgFluency)
	if err != nil {
		return nil, err
	}

	// Total translations, p95 latency, cache hit rate, avg tokens from jobs.
	var p95 float64
	err = e.db.QueryRow(`SELECT
		COUNT(*),
		COALESCE(AVG(tokens),0),
		COALESCE(100.0 * AVG(cache_hit::int), 0)
		FROM translation_jobs WHERE status = 'done'`).
		Scan(&k.TotalTranslations, &k.AvgTokens, &k.CacheHitRate)
	if err != nil {
		return nil, err
	}
	// p95 latency over completed translation jobs (percentile_cont needs the
	// array in order; simpler: ORDER BY with offset). Fall back to 0 on error.
	err = e.db.QueryRow(`
		SELECT COALESCE((
			SELECT latency_ms FROM translation_jobs
			WHERE status = 'done' AND latency_ms IS NOT NULL
			ORDER BY latency_ms
			OFFSET GREATEST(0, (SELECT COUNT(*) FROM translation_jobs WHERE status = 'done' AND latency_ms IS NOT NULL) * 95 / 100 - 1)
			LIMIT 1), 0)`).Scan(&p95)
	if err == nil {
		k.P95LatencyMS = float64(p95)
	}

	// Count evaluated vs total for a coverage signal.
	if err := e.db.QueryRow(`
		SELECT COUNT(*) FROM translation_evals WHERE status = 'done'
		  AND translation_job_id IS NOT NULL`).Scan(&k.EvaluatedTranslations); err != nil {
		k.EvaluatedTranslations = 0
	}
	k.EstCostUSD = (k.AvgTokens * float64(k.TotalTranslations) / 1000.0) * k.CostPer1KTokens
	return k, nil
}

// RequeueUnevaluated (re)scores translation jobs that have no eval row before
// a batch/sample backfill. It enqueues the most recent `limit` unevaluated done
// jobs and returns how many were enqueued. This is the FR-30 "nightly batch
// re-scores a sample" affordance, exposed on demand from the admin console.
func (e *QualityEvaluatorService) RequeueUnevaluated(limit int) (int, error) {
	if !e.Enabled() {
		return 0, nil
	}
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	count := 0
	// Translation jobs lacking a translation_eval row.
	rows, err := e.db.Query(`
		SELECT j.id FROM translation_jobs j
		WHERE j.status = 'done'
		  AND NOT EXISTS (SELECT 1 FROM translation_evals ev WHERE ev.translation_job_id = j.id)
		ORDER BY j.created_at DESC
		LIMIT $1`, limit)
	if err != nil {
		return count, err
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
		if err := e.EnqueueTranslation(id); err == nil {
			count++
		}
	}

	// Grammar jobs lacking a grammar_eval row.
	grows, err := e.db.Query(`
		SELECT j.id FROM grammar_jobs j
		WHERE j.status = 'done'
		  AND NOT EXISTS (SELECT 1 FROM grammar_evals ev WHERE ev.grammar_job_id = j.id)
		ORDER BY j.created_at DESC
		LIMIT $1`, limit)
	if err != nil {
		return count, err
	}
	var gids []string
	for grows.Next() {
		var id string
		if err := grows.Scan(&id); err == nil {
			gids = append(gids, id)
		}
	}
	grows.Close()
	for _, id := range gids {
		if err := e.EnqueueGrammar(id); err == nil {
			count++
		}
	}
	return count, nil
}

// num coerces a JSON number (float64 from decoding) into a float64 safely.
func num(v interface{}) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case int:
		return float64(x), true
	default:
		return 0, false
	}
}

// evalCostPer1K reads the configured USD-per-1k-token rate for cost KPIs.
func evalCostPer1K() float64 {
	rate := defaultCostPer1K
	if v := os.Getenv(evalCostPer1KEnv); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f >= 0 {
			rate = f
		}
	}
	return rate
}
