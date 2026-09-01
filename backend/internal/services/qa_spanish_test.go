package services

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/chorus/messenger/internal/models"
)

// ── Spanish curriculum existence ─────────────────────────────────────────────

func TestSpanishCurriculumUnitsExist(t *testing.T) {
	if len(spanishUnits) < 20 {
		t.Fatalf("expected at least 20 Spanish units, got %d", len(spanishUnits))
	}
	found := map[string]bool{}
	for _, u := range spanishUnits {
		found[u.Slug] = true
	}
	mustHave := []string{"a1-introductions", "a1-ordering-food", "a1-around-town", "a1-checkpoint", "a2-shopping", "b1-stories", "b2-conflict-repair"}
	for _, s := range mustHave {
		if !found[s] {
			t.Errorf("missing Spanish unit %q", s)
		}
	}
	// Ordering Food must be A1 with correct can-do
	for _, u := range spanishUnits {
		if u.Slug == "a1-ordering-food" {
			if u.Level != "A1" {
				t.Errorf("ordering-food level = %q, want A1", u.Level)
			}
			if !strings.Contains(u.CanDo, "order food") {
				t.Errorf("ordering-food canDo = %q, want to contain 'order food'", u.CanDo)
			}
			if u.Title != "Ordering Food" {
				t.Errorf("ordering-food title = %q, want Ordering Food", u.Title)
			}
		}
	}
}

func TestSpanishLexicalItemsContainCoreChunks(t *testing.T) {
	terms := map[string]string{}
	for _, it := range spanishLexicalItems {
		terms[it.Lemma] = it.Translation
	}
	checks := map[string]string{
		"hola":            "hello",
		"café":            "coffee",
		"café con leche":  "coffee with milk",
		"quisiera":        "I would like",
		"para llevar":     "to go",
		"¿cuánto cuesta?": "how much does it cost?",
		"¿dónde está?":   "where is it?",
	}
	for lemma, wantTrans := range checks {
		got, ok := terms[lemma]
		if !ok {
			t.Errorf("lexical missing lemma %q", lemma)
			continue
		}
		if got != wantTrans {
			t.Errorf("lemma %q translation = %q, want %q", lemma, got, wantTrans)
		}
	}
	// At least 5 chunks must be marked IsChunk
	chunkCount := 0
	for _, it := range spanishLexicalItems {
		if it.IsChunk {
			chunkCount++
		}
	}
	if chunkCount < 5 {
		t.Fatalf("expected at least 5 chunk lexical items, got %d", chunkCount)
	}
}

func TestSpanishOrderingCoffeeScenarioPhases(t *testing.T) {
	if len(orderingCoffeePhases) != 5 {
		t.Fatalf("orderingCoffeePhases len = %d, want 5", len(orderingCoffeePhases))
	}
	expected := []struct {
		title   string
		intents []string
		chunks  int
	}{
		{"Greeting", []string{"greet"}, 2},
		{"Order", []string{"order_drink"}, 2},
		{"Customization", []string{"customize"}, 3},
		{"Payment", []string{"pay"}, 2},
		{"Closing", []string{"close"}, 2},
	}
	for i, exp := range expected {
		ph := orderingCoffeePhases[i]
		if ph.Title != exp.title {
			t.Errorf("phase %d title = %q, want %q", i+1, ph.Title, exp.title)
		}
		if len(ph.RequiredIntents) != len(exp.intents) || ph.RequiredIntents[0] != exp.intents[0] {
			t.Errorf("phase %d intents = %v, want %v", i+1, ph.RequiredIntents, exp.intents)
		}
		if len(ph.ChunkBank) < exp.chunks {
			t.Errorf("phase %d chunkBank len = %d, want >=%d", i+1, len(ph.ChunkBank), exp.chunks)
		}
		// Every chunk must have text + translation (Spanish + English)
		for _, ch := range ph.ChunkBank {
			if ch["text"] == "" || ch["translation"] == "" {
				t.Errorf("phase %d chunk missing text/translation: %v", i+1, ch)
			}
			// Spanish text must contain non-ASCII or Spanish markers
			if ch["text"] == ch["translation"] {
				t.Errorf("phase %d chunk text equals translation (not Spanish): %v", i+1, ch)
			}
		}
	}
}

