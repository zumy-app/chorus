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

// ---------------------------------------------------------------------------
// Extended vocabulary cards
// ---------------------------------------------------------------------------

type VocabularyCard struct {
	ID                    string    `json:"id" db:"id"`
	UserID                string    `json:"userId" db:"user_id"`
	Term                  string    `json:"term" db:"term"`
	Language              string    `json:"language" db:"language"`
	Translation           string    `json:"translation" db:"translation"`
	Definition            string    `json:"definition" db:"definition"`
	Lemma                 string    `json:"lemma" db:"lemma"`
	NormalizedTerm        string    `json:"normalizedTerm" db:"normalized_term"`
	PartOfSpeech          string    `json:"partOfSpeech" db:"part_of_speech"`
	IsChunk               bool      `json:"isChunk" db:"is_chunk"`
	SourceType            string    `json:"sourceType" db:"source_type"`
	SourceMessageID       string    `json:"sourceMessageId,omitempty" db:"source_message_id"`
	CEFRLevel             string    `json:"cefrLevel,omitempty" db:"cefr_level"`
	CurriculumUnitID      string    `json:"curriculumUnitId,omitempty" db:"curriculum_unit_id"`
	RouteStatus           string    `json:"routeStatus" db:"route_status"`
	MasteryStage          int       `json:"masteryStage" db:"mastery_stage"`
	MasteryState          string    `json:"masteryState" db:"mastery_state"`
	EaseFactor            float64   `json:"easeFactor" db:"ease_factor"`
	Lapses                int       `json:"lapses" db:"lapses"`
	StageSuccessCount     int       `json:"stageSuccessCount" db:"stage_success_count"`
	ProductionSuccessCount int      `json:"productionSuccessCount" db:"production_success_count"`
	SpontaneousUseCount   int       `json:"spontaneousUseCount" db:"spontaneous_use_count"`
	TeachabilityScore     float64   `json:"teachabilityScore" db:"teachability_score"`
	Confidence            float64   `json:"confidence" db:"confidence"`
	ReviewCount           int       `json:"reviewCount" db:"review_count"`
	CorrectCount          int       `json:"correctCount" db:"correct_count"`
	IntervalDays          float64   `json:"intervalDays" db:"interval_days"`
	NextReview            time.Time `json:"nextReview" db:"next_review"`
	ContextSentence       string    `json:"contextSentence,omitempty" db:"context_sentence"`
	ContextMessageID      string    `json:"contextMessageId,omitempty" db:"context_message_id"`
	ContextChatID         string    `json:"contextChatId,omitempty" db:"context_chat_id"`
	CreatedAt             time.Time `json:"createdAt" db:"created_at"`
	FirstSeenAt           time.Time `json:"firstSeenAt" db:"first_seen_at"`
	LastSeenAt            time.Time `json:"lastSeenAt" db:"last_seen_at"`
}

// Stage constants (depth-of-processing: recognition -> production).
const (
	StageRecognition  = 1
	StageCuedRecall   = 2
	StageFreeRecall   = 3
	StageProduction   = 4
	StageSpontaneous = 5
)

const (
	StateNew       = "new"
	StateLearning  = "learning"
	StateReviewing = "reviewing"
	StateMastered  = "mastered"
	StateLeech     = "leech"
	StateIgnored   = "ignored"
)

// ---------------------------------------------------------------------------
// Word mining
// ---------------------------------------------------------------------------

type MinedItem struct {
	ID                   string    `json:"id" db:"id"`
	UserID               string    `json:"userId" db:"user_id"`
	JobID                string    `json:"jobId,omitempty" db:"job_id"`
	ChatID               string    `json:"chatId,omitempty" db:"chat_id"`
	MessageID            string    `json:"messageId,omitempty" db:"message_id"`
	SourceType           string    `json:"sourceType" db:"source_type"`
	SurfaceText          string    `json:"surfaceText" db:"surface_text"`
	Lemma                string    `json:"lemma" db:"lemma"`
	NormalizedText       string    `json:"normalizedText" db:"normalized_text"`
	Language             string    `json:"language" db:"language"`
	PartOfSpeech         string    `json:"partOfSpeech" db:"part_of_speech"`
	Translation          string    `json:"translation" db:"translation"`
	Definition           string    `json:"definition" db:"definition"`
	ContextSentence      string    `json:"contextSentence" db:"context_sentence"`
	TextSpan             any       `json:"textSpan,omitempty" db:"text_span"`
	CEFRLevel            string    `json:"cefrLevel,omitempty" db:"cefr_level"`
	Confidence           float64   `json:"confidence" db:"confidence"`
	TeachabilityScore    float64   `json:"teachabilityScore" db:"teachability_score"`
	IsChunk              bool      `json:"isChunk" db:"is_chunk"`
	IsProperNoun         bool      `json:"isProperNoun" db:"is_proper_noun"`
	GrammarTags          []string  `json:"grammarTags,omitempty" db:"grammar_tags"`
	CurriculumLexicalID  string    `json:"curriculumLexicalItemId,omitempty" db:"curriculum_lexical_item_id"`
	CurriculumUnitID     string    `json:"curriculumUnitId,omitempty" db:"curriculum_unit_id"`
	RouteStatus          string    `json:"routeStatus" db:"route_status"`
	Status               string    `json:"status" db:"status"`
	RouteReason          string    `json:"routeReason" db:"route_reason"`
	CreatedAt            time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt            time.Time `json:"updatedAt" db:"updated_at"`
}

