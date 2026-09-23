package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/chorus/messenger/internal/models"
)

// ---------------------------------------------------------------------------
// GrammarPointService
// ---------------------------------------------------------------------------

func TestGrammarPointService_GetDueItems_UnseenFirst(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()
	svc := NewGrammarPointService(db, nil)

	rows := sqlmock.NewRows([]string{
		"id::text", "grammar_point_id::text", "gp.title", "gp.cefr_level", "gi.ordinal", "gi.item_type",
		"gi.prompt", "sentence_with_blank", "choices", "correct", "note",
	})
	rows.AddRow("i1", "p1", "Ser vs estar", "A1", 1, "cloze", "Complete with ser.", "Pedro _____ de Madrid.", "{es,esta,son}", "es", "3rd person singular")
	rows.AddRow("i2", "p1", "Ser vs estar", "A1", 2, "mcq", "Pick the pronoun.", "¿De dónde _____ vosotros?", "{sois,somos,son,eres}", "sois", "vosotros -> sois")
	rows.AddRow("i3", "p1", "Ser vs estar", "A1", 3, "cloze", "Complete with ser.", "Nosotros _____ amigos.", "{somos,es,son}", "somos", "nosotros -> somos")
	rows.AddRow("i4", "p2", "Preterite", "A2", 1, "cloze", "Complete in preterite.", "Ayer _____ al cine.", "{fui,iba,voy}", "fui", "ir preterite")

	mock.ExpectQuery("SELECT gi.id::text, gi.grammar_point_id::text").
		WithArgs("user-1", "es", 3).
		WillReturnRows(rows)

	items, err := svc.GetDueItems(context.Background(), "user-1", "es", 3)
	if err != nil {
		t.Fatalf("GetDueItems: %v", err)
	}
	// 4 rows in, but max 2 drill items per grammar point per session.
	if len(items) != 3 {
		t.Fatalf("expected 3 items (2 for p1, 1 for p2), got %d", len(items))
	}
	if items[0].GrammarPointID != "p1" || items[0].Correct != "es" {
		t.Errorf("unexpected first item: %+v", items[0])
	}
	if len(items[0].Choices) != 3 {
		t.Errorf("choices not scanned: %+v", items[0].Choices)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestGrammarPointService_RecordItemAttempt_SM2Progression(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()
	svc := NewGrammarPointService(db, nil)

	// item -> point/course lookup.
	mock.ExpectQuery("SELECT gi.grammar_point_id::text, c.target_language").
		WithArgs("item-1").
		WillReturnRows(sqlmock.NewRows([]string{"grammar_point_id", "target_language"}).AddRow("p1", "es"))
	// attempt insert.
	mock.ExpectExec("INSERT INTO user_grammar_item_attempts").
		WithArgs("user-1", "item-1", "p1", true, 5).
		WillReturnResult(sqlmock.NewResult(1, 1))
	// mastery read: first sighting (no row).
	mock.ExpectQuery("SELECT mastery_stage, ease_factor, repetitions, interval_days").
		WithArgs("user-1", "p1").
		WillReturnError(sql.ErrNoRows)
	// mastery upsert: quality 5 -> repetitions 1, interval 1, stage 2.
	mock.ExpectExec("INSERT INTO user_grammar_mastery").
		WithArgs("user-1", "p1", "es", 0.3, 1, 0, 2, 2.6, 1, 1, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := svc.RecordItemAttempt(context.Background(), "user-1", "item-1", true, 5); err != nil {
		t.Fatalf("RecordItemAttempt: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestGrammarPointService_RecordItemAttempt_LapseResets(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()
	svc := NewGrammarPointService(db, nil)

	mock.ExpectQuery("SELECT gi.grammar_point_id::text, c.target_language").
		WithArgs("item-2").
		WillReturnRows(sqlmock.NewRows([]string{"grammar_point_id", "target_language"}).AddRow("p2", "es"))
	mock.ExpectExec("INSERT INTO user_grammar_item_attempts").
		WithArgs("user-1", "item-2", "p2", false, 1).
		WillReturnResult(sqlmock.NewResult(1, 1))
	// Existing state: stage 4, ease 2.5, reps 3, interval 15.
	mock.ExpectQuery("SELECT mastery_stage, ease_factor, repetitions, interval_days").
		WithArgs("user-1", "p2").
		WillReturnRows(sqlmock.NewRows([]string{"mastery_stage", "ease_factor", "repetitions", "interval_days"}).AddRow(4, 2.5, 3, 15))
	// quality 1 -> reps reset to 0, interval 1, stage drops to 3.
	mock.ExpectExec("INSERT INTO user_grammar_mastery").
		WithArgs("user-1", "p2", "es", 0.3, 0, 1, 3, 2.5, 0, 1, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := svc.RecordItemAttempt(context.Background(), "user-1", "item-2", false, 1); err != nil {
		t.Fatalf("RecordItemAttempt: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestGradeDrillAnswer_AccentPolicy(t *testing.T) {
	drill := models.GrammarDrillItem{Correct: "está"}
	if !GradeDrillAnswer(drill, "Esta", "A1") {
		t.Errorf("A1 should forgive missing accents")
	}
	if GradeDrillAnswer(drill, "Esta", "B1") {
		t.Errorf("B1+ should require accents")
	}
	if !GradeDrillAnswer(drill, "Está", "B2") {
		t.Errorf("accented answer should pass at B2")
	}
}

// ---------------------------------------------------------------------------
// GradingQueueService
// ---------------------------------------------------------------------------

type fakeGrader struct {
	result *models.GradingJobResult
	err    error
	calls  int
}

func (f *fakeGrader) GradeProduction(ctx context.Context, p models.GradingJobPayload) (*models.GradingJobResult, string, error) {
	f.calls++
	return f.result, "fake", f.err
}

func TestGradingQueue_EnqueueAndProcess(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	notifyCalls := 0
	grader := &fakeGrader{result: &models.GradingJobResult{Score: 8, MaxScore: 10, Feedback: "Great work."}}
	q := NewGradingQueueService(db, nil, grader, func(userID string, payload *models.GradingJobResult) {
		notifyCalls++
	})

	payload := models.GradingJobPayload{TaskType: "writing", Prompt: "Describe your city", UserText: "Mi ciudad es grande y bonita.", TargetLanguage: "es", NativeLanguage: "en"}
	payloadJSON, _ := jsonMarshal(payload)

	// Enqueue insert.
	mock.ExpectQuery("INSERT INTO grading_jobs").
		WithArgs("user-1", "writing", string(payloadJSON), nil).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("job-1"))

	jobID, err := q.Enqueue(context.Background(), "user-1", "writing", payload, "")
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	if jobID != "job-1" {
		t.Fatalf("expected job-1, got %s", jobID)
	}

	// processJob: load the claimed job.
	mock.ExpectQuery("SELECT id::text, user_id, COALESCE\\(submission_id::text,''\\)").
		WithArgs("job-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "submission_id", "job_type", "payload", "attempts"}).
			AddRow("job-1", "user-1", "", "writing", payloadJSON, 1))
	// finishJob: mark done.
	mock.ExpectExec("UPDATE grading_jobs").
		WithArgs(sqlmock.AnyArg(), "job-1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	q.processJob("job-1")
	if grader.calls != 1 {
		t.Fatalf("grader should have run once, ran %d", grader.calls)
	}
	if notifyCalls != 1 {
		t.Fatalf("notify should fire once on completion, fired %d", notifyCalls)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestGradingQueue_FailedGraderRetries(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	grader := &fakeGrader{err: errors.New("upstream down")}
	q := NewGradingQueueService(db, nil, grader, nil)

	payloadJSON, _ := jsonMarshal(models.GradingJobPayload{TaskType: "production", UserText: "hola", TargetLanguage: "es", NativeLanguage: "en"})
	mock.ExpectQuery("SELECT id::text, user_id, COALESCE\\(submission_id::text,''\\)").
		WithArgs("job-2").
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "submission_id", "job_type", "payload", "attempts"}).
			AddRow("job-2", "user-1", "", "production", payloadJSON, 1))
	mock.ExpectExec("UPDATE grading_jobs").
		WillReturnResult(sqlmock.NewResult(1, 1))

	q.processJob("job-2")
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestGradingQueue_HeuristicFallbackGrades(t *testing.T) {
	ai := NewLearningAIService(nil) // no endpoints
	res, provider, err := ai.GradeProduction(context.Background(), models.GradingJobPayload{
		TaskType: "writing", Prompt: "p", UserText: "uno dos tres cuatro cinco seis siete ocho nueve diez once doce trece catorce quince dieciseis diecisiete dieciocho",
		TargetLanguage: "es", NativeLanguage: "en", CEFRLevel: "A2",
	})
	if err != nil {
		t.Fatalf("GradeProduction: %v", err)
	}
	if provider != "heuristic-fallback" {
		t.Errorf("expected heuristic-fallback, got %q", provider)
	}
	if res.Score < 4 || res.MaxScore != 10 {
		t.Errorf("expected a passing engagement score, got %d/%d", res.Score, res.MaxScore)
	}
}

// ---------------------------------------------------------------------------
// RealTalkService
// ---------------------------------------------------------------------------

type fakeRealTalkCache struct {
	store map[string]string
}

func (f *fakeRealTalkCache) Get(ctx context.Context, key string) (string, error) {
	if v, ok := f.store[key]; ok {
		return v, nil
	}
	return "", errors.New("redis: nil")
}

func (f *fakeRealTalkCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	if f.store == nil {
		f.store = map[string]string{}
	}
	if s, ok := value.(string); ok {
		f.store[key] = s
	}
	return nil
}

func TestRealTalkService_DailyCache(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()
	cache := &fakeRealTalkCache{}
	svc := NewRealTalkService(db, cache)

	rows := sqlmock.NewRows([]string{"id::text", "category", "prompt_for_learner", "target_phrase", "why_useful", "follow_up_chunks"}).
		AddRow("rt-1", "conversation_starter", "Ask how the day is going.", "¿Qué tal el día?", "Natural opener.", `["Todo bien","Bastante ocupado"]`).
		AddRow("rt-2", "topic_injector", "Pivot to weekend plans.", "Por cierto, ¿planes?", "Smooth pivot.", `["Pues nada especial"]`)
	mock.ExpectQuery("SELECT id::text, category, prompt_for_learner").
		WithArgs("course-1", "A1").
		WillReturnRows(rows)

	prompts, err := svc.DailyPromptSet(context.Background(), "course-1", "A1")
	if err != nil {
		t.Fatalf("DailyPromptSet: %v", err)
	}
	if len(prompts) != 2 {
		t.Fatalf("expected 2 prompts, got %d", len(prompts))
	}
	if prompts[0].Category != "Icebreakers" || prompts[1].Category != "Deep Dives" {
		t.Errorf("category labels not mapped: %+v", prompts)
	}
	if prompts[0].TargetPhrase != "¿Qué tal el día?" || len(prompts[0].FollowUpChunks) != 2 {
		t.Errorf("enrichment fields missing: %+v", prompts[0])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}

	// Second call of the day must be served from cache (no DB expectations).
	cached, err := svc.DailyPromptSet(context.Background(), "course-1", "A1")
	if err != nil {
		t.Fatalf("cached DailyPromptSet: %v", err)
	}
	if len(cached) != 2 {
		t.Fatalf("cached set should hold 2 prompts, got %d", len(cached))
	}
}

// ---------------------------------------------------------------------------
// TeacherAssignmentService
// ---------------------------------------------------------------------------

func TestTeacherAssignment_CreateRequiresRelationship(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()
	svc := NewTeacherAssignmentService(db, nil)

	// No prior booking between the pair.
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("teacher-1", "student-1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	_, err = svc.CreateAssignment(context.Background(), "teacher-1", models.CreateTeacherAssignmentRequest{
		StudentID: "student-1", Type: "writing", Title: "Weekend essay",
		Instructions: "Write 5 sentences.", Content: map[string]any{"text": "Describe your weekend."},
		TargetLanguage: "es", NativeLanguage: "en",
	})
	if err == nil {
		t.Fatalf("expected relationship validation error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Session snapshot cache (transient Redis layer)
// ---------------------------------------------------------------------------

type fakeSessionCache struct {
	store map[string]string
}

func (f *fakeSessionCache) Get(ctx context.Context, key string) (string, error) {
	if v, ok := f.store[key]; ok {
		return v, nil
	}
	return "", errors.New("redis: nil")
}

func (f *fakeSessionCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	if f.store == nil {
		f.store = map[string]string{}
	}
	if s, ok := value.(string); ok {
		f.store[key] = s
	}
	return nil
}

func (f *fakeSessionCache) Del(ctx context.Context, keys ...string) error {
	for _, k := range keys {
		delete(f.store, k)
	}
	return nil
}

func TestSessionComposer_SnapshotRoundTrip(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()
	svc := NewSessionComposerService(db, nil, nil, nil, nil, nil)
	cache := &fakeSessionCache{}
	svc.SetSessionCache(cache)

	snap := &sessionSnapshot{
		SessionID: "sess-1", UserID: "user-1", Mode: "daily", Status: "in_progress",
		TargetLang: "es", Planned: 3, Completed: 1, Score: 150,
		Items: []snapshotItem{
			{ID: "it-1", Ordinal: 1, ItemType: "vocabulary", Status: "answered"},
			{ID: "it-2", Ordinal: 2, ItemType: "grammar", Status: "pending"},
			{ID: "it-3", Ordinal: 3, ItemType: "lesson_step", Status: "pending"},
		},
	}
	svc.saveSnapshot(context.Background(), snap)

	// Cached read serves the session without touching the DB.
	got := svc.loadSnapshot(context.Background(), "sess-1")
	if got == nil {
		t.Fatalf("snapshot should be cached")
	}
	if got.Completed != 1 || got.Planned != 3 || len(got.Items) != 3 {
		t.Fatalf("snapshot corrupted: %+v", got)
	}

	// Wrong owner is rejected (privacy).
	got2 := svc.loadSnapshot(context.Background(), "nope")
	if got2 != nil {
		t.Fatalf("unknown session should miss cache")
	}

	// syncSnapshot advances item state + counters.
	svc.syncSnapshot(context.Background(), "sess-1", "it-2", 200)
	got3 := svc.loadSnapshot(context.Background(), "sess-1")
	if got3.Completed != 2 || got3.Score != 350 {
		t.Fatalf("syncSnapshot did not advance counters: %+v", got3)
	}
	if got3.Items[1].Status != "answered" {
		t.Fatalf("syncSnapshot did not mark item answered: %+v", got3.Items[1])
	}

	// Completion drops the transient state.
	svc.dropSnapshot(context.Background(), "sess-1")
	if svc.loadSnapshot(context.Background(), "sess-1") != nil {
		t.Fatalf("dropSnapshot should clear the cache")
	}
}

func jsonMarshal(v any) ([]byte, error) {
	return json.Marshal(v)
}
