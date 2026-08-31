package services

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestQualityEvaluator_Disabled_Noop(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	// No endpoints -> Enqueue is a no-op even if called.
	e := NewQualityEvaluatorService(db, nil, nil)
	if e.Enabled() {
		t.Fatal("expected evaluator disabled with no endpoints")
	}
	if err := e.EnqueueTranslation("job-1"); err != nil {
		t.Fatalf("expected no-op, got %v", err)
	}
	if err := e.EnqueueGrammar("job-1"); err != nil {
		t.Fatalf("expected no-op, got %v", err)
	}
}

func TestQualityEvaluator_EnqueueTranslationInsertsRow(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	e := NewQualityEvaluatorService(db, deadRedis(), []GrammarEndpoint{
		NewGrammarEndpoint("openrouter", "openai", "https://openrouter.ai/api/v1", "", "gpt-4o-mini", 5),
	})

	mock.ExpectQuery(`INSERT INTO translation_evals.*RETURNING id`).
		WithArgs("job-1").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("eval-1"))

	if err := e.EnqueueTranslation("job-1"); err != nil {
		t.Fatalf("EnqueueTranslation failed: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestQualityEvaluator_EnqueueGrammarInsertsRow(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	e := NewQualityEvaluatorService(db, deadRedis(), []GrammarEndpoint{
		NewGrammarEndpoint("openrouter", "openai", "https://openrouter.ai/api/v1", "", "gpt-4o-mini", 5),
	})

	mock.ExpectQuery(`INSERT INTO grammar_evals.*RETURNING id`).
		WithArgs("grammar-1").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("eval-1"))

	if err := e.EnqueueGrammar("grammar-1"); err != nil {
		t.Fatalf("EnqueueGrammar failed: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestQualityEvaluator_PickEndpointPrefersDifferentModel(t *testing.T) {
	e := &QualityEvaluatorService{endpoints: []GrammarEndpoint{
		NewGrammarEndpoint("openrouter", "openai", "https://openrouter.ai", "", "gpt-4o-mini", 5),
		NewGrammarEndpoint("ollama", "ollama", "http://localhost:11434", "", "qwen2.5:3b", 5),
	}}
	// Producer is the model named openrouter -> evaluator should pick ollama.
	if got := e.pickEndpoint("openai(gpt-4o-mini)"); got.Name != "openrouter" {
		t.Fatalf("pickEndpoint: expected a first non-producer endpoint, got %q", got.Name)
	}
	// Producer matching nothing -> first endpoint.
	if got := e.pickEndpoint("nvidia"); got.Name != "openrouter" {
		t.Fatalf("pickEndpoint: expected first endpoint for unknown producer, got %q", got.Name)
	}
}

func TestQualityEvaluator_KPIs(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	e := NewQualityEvaluatorService(db, nil, nil)
	now := time.Now()

	mock.ExpectQuery(`SELECT COUNT\(\*\), COALESCE\(AVG\(accuracy_score\),0\), COALESCE\(AVG\(fluency_score\),0\) FROM translation_evals WHERE status = 'done'`).
		WillReturnRows(sqlmock.NewRows([]string{"count", "avg_accuracy", "avg_fluency"}).AddRow(10, 85.5, 88.25))

	mock.ExpectQuery(`SELECT COUNT\(\*\), COALESCE\(AVG\(tokens\),0\), COALESCE\(100\.0 \* AVG\(cache_hit::int\), 0\) FROM translation_jobs WHERE status = 'done'`).
		WillReturnRows(sqlmock.NewRows([]string{"count", "avg_tokens", "cache_hit"}).AddRow(100, 42, 35.5))

	mock.ExpectQuery(`SELECT COALESCE\(\s*\(.*latency_ms.*ORDER BY latency_ms.*LIMIT 1\), 0\)`).
		WillReturnRows(sqlmock.NewRows([]string{"p95"}).AddRow(float64(150)))

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM translation_evals WHERE status = 'done' AND translation_job_id IS NOT NULL`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(10))

	k, err := e.KPIs()
	if err != nil {
		t.Fatalf("KPIs failed: %v", err)
	}
	if k.Evaluations != 10 || k.AvgAccuracy != 85.5 || k.AvgFluency != 88.25 {
		t.Fatalf("unexpected kpi eval figures: %+v", k)
	}
	if k.TotalTranslations != 100 || k.AvgTokens != 42 || k.CacheHitRate != 35.5 {
		t.Fatalf("unexpected kpi job figures: %+v", k)
	}
	if k.P95LatencyMS != 150 {
		t.Fatalf("expected p95 150, got %v", k.P95LatencyMS)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
	_ = now
}

func TestQualityEvaluator_EvalBackoff(t *testing.T) {
	// Attempt 2 of 5 -> delay doubles once: 40s.
	status, delay := evalBackoff(2)
	if status != "failed" || delay != 40*time.Second {
		t.Fatalf("unexpected backoff for attempt 2: %s %v", status, delay)
	}
	// Attempt 5 of 5 -> exhausted: max-pending (24h).
	status, delay = evalBackoff(5)
	if status != "failed" || delay != evalMaxPending {
		t.Fatalf("unexpected backoff for exhausted attempt: %s %v", status, delay)
	}
}

func TestQualityEvaluator_RequeueUnevaluated(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	e := NewQualityEvaluatorService(db, deadRedis(), []GrammarEndpoint{
		NewGrammarEndpoint("openrouter", "openai", "https://openrouter.ai/api/v1", "", "gpt-4o-mini", 5),
	})

	// Translation jobs query returns one job, which EnqueueTranslation then inserts.
	mock.ExpectQuery(`SELECT j\.id FROM translation_jobs j.*NOT EXISTS.*LIMIT \$1`).
		WithArgs(200).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("job-1"))
	mock.ExpectQuery(`INSERT INTO translation_evals.*RETURNING id`).
		WithArgs("job-1").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("eval-1"))

	// Grammar jobs query returns none.
	mock.ExpectQuery(`SELECT j\.id FROM grammar_jobs j.*NOT EXISTS.*LIMIT \$1`).
		WithArgs(200).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	count, err := e.RequeueUnevaluated(200)
	if err != nil {
		t.Fatalf("RequeueUnevaluated failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 requeued eval, got %d", count)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
