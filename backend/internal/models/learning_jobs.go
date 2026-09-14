package models

import "time"

type GradingJobStatus string

const (
	GradingJobStatusPending    GradingJobStatus = "pending"
	GradingJobStatusProcessing GradingJobStatus = "processing"
	GradingJobStatusDone       GradingJobStatus = "done"
	GradingJobStatusFailed     GradingJobStatus = "failed"
)

type GradingJob struct {
	ID            string           `json:"id" db:"id"`
	UserID        string           `json:"userId" db:"user_id"`
	SubmissionID  *string          `json:"submissionId,omitempty" db:"submission_id"`
	VocabularyID  *string          `json:"vocabularyId,omitempty" db:"vocabulary_id"`
	JobType       string           `json:"jobType" db:"job_type"` // production, writing, placement_open
	Payload       any              `json:"payload" db:"payload"`
	Status        GradingJobStatus `json:"status" db:"status"`
	Result        any              `json:"result,omitempty" db:"result"`
	Attempts      int              `json:"attempts" db:"attempts"`
	LastError     *string          `json:"lastError,omitempty" db:"last_error"`
	CreatedAt     time.Time        `json:"createdAt" db:"created_at"`
	NextAttemptAt time.Time        `json:"nextAttemptAt" db:"next_attempt_at"`
	ProcessingAt  *time.Time       `json:"processingAt,omitempty" db:"processing_at"`
	CompletedAt   *time.Time       `json:"completedAt,omitempty" db:"completed_at"`
}

type GradingJobPayload struct {
	TaskType       string `json:"taskType"`
	Prompt         string `json:"prompt"`
	UserText       string `json:"userText"`
	CEFRLevel      string `json:"cefrLevel,omitempty"`
	TargetLanguage string `json:"targetLanguage"`
	NativeLanguage string `json:"nativeLanguage"`
	FocusGrammar   string `json:"focusGrammar,omitempty"`
	FocusVocab     string `json:"focusVocab,omitempty"`
}

type GradingJobResult struct {
	JobID         string          `json:"jobId"`
	Score         int             `json:"score"`
	MaxScore      int             `json:"maxScore"`
	Feedback      string          `json:"feedback"`
	Corrections   []ScenarioError `json:"corrections,omitempty"`
	CEFRBand      string          `json:"cefrBand,omitempty"`
	AccuracyScore float64         `json:"accuracyScore,omitempty"`
}
