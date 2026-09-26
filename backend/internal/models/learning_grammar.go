package models

import "time"

type GrammarItem struct {
	ID                string   `json:"id" db:"id"`
	GrammarPointID    string   `json:"grammarPointId" db:"grammar_point_id"`
	Ordinal           int      `json:"ordinal" db:"ordinal"`
	ItemType          string   `json:"itemType" db:"item_type"` // cloze, mcq, reconstruction
	Prompt            string   `json:"prompt" db:"prompt"`
	SentenceWithBlank string   `json:"sentenceWithBlank,omitempty" db:"sentence_with_blank"`
	Choices           []string `json:"choices" db:"choices"`
	Correct           string   `json:"correct" db:"correct"`
	AcceptVariants    []string `json:"acceptVariants" db:"accept_variants"`
	Note              string   `json:"note" db:"note"`
	IsActive          bool     `json:"isActive" db:"is_active"`
}

type GrammarItemAttempt struct {
	ID             string    `json:"id" db:"id"`
	UserID         string    `json:"userId" db:"user_id"`
	GrammarItemID  string    `json:"grammarItemId" db:"grammar_item_id"`
	GrammarPointID string    `json:"grammarPointId" db:"grammar_point_id"`
	Correct        bool      `json:"correct" db:"correct"`
	Quality        int       `json:"quality" db:"quality"`
	LatencyMs      int       `json:"latencyMs" db:"latency_ms"`
	CreatedAt      time.Time `json:"createdAt" db:"created_at"`
}

type UserGrammarMastery struct {
	UserID         string     `json:"userId" db:"user_id"`
	GrammarPointID string     `json:"grammarPointId" db:"grammar_point_id"`
	MasteryStage   int        `json:"masteryStage" db:"mastery_stage"`
	EaseFactor     float64    `json:"easeFactor" db:"ease_factor"`
	Repetitions    int        `json:"repetitions" db:"repetitions"`
	IntervalDays   int        `json:"intervalDays" db:"interval_days"`
	NextReviewAt   time.Time  `json:"nextReviewAt" db:"next_review_at"`
	LastReviewedAt *time.Time `json:"lastReviewedAt,omitempty" db:"last_reviewed_at"`
	TargetLanguage string     `json:"targetLanguage" db:"target_language"`
	Confidence     float64    `json:"confidence" db:"confidence"`
	SeenCount      int        `json:"seenCount" db:"seen_count"`
	CorrectCount   int        `json:"correctCount" db:"correct_count"`
	ErrorCount     int        `json:"errorCount" db:"error_count"`
}

type GrammarMicroLesson struct {
	PointID       string   `json:"pointId"`
	Title         string   `json:"title"`
	CEFRLevel     string   `json:"cefrLevel"`
	RuleText      string   `json:"ruleText"`
	CommonTrap    string   `json:"commonTrap"`
	Examples      any      `json:"examples"`
	Prerequisites []string `json:"prerequisites"`
}

type GrammarDrillItem struct {
	ID                string   `json:"id"`
	GrammarPointID    string   `json:"grammarPointId"`
	PointTitle        string   `json:"pointTitle"`
	CEFRLevel         string   `json:"cefrLevel,omitempty"`
	Ordinal           int      `json:"ordinal"`
	ItemType          string   `json:"itemType"`
	Prompt            string   `json:"prompt"`
	SentenceWithBlank string   `json:"sentenceWithBlank,omitempty"`
	Choices           []string `json:"choices,omitempty"`
	Correct           string   `json:"correct"`
	Note              string   `json:"note,omitempty"`
}

type GrammarPointDetail struct {
	Point   GrammarPoint        `json:"point"`
	Mastery *UserGrammarMastery `json:"mastery,omitempty"`
	Lesson  *GrammarMicroLesson `json:"lesson,omitempty"`
}

type RecordGrammarAttemptRequest struct {
	Correct   bool `json:"correct"`
	Quality   int  `json:"quality"`
	LatencyMs int  `json:"latencyMs"`
}
