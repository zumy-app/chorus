package services

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/chorus/messenger/internal/models"
)

func TestUpdatePlacementAbility_CorrectNeverDecreases(t *testing.T) {
	abilities := []float64{0, 30, 250, 400, 700, 1000}
	items := []int{100, 350, 650, 850}
	for _, a := range abilities {
		for _, v := range items {
			got := updatePlacementAbility(a, v, true)
			if got < a {
				t.Errorf("correct should never decrease ability: ability=%v item=%v got=%v", a, v, got)
			}
			if got > 1000 {
				t.Errorf("ability clamped: got %v", got)
			}
		}
	}
}

func TestUpdatePlacementAbility_WrongNeverIncreases(t *testing.T) {
	abilities := []float64{0, 30, 250, 400, 700, 1000}
	items := []int{100, 350, 650, 850}
	for _, a := range abilities {
		for _, v := range items {
			got := updatePlacementAbility(a, v, false)
			if got > a {
				t.Errorf("wrong should never increase ability: ability=%v item=%v got=%v", a, v, got)
			}
			if got < 0 {
				t.Errorf("ability clamped: got %v", got)
			}
		}
	}
}

func TestUpdatePlacementAbility_Clamps(t *testing.T) {
	if got := updatePlacementAbility(1000, 850, true); got != 1000 {
		t.Errorf("expected clamp to 1000, got %v", got)
	}
	if got := updatePlacementAbility(0, 100, false); got != 0 {
		t.Errorf("expected clamp to 0, got %v", got)
	}
}

// TestPlacementAbility_A2LearnerSimulation mirrors the question order the item
// bank produces (A1 x3, A2 x3, B1 x3, B2 x3) for a learner who knows A1/A2 but
// not B1/B2. The final ability must land in the A2 band (250-549), not B1.
func TestPlacementAbility_A2LearnerSimulation(t *testing.T) {
	sequence := []struct {
		level   string
		correct bool
	}{
		{"A1", true}, {"A1", true}, {"A1", true},
		{"A2", true}, {"A2", true}, {"A2", true},
		{"B1", false}, {"B1", false}, {"B1", false},
		{"B2", false}, {"B2", false}, {"B2", false},
	}
	ability := 250.0
	for _, s := range sequence {
		ability = updatePlacementAbility(ability, levelToValue(s.level), s.correct)
	}
	if level := levelFromAbility(int(ability)); level != "A2" {
		t.Fatalf("expected A2 for an A2-band learner, got %s (ability %.1f)", level, ability)
	}
}

func TestPlacementAbility_B1LearnerSimulation(t *testing.T) {
	sequence := []struct {
		level   string
		correct bool
	}{
		{"A1", true}, {"A1", true}, {"A1", true},
		{"A2", true}, {"A2", true}, {"A2", true},
		{"B1", true}, {"B1", true}, {"B1", true},
		{"B2", false}, {"B2", false}, {"B2", false},
	}
	ability := 250.0
	for _, s := range sequence {
		ability = updatePlacementAbility(ability, levelToValue(s.level), s.correct)
	}
	if level := levelFromAbility(int(ability)); level != "B1" {
		t.Fatalf("expected B1 for a B1-band learner, got %s (ability %.1f)", level, ability)
	}
}

// TestNormalizeAnswerForLevel verifies the accent policy: A1/A2 answers are
// forgiven for missing accents; B1+ answers must match diacritics exactly.
func TestNormalizeAnswerForLevel(t *testing.T) {
	cases := []struct {
		answer, expected, cefr string
		want                   bool
	}{
		{"esta", "está", "A1", true},
		{"ESTA ", "está", "A2", true},
		{"papa", "papá", "A2", true},
		{"esta", "está", "B1", false}, // accent required at B1+
		{"está", "está", "B1", true},
		{"está", "está", "B2", true},
		{"esta", "está", "B2", false},
		{"Carlos   es de Madrid", "carlos es de madrid", "A1", true}, // reconstruction: case/whitespace forgiving
		{"", "está", "A1", false},
	}
	for _, c := range cases {
		if got := normalizeAnswerForLevel(c.answer, c.expected, c.cefr); got != c.want {
			t.Errorf("normalizeAnswerForLevel(%q,%q,%s)=%v want %v", c.answer, c.expected, c.cefr, got, c.want)
		}
	}
}

