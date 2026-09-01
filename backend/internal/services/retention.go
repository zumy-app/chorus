package services

import (
	"context"
	"database/sql"
	"time"
)

const (
	DefaultMessageRetentionDays     = 365
	DefaultInboxRetentionDays       = 30
	DefaultTranslationRetentionDays = 90
	DefaultCallTranscriptRetentionDays = 90
)

type RetentionPolicy struct {
	MessageRetentionDays          int    `json:"messageRetentionDays"`
	InboxRetentionDays            int    `json:"inboxRetentionDays"`
	TranslationJobRetentionDays    int    `json:"translationJobRetentionDays"`
	CallTranscriptRetentionDays    int    `json:"callTranscriptRetentionDays"`
	EmailOutboxRetentionDays       int    `json:"emailOutboxRetentionDays"`
	Description                   string `json:"description"`
}

type RetentionResult struct {
	MessagesDeleted      int64 `json:"messagesDeleted"`
	InboxDeleted         int64 `json:"inboxDeleted"`
	TranslationJobsDeleted int64 `json:"translationJobsDeleted"`
	CallTranscriptsDeleted int64 `json:"callTranscriptsDeleted"`
}

type RetentionService struct {
	db *sql.DB
}

func NewRetentionService(db *sql.DB) *RetentionService {
	return &RetentionService{db: db}
}

func (s *RetentionService) GetPolicy() RetentionPolicy {
	return RetentionPolicy{
		MessageRetentionDays:          DefaultMessageRetentionDays,
		InboxRetentionDays:            DefaultInboxRetentionDays,
		TranslationJobRetentionDays:    DefaultTranslationRetentionDays,
		CallTranscriptRetentionDays:    DefaultCallTranscriptRetentionDays,
		EmailOutboxRetentionDays:       90,
		Description:                   "GDPR-oriented minimal retention: messages respect per-user message_retention_days (default 365d) and are hard-deleted after expiry; inbox 30d, translation jobs 90d, call transcripts 90d or transcript_recording off. See docs/DATA_RETENTION_GDPR.md",
	}
}

func (s *RetentionService) PurgeExpired(ctx context.Context) (RetentionResult, error) {
	var r RetentionResult
	if n, err := s.PurgeExpiredMessages(ctx); err != nil {
		return r, err
	} else {
		r.MessagesDeleted = n
	}
	if n, err := s.PurgeExpiredInbox(ctx); err != nil {
		return r, err
	} else {
		r.InboxDeleted = n
	}
	if n, err := s.PurgeExpiredTranslationJobs(ctx); err != nil {
		return r, err
	} else {
		r.TranslationJobsDeleted = n
	}
	if n, err := s.PurgeExpiredCallTranscripts(ctx); err != nil {
		return r, err
	} else {
		r.CallTranscriptsDeleted = n
	}
	return r, nil
}

func (s *RetentionService) PurgeExpiredMessages(ctx context.Context) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
		DELETE FROM messages
		WHERE deleted_at IS NULL
		AND created_at < NOW() - COALESCE(
			(SELECT (message_retention_days || ' days')::interval FROM user_settings WHERE user_settings.user_id = messages.sender_id),
			($1 || ' days')::interval
		)`)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}

func (s *RetentionService) PurgeExpiredInbox(ctx context.Context) (int64, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM inbox WHERE ttl < NOW()`)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}

func (s *RetentionService) PurgeExpiredTranslationJobs(ctx context.Context) (int64, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM translation_jobs WHERE status IN ('done','failed') AND completed_at < NOW() - INTERVAL '90 days'`)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}

func (s *RetentionService) PurgeExpiredCallTranscripts(ctx context.Context) (int64, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM call_transcripts WHERE created_at < NOW() - INTERVAL '90 days'`)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}

func (s *RetentionService) PurgeExpiredMedia(ctx context.Context) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
		DELETE FROM media_attachments
		WHERE message_id IN (SELECT id FROM messages WHERE deleted_at IS NOT NULL AND deleted_at < NOW() - INTERVAL '30 days')`)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}

func (s *RetentionService) StartScheduler(interval time.Duration) chan struct{} {
	stop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				_, _ = s.PurgeExpired(context.Background())
			case <-stop:
				return
			}
		}
	}()
	return stop
}