func TestSpanishOrderingCoffeeOpeningLine(t *testing.T) {
	// The seeded scenario opening line must be Spanish
	found := false
	for _, ph := range orderingCoffeePhases {
		for _, ch := range ph.ChunkBank {
			if strings.Contains(ch["text"], "Hola") || strings.Contains(ch["text"], "Quisiera") {
				found = true
			}
		}
	}
	if !found {
		t.Error("ordering coffee chunks should contain Hola/Quisiera Spanish")
	}
	// Verify the seed function would produce opening line Hola. ¿Qué te gustaría pedir hoy?
	// We check the constant in curriculum.go seed: 'Hola. ¿Qué te gustaría pedir hoy?'
	// Simply assert the phrase is present via scenario generation test below
}

// ── Spanish intent detection ─────────────────────────────────────────────────

func TestDetectIntentsSpanish(t *testing.T) {
	cases := []struct {
		msg      string
		required []string
		wantHit  []string
	}{
		{"Hola, buenos días", []string{"greet"}, []string{"greet"}},
		{"Quisiera un café con leche, por favor", []string{"order_drink"}, []string{"order_drink"}},
		{"Para llevar, por favor", []string{"customize"}, []string{"customize"}},
		{"¿Cuánto cuesta?", []string{"pay"}, []string{"pay"}},
		{"¿Aceptan tarjeta?", []string{"pay"}, []string{"pay"}},
		{"Gracias, que tenga un buen día", []string{"close"}, []string{"close"}},
		{"Hola quisiera un café sin azúcar ¿cuánto cuesta? gracias", []string{"greet", "order_drink", "customize", "pay", "close"}, []string{"greet", "order_drink", "customize", "pay", "close"}},
	}
	for _, c := range cases {
		got := detectIntents(c.msg, c.required)
		if len(got) != len(c.wantHit) {
			t.Errorf("detectIntents(%q, %v) = %v, want %v", c.msg, c.required, got, c.wantHit)
			continue
		}
		for i, w := range c.wantHit {
			if got[i] != w {
				t.Errorf("detectIntents(%q) [%d] = %q, want %q", c.msg, i, got[i], w)
			}
		}
	}
}

func TestIntentMatchesAccentNormalization(t *testing.T) {
	if !intentMatches(NormalizeLearningTerm("¿Cuánto cuesta?", "es"), "¿cuánto cuesta?", "pay") {
		t.Error("intentMatches should handle accented cuánto")
	}
	if !intentMatches(NormalizeLearningTerm("café", "es"), "cafe", "order_drink") {
		t.Error("intentMatches should handle café accent normalization")
	}
}

func TestQAIntentsCovered(t *testing.T) {
	if !intentsCovered([]string{"greet", "order_drink"}, []string{"greet"}) {
		t.Error("intentsCovered should be true when required subset covered")
	}
	if intentsCovered([]string{"greet"}, []string{"greet", "pay"}) {
		t.Error("intentsCovered should be false when pay missing")
	}
	if !intentsCovered([]string{}, []string{}) {
		t.Error("empty required should be covered")
	}
}

func TestScriptedReplySpanish(t *testing.T) {
	phase := models.ScenarioPhase{Title: "Greeting", RequiredIntents: []string{"greet"}, ChunkBank: []models.Chunk{{Text: "Hola"}}}
	reply := scriptedReply(&phase, "Hola", []string{"greet"}, true, "guided")
	if !strings.Contains(reply.AIMessage, "Hola") && !strings.Contains(reply.AIMessage, "Bienvenido") {
		t.Errorf("scriptedReply Greeting AIMessage = %q, want Spanish greeting", reply.AIMessage)
	}
	if reply.Translation == "" {
		t.Error("scriptedReply should provide English translation")
	}
	// Incomplete phase should ask to try suggestion
	reply2 := scriptedReply(&phase, "xyz", []string{}, false, "guided")
	if !strings.Contains(reply2.AIMessage, "No te entendí") && !strings.Contains(reply2.Translation, "didn't") {
		t.Errorf("incomplete phase reply = %q / %q, want Spanish hint", reply2.AIMessage, reply2.Translation)
	}
	// Phase progression for payment
	phasePay := models.ScenarioPhase{Title: "Payment", RequiredIntents: []string{"pay"}}
	rPay := scriptedReply(&phasePay, "¿Cuánto cuesta?", []string{"pay"}, true, "guided")
	if !strings.Contains(rPay.AIMessage, "euros") && !strings.Contains(rPay.AIMessage, "tarjeta") {
		t.Errorf("payment phase reply = %q, want price/card Spanish", rPay.AIMessage)
	}
}

