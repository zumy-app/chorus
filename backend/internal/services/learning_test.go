package services

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/chorus/messenger/internal/models"
)

func TestNormalizeLang(t *testing.T) {
	cases := map[string]string{
		"":           "",
		"EN":         "en",
		" es ":       "es",
		"es-MX":      "es",
		"zh-Hant":    "zh",
		"pt-BR":      "pt",
		"Spanish":    "spanish",
		"en-US-x":    "en",
	}
	for in, want := range cases {
		if got := normalizeLang(in); got != want {
			t.Errorf("normalizeLang(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestVocabOnlyCapability(t *testing.T) {
	c := vocabOnlyCapability("en", "fr")
	if c.SupportTier != string(models.LearningSupportVocabOnly) {
		t.Fatalf("expected vocab_only, got %s", c.SupportTier)
	}
	if !c.SRSEnabled || !c.MiningEnabled || !c.GrammarFeedbackEnabled {
		t.Fatalf("expected srs/mining/grammar enabled for vocab_only")
	}
	if c.PlacementEnabled || c.RoadmapEnabled || c.ScenariosEnabled {
		t.Fatalf("placement/roadmap/scenarios must be disabled for vocab_only")
	}
	if c.ActiveCourseID != "" {
		t.Fatalf("vocab_only must not reference a course")
	}
}

func TestDisabledCapability(t *testing.T) {
	c := disabledCapability("es", "es")
	if c.SupportTier != string(models.LearningSupportDisabled) {
		t.Fatalf("expected disabled, got %s", c.SupportTier)
	}
	if c.SRSEnabled || c.MiningEnabled || c.GrammarFeedbackEnabled {
		t.Fatalf("disabled pair must disable learning features")
	}
}

func TestGetCapability_FullCourse(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{
		"native_language", "target_language", "support_tier", "active_course_id",
		"placement_enabled", "roadmap_enabled", "scenarios_enabled",
		"srs_enabled", "mining_enabled", "grammar_feedback_enabled",
		"quality_notes", "created_at", "updated_at",
	}).AddRow("en", "es", "full_course", "course-1",
		true, true, true, true, true, true, "notes",
		time.Now(), time.Now())

	mock.ExpectQuery("SELECT native_language, target_language, support_tier").WithArgs("en", "es").WillReturnRows(rows)

	svc := NewLearningCapabilityService(db)
	c, err := svc.GetCapability(context.Background(), "en", "es")
	if err != nil {
		t.Fatalf("GetCapability: %v", err)
	}
	if c.SupportTier != string(models.LearningSupportFullCourse) {
		t.Fatalf("expected full_course, got %s", c.SupportTier)
	}
	if c.ActiveCourseID != "course-1" {
		t.Fatalf("expected course-1, got %s", c.ActiveCourseID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestGetCapability_VocabOnlyFallback(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT native_language, target_language, support_tier").WithArgs("es", "fr").WillReturnRows(sqlmock.NewRows([]string{
		"native_language", "target_language", "support_tier", "active_course_id",
		"placement_enabled", "roadmap_enabled", "scenarios_enabled",
		"srs_enabled", "mining_enabled", "grammar_feedback_enabled",
		"quality_notes", "created_at", "updated_at",
	}))

	svc := NewLearningCapabilityService(db)
	c, err := svc.GetCapability(context.Background(), "es", "fr")
	if err != nil {
		t.Fatalf("GetCapability: %v", err)
	}
	if c.SupportTier != string(models.LearningSupportVocabOnly) {
		t.Fatalf("expected vocab_only, got %s", c.SupportTier)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestFluencyLabel(t *testing.T) {
	cases := []struct {
		level string
		score int
		want  string
	}{
		{"A2", 740, "Building A2"},
		{"A2", 910, "Approaching B1"},
		{"B2", 940, "Maintaining B2"},
		{"A1", 100, "Building A1"},
	}
	for _, c := range cases {
		if got := fluencyLabel(c.level, c.score); got != c.want {
			t.Errorf("fluencyLabel(%s, %d) = %q, want %q", c.level, c.score, got, c.want)
		}
	}
}
