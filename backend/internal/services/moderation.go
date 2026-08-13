package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/chorus/messenger/internal/models"
)

var (
	ErrBlockSelf             = errors.New("cannot block yourself")
	ErrBlockUserNotFound     = errors.New("user not found")
	ErrReportSelf            = errors.New("cannot report yourself")
	ErrReportInvalidTarget   = errors.New("report must include a target user or message")
	ErrReportMessageNotFound = errors.New("message not found")
	ErrReportNotFound        = errors.New("report not found")
)

// Report statuses.
const (
	ReportOpen      = "open"
	ReportResolved  = "resolved"
	ReportDismissed = "dismissed"
)

// ModerationService owns the safety rails: user blocks and moderation reports.
type ModerationService struct {
	db *sql.DB
}

func NewModerationService(db *sql.DB) *ModerationService {
	return &ModerationService{db: db}
}

// Block records a directed block. The relationship is treated as mutual for
// enforcement (either direction blocks communication). Blocking yourself or a
// missing user is rejected.
func (s *ModerationService) Block(ctx context.Context, blockerID, blockedID, reason string) error {
	if blockerID == blockedID {
		return ErrBlockSelf
	}
	var exists bool
	if err := s.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM users WHERE id = $1 AND deleted_at IS NULL)`, blockedID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrBlockUserNotFound
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO blocked_users (blocker_id, blocked_id, reason)
		VALUES ($1, $2, $3)
		ON CONFLICT (blocker_id, blocked_id) DO UPDATE SET reason = EXCLUDED.reason`,
		blockerID, blockedID, strings.TrimSpace(reason))
	return err
}

// Unblock removes a directed block (idempotent).
func (s *ModerationService) Unblock(ctx context.Context, blockerID, blockedID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM blocked_users WHERE blocker_id = $1 AND blocked_id = $2`, blockerID, blockedID)
	return err
}

// GetBlocked returns the users a user has blocked, newest first, enriched with
// the blocked user's profile fields for display.
func (s *ModerationService) GetBlocked(ctx context.Context, userID string) ([]models.Block, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT b.id, b.blocker_id, b.blocked_id, b.reason, b.created_at,
		       u.id, u.username, u.display_name, u.email
		FROM blocked_users b
		INNER JOIN users u ON u.id = b.blocked_id
		WHERE b.blocker_id = $1
		ORDER BY b.created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	blocks := []models.Block{}
	for rows.Next() {
		var b models.Block
		var u models.User
		if err := rows.Scan(&b.ID, &b.BlockerID, &b.BlockedID, &b.Reason, &b.CreatedAt,
			&u.ID, &u.Username, &u.DisplayName, &u.Email); err != nil {
			return nil, err
		}
		b.Blocked = &u
		blocks = append(blocks, b)
	}
	return blocks, rows.Err()
}

// IsBlocked reports whether either direction of a block exists between two
// users (blocks are mutual for enforcement).
func (s *ModerationService) IsBlocked(ctx context.Context, userA, userB string) (bool, error) {
	var blocked bool
	err := s.db.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM blocked_users
			WHERE (blocker_id = $1 AND blocked_id = $2)
			   OR (blocker_id = $2 AND blocked_id = $1)
		)`, userA, userB).Scan(&blocked)
	return blocked, err
}

