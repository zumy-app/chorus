package services

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/chorus/messenger/internal/models"
)

// LearningProfileService owns the per-user, per-target-language learning
// profile: CEFR placement state, active course/unit, goals, and toggles. It
// also lazily creates a sane default profile the first time a learner asks for
// a supported pair so the dashboard has something real to render.
type LearningProfileService struct {
	db           *sql.DB
	capabilities *LearningCapabilityService
	curriculum   *CurriculumService
}

func NewLearningProfileService(db *sql.DB, capabilities *LearningCapabilityService, curriculum *CurriculumService) *LearningProfileService {
	return &LearningProfileService{db: db, capabilities: capabilities, curriculum: curriculum}
}

// GetProfile returns the profile for a pair, creating a default row when one
// does not exist. nativeLanguage defaults to "en" when empty.
func (s *LearningProfileService) GetProfile(ctx context.Context, userID, targetLanguage, nativeLanguage string) (*models.UserLanguageProfile, error) {
	targetLanguage = normalizeLang(targetLanguage)
	nativeLanguage = normalizeLang(nativeLanguage)
	if nativeLanguage == "" {
		nativeLanguage = "en"
	}
	if targetLanguage == "" {
		return nil, fmt.Errorf("target language is required")
	}

	profile, err := s.fetchProfile(ctx, userID, targetLanguage, nativeLanguage)
	if err == nil {
		return profile, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	return s.createDefaultProfile(ctx, userID, targetLanguage, nativeLanguage)
}

func (s *LearningProfileService) fetchProfile(ctx context.Context, userID, targetLanguage, nativeLanguage string) (*models.UserLanguageProfile, error) {
	var p models.UserLanguageProfile
	var activeCourseID, activeUnitID sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT user_id, native_language, target_language, current_cefr_level,
		       readiness_score, active_course_id::text, active_unit_id::text,
		       placement_status, primary_goal, daily_goal_items,
		       mining_enabled, nudges_enabled, scenario_hints_enabled,
		       created_at, updated_at
		FROM user_language_profiles
		WHERE user_id = $1 AND target_language = $2 AND native_language = $3
	`, userID, targetLanguage, nativeLanguage).Scan(
		&p.UserID, &p.NativeLanguage, &p.TargetLanguage, &p.CurrentCEFRLevel,
		&p.ReadinessScore, &activeCourseID, &activeUnitID,
		&p.PlacementStatus, &p.PrimaryGoal, &p.DailyGoalItems,
		&p.MiningEnabled, &p.NudgesEnabled, &p.ScenarioHintsEnabled,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if activeCourseID.Valid {
		p.ActiveCourseID = activeCourseID.String
	}
	if activeUnitID.Valid {
		p.ActiveUnitID = activeUnitID.String
	}
	return &p, nil
}

func (s *LearningProfileService) createDefaultProfile(ctx context.Context, userID, targetLanguage, nativeLanguage string) (*models.UserLanguageProfile, error) {
	capability, err := s.capabilities.GetCapability(ctx, nativeLanguage, targetLanguage)
	if err != nil {
		return nil, err
	}

	var activeCourseID, activeUnitID any
	activeCourseID = nil
	activeUnitID = nil
	if capability.SupportTier == string(models.LearningSupportFullCourse) && capability.ActiveCourseID != "" {
		activeCourseID = capability.ActiveCourseID
		if s.curriculum != nil {
			if unitID, err := s.curriculum.GetFirstUnitID(ctx, capability.ActiveCourseID); err == nil && unitID != "" {
				activeUnitID = unitID
			}
		}
	}

	var p models.UserLanguageProfile
	err = s.db.QueryRowContext(ctx, `
		INSERT INTO user_language_profiles (
			user_id, native_language, target_language, current_cefr_level,
			readiness_score, active_course_id, active_unit_id, placement_status,
			primary_goal, daily_goal_items, mining_enabled, nudges_enabled,
			scenario_hints_enabled
		)
		VALUES ($1, $2, $3, 'A1', 0, $4, $5, 'not_started',
		        'conversational_fluency', 10, true, true, true)
		ON CONFLICT (user_id, native_language, target_language) DO UPDATE SET
			updated_at = CURRENT_TIMESTAMP
		RETURNING user_id, native_language, target_language, current_cefr_level,
			readiness_score, active_course_id::text, active_unit_id::text,
			placement_status, primary_goal, daily_goal_items,
			mining_enabled, nudges_enabled, scenario_hints_enabled,
			created_at, updated_at
	`, userID, nativeLanguage, targetLanguage, activeCourseID, activeUnitID).Scan(
		&p.UserID, &p.NativeLanguage, &p.TargetLanguage, &p.CurrentCEFRLevel,
		&p.ReadinessScore, &activeCourseID, &activeUnitID,
		&p.PlacementStatus, &p.PrimaryGoal, &p.DailyGoalItems,
		&p.MiningEnabled, &p.NudgesEnabled, &p.ScenarioHintsEnabled,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create learning profile: %w", err)
	}
	if activeCourseID != nil {
		p.ActiveCourseID = activeCourseID.(string)
	}
	if activeUnitID != nil {
		p.ActiveUnitID = activeUnitID.(string)
	}
	return &p, nil
}

// UpdateProfile applies user-editable profile settings. It returns the updated
// profile. Unset fields are left unchanged.
func (s *LearningProfileService) UpdateProfile(ctx context.Context, userID string, req models.LearningProfileUpdateRequest) (*models.UserLanguageProfile, error) {
	targetLanguage := normalizeLang(req.TargetLanguage)
	nativeLanguage := normalizeLang(req.NativeLanguage)
	if nativeLanguage == "" {
		nativeLanguage = "en"
	}
	if targetLanguage == "" {
		return nil, fmt.Errorf("target language is required")
	}

	if _, err := s.GetProfile(ctx, userID, targetLanguage, nativeLanguage); err != nil {
		return nil, err
	}

	_, err := s.db.ExecContext(ctx, `
		UPDATE user_language_profiles
		SET primary_goal = COALESCE(NULLIF($3, ''), primary_goal),
		    daily_goal_items = CASE WHEN $4 > 0 THEN $4 ELSE daily_goal_items END,
		    mining_enabled = COALESCE($5, mining_enabled),
		    nudges_enabled = COALESCE($6, nudges_enabled),
		    scenario_hints_enabled = COALESCE($7, scenario_hints_enabled),
		    updated_at = CURRENT_TIMESTAMP
		WHERE user_id = $1 AND target_language = $2 AND native_language = $8
	`, userID, targetLanguage, strings.TrimSpace(req.PrimaryGoal), req.DailyGoalItems,
		req.MiningEnabled, req.NudgesEnabled, req.ScenarioHintsEnabled, nativeLanguage)
	if err != nil {
		return nil, err
	}

	return s.fetchProfile(ctx, userID, targetLanguage, nativeLanguage)
}

// SetActiveUnit updates the profile's active unit after placement or unit
// completion. It is kept separate from UpdateProfile because it is only ever
// written server-side.
func (s *LearningProfileService) SetActiveUnit(ctx context.Context, userID, targetLanguage, nativeLanguage, unitID string) error {
	if nativeLanguage == "" {
		nativeLanguage = "en"
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE user_language_profiles
		SET active_unit_id = $4, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = $1 AND target_language = $2 AND native_language = $3
	`, userID, targetLanguage, nativeLanguage, nullStr(unitID))
	return err
}
