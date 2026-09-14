package models

import "time"

// Stage constants (depth-of-processing: recognition -> production).
const (
	StageRecognition = 1
	StageCuedRecall  = 2
	StageFreeRecall  = 3
	StageProduction  = 4
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

type VocabularyCard struct {
	ID                     string    `json:"id" db:"id"`
	UserID                 string    `json:"userId" db:"user_id"`
	Term                   string    `json:"term" db:"term"`
	Language               string    `json:"language" db:"language"`
	Translation            string    `json:"translation" db:"translation"`
	Definition             string    `json:"definition" db:"definition"`
	Lemma                  string    `json:"lemma" db:"lemma"`
	NormalizedTerm         string    `json:"normalizedTerm" db:"normalized_term"`
	PartOfSpeech           string    `json:"partOfSpeech" db:"part_of_speech"`
	IsChunk                bool      `json:"isChunk" db:"is_chunk"`
	SourceType             string    `json:"sourceType" db:"source_type"`
	SourceMessageID        string    `json:"sourceMessageId,omitempty" db:"source_message_id"`
	CEFRLevel              string    `json:"cefrLevel,omitempty" db:"cefr_level"`
	CurriculumUnitID       string    `json:"curriculumUnitId,omitempty" db:"curriculum_unit_id"`
	RouteStatus            string    `json:"routeStatus" db:"route_status"`
	MasteryStage           int       `json:"masteryStage" db:"mastery_stage"`
	MasteryState           string    `json:"masteryState" db:"mastery_state"`
	EaseFactor             float64   `json:"easeFactor" db:"ease_factor"`
	Lapses                 int       `json:"lapses" db:"lapses"`
	StageSuccessCount      int       `json:"stageSuccessCount" db:"stage_success_count"`
	ProductionSuccessCount int       `json:"productionSuccessCount" db:"production_success_count"`
	SpontaneousUseCount    int       `json:"spontaneousUseCount" db:"spontaneous_use_count"`
	TeachabilityScore      float64   `json:"teachabilityScore" db:"teachability_score"`
	Confidence             float64   `json:"confidence" db:"confidence"`
	ReviewCount            int       `json:"reviewCount" db:"review_count"`
	CorrectCount           int       `json:"correctCount" db:"correct_count"`
	IntervalDays           float64   `json:"intervalDays" db:"interval_days"`
	NextReview             time.Time `json:"nextReview" db:"next_review"`
	ContextSentence        string    `json:"contextSentence,omitempty" db:"context_sentence"`
	ContextMessageID       string    `json:"contextMessageId,omitempty" db:"context_message_id"`
	ContextChatID          string    `json:"contextChatId,omitempty" db:"context_chat_id"`
	CreatedAt              time.Time `json:"createdAt" db:"created_at"`
	FirstSeenAt            time.Time `json:"firstSeenAt" db:"first_seen_at"`
	LastSeenAt             time.Time `json:"lastSeenAt" db:"last_seen_at"`
}

type LearningSession struct {
	ID                 string                `json:"id" db:"id"`
	UserID             string                `json:"userId" db:"user_id"`
	TargetLanguage     string                `json:"targetLanguage" db:"target_language"`
	Mode               string                `json:"mode" db:"mode"`
	Status             string                `json:"status" db:"status"`
	SourceUnitID       string                `json:"sourceUnitId,omitempty" db:"source_unit_id"`
	SourceLessonID     string                `json:"sourceLessonId,omitempty" db:"source_lesson_id"`
	PlannedItemCount   int                   `json:"plannedItemCount" db:"planned_item_count"`
	CompletedItemCount int                   `json:"completedItemCount" db:"completed_item_count"`
	Score              int                   `json:"score" db:"score"`
	XPAwarded          int                   `json:"xpAwarded" db:"xp_awarded"`
	StartedAt          time.Time             `json:"startedAt" db:"started_at"`
	CompletedAt        time.Time             `json:"completedAt,omitempty" db:"completed_at"`
	ProgressPct        int                   `json:"progressPct"`
	Items              []LearningSessionItem `json:"items,omitempty"`
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

type SessionQuestion struct {
	ID           string        `json:"id"`
	ItemType     string        `json:"itemType"`
	ActivityType string        `json:"activityType"`
	PromptType   string        `json:"promptType"`
	Prompt       SessionPrompt `json:"prompt"`
}

type SessionPrompt struct {
	Text        string   `json:"text,omitempty"`
	Source      string   `json:"source,omitempty"`
	Translation string   `json:"translation,omitempty"`
	Choices     []string `json:"choices,omitempty"`
	Term        string   `json:"term,omitempty"`
	Tone        string   `json:"tone,omitempty"`
	GrammarHint string   `json:"grammarHint,omitempty"`
}

type AnswerSessionItemRequest struct {
	Answer    SessionAnswerRequest `json:"answer"`
	LatencyMs int                  `json:"latencyMs"`
}

type SessionAnswerRequest struct {
	Text   string `json:"text"`
	Choice string `json:"choice"`
}

type AnswerSessionItemResponse struct {
	Correct  bool             `json:"correct"`
	Quality  int              `json:"quality"`
	Feedback SessionFeedback  `json:"feedback"`
	NextItem *SessionQuestion `json:"nextItem,omitempty"`
}

type SessionFeedback struct {
	Message        string `json:"message"`
	CorrectAnswer  string `json:"correctAnswer,omitempty"`
	GrammarPointID string `json:"grammarPointId,omitempty"`
	MasteryState   string `json:"masteryState,omitempty"`
}

const (
	QueueOriginSeed     = "seed"
	QueueOriginPersonal = "personal"
	QueueOriginGrammar  = "grammar"
)

type SRSQueueItem struct {
	ID         string          `json:"id"`
	Origin     string          `json:"origin"`
	ItemType   string          `json:"itemType"`
	Due        bool            `json:"due"`
	NextReview time.Time       `json:"nextReview,omitempty"`
	Answer     string          `json:"answer,omitempty"`
	Question   SessionQuestion `json:"question"`
}

type SRSQueue struct {
	TargetLanguage string         `json:"targetLanguage"`
	TotalDue       int            `json:"totalDue"`
	TotalNew       int            `json:"totalNew"`
	Seeded         int            `json:"seeded"`
	Items          []SRSQueueItem `json:"items"`
}