// ---------------------------------------------------------------------------
// Learning sessions
// ---------------------------------------------------------------------------

type LearningSession struct {
	ID                string    `json:"id" db:"id"`
	UserID            string    `json:"userId" db:"user_id"`
	TargetLanguage    string    `json:"targetLanguage" db:"target_language"`
	Mode              string    `json:"mode" db:"mode"`
	Status            string    `json:"status" db:"status"`
	SourceUnitID      string    `json:"sourceUnitId,omitempty" db:"source_unit_id"`
	SourceLessonID    string    `json:"sourceLessonId,omitempty" db:"source_lesson_id"`
	PlannedItemCount  int       `json:"plannedItemCount" db:"planned_item_count"`
	CompletedItemCount int      `json:"completedItemCount" db:"completed_item_count"`
	Score             int       `json:"score" db:"score"`
	XPAwarded         int       `json:"xpAwarded" db:"xp_awarded"`
	StartedAt         time.Time `json:"startedAt" db:"started_at"`
	CompletedAt       time.Time `json:"completedAt,omitempty" db:"completed_at"`
	ProgressPct       int       `json:"progressPct"`
	Items             []LearningSessionItem `json:"items,omitempty"`
}

type LearningSessionItem struct {
	ID             string    `json:"id" db:"id"`
	SessionID      string    `json:"sessionId" db:"session_id"`
	Ordinal        int       `json:"ordinal" db:"ordinal"`
	ItemType       string    `json:"itemType" db:"item_type"`
	ActivityType   string    `json:"activityType"`
	VocabularyID   string    `json:"vocabularyId,omitempty" db:"vocabulary_id"`
	GrammarPointID string    `json:"grammarPointId,omitempty" db:"grammar_point_id"`
	LessonStepID   string    `json:"lessonStepId,omitempty" db:"lesson_step_id"`
	Payload        any       `json:"payload" db:"payload"`
	Result         any       `json:"result,omitempty" db:"result"`
	Status         string    `json:"status" db:"status"`
	CreatedAt      time.Time `json:"createdAt" db:"created_at"`
}

type StartSessionRequest struct {
	TargetLanguage string `json:"targetLanguage"`
	NativeLanguage string `json:"nativeLanguage"`
	Mode           string `json:"mode"`
	Source         string `json:"source"`
}

type StartSessionResponse struct {
	Session *LearningSession  `json:"session"`
	Items   []SessionQuestion `json:"items"`
}

// SessionQuestion is the client-facing shape of a single practice item.
type SessionQuestion struct {
	ID           string      `json:"id"`
	ItemType     string      `json:"itemType"`
	ActivityType string      `json:"activityType"`
	PromptType   string      `json:"promptType"`
	Prompt       SessionPrompt `json:"prompt"`
}

type SessionPrompt struct {
	Text            string   `json:"text,omitempty"`
	Source          string   `json:"source,omitempty"`
	Translation     string   `json:"translation,omitempty"`
	Choices         []string `json:"choices,omitempty"`
	Term            string   `json:"term,omitempty"`
	Tone            string   `json:"tone,omitempty"`
	GrammarHint     string   `json:"grammarHint,omitempty"`
}

type AnswerSessionItemRequest struct {
	Answer    SessionAnswerRequest `json:"answer"`
	LatencyMs int                  `json:"latencyMs"`
}

type SessionAnswerRequest struct {
	Text    string `json:"text"`
	Choice  string `json:"choice"`
}

type AnswerSessionItemResponse struct {
	Correct       bool           `json:"correct"`
	Quality       int            `json:"quality"`
	Feedback      SessionFeedback `json:"feedback"`
	NextItem      *SessionQuestion `json:"nextItem,omitempty"`
}

type SessionFeedback struct {
	Message       string `json:"message"`
	CorrectAnswer string `json:"correctAnswer,omitempty"`
	GrammarPointID string `json:"grammarPointId,omitempty"`
	MasteryState  string `json:"masteryState,omitempty"`
}

// ---------------------------------------------------------------------------
// Lessons
// ---------------------------------------------------------------------------

