package models

import "time"

type ScenarioScript struct {
	ID                 string          `json:"id" db:"id"`
	CourseID           string          `json:"courseId" db:"course_id"`
	UnitID             string          `json:"unitId,omitempty" db:"unit_id"`
	Slug               string          `json:"slug" db:"slug"`
	Title              string          `json:"title" db:"title"`
	Domain             string          `json:"domain" db:"domain"`
	CEFRLevel          string          `json:"cefrLevel" db:"cefr_level"`
	CanDoStatement     string          `json:"canDoStatement" db:"can_do_statement"`
	AIRoleName         string          `json:"aiRoleName" db:"ai_role_name"`
	AIRoleDescription  string          `json:"aiRoleDescription" db:"ai_role_description"`
	OpeningLine        string          `json:"openingLine" db:"opening_line"`
	MaxTurns           int             `json:"maxTurns" db:"max_turns"`
	EstimatedMinutes   int             `json:"estimatedMinutes" db:"estimated_minutes"`
	ScaffoldInitial    string          `json:"scaffoldInitial,omitempty" db:"scaffold_initial"`
	CompletionCriteria any             `json:"completionCriteria,omitempty" db:"completion_criteria"`
	Phases             []ScenarioPhase `json:"phases,omitempty"`
	Metadata           any             `json:"metadata,omitempty" db:"-"`
}

type ScenarioPhase struct {
	ID              string   `json:"id" db:"id"`
	ScenarioID      string   `json:"scenarioId" db:"scenario_id"`
	Ordinal         int      `json:"ordinal" db:"ordinal"`
	Title           string   `json:"title" db:"title"`
	LearnerGoal     string   `json:"learnerGoal" db:"learner_goal"`
	RequiredIntents []string `json:"requiredIntents" db:"required_intents"`
	ScaffoldHints   any      `json:"scaffoldHints,omitempty" db:"scaffold_hints"`
	AIFollowUp      string   `json:"aiFollowUp,omitempty" db:"ai_follow_up"`
	SuccessExamples any      `json:"successExamples,omitempty" db:"success_examples"`
	ChunkBank       []Chunk  `json:"chunkBank" db:"chunk_bank"`
}

type Chunk struct {
	Text        string `json:"text"`
	Translation string `json:"translation"`
}

type ScenarioRun struct {
	ID                  string         `json:"id" db:"id"`
	UserID              string         `json:"userId" db:"user_id"`
	ScenarioID          string         `json:"scenarioId" db:"scenario_id"`
	TargetLanguage      string         `json:"targetLanguage" db:"target_language"`
	NativeLanguage      string         `json:"nativeLanguage" db:"native_language"`
	Status              string         `json:"status" db:"status"`
	ScaffoldLevel       string         `json:"scaffoldLevel" db:"scaffold_level"`
	CurrentPhaseOrdinal int            `json:"currentPhaseOrdinal" db:"current_phase_ordinal"`
	PhaseScores         any            `json:"phaseScores" db:"phase_scores"`
	CoveredIntents      []string       `json:"coveredIntents" db:"covered_intents"`
	Score               int            `json:"score" db:"score"`
	XPAwarded           int            `json:"xpAwarded" db:"xp_awarded"`
	AssignmentID        *string        `json:"assignmentId,omitempty" db:"assignment_id"`
	StartedAt           time.Time      `json:"startedAt" db:"started_at"`
	CompletedAt         time.Time      `json:"completedAt,omitempty" db:"completed_at"`
	Turns               []ScenarioTurn `json:"turns,omitempty"`
	CurrentPhase        *ScenarioPhase `json:"currentPhase,omitempty"`
	SuggestedChunks     []Chunk        `json:"suggestedChunks,omitempty"`
}

type ScenarioTurn struct {
	ID           string    `json:"id" db:"id"`
	RunID        string    `json:"runId" db:"run_id"`
	Ordinal      int       `json:"ordinal" db:"ordinal"`
	Speaker      string    `json:"speaker" db:"speaker"`
	Text         string    `json:"text" db:"text"`
	Translation  string    `json:"translation" db:"translation"`
	PhaseOrdinal int       `json:"phaseOrdinal" db:"phase_ordinal"`
	Evaluation   any       `json:"evaluation,omitempty" db:"evaluation"`
	CreatedAt    time.Time `json:"createdAt" db:"created_at"`
}

type ScenarioStartResponse struct {
	Run        *ScenarioRun    `json:"run"`
	AIResponse ScenarioAIReply `json:"aiResponse"`
}

type SendScenarioMessageRequest struct {
	Message string `json:"message" binding:"required"`
}

type ScenarioAIReply struct {
	AIMessage         string                 `json:"aiMessage"`
	Translation       string                 `json:"translation"`
	PhaseComplete     bool                   `json:"phaseComplete"`
	NextPhaseOrdinal  int                    `json:"nextPhaseOrdinal,omitempty"`
	CoveredIntents    []string               `json:"coveredIntents,omitempty"`
	Errors            []ScenarioError        `json:"errors,omitempty"`
	Score             int                    `json:"score"`
	RunCompleted      bool                   `json:"runCompleted"`
	Summary           *ScenarioSummaryResult `json:"summary,omitempty"`
	SuggestedChunks   []Chunk                `json:"suggestedChunks,omitempty"`
	ShouldSelfCorrect bool                   `json:"shouldSelfCorrect,omitempty"`
	FirstPassDone     bool                   `json:"firstPassDone,omitempty"`
}

type ScenarioError struct {
	Span        string `json:"span,omitempty"`
	Correction  string `json:"correction,omitempty"`
	GrammarTag  string `json:"grammarTag,omitempty"`
	Explanation string `json:"explanation,omitempty"`
}

type ScenarioSummaryResult struct {
	Score           int `json:"score"`
	XPAwarded       int `json:"xpAwarded"`
	Mins            int `json:"minutes"`
	VocabularyAdded int `json:"vocabularyAdded"`
}
