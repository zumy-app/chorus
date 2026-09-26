package services

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

func newMockFlagService(t *testing.T) (*FeatureFlagService, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	// Short cache TTL so tests don't interfere with each other.
	svc := NewFeatureFlagService(db, true)
	svc.cacheTTL = 1 * time.Millisecond
	return svc, mock
}

func seedDefaultFlags(t *testing.T, mock sqlmock.Sqlmock) {
	rows := sqlmock.NewRows([]string{"key", "default_state", "admin_only", "beta_access", "stable"}).
		AddRow("grammar_insights", false, false, true, true).
		AddRow("word_collector", false, false, true, true).
		AddRow("video_calls", false, true, false, false).
		AddRow("word_flashcards", false, true, false, false)
	mock.ExpectQuery("SELECT key, default_state, admin_only, beta_access, stable FROM feature_flags").
		WillReturnRows(rows)

	overrideRows := sqlmock.NewRows([]string{"flag_key", "enabled"})
	mock.ExpectQuery("SELECT flag_key, enabled FROM feature_flag_overrides WHERE user_id = \\$1").
		WillReturnRows(overrideRows)
}

func TestFeatureFlagService_ResolveForUser_General(t *testing.T) {
	svc, mock := newMockFlagService(t)
	seedDefaultFlags(t, mock)

	uid := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	flags, err := svc.ResolveForUser(context.Background(), uid, false, false)
	if err != nil {
		t.Fatalf("ResolveForUser failed: %v", err)
	}

	if !flags["grammar_insights"] {
		t.Errorf("expected grammar_insights to be enabled for general user")
	}
	if !flags["word_collector"] {
		t.Errorf("expected word_collector to be enabled for general user")
	}
	if flags["video_calls"] {
		t.Errorf("expected video_calls to be disabled for general user")
	}
	if flags["word_flashcards"] {
		t.Errorf("expected word_flashcards to be disabled for general user")
	}
}

func TestFeatureFlagService_ResolveForUser_Admin(t *testing.T) {
	svc, mock := newMockFlagService(t)
	seedDefaultFlags(t, mock)

	uid := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	flags, err := svc.ResolveForUser(context.Background(), uid, true, false)
	if err != nil {
		t.Fatalf("ResolveForUser failed: %v", err)
	}

	if !flags["video_calls"] {
		t.Errorf("expected video_calls to be enabled for admin")
	}
	if !flags["word_flashcards"] {
		t.Errorf("expected word_flashcards to be enabled for admin")
	}
}

func TestFeatureFlagService_PerUserOverride(t *testing.T) {
	svc, mock := newMockFlagService(t)

	rows := sqlmock.NewRows([]string{"key", "default_state", "admin_only", "beta_access", "stable"}).
		AddRow("video_calls", false, true, false, false)
	mock.ExpectQuery("SELECT key, default_state, admin_only, beta_access, stable FROM feature_flags").
		WillReturnRows(rows)

	uid := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	overrideRows := sqlmock.NewRows([]string{"flag_key", "enabled"}).
		AddRow("video_calls", true)
	mock.ExpectQuery("SELECT flag_key, enabled FROM feature_flag_overrides WHERE user_id = \\$1").
		WithArgs(uid).
		WillReturnRows(overrideRows)

	flags, err := svc.ResolveForUser(context.Background(), uid, false, false)
	if err != nil {
		t.Fatalf("ResolveForUser failed: %v", err)
	}

	if !flags["video_calls"] {
		t.Errorf("expected video_calls to be enabled via per-user override")
	}
}

func TestFeatureFlagService_DefaultFlags_ContainsCore(t *testing.T) {
	flags := DefaultFlags()
	m := make(map[string]bool)
	for _, f := range flags {
		m[f.Key] = true
	}
	required := []string{"grammar_insights", "word_collector", "feature_voting", "referral_bump", "video_calls", "teacher_marketplace", "voice_message", "media_sharing", "document_sharing", "location_sharing", "real_talk", "translate_as_you_type", "google_oauth", "group_chat", "gdpr_export", "payout_teacher", "learning_v3_engine", "redis_session_cache"}
	for _, key := range required {
		if !m[key] {
			t.Errorf("expected default flag set to contain %q", key)
		}
	}
}