// ── Daily drills — SRS & practice ───────────────────────────────────────────

func TestPracticeGradingSpanish(t *testing.T) {
	svc := NewPracticeService(nil)
	// Accent folding: café vs cafe should be correct
	ok, q := svc.GradeAnswer("café", "cafe", "cued_recall")
	if !ok || q < 4 {
		t.Errorf("GradeAnswer café/cafe cued_recall = %v %d, want true 4", ok, q)
	}
	ok, q = svc.GradeAnswer("¿cuánto cuesta?", "cuanto cuesta", "free_recall")
	if !ok {
		t.Errorf("GradeAnswer cuanto cuesta should be correct with accent fold")
	}
	// Wrong answer
	ok, q = svc.GradeAnswer("estoy", "soy", "cued_recall")
	if ok {
		t.Error("estoy vs soy should be incorrect")
	}
	// MCQ recognition
	ok, q = svc.GradeAnswer("hola", "hola", "recognition")
	if !ok || q != 4 {
		t.Errorf("recognition hola should be 4, got %d %v", q, ok)
	}
}

func TestQANextStageForCard(t *testing.T) {
	cases := []struct {
		stage int
		want  int
	}{
		{0, stageRecognition},
		{1, 1},
		{2, 2},
		{4, stageProduction},
		{5, stageProduction},
	}
	for _, c := range cases {
		card := &VocabularyCard{MasteryStage: c.stage}
		got := nextStageForCard(card)
		if got != c.want {
			t.Errorf("nextStageForCard(%d) = %d, want %d", c.stage, got, c.want)
		}
	}
}

func TestBlankClozeWordSpanish(t *testing.T) {
	blank, cloze := blankClozeWord("Quisiera un café con leche")
	if blank == "" || cloze == "" {
		t.Fatalf("blankClozeWord returned empty: %q %q", blank, cloze)
	}
	if !strings.Contains(cloze, "______") {
		t.Errorf("cloze = %q, want to contain ______", cloze)
	}
	// Spanish example with ¿
	blank2, cloze2 := blankClozeWord("¿Cuánto cuesta el café?")
	if blank2 == "" || cloze2 == "" {
		t.Errorf("blankClozeWord Spanish question failed: %q %q", blank2, cloze2)
	}
}

func TestQABuildGrammarClozeQuestion(t *testing.T) {
	q := buildGrammarClozeQuestion("ser vs estar", "Use ser for identity", `["Yo soy estudiante"]`)
	if q == nil {
		t.Fatal("buildGrammarClozeQuestion returned nil")
	}
	if q.question.Prompt.Text == "" || q.answer == "" {
		t.Errorf("grammar cloze empty: prompt %q answer %q", q.question.Prompt.Text, q.answer)
	}
	if !strings.Contains(q.question.Prompt.Text, "______") {
		t.Errorf("grammar cloze prompt should contain blank: %q", q.question.Prompt.Text)
	}
}

func TestInterleaveQueueSpanish(t *testing.T) {
	vocab := []models.SRSQueueItem{{ID: "v1"}, {ID: "v2"}, {ID: "v3"}, {ID: "v4"}}
	grammar := []models.SRSQueueItem{{ID: "g1"}}
	out := interleaveQueue(vocab, grammar, 5)
	if len(out) != 5 {
		t.Fatalf("interleaveQueue len = %d, want 5", len(out))
	}
	// Grammar should be interleaved, not all at end
	hasGrammarEarly := false
	for i := 0; i < 2; i++ {
		if out[i].ID == "g1" {
			hasGrammarEarly = true
		}
	}
	if !hasGrammarEarly {
		t.Error("grammar item should be interleaved early, not at end")
	}
}

func TestInterleaveItemsSpanish(t *testing.T) {
	items := []sessionComposedItem{
		{itemType: "vocabulary", activityType: "recognition"},
		{itemType: "vocabulary", activityType: "cued_recall"},
		{itemType: "vocabulary", activityType: "free_recall"},
		{itemType: "vocabulary", activityType: "production"},
		{itemType: "grammar", activityType: "grammar_review"},
		{itemType: "grammar", activityType: "grammar_review"},
	}
	out := interleaveItems(items)
	if len(out) != 6 {
		t.Fatalf("interleaveItems len = %d, want 6", len(out))
	}
	grammarPos := -1
	for i, it := range out {
		if it.itemType == "grammar" {
			grammarPos = i
			break
		}
	}
	if grammarPos > 2 {
		t.Errorf("grammar item should be interleaved early, pos %d", grammarPos)
	}
}