// ChatBlocked reports whether the sender is blocked from the chat: either the
// sender blocked any participant, or any participant blocked the sender.
func (s *ModerationService) ChatBlocked(ctx context.Context, chatID, senderID string) (bool, error) {
	var blocked bool
	err := s.db.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM blocked_users b
			INNER JOIN chat_participants cp ON cp.chat_id = $1
			WHERE cp.user_id != $2
			  AND ((cp.user_id = b.blocked_id AND b.blocker_id = $2)
			    OR (cp.user_id = b.blocker_id AND b.blocked_id = $2))
		)`, chatID, senderID).Scan(&blocked)
	return blocked, err
}

// Report files a moderation report. For message reports the reported user and
// chat are resolved from the message row; the reporter cannot report
// themselves.
func (s *ModerationService) Report(ctx context.Context, reporterID string, req models.ReportRequest) (*models.Report, error) {
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		return nil, ErrReportInvalidTarget
	}

	report := &models.Report{
		ReporterID: reporterID,
		Type:       req.Type,
		Reason:     reason,
		Status:     ReportOpen,
	}

	var err error
	switch req.Type {
	case "message":
		if req.MessageID == "" {
			return nil, ErrReportInvalidTarget
		}
		var senderID string
		var chatID *string
		err = s.db.QueryRowContext(ctx, `
			SELECT sender_id, chat_id FROM messages WHERE id = $1`, req.MessageID).Scan(&senderID, &chatID)
		if err == sql.ErrNoRows {
			return nil, ErrReportMessageNotFound
		}
		if err != nil {
			return nil, err
		}
		if senderID == reporterID {
			return nil, ErrReportSelf
		}
		report.ReportedUserID = senderID
		report.MessageID = &req.MessageID
		report.ChatID = chatID
	case "user":
		if req.ReportedUserID == "" {
			return nil, ErrReportInvalidTarget
		}
		if req.ReportedUserID == reporterID {
			return nil, ErrReportSelf
		}
		var exists bool
		if err := s.db.QueryRowContext(ctx,
			`SELECT EXISTS(SELECT 1 FROM users WHERE id = $1 AND deleted_at IS NULL)`, req.ReportedUserID).Scan(&exists); err != nil {
			return nil, err
		}
		if !exists {
			return nil, ErrBlockUserNotFound
		}
		report.ReportedUserID = req.ReportedUserID
	default:
		return nil, ErrReportInvalidTarget
	}

	err = s.db.QueryRowContext(ctx, `
		INSERT INTO reports (reporter_id, type, reported_user_id, message_id, chat_id, reason, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at`, report.ReporterID, report.Type, report.ReportedUserID,
		report.MessageID, report.ChatID, report.Reason, ReportOpen).Scan(&report.ID, &report.CreatedAt)
	if err != nil {
		return nil, err
	}
	return report, nil
}

// ListReports returns reports filtered by status (empty = all) and an optional
// query matching the reported user's username/email, along with the total
// count. Enriched with reporter and reported user profile fields.
func (s *ModerationService) ListReports(ctx context.Context, status, q string, limit, offset int) ([]models.Report, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	where := "WHERE 1=1"
	args := []interface{}{}
	if status != "" && status != "all" {
		args = append(args, status)
		where += fmt.Sprintf(` AND r.status = $%d`, len(args))
	}
	if q = strings.TrimSpace(q); q != "" {
		args = append(args, "%"+strings.ToLower(q)+"%")
		where += fmt.Sprintf(` AND (LOWER(repu.username) LIKE $%d OR LOWER(repu.email) LIKE $%d)`, len(args), len(args))
	}

	var total int
	countArgs := append([]interface{}{}, args...)
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM reports r INNER JOIN users repu ON repu.id = r.reported_user_id `+where, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	queryArgs := append(append([]interface{}{}, args...), limit, offset)
	query := `
		SELECT r.id, r.reporter_id, r.type, r.reported_user_id, r.message_id, r.chat_id,
		       r.reason, r.status, r.resolver_id, r.resolution_note, r.created_at, r.resolved_at,
		       rep.id, rep.username, rep.display_name, rep.email,
		       repu.id, repu.username, repu.display_name, repu.email
		FROM reports r
		INNER JOIN users rep ON rep.id = r.reporter_id
		INNER JOIN users repu ON repu.id = r.reported_user_id ` + where + `
		ORDER BY r.created_at DESC
		LIMIT $` + fmt.Sprintf("%d", len(args)+1) + ` OFFSET $` + fmt.Sprintf("%d", len(args)+2)

	rows, err := s.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	reports := []models.Report{}
	for rows.Next() {
		var r models.Report
		var rep, repu models.User
		if err := rows.Scan(&r.ID, &r.ReporterID, &r.Type, &r.ReportedUserID, &r.MessageID, &r.ChatID,
			&r.Reason, &r.Status, &r.ResolverID, &r.ResolutionNote, &r.CreatedAt, &r.ResolvedAt,
			&rep.ID, &rep.Username, &rep.DisplayName, &rep.Email,
			&repu.ID, &repu.Username, &repu.DisplayName, &repu.Email); err != nil {
			return nil, 0, err
		}
		r.Reporter = &rep
		r.ReportedUser = &repu
		reports = append(reports, r)
	}
	return reports, total, rows.Err()
}

// GetReportStats aggregates moderation figures for the admin console.
func (s *ModerationService) GetReportStats(ctx context.Context) (models.ReportStats, error) {
	var stats models.ReportStats
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM reports WHERE status = 'open'`).Scan(&stats.OpenReports); err != nil {
		return stats, err
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM reports WHERE type = 'user' AND status = 'open'`).Scan(&stats.UserReports); err != nil {
		return stats, err
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM reports WHERE type = 'message' AND status = 'open'`).Scan(&stats.MessageReports); err != nil {
		return stats, err
	}
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM reports
		WHERE status = 'resolved' AND resolved_at >= CURRENT_DATE`).Scan(&stats.ResolvedToday); err != nil {
		return stats, err
	}
	return stats, nil
}

// ResolveReport marks a report resolved and records the acting moderator.
func (s *ModerationService) ResolveReport(ctx context.Context, resolverID, reportID string) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE reports SET status = 'resolved', resolver_id = $2, resolved_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND status = 'open'`, reportID, resolverID)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return ErrReportNotFound
	}
	return nil
}

// DismissReport marks a report dismissed with a moderator note.
func (s *ModerationService) DismissReport(ctx context.Context, resolverID, reportID, note string) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE reports SET status = 'dismissed', resolver_id = $2, resolution_note = $3, resolved_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND status = 'open'`, reportID, resolverID, strings.TrimSpace(note))
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return ErrReportNotFound
	}
	return nil
}