type CurriculumLesson struct {
	ID               string          `json:"id" db:"id"`
	UnitID           string          `json:"unitId" db:"unit_id"`
	Ordinal          int             `json:"ordinal" db:"ordinal"`
	Slug             string          `json:"slug" db:"slug"`
	Type             string          `json:"type" db:"type"`
	Title            string          `json:"title" db:"title"`
	Objective        string          `json:"objective" db:"objective"`
	EstimatedMinutes int             `json:"estimatedMinutes" db:"estimated_minutes"`
	Steps            []CurriculumStep `json:"steps,omitempty"`
}

type CurriculumStep struct {
	ID          string    `json:"id" db:"id"`
	LessonID    string    `json:"lessonId" db:"lesson_id"`
	Ordinal     int       `json:"ordinal" db:"ordinal"`
	Type        string    `json:"type" db:"type"`
	Prompt      any       `json:"prompt" db:"prompt"`
	AnswerKey   any       `json:"answerKey,omitempty" db:"answer_key"`
	ContentRefs any       `json:"contentRefs,omitempty" db:"content_refs"`
}

type LessonAttempt struct {
	ID             string    `json:"id" db:"id"`
	UserID         string    `json:"userId" db:"user_id"`
	LessonID       string    `json:"lessonId" db:"lesson_id"`
	TargetLanguage string    `json:"targetLanguage" db:"target_language"`
	Status         string    `json:"status" db:"status"`
	Score          int       `json:"score" db:"score"`
	CorrectCount   int       `json:"correctCount" db:"correct_count"`
	TotalCount     int       `json:"totalCount" db:"total_count"`
	StartedAt      time.Time `json:"startedAt" db:"started_at"`
	CompletedAt    time.Time `json:"completedAt,omitempty" db:"completed_at"`
	Steps          []LessonStepResult `json:"steps,omitempty"`
}

type LessonStepResult struct {
	ID        string `json:"id" db:"id"`
	StepID    string `json:"stepId" db:"step_id"`
	UserAnswer any   `json:"userAnswer" db:"user_answer"`
	Correct   bool   `json:"correct" db:"correct"`
	Score     int    `json:"score" db:"score"`
	Feedback  any    `json:"feedback" db:"feedback"`
}

type AnswerLessonStepRequest struct {
	UserAnswer string `json:"answer" binding:"required"`
}

type LessonStartResponse struct {
	Attempt *LessonAttempt    `json:"attempt"`
	Steps   []CurriculumStep  `json:"steps"`
}

// ---------------------------------------------------------------------------
// Placement
// ---------------------------------------------------------------------------

type PlacementQuestion struct {
	ID        string      `json:"id"`
	Ref       string      `json:"ref"`
	ItemType  string      `json:"itemType"`
	CEFRLevel string      `json:"cefrLevel"`
	Prompt    any         `json:"prompt"`
	Choices   []string    `json:"choices,omitempty"`
}

type PlacementStartResponse struct {
	AttemptID string             `json:"attemptId"`
	Status    string             `json:"status"`
	Question  PlacementQuestion  `json:"question"`
	TotalQuestions int           `json:"totalQuestions"`
}

type PlacementAnswerRequest struct {
	Answer string `json:"answer" binding:"required"`
}

type PlacementResult struct {
	AttemptID      string `json:"attemptId"`
	EstimatedCEFR  string `json:"estimatedCefr"`
	ReadinessScore int    `json:"readinessScore"`
	ActiveUnitID   string `json:"activeUnitId"`
	UnitsSkipped   []string `json:"skippedUnits,omitempty"`
}

// ---------------------------------------------------------------------------
// Scenarios
// ---------------------------------------------------------------------------

type ScenarioScript struct {
	ID                string          `json:"id" db:"id"`
	CourseID          string          `json:"courseId" db:"course_id"`
	UnitID            string          `json:"unitId,omitempty" db:"unit_id"`
	Slug              string          `json:"slug" db:"slug"`
	Title             string          `json:"title" db:"title"`
	Domain            string          `json:"domain" db:"domain"`
	CEFRLevel         string          `json:"cefrLevel" db:"cefr_level"`
	CanDoStatement    string          `json:"canDoStatement" db:"can_do_statement"`
	AIRoleName        string          `json:"aiRoleName" db:"ai_role_name"`
	AIRoleDescription string          `json:"aiRoleDescription" db:"ai_role_description"`
	OpeningLine       string          `json:"openingLine" db:"opening_line"`
	MaxTurns          int             `json:"maxTurns" db:"max_turns"`
	EstimatedMinutes  int             `json:"estimatedMinutes" db:"estimated_minutes"`
	CompletionCriteria any             `json:"completionCriteria,omitempty" db:"completion_criteria"`
	Phases            []ScenarioPhase `json:"phases,omitempty"`
	Metadata          any             `json:"metadata,omitempty" db:"-"`
}

