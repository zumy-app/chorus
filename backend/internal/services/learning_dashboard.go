package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/chorus/messenger/internal/models"
)

// LearningDashboardService aggregates the Learn dashboard in one place so the
// web and mobile clients render identical data. It composes profile, pair
// capability, SRS/vocabulary counts, grammar weakness, scenario recommendation,
// and activity/stats snapshots. Empty states return zeroed summaries rather
// than errors so an unused account renders a clean "get started" dashboard.
type LearningDashboardService struct {
	db           *sql.DB
	capabilities *LearningCapabilityService
	profiles     *LearningProfileService
	curriculum   *CurriculumService
}

func NewLearningDashboardService(db *sql.DB, capabilities *LearningCapabilityService, profiles *LearningProfileService, curriculum *CurriculumService) *LearningDashboardService {
	return &LearningDashboardService{db: db, capabilities: capabilities, profiles: profiles, curriculum: curriculum}
}

func (s *LearningDashboardService) GetDashboard(ctx context.Context, userID, targetLanguage, nativeLanguage string) (*models.LearningDashboard, error) {
	targetLanguage = normalizeLang(targetLanguage)
	nativeLanguage = normalizeLang(nativeLanguage)
	if nativeLanguage == "" {
		nativeLanguage = "en"
	}
	if targetLanguage == "" {
		return nil, fmt.Errorf("target language is required")
	}

	capability, err := s.capabilities.GetCapability(ctx, nativeLanguage, targetLanguage)
	if err != nil {
		return nil, err
	}
	profile, err := s.profiles.GetProfile(ctx, userID, targetLanguage, nativeLanguage)
	if err != nil {
		return nil, err
	}

	dash := &models.LearningDashboard{
		Capability:            *capability,
		Profile:               *profile,
		DailyGoal:             s.dailyGoal(ctx, userID, targetLanguage, profile),
		Streak:                s.streak(ctx, userID, targetLanguage),
		Fluency:               s.fluency(profile),
		Vocabulary:            s.vocabulary(ctx, userID, targetLanguage),
		Grammar:               s.grammar(ctx, userID, targetLanguage),
		Scenario:              s.scenario(ctx, userID, targetLanguage, profile, capability),
		RecommendedActivities: s.recommended(profile, capability),
		WeeklyActivity:        s.weeklyActivity(ctx, userID, targetLanguage),
		MonthlyActivity:       s.monthlyActivity(ctx, userID, targetLanguage),
	}

	if capability.SupportTier == string(models.LearningSupportFullCourse) {
		if unit := s.currentUnit(ctx, userID, profile); unit != nil {
			dash.CurrentUnit = unit
			dash.NextLesson = s.nextLesson(ctx, userID, unit.ID)
		}
	}

	return dash, nil
}

