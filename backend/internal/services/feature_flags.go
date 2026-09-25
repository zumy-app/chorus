package services

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

// FeatureFlagService is the single source of truth for feature-flag resolution.
// It replaces the previous env-var/canary service. Flags are stored in Postgres
// and resolved per user on every request. A short in-process cache (30s) bounds
// DB load while still allowing fast promotion/demotion.
type FeatureFlagService struct {
	db *sql.DB

	mu         sync.RWMutex
	cache      map[string]*resolvedFlagCacheEntry
	cacheTTL   time.Duration
	defaultOff bool // if true, unknown flags resolve to false (production)
}

type resolvedFlagCacheEntry struct {
	flags      map[string]bool
	expiresAt  time.Time
	isAdmin    bool
	isBeta     bool
}

// NewFeatureFlagService creates a DB-backed flag service. defaultOff should be
// true in production so unknown flags never leak; tests may pass false.
func NewFeatureFlagService(db *sql.DB, defaultOff bool) *FeatureFlagService {
	return &FeatureFlagService{
		db:         db,
		cache:      make(map[string]*resolvedFlagCacheEntry),
		cacheTTL:   30 * time.Second,
		defaultOff: defaultOff,
	}
}

// ResolveForUser returns the resolved flag map for a user.
// It is the single source of truth used by both client visibility and server-side enforcement.
func (s *FeatureFlagService) ResolveForUser(ctx context.Context, userID uuid.UUID, isAdmin, isBeta bool) (map[string]bool, error) {
	cacheKey := userID.String() + "|" + roleKey(isAdmin, isBeta)

	// 1. Check in-process cache.
	s.mu.RLock()
	entry, ok := s.cache[cacheKey]
	s.mu.RUnlock()
	if ok && time.Now().Before(entry.expiresAt) && entry.isAdmin == isAdmin && entry.isBeta == isBeta {
		return copyFlagMap(entry.flags), nil
	}

	// 2. Load flags and overrides from DB.
	flags, err := s.resolveFromDB(ctx, userID, isAdmin, isBeta)
	if err != nil {
		return nil, err
	}

	// 3. Populate cache.
	s.mu.Lock()
	s.cache[cacheKey] = &resolvedFlagCacheEntry{
		flags:     flags,
		expiresAt: time.Now().Add(s.cacheTTL),
		isAdmin:   isAdmin,
		isBeta:    isBeta,
	}
	s.mu.Unlock()

	return copyFlagMap(flags), nil
}

// IsEnabled resolves a single flag for a user.
func (s *FeatureFlagService) IsEnabled(ctx context.Context, userID uuid.UUID, key string, isAdmin, isBeta bool) (bool, error) {
	flags, err := s.ResolveForUser(ctx, userID, isAdmin, isBeta)
	if err != nil {
		return false, err
	}
	enabled, ok := flags[key]
	if !ok {
		return !s.defaultOff, nil
	}
	return enabled, nil
}