type ScenarioPhase struct {
	ID             string     `json:"id" db:"id"`
	ScenarioID     string     `json:"scenarioId" db:"scenario_id"`
	Ordinal        int        `json:"ordinal" db:"ordinal"`
	Title          string     `json:"title" db:"title"`
	LearnerGoal    string     `json:"learnerGoal" db:"learner_goal"`
	RequiredIntents []string  `json:"requiredIntents" db:"required_intents"`
	ChunkBank      []Chunk    `json:"chunkBank" db:"chunk_bank"`
}

type Chunk struct {
	Text        string `json:"text"`
	Translation string `json:"translation"`
}

type ScenarioRun struct {
	ID                 string    `json:"id" db:"id"`
	UserID             string    `json:"userId" db:"user_id"`
	ScenarioID         string    `json:"scenarioId" db:"scenario_id"`
	TargetLanguage     string    `json:"targetLanguage" db:"target_language"`
	NativeLanguage     string    `json:"nativeLanguage" db:"native_language"`
	Status             string    `json:"status" db:"status"`
	ScaffoldLevel      string    `json:"scaffoldLevel" db:"scaffold_level"`
	CurrentPhaseOrdinal int      `json:"currentPhaseOrdinal" db:"current_phase_ordinal"`
	PhaseScores        any       `json:"phaseScores" db:"phase_scores"`
	CoveredIntents     []string  `json:"coveredIntents" db:"covered_intents"`
	Score              int       `json:"score" db:"score"`
	XPAwarded          int       `json:"xpAwarded" db:"xp_awarded"`
	StartedAt          time.Time `json:"startedAt" db:"started_at"`
	CompletedAt        time.Time `json:"completedAt,omitempty" db:"completed_at"`
	Turns              []ScenarioTurn `json:"turns,omitempty"`
	CurrentPhase       *ScenarioPhase `json:"currentPhase,omitempty"`
	SuggestedChunks    []Chunk        `json:"suggestedChunks,omitempty"`
}

type ScenarioTurn struct {
	ID          string   `json:"id" db:"id"`
	RunID       string   `json:"runId" db:"run_id"`
	Ordinal     int      `json:"ordinal" db:"ordinal"`
	Speaker     string   `json:"speaker" db:"speaker"`
	Text        string   `json:"text" db:"text"`
	Translation string   `json:"translation" db:"translation"`
	PhaseOrdinal int     `json:"phaseOrdinal" db:"phase_ordinal"`
	Evaluation  any      `json:"evaluation,omitempty" db:"evaluation"`
	CreatedAt   time.Time `json:"createdAt" db:"created_at"`
}

type ScenarioStartResponse struct {
	Run        *ScenarioRun  `json:"run"`
	AIResponse ScenarioAIReply `json:"aiResponse"`
}

type SendScenarioMessageRequest struct {
	Message string `json:"message" binding:"required"`
}

type ScenarioAIReply struct {
	AIMessage       string        `json:"aiMessage"`
	Translation     string        `json:"translation"`
	PhaseComplete   bool          `json:"phaseComplete"`
	NextPhaseOrdinal int          `json:"nextPhaseOrdinal,omitempty"`
	CoveredIntents  []string      `json:"coveredIntents,omitempty"`
	Errors          []ScenarioError `json:"errors,omitempty"`
	Score           int           `json:"score"`
	RunCompleted    bool          `json:"runCompleted"`
	Summary         *ScenarioSummaryResult `json:"summary,omitempty"`
	SuggestedChunks []Chunk        `json:"suggestedChunks,omitempty"`
	ShouldSelfCorrect bool         `json:"shouldSelfCorrect,omitempty"`
	FirstPassDone    bool          `json:"firstPassDone,omitempty"`
}

type ScenarioError struct {
	Span        string `json:"span,omitempty"`
	Correction  string `json:"correction,omitempty"`
	GrammarTag  string `json:"grammarTag,omitempty"`
	Explanation string `json:"explanation,omitempty"`
}

type ScenarioSummaryResult struct {
	Score          int `json:"score"`
	XPAwarded      int `json:"xpAwarded"`
	Mins           int `json:"minutes"`
	VocabularyAdded int `json:"vocabularyAdded"`
}

// ---------------------------------------------------------------------------
// Real talk / nudges / streak recovery
// ---------------------------------------------------------------------------

type RealTalkPrompt struct {
	ID        string `json:"id"`
	Category  string `json:"category"`
	Text      string `json:"text"`
	SourcePrompt bool `json:"sourcePrompt,omitempty"`
}

type StreakRecoverResult struct {
	NewStreak int `json:"newStreak"`
	Recovered bool `json:"recovered"`
}