func (s *LearningDashboardService) dailyGoal(ctx context.Context, userID, targetLanguage string, profile *models.UserLanguageProfile) models.DailyGoalSummary {
	summary := models.DailyGoalSummary{TargetItems: profile.DailyGoalItems}
	if summary.TargetItems <= 0 {
		summary.TargetItems = 10
	}
	_ = s.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(items_completed), 0)
		FROM daily_learning_stats
		WHERE user_id = $1 AND target_language = $2 AND activity_date = CURRENT_DATE
	`, userID, targetLanguage).Scan(&summary.CompletedItems)
	if summary.TargetItems > 0 {
		summary.Percent = summary.CompletedItems * 100 / summary.TargetItems
		if summary.Percent > 100 {
			summary.Percent = 100
		}
	}
	return summary
}

func (s *LearningDashboardService) streak(ctx context.Context, userID, targetLanguage string) models.StreakSummary {
	rows, err := s.db.QueryContext(ctx, `
		SELECT DISTINCT activity_date
		FROM daily_learning_stats
		WHERE user_id = $1 AND target_language = $2
		ORDER BY activity_date DESC
		LIMIT 365
	`, userID, targetLanguage)
	if err != nil {
		return models.StreakSummary{}
	}
	defer rows.Close()

	dates := make(map[string]bool)
	var ordered []time.Time
	for rows.Next() {
		var d time.Time
		if err := rows.Scan(&d); err != nil {
			continue
		}
		key := d.Format("2006-01-02")
		if !dates[key] {
			dates[key] = true
			ordered = append(ordered, d)
		}
	}

	if len(ordered) == 0 {
		return models.StreakSummary{}
	}

	// Anchor to today; if today is not yet active, a streak can still be alive
	// from yesterday.
	cursor := time.Now()
	if _, ok := dates[cursor.Format("2006-01-02")]; !ok {
		cursor = cursor.AddDate(0, 0, -1)
	}
	days := 0
	for {
		if _, ok := dates[cursor.Format("2006-01-02")]; !ok {
			break
		}
		days++
		cursor = cursor.AddDate(0, 0, -1)
	}

	summary := models.StreakSummary{Days: days}
	summary.CanRecover = days == 0 && len(ordered) > 0
	// At risk when the streak exists but today has no activity yet.
	_, todayActive := dates[time.Now().Format("2006-01-02")]
	summary.AtRisk = days > 0 && !todayActive
	return summary
}

func (s *LearningDashboardService) fluency(profile *models.UserLanguageProfile) models.FluencySummary {
	score := profile.ReadinessScore
	summary := models.FluencySummary{
		ReadinessScore:   score,
		ReadinessPercent: score / 10,
		Label:            fluencyLabel(profile.CurrentCEFRLevel, score),
		ComponentScores:  map[string]int{},
	}
	if summary.ReadinessPercent > 100 {
		summary.ReadinessPercent = 100
	}
	return summary
}

func fluencyLabel(level string, score int) string {
	if score >= 900 {
		switch level {
		case "A1":
			return "Approaching A2"
		case "A2":
			return "Approaching B1"
		case "B1":
			return "Approaching B2"
		case "B2":
			return "Maintaining B2"
		}
	}
	return "Building " + level
}

func (s *LearningDashboardService) vocabulary(ctx context.Context, userID, targetLanguage string) models.VocabularySummary {
	var summary models.VocabularySummary
	_ = s.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE next_review <= CURRENT_TIMESTAMP),
			COUNT(*) FILTER (WHERE mastery_state = 'mastered'),
			COUNT(*) FILTER (WHERE source_type IN ('chat','scenario') AND created_at >= CURRENT_DATE - INTERVAL '7 days')
		FROM vocabulary
		WHERE user_id = $1 AND language = $2
	`, userID, targetLanguage).Scan(&summary.Total, &summary.DueToday, &summary.Mastered, &summary.NewFromChats)
	return summary
}

func (s *LearningDashboardService) grammar(ctx context.Context, userID, targetLanguage string) models.GrammarSummary {
	summary := models.GrammarSummary{}
	var confidence float64
	_ = s.db.QueryRowContext(ctx, `
		SELECT COALESCE(g.title, ''), COALESCE(m.confidence, 0)
		FROM user_grammar_mastery m
		JOIN grammar_points g ON g.id = m.grammar_point_id
		WHERE m.user_id = $1 AND m.target_language = $2
		ORDER BY m.confidence ASC, m.next_review_at ASC
		LIMIT 1
	`, userID, targetLanguage).Scan(&summary.WeakestPointTitle, &confidence)

	// confidence is stored 0..1; expose a 0..100 integer percent.
	summary.ConfidencePct = int(confidence * 100)

	_ = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM user_grammar_mastery
		WHERE user_id = $1 AND target_language = $2 AND next_review_at <= CURRENT_TIMESTAMP
	`, userID, targetLanguage).Scan(&summary.DueToday)
	return summary
}

func (s *LearningDashboardService) scenario(ctx context.Context, userID, targetLanguage string, profile *models.UserLanguageProfile, capability *models.LearningPairCapability) models.ScenarioSummary {
	summary := models.ScenarioSummary{}
	if capability.SupportTier != string(models.LearningSupportFullCourse) || capability.ActiveCourseID == "" {
		return summary
	}
	_ = s.db.QueryRowContext(ctx, `
		SELECT sc.id::text, sc.title
		FROM scenario_scripts sc
		WHERE sc.course_id = $1
		  AND NOT EXISTS (
			SELECT 1 FROM scenario_runs r
			WHERE r.scenario_id = sc.id AND r.user_id = $2 AND r.status = 'completed'
		  )
		ORDER BY CASE sc.cefr_level WHEN 'A1' THEN 1 WHEN 'A2' THEN 2 WHEN 'B1' THEN 3 ELSE 4 END, sc.created_at
		LIMIT 1
	`, capability.ActiveCourseID, userID).Scan(&summary.NextScenarioID, &summary.Title)

	summary.HasNewWords = s.hasCandidateMinedItems(ctx, userID, targetLanguage)
	return summary
}

func (s *LearningDashboardService) hasCandidateMinedItems(ctx context.Context, userID, targetLanguage string) bool {
	var n int
	_ = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM mined_items
		WHERE user_id = $1 AND language = $2 AND status = 'candidate'
	`, userID, targetLanguage).Scan(&n)
	return n > 0
}