func TestGradePlacementAnswer_AcceptVariants(t *testing.T) {
	item := placementItem{CEFR: "A1", Correct: "five", AcceptVariants: []string{"5"}}
	if !gradePlacementAnswer(item, "5") {
		t.Errorf("accept_variants should be accepted")
	}
	if !gradePlacementAnswer(item, "Five") {
		t.Errorf("case-insensitive match expected")
	}
	if gradePlacementAnswer(item, "fifty") {
		t.Errorf("wrong answer should not pass")
	}
}

// TestUpdatePlacementAbility_IRTBounds pins the logistic formula shape: far
// items move the estimate little, near items move it a lot, sign always
// matches correctness.
func TestUpdatePlacementAbility_IRTBounds(t *testing.T) {
	// Easy item (b=100) for a strong learner (theta=900): near-zero gain.
	got := updatePlacementAbility(900, 100, true)
	if got-900 > 8 {
		t.Errorf("easy correct item should barely move ability, got %v", got)
	}
	// Hard item (b=850) for a weak learner (theta=100): tiny loss on wrong.
	got = updatePlacementAbility(100, 850, false)
	if 100-got > 8 {
		t.Errorf("hard wrong item should barely move ability, got %v", got)
	}
	// Boundary item carries the most information.
	got = updatePlacementAbility(500, 500, true)
	if got-500 < 30 || got-500 > 50 {
		t.Errorf("boundary correct item should move ability ~40, got %v", got-500)
	}
}

