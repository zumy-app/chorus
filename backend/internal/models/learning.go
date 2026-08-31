package models

import "time"

type LearningSupportTier string

const (
	LearningSupportFullCourse     LearningSupportTier = "full_course"
	LearningSupportBetaAIAssisted LearningSupportTier = "beta_ai_assisted"
	LearningSupportVocabOnly      LearningSupportTier = "vocab_only"
	LearningSupportDisabled       LearningSupportTier = "disabled"
)

type LearningPairCapability struct {
	NativeLanguage         string    `json:"nativeLanguage" db:"native_language"`
	TargetLanguage         string    `json:"targetLanguage" db:"target_language"`
	SupportTier            string    `json:"supportTier" db:"support_tier"`
	ActiveCourseID         string    `json:"activeCourseId,omitempty" db:"active_course_id"`
	PlacementEnabled       bool      `json:"placementEnabled" db:"placement_enabled"`
	RoadmapEnabled         bool      `json:"roadmapEnabled" db:"roadmap_enabled"`
	ScenariosEnabled       bool      `json:"scenariosEnabled" db:"scenarios_enabled"`
	SRSEnabled             bool      `json:"srsEnabled" db:"srs_enabled"`
	MiningEnabled          bool      `json:"miningEnabled" db:"mining_enabled"`
	GrammarFeedbackEnabled bool      `json:"grammarFeedbackEnabled" db:"grammar_feedback_enabled"`
	QualityNotes           string    `json:"qualityNotes" db:"quality_notes"`
	CreatedAt              time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt              time.Time `json:"updatedAt" db:"updated_at"`
}

type UserLanguageProfile struct {
	UserID               string    `json:"userId" db:"user_id"`
	NativeLanguage       string    `json:"nativeLanguage" db:"native_language"`
	TargetLanguage       string    `json:"targetLanguage" db:"target_language"`
	CurrentCEFRLevel     string    `json:"currentCefrLevel" db:"current_cefr_level"`
	ReadinessScore       int       `json:"readinessScore" db:"readiness_score"`
	ActiveCourseID       string    `json:"activeCourseId,omitempty" db:"active_course_id"`
	ActiveUnitID         string    `json:"activeUnitId,omitempty" db:"active_unit_id"`
	PlacementStatus      string    `json:"placementStatus" db:"placement_status"`
	PrimaryGoal          string    `json:"primaryGoal" db:"primary_goal"`
	DailyGoalItems       int       `json:"dailyGoalItems" db:"daily_goal_items"`
	MiningEnabled        bool      `json:"miningEnabled" db:"mining_enabled"`
	NudgesEnabled        bool      `json:"nudgesEnabled" db:"nudges_enabled"`
	ScenarioHintsEnabled bool      `json:"scenarioHintsEnabled" db:"scenario_hints_enabled"`
	CreatedAt            time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt            time.Time `json:"updatedAt" db:"updated_at"`
}

type CurriculumCourse struct {
	ID             string    `json:"id" db:"id"`
	TargetLanguage string    `json:"targetLanguage" db:"target_language"`
	NativeLanguage string    `json:"nativeLanguage" db:"native_language"`
	Title          string    `json:"title" db:"title"`
	Version        string    `json:"version" db:"version"`
	IsActive       bool      `json:"isActive" db:"is_active"`
	SupportTier    string    `json:"supportTier" db:"support_tier"`
	Metadata       any       `json:"metadata,omitempty" db:"metadata"`
	CreatedAt      time.Time `json:"createdAt" db:"created_at"`
}

type CurriculumUnit struct {
	ID                 string    `json:"id" db:"id"`
	CourseID           string    `json:"courseId" db:"course_id"`
	CEFRLevel          string    `json:"cefrLevel" db:"cefr_level"`
	Ordinal            int       `json:"ordinal" db:"ordinal"`
	Slug               string    `json:"slug" db:"slug"`
	Title              string    `json:"title" db:"title"`
	CanDoStatement     string    `json:"canDoStatement" db:"can_do_statement"`
	Description        string    `json:"description" db:"description"`
	EstimatedMinutes   int       `json:"estimatedMinutes" db:"estimated_minutes"`
	CheckpointRequired bool      `json:"checkpointRequired" db:"checkpoint_required"`
	CreatedAt          time.Time `json:"createdAt" db:"created_at"`
}

type LessonSummary struct {
	ID               string `json:"id"`
	UnitID           string `json:"unitId"`
	Ordinal          int    `json:"ordinal"`
	Slug             string `json:"slug"`
	Type             string `json:"type"`
	Title            string `json:"title"`
	Objective        string `json:"objective"`
	EstimatedMinutes int    `json:"estimatedMinutes"`
	Status           string `json:"status"`
}