// resolveFromDB loads all flag definitions and the user's overrides, then applies
// the resolution rules: override > stable > beta_access > admin_only > default_state.
func (s *FeatureFlagService) resolveFromDB(ctx context.Context, userID uuid.UUID, isAdmin, isBeta bool) (map[string]bool, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT key, default_state, admin_only, beta_access, stable
		FROM feature_flags
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	flags := make(map[string]bool)
	for rows.Next() {
		var key string
		var defaultState, adminOnly, betaAccess, stable bool
		if err := rows.Scan(&key, &defaultState, &adminOnly, &betaAccess, &stable); err != nil {
			return nil, err
		}

		enabled := stable || (betaAccess && isBeta) || (adminOnly && isAdmin) || defaultState
		flags[key] = enabled
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Apply per-user overrides.
	overrideRows, err := s.db.QueryContext(ctx, `
		SELECT flag_key, enabled
		FROM feature_flag_overrides
		WHERE user_id = $1
	`, userID)
	if err != nil {
		return nil, err
	}
	defer overrideRows.Close()

	for overrideRows.Next() {
		var key string
		var enabled bool
		if err := overrideRows.Scan(&key, &enabled); err != nil {
			return nil, err
		}
		flags[key] = enabled
	}
	if err := overrideRows.Err(); err != nil {
		return nil, err
	}

	return flags, nil
}

// InvalidateCacheForUser drops a user's cached flag map so the next request picks
// up fresh values. Call this after writing an override or changing a flag tier.
func (s *FeatureFlagService) InvalidateCacheForUser(userID uuid.UUID) {
	prefix := userID.String() + "|"
	s.mu.Lock()
	for k := range s.cache {
		if len(k) > len(prefix) && k[:len(prefix)] == prefix {
			delete(s.cache, k)
		}
	}
	s.mu.Unlock()
}

// InvalidateCache drops the entire cache. Use sparingly (e.g. flag tier changes).
func (s *FeatureFlagService) InvalidateCache() {
	s.mu.Lock()
	s.cache = make(map[string]*resolvedFlagCacheEntry)
	s.mu.Unlock()
}

// ListFlags returns all flag definitions for the admin panel.
func (s *FeatureFlagService) ListFlags(ctx context.Context) ([]FeatureFlagRow, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT key, description, default_state, admin_only, beta_access, stable, created_at, updated_at
		FROM feature_flags
		ORDER BY key
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []FeatureFlagRow
	for rows.Next() {
		var f FeatureFlagRow
		if err := rows.Scan(&f.Key, &f.Description, &f.DefaultState, &f.AdminOnly, &f.BetaAccess, &f.Stable, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// GetFlag returns a single flag definition.
func (s *FeatureFlagService) GetFlag(ctx context.Context, key string) (*FeatureFlagRow, error) {
	var f FeatureFlagRow
	err := s.db.QueryRowContext(ctx, `
		SELECT key, description, default_state, admin_only, beta_access, stable, created_at, updated_at
		FROM feature_flags
		WHERE key = $1
	`, key).Scan(&f.Key, &f.Description, &f.DefaultState, &f.AdminOnly, &f.BetaAccess, &f.Stable, &f.CreatedAt, &f.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}

// SetFlagTiers updates the audience tiers for a flag and clears the cache.
func (s *FeatureFlagService) SetFlagTiers(ctx context.Context, key string, adminOnly, betaAccess, stable bool) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE feature_flags
		SET admin_only = $1, beta_access = $2, stable = $3, updated_at = NOW()
		WHERE key = $4
	`, adminOnly, betaAccess, stable, key)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("flag not found")
	}
	s.InvalidateCache()
	return nil
}

// SetUserOverride creates or updates a per-user override.
func (s *FeatureFlagService) SetUserOverride(ctx context.Context, userID uuid.UUID, key string, enabled bool) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO feature_flag_overrides (user_id, flag_key, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		ON CONFLICT (user_id, flag_key)
		DO UPDATE SET enabled = EXCLUDED.enabled, updated_at = NOW()
	`, userID, key, enabled)
	if err != nil {
		return err
	}
	s.InvalidateCacheForUser(userID)
	return nil
}

// DeleteUserOverride removes a per-user override.
func (s *FeatureFlagService) DeleteUserOverride(ctx context.Context, userID uuid.UUID, key string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM feature_flag_overrides
		WHERE user_id = $1 AND flag_key = $2
	`, userID, key)
	if err != nil {
		return err
	}
	s.InvalidateCacheForUser(userID)
	return nil
}

// SeedFlags inserts initial flag rows if they do not exist. Called on startup.
func (s *FeatureFlagService) SeedFlags(ctx context.Context, flags []FeatureFlagRow) error {
	for _, f := range flags {
		_, err := s.db.ExecContext(ctx, `
			INSERT INTO feature_flags (key, description, default_state, admin_only, beta_access, stable, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
			ON CONFLICT (key) DO NOTHING
		`, f.Key, f.Description, f.DefaultState, f.AdminOnly, f.BetaAccess, f.Stable)
		if err != nil {
			return err
		}
	}
	return nil
}

// FeatureFlagRow is the DB representation of a flag definition.
type FeatureFlagRow struct {
	Key         string    `json:"key"`
	Description string    `json:"description"`
	DefaultState bool     `json:"defaultState"`
	AdminOnly   bool      `json:"adminOnly"`
	BetaAccess  bool      `json:"betaAccess"`
	Stable      bool      `json:"stable"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// DefaultFlags returns the canonical flag seed set for Release 1.
func DefaultFlags() []FeatureFlagRow {
	return []FeatureFlagRow{
		{Key: "grammar_insights", Description: "Basic grammar analysis in chat", DefaultState: false, AdminOnly: false, BetaAccess: true, Stable: true},
		{Key: "word_collector", Description: "Tap-to-save vocabulary from messages", DefaultState: false, AdminOnly: false, BetaAccess: true, Stable: true},
		{Key: "feature_voting", Description: "Community feature voting in Hub", DefaultState: false, AdminOnly: false, BetaAccess: true, Stable: true},
		{Key: "referral_bump", Description: "Waitlist referral position bump", DefaultState: false, AdminOnly: false, BetaAccess: true, Stable: true},
		{Key: "google_oauth", Description: "Continue with Google login", DefaultState: false, AdminOnly: true, BetaAccess: false, Stable: false},
		{Key: "push_notifications", Description: "FCM token registration and delivery", DefaultState: false, AdminOnly: true, BetaAccess: false, Stable: false},
		{Key: "word_flashcards", Description: "Spaced-repetition flashcard practice", DefaultState: false, AdminOnly: true, BetaAccess: false, Stable: false},
		{Key: "ai_writing_assistant", Description: "Help me write composer feature", DefaultState: false, AdminOnly: true, BetaAccess: false, Stable: false},
		{Key: "placement_test", Description: "Full CEFR placement quiz", DefaultState: false, AdminOnly: true, BetaAccess: false, Stable: false},
		{Key: "scenario_roleplay", Description: "AI scenario role-play", DefaultState: false, AdminOnly: true, BetaAccess: false, Stable: false},
		{Key: "voice_message", Description: "In-chat voice recording", DefaultState: false, AdminOnly: true, BetaAccess: false, Stable: false},
		{Key: "media_sharing", Description: "Photo/video sharing in chat", DefaultState: false, AdminOnly: true, BetaAccess: false, Stable: false},
		{Key: "document_sharing", Description: "PDF/doc sharing in chat", DefaultState: false, AdminOnly: true, BetaAccess: false, Stable: false},
		{Key: "location_sharing", Description: "Location sharing in chat", DefaultState: false, AdminOnly: true, BetaAccess: false, Stable: false},
		{Key: "video_calls", Description: "WebRTC audio/video calls", DefaultState: false, AdminOnly: true, BetaAccess: false, Stable: false},
		{Key: "study_pods", Description: "Group study pods", DefaultState: false, AdminOnly: true, BetaAccess: false, Stable: false},
		{Key: "teacher_marketplace", Description: "Teacher booking & payouts", DefaultState: false, AdminOnly: true, BetaAccess: false, Stable: false},
		{Key: "payout_teacher", Description: "Payout dashboard for teachers", DefaultState: false, AdminOnly: true, BetaAccess: false, Stable: false},
		{Key: "xp_leaderboard", Description: "League/ranking tables", DefaultState: false, AdminOnly: true, BetaAccess: false, Stable: false},
		{Key: "group_chat", Description: "Multi-person chat rooms", DefaultState: false, AdminOnly: true, BetaAccess: false, Stable: false},
		{Key: "gdpr_export", Description: "GDPR data export flow", DefaultState: false, AdminOnly: true, BetaAccess: false, Stable: false},
		{Key: "social_feed", Description: "Public community activity feed", DefaultState: false, AdminOnly: true, BetaAccess: false, Stable: false},
		// Legacy env flags migrated to DB model.
		{Key: "learning_v3_engine", Description: "Learning V3 engine", DefaultState: true, AdminOnly: false, BetaAccess: false, Stable: false},
		{Key: "redis_session_cache", Description: "Redis session cache", DefaultState: true, AdminOnly: false, BetaAccess: false, Stable: false},
	}
}

func roleKey(isAdmin, isBeta bool) string {
	if isAdmin {
		return "admin"
	}
	if isBeta {
		return "beta"
	}
	return "general"
}

func copyFlagMap(m map[string]bool) map[string]bool {
	out := make(map[string]bool, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