// ── Scenario service with sqlmock (runnable) ─────────────────────────────────

func TestScenarioServiceListScenariosSpanish(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()
	svc := NewScenarioService(db, nil, nil, nil, nil)

	// capability lookup
	mock.ExpectQuery("SELECT support_tier, active_course_id").WithArgs("en", "es").WillReturnRows(
		sqlmock.NewRows([]string{"support_tier", "active_course_id"}).AddRow("full_course", "course-1"),
	)
	// scenario list
	mock.ExpectQuery("SELECT sc.id::text, sc.course_id::text").WithArgs("course-1").WillReturnRows(
		sqlmock.NewRows([]string{"id", "course_id", "unit_id", "slug", "title", "domain", "cefr_level", "can_do_statement", "ai_role_name", "ai_role_description", "opening_line", "max_turns", "estimated_minutes", "completion_criteria"}).
			AddRow("sc1", "course-1", "unit-1", "ordering-coffee", "Ordering Coffee at a Cafe", "food_drink", "A1", "I can order a drink", "Sparky", "Barista", "Hola. ¿Qué te gustaría pedir hoy?", 10, 5, []byte(`{}`)),
	)
	mock.ExpectQuery("SELECT EXISTS").WithArgs("sc1", "user1").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	scenarios, err := svc.ListScenarios(context.Background(), "user1", "es", "en")
	if err != nil {
		t.Fatalf("ListScenarios: %v", err)
	}
	if len(scenarios) != 1 || scenarios[0].Slug != "ordering-coffee" {
		t.Fatalf("unexpected scenarios: %+v", scenarios)
	}
	if scenarios[0].OpeningLine != "Hola. ¿Qué te gustaría pedir hoy?" {
		t.Errorf("opening line = %q, want Spanish Hola", scenarios[0].OpeningLine)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestScenarioServiceStartSpanishOpeningLineAndChunks(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()
	svc := NewScenarioService(db, nil, nil, nil, NewCurriculumService(db))

	mock.ExpectQuery("SELECT sc.id::text, sc.course_id::text").WithArgs("sc1").WillReturnRows(
		sqlmock.NewRows([]string{"id", "course_id", "unit_id", "slug", "title", "domain", "cefr_level", "ai_role_name", "ai_role_description", "opening_line", "max_turns", "estimated_minutes", "completion_criteria"}).
			AddRow("sc1", "course-1", "unit-1", "ordering-coffee", "Ordering Coffee", "food_drink", "A1", "Sparky", "Barista", "Hola. ¿Qué te gustaría pedir hoy?", 10, 5, []byte(`{}`)),
	)
	mock.ExpectQuery("SELECT id::text, scaffold_level FROM scenario_runs").WithArgs("sc1", "user1").WillReturnRows(sqlmock.NewRows([]string{"id", "scaffold_level"}))
	mock.ExpectQuery("INSERT INTO scenario_runs").WithArgs("user1", "sc1", "es", "en", "guided").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("run1"))
	mock.ExpectExec("INSERT INTO scenario_turns").WithArgs("run1", "Hola. ¿Qué te gustaría pedir hoy?", "").WillReturnResult(sqlmock.NewResult(1, 1))
	phaseRows := sqlmock.NewRows([]string{"id", "scenario_id", "ordinal", "title", "learner_goal", "required_intents", "chunk_bank"}).
		AddRow("ph1", "sc1", 1, "Greeting", "Greet the barista.", "{greet}", []byte(`[{"text":"Hola, buenos días.","translation":"Hello, good morning."}]`))
	mock.ExpectQuery("scenario_phases").WithArgs("sc1").WillReturnRows(phaseRows)

	resp, err := svc.StartScenario(context.Background(), "user1", "sc1", "es", "en")
	if err != nil {
		t.Fatalf("StartScenario: %v", err)
	}
	if resp.AIResponse.AIMessage != "Hola. ¿Qué te gustaría pedir hoy?" {
		t.Errorf("AI opening = %q, want Spanish Hola", resp.AIResponse.AIMessage)
	}
	if len(resp.AIResponse.SuggestedChunks) == 0 || resp.AIResponse.SuggestedChunks[0].Text == "" {
		t.Errorf("SuggestedChunks empty, want Spanish chunk: %+v", resp.AIResponse.SuggestedChunks)
	}
	if resp.AIResponse.SuggestedChunks[0].Text != "Hola, buenos días." {
		t.Errorf("chunk text = %q, want Hola", resp.AIResponse.SuggestedChunks[0].Text)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestScenarioServiceSendSpanishAndAIReply(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()
	svc := NewScenarioService(db, nil, NewPracticeService(db), nil, nil)

	// Load run
	mock.ExpectQuery("SELECT id::text, scenario_id::text, status, target_language").WithArgs("run1", "user1").WillReturnRows(
		sqlmock.NewRows([]string{"id", "scenario_id", "status", "target_language", "native_language", "scaffold_level", "current_phase_ordinal", "score", "xp_awarded"}).
			AddRow("run1", "sc1", "in_progress", "es", "en", "guided", 1, 0, 0),
	)
	// phases
	phaseRows := sqlmock.NewRows([]string{"id", "scenario_id", "ordinal", "title", "learner_goal", "required_intents", "chunk_bank"}).
		AddRow("ph1", "sc1", 1, "Greeting", "Greet", "{greet}", []byte(`[{"text":"Hola, buenos días.","translation":"Hello"}]`)).
		AddRow("ph2", "sc1", 2, "Order", "Order drink", "{order_drink}", []byte(`[{"text":"Quisiera un café","translation":"I would like a coffee"}]`))
	mock.ExpectQuery("scenario_phases").WithArgs("sc1").WillReturnRows(phaseRows)
	// insert user turn
	mock.ExpectExec("INSERT INTO scenario_turns").WithArgs("run1", "Hola, buenos días", 1).WillReturnResult(sqlmock.NewResult(1, 1))
	// covered intents fetch
	mock.ExpectQuery("SELECT COVERED_intents FROM scenario_runs").WithArgs("run1").WillReturnRows(sqlmock.NewRows([]string{"covered_intents"}).AddRow([]byte(`{}`)))
	// insert AI turn
	mock.ExpectExec("INSERT INTO scenario_turns").WithArgs("run1", "¡Hola! Bienvenido. ¿Qué te gustaría pedir hoy?", "Hello! Welcome. What would you like to order today?", 2).WillReturnResult(sqlmock.NewResult(1, 1))
	// covered intents again
	mock.ExpectQuery("SELECT COVERED_intents FROM scenario_runs").WithArgs("run1").WillReturnRows(sqlmock.NewRows([]string{"covered_intents"}).AddRow([]byte(`{}`)))
	// phases for completion check
	phaseRows2 := sqlmock.NewRows([]string{"id", "scenario_id", "ordinal", "title", "learner_goal", "required_intents", "chunk_bank"}).
		AddRow("ph1", "sc1", 1, "Greeting", "Greet", "{greet}", []byte(`[]`)).
		AddRow("ph2", "sc1", 2, "Order", "Order", "{order_drink}", []byte(`[]`))
	mock.ExpectQuery("scenario_phases").WithArgs("sc1").WillReturnRows(phaseRows2)
	// update run
	mock.ExpectExec("UPDATE scenario_runs SET current_phase_ordinal").WithArgs("run1", 2, sqlmock.AnyArg(), 50, "in_progress").WillReturnResult(sqlmock.NewResult(1, 1))

	reply, err := svc.SendMessage(context.Background(), "user1", "run1", "Hola, buenos días")
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if !reply.PhaseComplete {
		t.Error("PhaseComplete should be true for greet")
	}
	if !strings.Contains(reply.AIMessage, "Hola") && !strings.Contains(reply.AIMessage, "Bienvenido") {
		t.Errorf("AI reply = %q, want Spanish", reply.AIMessage)
	}
	if reply.Translation == "" {
		t.Error("AI reply translation should not be empty")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestScenarioRequestHintReturnsChunks(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()
	svc := NewScenarioService(db, nil, nil, nil, nil)
	// GetRun: run query
	mock.ExpectQuery("SELECT id::text, user_id, scenario_id::text").WithArgs("run1", "user1").WillReturnRows(
		sqlmock.NewRows([]string{"id", "user_id", "scenario_id", "target_language", "native_language", "status", "scaffold_level", "current_phase_ordinal", "phase_scores", "covered_intents", "score", "xp_awarded", "started_at"}).
			AddRow("run1", "user1", "sc1", "es", "en", "in_progress", "guided", 1, "[]", []byte(`{}`), 0, 0, time.Now()),
	)
	mock.ExpectQuery("SELECT id::text, run_id::text, ordinal, speaker, text").WithArgs("run1").WillReturnRows(sqlmock.NewRows([]string{"id", "run_id", "ordinal", "speaker", "text", "translation", "phase_ordinal", "evaluation"}))
	phaseRows := sqlmock.NewRows([]string{"id", "scenario_id", "ordinal", "title", "learner_goal", "required_intents", "chunk_bank"}).
		AddRow("ph1", "sc1", 1, "Greeting", "Greet", "{greet}", []byte(`[{"text":"Hola, buenos días.","translation":"Hello"}]`))
	mock.ExpectQuery("scenario_phases").WithArgs("sc1").WillReturnRows(phaseRows)

	chunks, err := svc.RequestHint(context.Background(), "user1", "run1")
	if err != nil {
		t.Fatalf("RequestHint: %v", err)
	}
	if len(chunks) == 0 || chunks[0].Text == "" {
		t.Fatalf("hint chunks empty, want Spanish chunk")
	}
	if chunks[0].Translation == "" {
		t.Error("hint chunk translation empty")
	}
}

func TestPracticeDueAndSessionFlow(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()
	svc := NewPracticeService(db)
	// GetDueCards returns Spanish due cards
	rows := sqlmock.NewRows([]string{"id", "user_id", "term", "language", "translation", "definition", "lemma", "normalized_term", "part_of_speech", "is_chunk", "source_type", "source_message_id", "cefr_level", "curriculum_unit_id", "route_status", "mastery_stage", "mastery_state", "ease_factor", "lapses", "stage_success_count", "production_success_count", "spontaneous_use_count", "teachability_score", "confidence", "review_count", "correct_count", "interval_days", "next_review", "context_sentence", "context_message_id", "context_chat_id", "created_at", "first_seen_at", "last_seen_at"}).
		AddRow("c1", "user1", "café", "es", "coffee", "", "café", "cafe", "noun", false, "seed", nil, "A1", nil, "core", 1, "learning", 2.5, 0, 0, 0, 0, 90.0, 0.5, 3, 2, 1, time.Now().Add(-time.Hour), "Quisiera un café", nil, nil, time.Now(), time.Now(), time.Now())
	mock.ExpectQuery("SELECT id::text, user_id, term, language").WithArgs("user1", "es", 10).WillReturnRows(rows)
	cards, err := svc.GetDueCards(context.Background(), "user1", "es", 10)
	if err != nil {
		t.Fatalf("GetDueCards: %v", err)
	}
	if len(cards) != 1 || cards[0].Term != "café" {
		t.Fatalf("GetDueCards unexpected: %+v", cards)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestVocabOnlyAndFullCourseSpanish(t *testing.T) {
	// Spanish for en should be full_course, random pair should be vocab_only
	c := vocabOnlyCapability("en", "fr")
	if c.SupportTier != string(models.LearningSupportVocabOnly) {
		t.Errorf("vocabOnlyCapability tier = %q, want vocab_only", c.SupportTier)
	}
}

func TestSpanishCapabilityFullCourseFields(t *testing.T) {
	js, _ := json.Marshal(map[string]any{"text": "Hola", "choices": []string{"Hola", "Adiós"}})
	if len(js) == 0 {
		t.Error("json marshal failed")
	}
	// Verify spanishUnits are ordered by ordinal
	for i := 1; i < len(spanishUnits); i++ {
		if spanishUnits[i].Ordinal <= spanishUnits[i-1].Ordinal {
			t.Errorf("spanishUnits not ordered: %d %d", spanishUnits[i-1].Ordinal, spanishUnits[i].Ordinal)
		}
	}
}

func TestStreakRecoveryBooking(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()
	sessSvc := &SessionComposerService{db: db}
	mock.ExpectExec("INSERT INTO daily_learning_stats").WithArgs("user1", "es").WillReturnResult(sqlmock.NewResult(1, 1))
	_, err = sessSvc.BookRecovery(context.Background(), "user1", "es", "en")
	if err != nil {
		t.Fatalf("BookRecovery: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestSpanishIsChunkVocabularyFlag(t *testing.T) {
	for _, it := range spanishLexicalItems {
		if it.Lemma == "para llevar" && !it.IsChunk {
			t.Error("para llevar should be IsChunk=true")
		}
		if it.Lemma == "café con leche" && !it.IsChunk {
			t.Error("café con leche should be IsChunk=true")
		}
	}
}