func (s *LearningDashboardService) currentUnit(ctx context.Context, userID string, profile *models.UserLanguageProfile) *models.UnitProgressSummary {
	if profile.ActiveUnitID == "" {
		return nil
	}
	var unit models.UnitProgressSummary
	var checkpointScore sql.NullInt64
	var startedAt, completedAt sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT u.id::text, u.course_id::text, u.cefr_level, u.ordinal, u.slug, u.title,
		       u.can_do_statement, u.description, u.estimated_minutes, u.checkpoint_required,
		       COALESCE(p.status, 'available') AS status,
		       COALESCE(p.progress_pct, 0) AS progress_pct,
		       COALESCE(p.competency_score, 0) AS competency_score,
		       COALESCE(p.lessons_completed, 0) AS lessons_completed,
		       p.checkpoint_score, p.started_at, p.completed_at
		FROM curriculum_units u
		LEFT JOIN user_unit_progress p ON p.unit_id = u.id AND p.user_id = $2
		WHERE u.id = $1
	`, profile.ActiveUnitID, userID).Scan(
		&unit.ID, &unit.CourseID, &unit.CEFRLevel, &unit.Ordinal, &unit.Slug, &unit.Title,
		&unit.CanDoStatement, &unit.Description, &unit.EstimatedMinutes, &unit.CheckpointRequired,
		&unit.Status, &unit.ProgressPct, &unit.CompetencyScore, &unit.LessonsCompleted,
		&checkpointScore, &startedAt, &completedAt,
	)
	if err != nil {
		return nil
	}
	if checkpointScore.Valid {
		v := int(checkpointScore.Int64)
		unit.CheckpointScore = &v
	}
	if startedAt.Valid {
		unit.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		unit.CompletedAt = &completedAt.Time
	}
	return &unit
}

func (s *LearningDashboardService) nextLesson(ctx context.Context, userID, unitID string) *models.LessonSummary {
	var lesson models.LessonSummary
	err := s.db.QueryRowContext(ctx, `
		SELECT l.id::text, l.unit_id::text, l.ordinal, l.slug, l.type, l.title, l.objective, l.estimated_minutes
		FROM curriculum_lessons l
		WHERE l.unit_id = $1
		  AND NOT EXISTS (
			SELECT 1 FROM user_lesson_attempts a
			WHERE a.user_id = $2 AND a.lesson_id = l.id AND a.status = 'completed'
		  )
		ORDER BY l.ordinal
		LIMIT 1
	`, unitID, userID).Scan(
		&lesson.ID, &lesson.UnitID, &lesson.Ordinal, &lesson.Slug, &lesson.Type,
		&lesson.Title, &lesson.Objective, &lesson.EstimatedMinutes,
	)
	if err != nil {
		return nil
	}
	lesson.Status = "available"
	return &lesson
}

func (s *LearningDashboardService) weeklyActivity(ctx context.Context, userID, targetLanguage string) []models.DailyActivityPoint {
	points := make([]models.DailyActivityPoint, 0, 7)
	rows, err := s.db.QueryContext(ctx, `
		SELECT activity_date, COALESCE(xp, 0), COALESCE(items_completed, 0)
		FROM daily_learning_stats
		WHERE user_id = $1 AND target_language = $2
		  AND activity_date >= CURRENT_DATE - INTERVAL '6 days'
		ORDER BY activity_date
	`, userID, targetLanguage)
	if err != nil {
		return points
	}
	defer rows.Close()

	byDate := make(map[string]models.DailyActivityPoint)
	for rows.Next() {
		var d time.Time
		var p models.DailyActivityPoint
		if err := rows.Scan(&d, &p.XP, &p.ItemsCompleted); err != nil {
			continue
		}
		p.Date = d.Format("2006-01-02")
		byDate[p.Date] = p
	}

	for i := 6; i >= 0; i-- {
		d := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
		if p, ok := byDate[d]; ok {
			points = append(points, p)
		} else {
			points = append(points, models.DailyActivityPoint{Date: d})
		}
	}
	return points
}

func (s *LearningDashboardService) recommended(profile *models.UserLanguageProfile, capability *models.LearningPairCapability) []models.RecommendedActivity {
	acts := make([]models.RecommendedActivity, 0, 3)
	if capability.SRSEnabled {
		acts = append(acts, models.RecommendedActivity{
			ID:               "vocabulary",
			Type:             "vocabulary",
			Title:            "Vocabulary Review",
			Description:      "Clear your due words with a quick review.",
			Priority:         "high",
			EstimatedMinutes: 3,
			Action:           "start_session",
		})
	}
	if capability.SupportTier == string(models.LearningSupportFullCourse) && profile.ActiveUnitID != "" {
		acts = append(acts, models.RecommendedActivity{
			ID:               "continue",
			Type:             "lesson",
			Title:            "Continue Learning",
			Description:      "Keep going on your active unit.",
			Priority:         "medium",
			EstimatedMinutes: 5,
			Action:           "start_session",
		})
	}
	if capability.ScenariosEnabled {
		acts = append(acts, models.RecommendedActivity{
			ID:               "scenario",
			Type:             "scenario",
			Title:            "Real-World Scenario",
			Description:      "Practice a real conversation.",
			Priority:         "medium",
			EstimatedMinutes: 5,
			Action:           "open_scenarios",
		})
	}
	return acts
}

// monthlyActivity returns time-bucketed per-month learning metrics for FR-31.
// Words learned is derived from the vocabulary bank (words saved/learned for
// the target language). Sentences understood is derived from messages carrying a
// translation that the user can consume in their chats. The last 12 months are
// always returned (zero-filled), so clients can render a month picker with real
// and empty states.
func (s *LearningDashboardService) monthlyActivity(ctx context.Context, userID, targetLanguage string) []models.MonthlyActivityPoint {
	wordsByMonth := make(map[string]int)
	sentencesByMonth := make(map[string]int)

	if rows, err := s.db.QueryContext(ctx, `
		SELECT to_char(date_trunc('month', created_at), 'YYYY-MM') AS month, COUNT(*)
		FROM vocabulary
		WHERE user_id = $1 AND language = $2 AND created_at IS NOT NULL
		GROUP BY month
	`, userID, targetLanguage); err == nil {
		for rows.Next() {
			var m string
			var n int
			if rows.Scan(&m, &n) == nil {
				wordsByMonth[m] = n
			}
		}
		rows.Close()
	}

	if rows, err := s.db.QueryContext(ctx, `
		SELECT to_char(date_trunc('month', m.created_at), 'YYYY-MM') AS month, COUNT(*)
		FROM messages m
		JOIN chat_participants cp ON cp.chat_id = m.chat_id
		WHERE cp.user_id = $1
		  AND m.created_at IS NOT NULL
		  AND m.translations IS NOT NULL
		  AND m.translations <> '{}'::jsonb
		GROUP BY month
	`, userID); err == nil {
		for rows.Next() {
			var m string
			var n int
			if rows.Scan(&m, &n) == nil {
				sentencesByMonth[m] = n
			}
		}
		rows.Close()
	}

	// Build the last 12 month buckets (ascending) so the month picker always has
	// a full range, even across a year boundary.
	points := make([]models.MonthlyActivityPoint, 0, 12)
	now := time.Now()
	anchor := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	for i := 11; i >= 0; i-- {
		month := anchor.AddDate(0, -i, 0)
		key := month.Format("2006-01")
		points = append(points, models.MonthlyActivityPoint{
			Month:               key,
			WordsLearned:        wordsByMonth[key],
			SentencesUnderstood: sentencesByMonth[key],
		})
	}
	return points
}
