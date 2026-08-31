package services

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/chorus/messenger/internal/models"
)

// LearningCapabilityService resolves how much structured learning support a
// native-language/target-language pair currently has. Translation and grammar
// may work broadly; full CEFR progression is enabled only for curated courses.
type LearningCapabilityService struct {
	db *sql.DB
}

func NewLearningCapabilityService(db *sql.DB) *LearningCapabilityService {
	return &LearningCapabilityService{db: db}
}

func (s *LearningCapabilityService) GetCapability(ctx context.Context, nativeLanguage, targetLanguage string) (*models.LearningPairCapability, error) {
	nativeLanguage = normalizeLang(nativeLanguage)
	targetLanguage = normalizeLang(targetLanguage)
	if nativeLanguage == "" {
		nativeLanguage = "en"
	}
	if targetLanguage == "" {
		targetLanguage = "es"
	}

	if nativeLanguage == targetLanguage {
		return disabledCapability(nativeLanguage, targetLanguage), nil
	}

	capability := &models.LearningPairCapability{}
	var activeCourseID sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT native_language, target_language, support_tier,
		       active_course_id::text,
		       placement_enabled, roadmap_enabled, scenarios_enabled,
		       srs_enabled, mining_enabled, grammar_feedback_enabled,
		       quality_notes, created_at, updated_at
		FROM learning_pair_capabilities
		WHERE native_language = $1 AND target_language = $2
	`, nativeLanguage, targetLanguage).Scan(
		&capability.NativeLanguage,
		&capability.TargetLanguage,
		&capability.SupportTier,
		&activeCourseID,
		&capability.PlacementEnabled,
		&capability.RoadmapEnabled,
		&capability.ScenariosEnabled,
		&capability.SRSEnabled,
		&capability.MiningEnabled,
		&capability.GrammarFeedbackEnabled,
		&capability.QualityNotes,
		&capability.CreatedAt,
		&capability.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return vocabOnlyCapability(nativeLanguage, targetLanguage), nil
	}
	if err != nil {
		return nil, err
	}
	if activeCourseID.Valid {
		capability.ActiveCourseID = activeCourseID.String
	}
	return capability, nil
}

func normalizeLang(lang string) string {
	lang = strings.TrimSpace(strings.ToLower(lang))
	if idx := strings.Index(lang, "-"); idx > 0 {
		return lang[:idx]
	}
	return lang
}

func vocabOnlyCapability(nativeLanguage, targetLanguage string) *models.LearningPairCapability {
	now := time.Now()
	return &models.LearningPairCapability{
		NativeLanguage:         nativeLanguage,
		TargetLanguage:         targetLanguage,
		SupportTier:            string(models.LearningSupportVocabOnly),
		SRSEnabled:             true,
		MiningEnabled:          true,
		GrammarFeedbackEnabled: true,
		QualityNotes:           "Translation, grammar analysis, chat-mined vocabulary, and SRS are available. Structured lessons for this pair are coming soon.",
		CreatedAt:              now,
		UpdatedAt:              now,
	}
}

func disabledCapability(nativeLanguage, targetLanguage string) *models.LearningPairCapability {
	now := time.Now()
	return &models.LearningPairCapability{
		NativeLanguage: nativeLanguage,
		TargetLanguage: targetLanguage,
		SupportTier:    string(models.LearningSupportDisabled),
		QualityNotes:   "Choose a different target language to start learning.",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}