type UnitProgressSummary struct {
	ID                 string          `json:"id"`
	CourseID           string          `json:"courseId"`
	CEFRLevel          string          `json:"cefrLevel"`
	Ordinal            int             `json:"ordinal"`
	Slug               string          `json:"slug"`
	Title              string          `json:"title"`
	CanDoStatement     string          `json:"canDoStatement"`
	Description        string          `json:"description"`
	EstimatedMinutes   int             `json:"estimatedMinutes"`
	CheckpointRequired bool            `json:"checkpointRequired"`
	Status             string          `json:"status"`
	ProgressPct        int             `json:"progressPct"`
	CompetencyScore    int             `json:"competencyScore"`
	LessonsCompleted   int             `json:"lessonsCompleted"`
	CheckpointScore    *int            `json:"checkpointScore,omitempty"`
	StartedAt          *time.Time      `json:"startedAt,omitempty"`
	CompletedAt        *time.Time      `json:"completedAt,omitempty"`
	Lessons            []LessonSummary `json:"lessons,omitempty"`
}

type LearningPath struct {
	Capability LearningPairCapability `json:"capability"`
	Profile    UserLanguageProfile    `json:"profile"`
	Units      []UnitProgressSummary  `json:"units"`
}

type DailyGoalSummary struct {
	TargetItems    int `json:"targetItems"`
	CompletedItems int `json:"completedItems"`
	Percent        int `json:"percent"`
}

type StreakSummary struct {
	Days       int  `json:"days"`
	AtRisk     bool `json:"atRisk"`
	CanRecover bool `json:"canRecover"`
}

type FluencySummary struct {
	ReadinessScore   int            `json:"readinessScore"`
	ReadinessPercent int            `json:"readinessPercent"`
	Label            string         `json:"label"`
	ComponentScores  map[string]int `json:"componentScores"`
}

type VocabularySummary struct {
	Total        int `json:"total"`
	DueToday     int `json:"dueToday"`
	Mastered     int `json:"mastered"`
	NewFromChats int `json:"newFromChats"`
}

type GrammarSummary struct {
	WeakestPointTitle string `json:"weakestPointTitle"`
	ConfidencePct     int    `json:"confidencePct"`
	DueToday          int    `json:"dueToday"`
}

type ScenarioSummary struct {
	NextScenarioID string `json:"nextScenarioId,omitempty"`
	Title          string `json:"title,omitempty"`
	ProgressPct    int    `json:"progressPct"`
	HasNewWords    bool   `json:"hasNewWords"`
}

type RecommendedActivity struct {
	ID               string `json:"id"`
	Type             string `json:"type"`
	Title            string `json:"title"`
	Description      string `json:"description"`
	Priority         string `json:"priority"`
	EstimatedMinutes int    `json:"estimatedMinutes"`
	Action           string `json:"action"`
}

type DailyActivityPoint struct {
	Date           string `json:"date"`
	XP             int    `json:"xp"`
	ItemsCompleted int    `json:"itemsCompleted"`
}

type LearningDashboard struct {
	Capability            LearningPairCapability `json:"capability"`
	Profile               UserLanguageProfile    `json:"profile"`
	DailyGoal             DailyGoalSummary       `json:"dailyGoal"`
	Streak                StreakSummary          `json:"streak"`
	Fluency               FluencySummary         `json:"fluency"`
	CurrentUnit           *UnitProgressSummary   `json:"currentUnit,omitempty"`
	NextLesson            *LessonSummary         `json:"nextLesson,omitempty"`
	Vocabulary            VocabularySummary      `json:"vocabulary"`
	Grammar               GrammarSummary         `json:"grammar"`
	Scenario              ScenarioSummary        `json:"scenario"`
	RecommendedActivities []RecommendedActivity `json:"recommendedActivities"`
	WeeklyActivity        []DailyActivityPoint   `json:"weeklyActivity"`
}

type LearningProfileUpdateRequest struct {
	NativeLanguage       string `json:"nativeLanguage"`
	TargetLanguage       string `json:"targetLanguage"`
	PrimaryGoal          string `json:"primaryGoal"`
	DailyGoalItems       int    `json:"dailyGoalItems"`
	MiningEnabled        *bool  `json:"miningEnabled"`
	NudgesEnabled        *bool  `json:"nudgesEnabled"`
	ScenarioHintsEnabled *bool  `json:"scenarioHintsEnabled"`
}
