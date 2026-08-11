package services

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/chorus/messenger/internal/models"
)

const (
	notificationMaxAttempts = 6
	notificationBaseDelay   = 1 * time.Minute
	notificationMaxDelay    = 1 * time.Hour
)

// NotificationService is a durable, retrying wrapper around EmailSender. Every
// notification is persisted to the email_outbox table before delivery; failed
// sends are retried by a background worker with exponential backoff until they
// succeed or exhaust the attempt budget. It implements EmailSender so all
// existing handlers keep working unchanged.
type NotificationService struct {
	db          *sql.DB
	sender      EmailSender
	maxAttempts int
	baseDelay   time.Duration
	maxDelay    time.Duration

	stopCh chan struct{}
	wg     sync.WaitGroup
}

func NewNotificationService(db *sql.DB, sender EmailSender) *NotificationService {
	return &NotificationService{
		db:          db,
		sender:      sender,
		maxAttempts: notificationMaxAttempts,
		baseDelay:   notificationBaseDelay,
		maxDelay:    notificationMaxDelay,
		stopCh:      make(chan struct{}),
	}
}

// Start launches the background retry worker.
func (n *NotificationService) Start() {
	n.wg.Add(1)
	go n.retryLoop()
}

// Stop halts the background worker and waits for it to finish.
func (n *NotificationService) Stop() {
	close(n.stopCh)
	n.wg.Wait()
}

func (n *NotificationService) retryLoop() {
	defer n.wg.Done()
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			n.retryDue()
		case <-n.stopCh:
			return
		}
	}
}

func (n *NotificationService) retryDue() {
	rows, err := n.db.Query(`
		SELECT id, recipient, subject, body FROM email_outbox
		WHERE status = 'pending' AND next_attempt_at <= CURRENT_TIMESTAMP
		ORDER BY created_at LIMIT 50`)
	if err != nil {
		log.Printf("[email] failed to load due emails: %v", err)
		return
	}
	defer rows.Close()
	type item struct{ id, recipient, subject, body string }
	var items []item
	for rows.Next() {
		var it item
		if err := rows.Scan(&it.id, &it.recipient, &it.subject, &it.body); err != nil {
			log.Printf("[email] scan failed: %v", err)
			continue
		}
		items = append(items, it)
	}
	for _, it := range items {
		if err := n.sender.Send(it.recipient, it.subject, it.body); err != nil {
			n.markFailed(it.id, err)
		} else {
			n.markSent(it.id)
		}
	}
}

// Send implements EmailSender. It persists the email, attempts delivery once,
// and returns the send error (if any) so callers keep their existing behavior;
// failed emails remain pending and are retried by the worker.
func (n *NotificationService) Send(to, subject, html string) error {
	to = strings.ToLower(strings.TrimSpace(to))
	var id string
	err := n.db.QueryRow(`INSERT INTO email_outbox (recipient, subject, body)
		VALUES ($1, $2, $3) RETURNING id`, to, subject, html).Scan(&id)
	if err != nil {
		return fmt.Errorf("failed to persist email: %w", err)
	}
	if n.sender == nil {
		err := fmt.Errorf("email sender not configured")
		n.markFailed(id, err)
		return err
	}
	if err := n.sender.Send(to, subject, html); err != nil {
		n.markFailed(id, err)
		return err
	}
	n.markSent(id)
	return nil
}

func (n *NotificationService) markSent(id string) {
	if _, err := n.db.Exec(`UPDATE email_outbox
		SET status = 'sent', sent_at = CURRENT_TIMESTAMP, last_error = NULL WHERE id = $1`, id); err != nil {
		log.Printf("[email] failed to mark %s sent: %v", id, err)
	}
}

func (n *NotificationService) markFailed(id string, sendErr error) {
	var attempts int
	if err := n.db.QueryRow(`SELECT attempts FROM email_outbox WHERE id = $1`, id).Scan(&attempts); err != nil {
		log.Printf("[email] failed to read attempts for %s: %v", id, err)
		return
	}
	attempts++
	status := "pending"
	if attempts >= n.maxAttempts {
		status = "failed"
	}
	delay := n.baseDelay
	for i := 1; i < attempts; i++ {
		if delay >= n.maxDelay {
			delay = n.maxDelay
			break
		}
		delay *= 2
	}
	if _, err := n.db.Exec(`UPDATE email_outbox
		SET attempts = $1, status = $2, last_error = $3, next_attempt_at = CURRENT_TIMESTAMP + ($4 * INTERVAL '1 second')
		WHERE id = $5`, attempts, status, sendErr.Error(), int(delay.Seconds()), id); err != nil {
		log.Printf("[email] failed to update %s: %v", id, err)
	}
}

// List returns recent outbox entries, optionally filtered by status
// ("" or "all" returns every entry).
func (n *NotificationService) List(status string) ([]models.EmailOutboxEntry, error) {
	query := `SELECT id, recipient, subject, status, attempts, COALESCE(last_error, ''), created_at, next_attempt_at, sent_at
		FROM email_outbox`
	var args []interface{}
	if status != "" && status != "all" {
		query += ` WHERE status = $1`
		args = append(args, status)
	}
	query += ` ORDER BY created_at DESC LIMIT 100`
	rows, err := n.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := []models.EmailOutboxEntry{}
	for rows.Next() {
		var e models.EmailOutboxEntry
		var lastError string
		if err := rows.Scan(&e.ID, &e.Recipient, &e.Subject, &e.Status, &e.Attempts, &lastError,
			&e.CreatedAt, &e.NextAttemptAt, &e.SentAt); err != nil {
			return nil, err
		}
		e.LastError = lastError
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// RetryByID immediately re-attempts a pending or failed email. It refuses to
// re-send an email that already succeeded.
func (n *NotificationService) RetryByID(id string) error {
	var recipient, subject, body, status string
	err := n.db.QueryRow(`SELECT recipient, subject, body, status FROM email_outbox WHERE id = $1`, id).
		Scan(&recipient, &subject, &body, &status)
	if err != nil {
		return err
	}
	if status == "sent" {
		return fmt.Errorf("email already sent")
	}
	if n.sender == nil {
		err := fmt.Errorf("email sender not configured")
		n.markFailed(id, err)
		return err
	}
	if err := n.sender.Send(recipient, subject, body); err != nil {
		n.markFailed(id, err)
		return err
	}
	n.markSent(id)
	return nil
}