func TestCefrFromSelfSelection(t *testing.T) {
	cases := map[string]string{
		"beginner":     "A1",
		"beginner ":    "A1",
		"BEGINNER":     "A1",
		"intermediate": "B1",
		"Intermediate": "B1",
		"advanced":     "B2",
		"ADVANCED":     "B2",
		"expert":       "",
		"":             "",
		"b2":           "",
	}
	for in, want := range cases {
		if got := cefrFromSelfSelection(in); got != want {
			t.Errorf("cefrFromSelfSelection(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestReadinessSeedForSelfSelection(t *testing.T) {
	cases := map[string]int{"A1": 0, "A2": 250, "B1": 550, "B2": 800}
	for level, want := range cases {
		if got := readinessSeedForSelfSelection(level); got != want {
			t.Errorf("readinessSeedForSelfSelection(%q) = %d, want %d", level, got, want)
		}
	}
	if got := readinessSeedForSelfSelection("C2"); got != 0 {
		t.Errorf("unknown level should seed 0, got %d", got)
	}
}

func TestSelectLevel_InvalidLevel(t *testing.T) {
	svc := NewPlacementService(nil, nil, nil)
	_, err := svc.SelectLevel(context.Background(), "user-1", "expert", "es", "en")
	if err == nil {
		t.Fatalf("expected an error for an invalid self-selected level")
	}
	if !strings.Contains(err.Error(), "invalid self-selected level") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSelectLevel_SeedsProfile(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	// LearningProfileService GetProfile -> fetchProfile SELECT.
	mock.ExpectQuery("SELECT user_id, native_language, target_language, current_cefr_level").
		WithArgs("user-1", "es", "en").
		WillReturnRows(sqlmock.NewRows([]string{
			"user_id", "native_language", "target_language", "current_cefr_level",
			"readiness_score", "active_course_id", "active_unit_id",
			"placement_status", "primary_goal", "daily_goal_items",
			"mining_enabled", "nudges_enabled", "scenario_hints_enabled",
			"created_at", "updated_at",
		}).AddRow("user-1", "en", "es", "A1", 0, "course-1", "unit-A1",
			"not_started", "conversational_fluency", 10, true, true, true,
			time.Now(), time.Now()))

	// startUnitID -> capabilitiesFor SELECT.
	mock.ExpectQuery("SELECT support_tier, active_course_id::text FROM learning_pair_capabilities").
		WithArgs("en", "es").
		WillReturnRows(sqlmock.NewRows([]string{
			"support_tier", "active_course_id::text",
		}).AddRow("full_course", "course-1"))

	// startUnitID -> curriculum unit SELECT.
	mock.ExpectQuery("SELECT id::text FROM curriculum_units WHERE course_id").
		WithArgs("course-1", "B1").
		WillReturnRows(sqlmock.NewRows([]string{"id::text"}).AddRow("unit-B1"))

	// SelectLevel profile UPDATE.
	mock.ExpectExec("UPDATE user_language_profiles").
		WithArgs("user-1", "es", "en", "B1", 550, models.PlacementStatusSelfSelected, "unit-B1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	svc := NewPlacementService(db, nil, NewLearningProfileService(db, nil, nil))
	res, err := svc.SelectLevel(context.Background(), "user-1", "Intermediate", "es", "en")
	if err != nil {
		t.Fatalf("SelectLevel: %v", err)
	}
	if res.EstimatedCEFR != "B1" {
		t.Fatalf("expected B1, got %s", res.EstimatedCEFR)
	}
	if res.ReadinessScore != 550 {
		t.Fatalf("expected readiness 550, got %d", res.ReadinessScore)
	}
	if res.ActiveUnitID != "unit-B1" {
		t.Fatalf("expected active unit unit-B1, got %s", res.ActiveUnitID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestLevelFromAbility_Boundaries(t *testing.T) {
	cases := map[int]string{
		0:    "A1",
		100:  "A1",
		249:  "A1",
		250:  "A2",
		549:  "A2",
		550:  "B1",
		799:  "B1",
		800:  "B2",
		1000: "B2",
	}
	for ability, want := range cases {
		if got := levelFromAbility(ability); got != want {
			t.Errorf("levelFromAbility(%d)=%s, want %s", ability, got, want)
		}
	}
}

// TestBuildItemBank_CalibratedFirst verifies the V3 redesign: the calibrated
// placement_items bank is used when present, the fixed module structure
// (8 receptive -> 8 grammar -> 4 discourse) is respected, and the hardcoded
// fallback only runs when the course has no calibrated items.
func TestBuildItemBank_CalibratedFirst(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	// capabilitiesFor -> active course.
	mock.ExpectQuery("SELECT support_tier, active_course_id::text FROM learning_pair_capabilities").
		WithArgs("en", "es").
		WillReturnRows(sqlmock.NewRows([]string{"support_tier", "active_course_id::text"}).AddRow("full_course", "course-1"))

	// Calibrated bank: 3 receptive + 3 grammar + 2 discourse.
	rows := sqlmock.NewRows([]string{
		"id::text", "cefr_level", "module", "item_type", "prompt", "choices", "correct", "accept_variants", "difficulty_value",
	})
	rx := func(id, cefr, module, itemType, prompt, correct string, difficulty int, choices ...string) {
		ch := "{}"
		if len(choices) > 0 {
			ch = "{" + strings.Join(choices, ",") + "}"
		}
		rows.AddRow(id, cefr, module, itemType, prompt, ch, correct, "{}", difficulty)
	}
	rx("p1", "A1", "receptive_vocab", "L2_to_L1_context", `{"sentence":"El niño tiene cinco años.","target_word":"cinco","context_translation":"The child is ___ years old."}`, "five", 125, "five", "fifteen", "yesterday", "many")
	rx("p2", "A2", "receptive_vocab", "L2_to_L1_context", `{"sentence":"Ayer compré una camisa nueva.","target_word":"compré","context_translation":"Yesterday I ___ a new shirt."}`, "bought", 350, "bought", "sold", "wore", "washed")
	rx("p3", "B1", "receptive_vocab", "L2_to_L1_context", `{"sentence":"...","target_word":"x","context_translation":"..."}`, "x", 600, "x", "y", "z", "w")
	rx("g1", "A1", "grammar_production", "gap_fill_type", `{"sentence_with_blank":"Mi hermana _____ en una tienda.","verb":"trabajar","translation":"My sister works in a store."}`, "trabaja", 180)
	rx("g2", "A2", "grammar_production", "gap_fill_type", `{"sentence_with_blank":"...","verb":"viajar","translation":"..."}`, "viajamos", 400)
	rx("g3", "B2", "grammar_production", "gap_fill_type", `{"sentence_with_blank":"...","verb":"saber","translation":"..."}`, "sepan", 800)
	rx("d1", "B1", "discourse_reading", "passage_mcq", `{"passage":"...","question":"..."}`, "answer1", 650, "answer1", "b", "c", "d")
	rx("d2", "B2", "discourse_reading", "passage_mcq", `{"passage":"...","question":"..."}`, "answer2", 850, "answer2", "b", "c", "d")

	mock.ExpectQuery("SELECT id::text, cefr_level, module, item_type").
		WithArgs("course-1").
		WillReturnRows(rows)

	svc := NewPlacementService(db, nil, nil)
	items, err := svc.buildItemBank(context.Background(), "es", "en")
	if err != nil {
		t.Fatalf("buildItemBank: %v", err)
	}
	// 8 calibrated items in module order, padded with fallback MCQs up to the
	// 20-item cap (thin calibrated bank < placementMinBankItems).
	if len(items) < placementMinBankItems {
		t.Fatalf("bank should be padded to at least %d items, got %d", placementMinBankItems, len(items))
	}
	if len(items) > placementTotalQuestions {
		t.Fatalf("bank must never exceed %d items, got %d", placementTotalQuestions, len(items))
	}
	if len(items) != placementTotalQuestions {
		t.Fatalf("8 calibrated + fallback padding should fill the full %d-item structure, got %d", placementTotalQuestions, len(items))
	}
	// Module ordering: receptive block first, then grammar, then discourse.
	for i := 0; i < 3; i++ {
		if items[i].Module != "receptive_vocab" {
			t.Fatalf("item %d should be receptive_vocab, got %s", i, items[i].Module)
		}
	}
	for i := 3; i < 6; i++ {
		if items[i].Module != "grammar_production" {
			t.Fatalf("item %d should be grammar_production, got %s", i, items[i].Module)
		}
	}
	for i := 6; i < 8; i++ {
		if items[i].Module != "discourse_reading" {
			t.Fatalf("item %d should be discourse_reading, got %s", i, items[i].Module)
		}
	}
	for _, it := range items {
		if it.Difficulty == 0 {
			t.Fatalf("all items (calibrated and fallback) must carry difficulty values")
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestBuildItemBank_FallbackWhenNoCourse(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()
	// No capability row -> hardcoded fallback bank, no item query issued.
	mock.ExpectQuery("SELECT support_tier, active_course_id::text FROM learning_pair_capabilities").
		WithArgs("en", "es").
		WillReturnError(sql.ErrNoRows)

	svc := NewPlacementService(db, nil, nil)
	items, err := svc.buildItemBank(context.Background(), "es", "en")
	if err != nil {
		t.Fatalf("buildItemBank: %v", err)
	}
	if len(items) != 12 {
		t.Fatalf("fallback bank should hold 12 items, got %d", len(items))
	}
	for _, it := range items {
		if it.Difficulty == 0 {
			t.Fatalf("fallback items must be seeded with band difficulty")
		}
		if _, ok := it.Prompt.(map[string]any); !ok {
			t.Fatalf("fallback prompts must be wrapped for the shared render path")
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
