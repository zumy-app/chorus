package services

import (
	"context"
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

func TestBuildChoices_ContainsCorrectAndFourDistinct(t *testing.T) {
	pool := []string{"hola", "gracias", "me llamo", "café", "idioma", "ayer", "voy a", "podríamos"}
	for i := 0; i < 50; i++ {
		choices := buildChoices("hola", pool)
		if len(choices) != 4 {
			t.Fatalf("expected 4 choices, got %d (%v)", len(choices), choices)
		}
		seen := map[string]bool{}
		for _, c := range choices {
			if seen[c] {
				t.Fatalf("duplicate choice %q (%v)", c, choices)
			}
			seen[c] = true
			if c == "Opción 4" || strings.Contains(c, "pción") {
				t.Fatalf("found placeholder filler choice %q (%v)", c, choices)
			}
		}
		if !seen["hola"] {
			t.Fatalf("correct answer must be present (%v)", choices)
		}
	}
}

func TestBuildChoices_CorrectNotAlwaysFirst(t *testing.T) {
	pool := []string{"hola", "gracias", "me llamo", "café", "idioma", "ayer", "voy a", "podríamos"}
	firstIndexes := map[int]bool{}
	for i := 0; i < 100; i++ {
		choices := buildChoices("hola", pool)
		for j, c := range choices {
			if c == "hola" {
				firstIndexes[j] = true
				break
			}
		}
	}
	if len(firstIndexes) < 2 {
		t.Fatalf("correct answer should appear in more than one position over 100 runs, got %v", firstIndexes)
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
