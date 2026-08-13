package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/chorus/messenger/internal/models"
)

// Pricing used for MRR projections on the admin analytics endpoint.
const (
	MonthlyPrice = 7.99
	YearlyPrice  = 79.90
)

// Plan keys used by the checkout endpoint.
const (
	PlanKeyMonthly = "monthly"
	PlanKeyAnnual  = "annual"
)

var (
	ErrInvalidGrace   = errors.New("invalid grace window")
	ErrUserNotPremium = errors.New("user has no active premium to revoke")
)

// Premium notification kinds passed to a BillingNotifier.
const (
	NotifyActivated  = "activated"
	NotifyEnterGrace = "grace"
	NotifyDowngrade  = "downgraded"
)

// BillingNotifier receives premium lifecycle events so the app can email users
// (P15). kind is one of NotifyActivated/NotifyEnterGrace/NotifyDowngrade;
// graceUntil is set for grace notifications.
type BillingNotifier func(ctx context.Context, user *models.User, kind string, graceUntil *time.Time)

const graceSweepInterval = 5 * time.Minute

// BillingService owns subscription state: grants, revokes, PayPal webhook
// processing, analytics, and the user-facing subscription view. Plan changes
// are always audited into plan_changes.
type BillingService struct {
	db            *sql.DB
	paypal        *PayPalClient
	entitlements  *EntitlementService
	planMonthlyID string
	planYearlyID  string
	notifier      BillingNotifier
	now           func() time.Time
	manageURLFunc func(subscriptionID string) string

	stopCh chan struct{}
	wg     sync.WaitGroup
}

func NewBillingService(db *sql.DB, paypal *PayPalClient, entitlements *EntitlementService, planMonthlyID, planYearlyID string) *BillingService {
	return &BillingService{
		db:            db,
		paypal:        paypal,
		entitlements:  entitlements,
		planMonthlyID: planMonthlyID,
		planYearlyID:  planYearlyID,
		now:           time.Now,
		stopCh:        make(chan struct{}),
	}
}

// SetNotifier registers the premium lifecycle email hook (P15). Leave unset
// for silent billing behavior (tests, dev).
func (s *BillingService) SetNotifier(fn BillingNotifier) {
	s.notifier = fn
}

// notify fires the lifecycle hook when configured. Errors are logged by the
// caller's hook; a nil user or empty email is skipped defensively.
func (s *BillingService) notify(ctx context.Context, user *models.User, kind string, graceUntil *time.Time) {
	if s.notifier == nil || user == nil || strings.TrimSpace(user.Email) == "" {
		return
	}
	s.notifier(ctx, user, kind, graceUntil)
}

// StartGraceSweeper launches the background job that emails users whose paid
// grace window has just expired ("premium ended"). It is idempotent via the
// grace_notified_at marker.
func (s *BillingService) StartGraceSweeper() {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(graceSweepInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.sweepExpiredGrace(context.Background())
			case <-s.stopCh:
				return
			}
		}
	}()
}

// StopGraceSweeper halts the background job and waits for it to finish.
func (s *BillingService) StopGraceSweeper() {
	close(s.stopCh)
	s.wg.Wait()
}

// sweepExpiredGrace notifies every user whose grace expired and who has not
// been told yet, then marks them so a downgrade email is sent exactly once.
func (s *BillingService) sweepExpiredGrace(ctx context.Context) {
	if s.notifier == nil {
		return
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, username, display_name, email
		FROM users
		WHERE plan = 'free'
		  AND plan_grace_until IS NOT NULL
		  AND plan_grace_until <= CURRENT_TIMESTAMP
		  AND grace_notified_at IS NULL
		  AND deleted_at IS NULL
		LIMIT 100`)
	if err != nil {
		log.Printf("[billing] grace sweep query failed: %v", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Username, &u.DisplayName, &u.Email); err != nil {
			log.Printf("[billing] grace sweep scan failed: %v", err)
			continue
		}
		// Claim the row first so concurrent instances do not double-send, then
		// fire the hook.
		if _, err := s.db.ExecContext(ctx, `
			UPDATE users SET grace_notified_at = CURRENT_TIMESTAMP WHERE id = $1 AND grace_notified_at IS NULL`, u.ID); err != nil {
			log.Printf("[billing] grace sweep claim failed for %s: %v", u.ID, err)
			continue
		}
		s.notify(ctx, &u, NotifyDowngrade, nil)
	}
}

// SetManageURLFunc overrides how manage links are built (e.g. in tests).
func (s *BillingService) SetManageURLFunc(f func(subscriptionID string) string) {
	s.manageURLFunc = f
}

func (s *BillingService) manageURL(subID string) string {
	if s.manageURLFunc != nil {
		return s.manageURLFunc(subID)
	}
	if s.paypal != nil {
		return s.paypal.ManageURL(subID)
	}
	return ""
}

func (s *BillingService) loadUser(ctx context.Context, userID string) (*models.User, error) {
	return scanUser(s.db.QueryRowContext(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1`, userID))
}

// ---------------------------------------------------------------------------
// User-facing subscription view
// ---------------------------------------------------------------------------

// GetSubscription returns the subscription view for the current user, with the
// effective plan resolved via the entitlement service (grace-aware).
func (s *BillingService) GetSubscription(ctx context.Context, userID string) (*models.SubscriptionInfo, error) {
	user, err := s.loadUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	e := s.entitlements.ResolveNow(user)
	info := &models.SubscriptionInfo{
		Plan:          e.Plan,
		EffectivePlan: e.EffectivePlan,
		InGrace:       e.EffectivePlan == EffectivePremium && e.Plan != PlanPremium,
		PremiumSince:  user.PremiumSince,
		Status:        "",
		Provider:      "",
		ManageURL:     "",
		AutoGrammar:   e.Features.AutoGrammar,
		WordLimit:     e.Features.TranslationWordLimit,
	}
	if user.SubscriptionID != nil {
		info.SubscriptionID = *user.SubscriptionID
	}
	if user.SubscriptionProvider != "" {
		info.Provider = user.SubscriptionProvider
	}
	if user.SubscriptionStatus != nil {
		info.Status = *user.SubscriptionStatus
	}
	info.NextBillingDate = user.NextBillingDate
	info.GraceUntil = user.PlanGraceUntil
	if user.SubscriptionID != nil {
		info.ManageURL = s.manageURL(*user.SubscriptionID)
	}
	return info, nil
}

// ---------------------------------------------------------------------------
// Admin grant / revoke
// ---------------------------------------------------------------------------

// GrantPremium applies an admin grant. Mode semantics:
//   - indefinite: stored plan becomes premium, grace cleared.
//   - days / until: the user keeps equivalent premium entitlements via a grace
//     window that expires at now+days / at <until>; the stored plan is free
//     afterwards, matching the existing cutover-grace machinery (no scheduler
//     needed — entitlement resolution is time-aware).
//
// The acting admin is recorded in plan_changes. Setting Plan=free delegates to
// RevokePremium.
func (s *BillingService) GrantPremium(ctx context.Context, actorID *string, targetUserID string, req models.GrantPlanRequest) (*models.User, error) {
	now := s.now()
	user, err := s.loadUser(ctx, targetUserID)
	if err != nil {
		return nil, err
	}
	fromPlan := user.Plan

	var toPlan string
	var graceUntil *time.Time
	switch req.Plan {
	case PlanPremium:
		switch req.Mode {
		case "", "indefinite":
			toPlan = PlanPremium
		case "days":
			if req.Days <= 0 {
				return nil, ErrInvalidGrace
			}
			g := now.Add(time.Duration(req.Days) * 24 * time.Hour)
			graceUntil = &g
			toPlan = PlanFree
		case "until":
			until, err := time.Parse(time.RFC3339, req.Until)
			if err != nil || !until.After(now) {
				return nil, ErrInvalidGrace
			}
			graceUntil = &until
			toPlan = PlanFree
		}
	case PlanFree:
		return s.RevokePremium(ctx, actorID, targetUserID, req)
	default:
		return nil, errors.New("unsupported plan")
	}

	// Extending an active grace keeps the later deadline.
	if graceUntil != nil && hasActiveGrace(user, now) && user.PlanGraceUntil != nil && user.PlanGraceUntil.After(*graceUntil) {
		graceUntil = user.PlanGraceUntil
	}

	premiumSince := user.PremiumSince
	if premiumSince == nil {
		p := now
		premiumSince = &p
	}

	if _, err := s.db.ExecContext(ctx, `
		UPDATE users
		SET plan = $1, plan_grace_until = $2, premium_since = $3
		WHERE id = $4`, toPlan, graceUntil, premiumSince, targetUserID); err != nil {
		return nil, err
	}
	if err := s.recordPlanChange(ctx, targetUserID, actorID, fromPlan, toPlan, graceUntil, "admin", req.Reason); err != nil {
		return nil, err
	}
	// Lifecycle emails (P15): fire only on a real transition so extending an
	// existing premium/grace account stays silent.
	wasPremium := s.entitlements.ResolveNow(user).EffectivePlan == EffectivePremium
	if !wasPremium {
		if toPlan == PlanPremium {
			s.notify(ctx, user, NotifyActivated, nil)
		} else if graceUntil != nil {
			s.notify(ctx, user, NotifyEnterGrace, graceUntil)
		}
	}
	return s.loadUser(ctx, targetUserID)
}

// RevokePremium drops a user to the free plan. With ClearGraceInDays > 0 the
// user keeps premium entitlements for that many days (a manual grace window).
func (s *BillingService) RevokePremium(ctx context.Context, actorID *string, targetUserID string, req models.GrantPlanRequest) (*models.User, error) {
	now := s.now()
	user, err := s.loadUser(ctx, targetUserID)
	if err != nil {
		return nil, err
	}
	e := s.entitlements.ResolveNow(user)
	if e.EffectivePlan != EffectivePremium {
		return nil, ErrUserNotPremium
	}
	fromPlan := user.Plan

	var graceUntil *time.Time
	if req.ClearGraceInDays > 0 {
		g := now.Add(time.Duration(req.ClearGraceInDays) * 24 * time.Hour)
		graceUntil = &g
	}
	if _, err := s.db.ExecContext(ctx, `
		UPDATE users
		SET plan = 'free', plan_grace_until = $1, next_billing_date = NULL, subscription_status = NULL
		WHERE id = $2`, graceUntil, targetUserID); err != nil {
		return nil, err
	}
	if err := s.recordPlanChange(ctx, targetUserID, actorID, fromPlan, PlanFree, graceUntil, "admin", req.Reason); err != nil {
		return nil, err
	}
	if graceUntil != nil {
		s.notify(ctx, user, NotifyEnterGrace, graceUntil)
	} else {
		s.notify(ctx, user, NotifyDowngrade, nil)
	}
	return s.loadUser(ctx, targetUserID)
}

// Checkout creates a PayPal subscription for the user and returns the approval
// URL the client should redirect the browser to. The user id is stored as
// custom_id so the ACTIVATED webhook can attribute it.
func (s *BillingService) Checkout(ctx context.Context, userID, planKey, returnURL, cancelURL string) (*models.CheckoutResponse, error) {
	if s.paypal == nil || !s.paypal.Enabled() {
		return nil, ErrPayPalUnconfigured
	}
	subID, approvalURL, err := s.paypal.CreateSubscription(ctx, planKey, userID, returnURL, cancelURL)
	if err != nil {
		return nil, err
	}
	// Stash the provider subscription id ahead of activation so sales events
	// and cancel/expiry webhooks can attribute the user even before ACTIVATED.
	if _, err := s.db.ExecContext(ctx, `
		UPDATE users
		SET subscription_id = $1, subscription_plan_id = $2, subscription_status = 'PENDING'
		WHERE id = $3 AND (subscription_id IS NULL OR subscription_id = $1)`, subID, planKey, userID); err != nil {
		return nil, err
	}
	return &models.CheckoutResponse{ApprovalURL: approvalURL, PlanKey: planKey}, nil
}

// PlanHistory returns the audit trail for a user.
func (s *BillingService) PlanHistory(ctx context.Context, userID string) ([]models.PlanChange, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, actor_id, from_plan, to_plan, grace_until, source, reason, created_at
		FROM plan_changes WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.PlanChange, 0)
	for rows.Next() {
		var pc models.PlanChange
		if err := rows.Scan(&pc.ID, &pc.UserID, &pc.ActorID, &pc.FromPlan, &pc.ToPlan, &pc.GraceUntil, &pc.Source, &pc.Reason, &pc.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, pc)
	}
	return out, rows.Err()
}

const premiumUserColumns = `u.id, u.username, u.display_name, u.email, u.plan, u.subscription_id, u.subscription_status,
	u.premium_since, u.next_billing_date, u.plan_grace_until, u.created_at`

// ListPremiumUsers returns premium (stored or in-grace) users for the admin
// console, with message volume. Filters: q (substring), limit, offset.
func (s *BillingService) ListPremiumUsers(ctx context.Context, q string, limit, offset int) ([]models.PremiumUserRow, int, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	where := `u.deleted_at IS NULL AND (u.plan = 'premium' OR (u.plan = 'free' AND u.plan_grace_until > CURRENT_TIMESTAMP))`
	args := []interface{}{}
	addArg := func(v interface{}) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}
	if q != "" {
		where += fmt.Sprintf(` AND (u.username ILIKE %s OR u.display_name ILIKE %s OR u.email ILIKE %s)`,
			addArg("%"+q+"%"), addArg("%"+q+"%"), addArg("%"+q+"%"))
	}

	var total int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users u WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `SELECT ` + premiumUserColumns + `,
		(SELECT COUNT(*) FROM messages m WHERE m.sender_id = u.id) AS messages_sent
		FROM users u WHERE ` + where +
		` ORDER BY COALESCE(u.premium_since, u.created_at) DESC LIMIT ` + addArg(limit) + ` OFFSET ` + addArg(offset)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := make([]models.PremiumUserRow, 0)
	for rows.Next() {
		row, err := s.scanPremiumUserRow(rows)
		if err != nil {
			return nil, 0, err
		}
		s.decoratePremiumUser(&row)
		out = append(out, row)
	}
	return out, total, rows.Err()
}

// PremiumAnalytics aggregates premium metrics for the admin dashboard.
func (s *BillingService) PremiumAnalytics(ctx context.Context) (*models.PremiumAnalytics, error) {
	stats := &models.PremiumAnalytics{}
	count := func(query string, arg ...interface{}) (int, error) {
		var n int
		err := s.db.QueryRowContext(ctx, query, arg...).Scan(&n)
		return n, err
	}
	var err error
	if stats.StoredPremium, err = count(`SELECT COUNT(*) FROM users WHERE deleted_at IS NULL AND plan = 'premium'`); err != nil {
		return nil, err
	}
	if stats.InGrace, err = count(`SELECT COUNT(*) FROM users WHERE deleted_at IS NULL AND plan = 'free' AND plan_grace_until > CURRENT_TIMESTAMP`); err != nil {
		return nil, err
	}
	stats.TotalPremiumUsers = stats.StoredPremium + stats.InGrace

	now := s.now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).UTC()
	if stats.NewThisMonth, err = count(`SELECT COUNT(*) FROM users WHERE deleted_at IS NULL AND premium_since >= $1`, monthStart); err != nil {
		return nil, err
	}
	if stats.ChurnedThisMonth, err = count(`SELECT COUNT(*) FROM plan_changes WHERE to_plan = 'free' AND created_at >= $1`, monthStart); err != nil {
		return nil, err
	}

	if s.planMonthlyID != "" {
		if stats.MonthlySubscriptions, err = count(`SELECT COUNT(*) FROM users WHERE deleted_at IS NULL AND plan = 'premium' AND subscription_plan_id = $1`, s.planMonthlyID); err != nil {
			return nil, err
		}
	}
	if s.planYearlyID != "" {
		if stats.YearlySubscriptions, err = count(`SELECT COUNT(*) FROM users WHERE deleted_at IS NULL AND plan = 'premium' AND subscription_plan_id = $1`, s.planYearlyID); err != nil {
			return nil, err
		}
	}
	// Premium users without a classified provider plan (admin grants, missing
	// plan-id config, legacy rows) are lumped into the monthly bucket so the
	// MRR projection stays conservative.
	unclassified := stats.StoredPremium - stats.MonthlySubscriptions - stats.YearlySubscriptions
	if unclassified < 0 {
		unclassified = 0
	}
	stats.MonthlySubscriptions += unclassified

	stats.ProjectedMRR = float64(stats.MonthlySubscriptions)*MonthlyPrice + float64(stats.YearlySubscriptions)*(YearlyPrice/12)
	stats.RevenueLastYear = stats.ProjectedMRR * 12

	if stats.TopUsersByUsage, err = s.TopUsersByUsage(ctx, 10); err != nil {
		return nil, err
	}
	return stats, nil
}

// TopUsersByUsage returns the most active premium (or in-grace) users.
func (s *BillingService) TopUsersByUsage(ctx context.Context, limit int) ([]models.PremiumUserRow, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+premiumUserColumns+`, COUNT(m.id) AS messages_sent
		FROM users u
		JOIN messages m ON m.sender_id = u.id
		WHERE u.deleted_at IS NULL
		  AND (u.plan = 'premium' OR (u.plan = 'free' AND u.plan_grace_until > CURRENT_TIMESTAMP))
		GROUP BY u.id
		ORDER BY messages_sent DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.PremiumUserRow, 0)
	for rows.Next() {
		row, err := s.scanPremiumUserRow(rows)
		if err != nil {
			return nil, err
		}
		s.decoratePremiumUser(&row)
		out = append(out, row)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].MessagesSent > out[j].MessagesSent })
	return out, rows.Err()
}

type premiumRowScanner interface {
	Scan(dest ...interface{}) error
}

func (s *BillingService) scanPremiumUserRow(sc premiumRowScanner) (models.PremiumUserRow, error) {
	var row models.PremiumUserRow
	err := sc.Scan(&row.ID, &row.Username, &row.DisplayName, &row.Email, &row.Plan,
		&row.SubscriptionID, &row.SubscriptionStatus, &row.PremiumSince, &row.NextBillingDate,
		&row.GraceUntil, &row.CreatedAt, &row.MessagesSent)
	return row, err
}

func (s *BillingService) decoratePremiumUser(row *models.PremiumUserRow) {
	e := s.entitlements.Resolve(&models.User{Plan: row.Plan, PlanGraceUntil: row.GraceUntil}, s.now())
	row.EffectivePlan = e.EffectivePlan
	row.InGrace = e.EffectivePlan == EffectivePremium && e.Plan != PlanPremium
}

func (s *BillingService) recordPlanChange(ctx context.Context, userID string, actorID *string, fromPlan, toPlan string, graceUntil *time.Time, source, reason string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO plan_changes (user_id, actor_id, from_plan, to_plan, grace_until, source, reason)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		userID, actorID, fromPlan, toPlan, graceUntil, source, reason)
	return err
}

// ---------------------------------------------------------------------------
// PayPal webhooks
// ---------------------------------------------------------------------------

// providerEvent is the envelope PayPal sends to a webhook endpoint.
type providerEvent struct {
	ID           string          `json:"id"`
	EventType    string          `json:"event_type"`
	Summary      string          `json:"summary"`
	ResourceType string          `json:"resource_type"`
	CreateTime   string          `json:"create_time"`
	Resource     json.RawMessage `json:"resource"`
}

type subscriptionResource struct {
	ID          string `json:"id"`
	CustomID    string `json:"custom_id"`
	PlanID      string `json:"plan_id"`
	Status      string `json:"status"`
	StartTime   string `json:"start_time"`
	CreateTime  string `json:"create_time"`
	BillingInfo struct {
		NextBillingTime string `json:"next_billing_time"`
		LastPayment     struct {
			Time   string `json:"time"`
			Amount struct {
				Value string `json:"value"`
			} `json:"amount"`
		} `json:"last_payment"`
	} `json:"billing_info"`
}

type saleResource struct {
	ID                 string `json:"id"`
	SubscriptionID     string `json:"subscription_id"`
	BillingAgreementID string `json:"billing_agreement_id"`
	Custom             string `json:"custom"`
	CustomID           string `json:"custom_id"`
	State              string `json:"state"`
	Status             string `json:"status"`
	CreateTime         string `json:"create_time"`
	UpdateTime         string `json:"update_time"`
	Payer              struct {
		EmailAddress string `json:"email_address"`
	} `json:"payer"`
	SupplementaryData struct {
		RelatedIds struct {
			SubscriptionID string `json:"subscription_id"`
		} `json:"related_ids"`
	} `json:"supplementary_data"`
}

// ApplyWebhook ingests one verified webhook event. It is idempotent: events
// are keyed by provider_event_id (provider + event id), so double delivery
// never double-applies.
func (s *BillingService) ApplyWebhook(ctx context.Context, body []byte) error {
	var ev providerEvent
	if err := json.Unmarshal(body, &ev); err != nil {
		return fmt.Errorf("malformed webhook payload: %w", err)
	}
	if ev.ID == "" || ev.EventType == "" {
		return errors.New("webhook event missing id/event_type")
	}

	// Idempotency gate.
	inserted, err := s.ingestEvent(ctx, &ev)
	if err != nil {
		return err
	}
	if !inserted {
		return nil // already processed
	}

	action, planKey, sub, sale, err := s.mapEvent(ev)
	if err != nil {
		s.failEvent(ctx, ev.ID, err.Error())
		return nil
	}
	if action == "ignore" {
		return s.markEvent(ctx, ev.ID, "ignored")
	}

	userID, err := s.resolveEventUser(ctx, sub, sale)
	if err != nil {
		// Payer email could not be mapped — keep the event for inspection.
		s.failEvent(ctx, ev.ID, err.Error())
		return nil
	}

	switch action {
	case "grant", "renew":
		if err := s.grantFromWebhook(ctx, userID, planKey, sub, sale); err != nil {
			s.failEvent(ctx, ev.ID, err.Error())
			return err
		}
	case "revoke":
		graceUntil := graceFromSub(sub)
		if err := s.revokeFromWebhook(ctx, userID, ev.EventType, sub, sale, graceUntil); err != nil {
			s.failEvent(ctx, ev.ID, err.Error())
			return err
		}
	}
	return s.markEvent(ctx, ev.ID, "processed")
}

// ingestEvent records the event and reports whether it is new (inserted).
func (s *BillingService) ingestEvent(ctx context.Context, ev *providerEvent) (bool, error) {
	payload := json.RawMessage("{}")
	if len(ev.Resource) > 0 {
		payload = ev.Resource
	}
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO subscription_events (provider, provider_event_id, event_type, payload, status)
		VALUES ('paypal', $1, $2, $3, 'received')
		ON CONFLICT (provider, provider_event_id) DO NOTHING`, ev.ID, ev.EventType, payload)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n == 1, nil
}

func (s *BillingService) markEvent(ctx context.Context, eventID, status string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE subscription_events SET status = $1, handled_at = CURRENT_TIMESTAMP, error = NULL WHERE provider_event_id = $2`, status, eventID)
	return err
}

func (s *BillingService) failEvent(ctx context.Context, eventID, message string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE subscription_events SET status = 'failed', error = $1 WHERE provider_event_id = $2`, message, eventID)
	return err
}

// mapEvent decides what a webhook event means. Actions: grant, renew, revoke,
// ignore. planKey is "monthly"/"annual" when extractable from the plan id.
func (s *BillingService) mapEvent(ev providerEvent) (action, planKey string, sub *subscriptionResource, sale *saleResource, err error) {
	etype := strings.ToUpper(ev.EventType)
	switch {
	case strings.HasPrefix(etype, "BILLING.SUBSCRIPTION.ACTIVATED"),
		strings.HasPrefix(etype, "BILLING.SUBSCRIPTION.APPROVED"),
		strings.HasPrefix(etype, "BILLING.SUBSCRIPTION.STARTED"):
		action = "grant"
	case strings.HasPrefix(etype, "BILLING.SUBSCRIPTION.CANCELLED"),
		strings.HasPrefix(etype, "BILLING.SUBSCRIPTION.EXPIRED"),
		strings.HasPrefix(etype, "BILLING.SUBSCRIPTION.SUSPENDED"):
		action = "revoke"
	case strings.HasPrefix(etype, "PAYMENT.SALE.COMPLETED"),
		strings.HasPrefix(etype, "PAYMENT.CAPTURE.COMPLETED"),
		strings.HasPrefix(etype, "PAYMENT.PAYMENTS.CREATED"),
		strings.HasPrefix(etype, "PAYMENT.SALE.PENDING"),
		strings.HasPrefix(etype, "BILLING.SUBSCRIPTION.PAYMENT."):
		action = "renew"
	case strings.HasPrefix(etype, "PAYMENT.SALE.REVERSED"),
		strings.HasPrefix(etype, "PAYMENT.SALE.REFUNDED"),
		strings.HasPrefix(etype, "PAYMENT.CAPTURE.REVERSED"),
		strings.HasPrefix(etype, "PAYMENT.CAPTURE.REFUNDED"):
		action = "revoke"
	default:
		action = "ignore"
	}

	if len(ev.Resource) == 0 {
		return
	}
	// Prefer parsing as a subscription resource when the type says so.
	if ev.ResourceType == "subscription" {
		var sr subscriptionResource
		if json.Unmarshal(ev.Resource, &sr) == nil && sr.ID != "" {
			sub = &sr
			planKey = s.planKeyForID(sr.PlanID)
			return
		}
	}
	var sl saleResource
	if json.Unmarshal(ev.Resource, &sl) == nil && sl.ID != "" {
		sale = &sl
	}
	return
}

func (s *BillingService) planKeyForID(planID string) string {
	switch planID {
	case s.planMonthlyID:
		return PlanKeyMonthly
	case s.planYearlyID:
		return PlanKeyAnnual
	}
	return ""
}

// resolveEventUser maps a webhook event to a chorus user id.
func (s *BillingService) resolveEventUser(ctx context.Context, sub *subscriptionResource, sale *saleResource) (string, error) {
	if sub != nil {
		if sub.CustomID != "" {
			if ok, err := s.userExists(ctx, sub.CustomID); err == nil && ok {
				return sub.CustomID, nil
			}
		}
		if sub.ID != "" {
			if id, ok := s.userIDBySubscription(ctx, sub.ID); ok {
				return id, nil
			}
		}
	}
	if sale != nil {
		for _, cand := range []string{sale.CustomID, sale.Custom, sale.SubscriptionID, sale.BillingAgreementID, sale.SupplementaryData.RelatedIds.SubscriptionID} {
			if cand == "" {
				continue
			}
			if id, ok := s.userIDBySubscription(ctx, cand); ok {
				return id, nil
			}
		}
		if sale.Payer.EmailAddress != "" {
			if id, ok := s.userIDByEmail(ctx, sale.Payer.EmailAddress); ok {
				return id, nil
			}
		}
	}
	return "", errors.New("webhook event could not be attributed to a user (custom_id, subscription_id, payer email all unmatched)")
}

func (s *BillingService) userExists(ctx context.Context, id string) (bool, error) {
	var ok bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id = $1 AND deleted_at IS NULL)`, id).Scan(&ok)
	return ok, err
}

func (s *BillingService) userIDBySubscription(ctx context.Context, subscriptionID string) (string, bool) {
	var id string
	err := s.db.QueryRowContext(ctx, `SELECT id FROM users WHERE subscription_id = $1 AND deleted_at IS NULL`, subscriptionID).Scan(&id)
	if err != nil {
		return "", false
	}
	return id, true
}

func (s *BillingService) userIDByEmail(ctx context.Context, email string) (string, bool) {
	var id string
	err := s.db.QueryRowContext(ctx, `SELECT id FROM users WHERE LOWER(email) = LOWER($1) AND deleted_at IS NULL`, email).Scan(&id)
	if err != nil {
		return "", false
	}
	return id, true
}

// grantFromWebhook activates premium for the user from a subscription event
// (activation/renewal or a completed payment).
func (s *BillingService) grantFromWebhook(ctx context.Context, userID, planKey string, sub *subscriptionResource, sale *saleResource) error {
	now := s.now()
	user, err := s.loadUser(ctx, userID)
	if err != nil {
		return err
	}
	fromPlan := user.Plan

	subID := ""
	planID := ""
	status := "ACTIVE"
	var nextBilling *time.Time
	var paidAt *time.Time

	if sub != nil {
		subID = sub.ID
		planID = sub.PlanID
		if sub.BillingInfo.NextBillingTime != "" {
			if t, err := parsePayPalTime(sub.BillingInfo.NextBillingTime); err == nil {
				nextBilling = &t
			}
		}
		if sub.BillingInfo.LastPayment.Time != "" {
			if t, err := parsePayPalTime(sub.BillingInfo.LastPayment.Time); err == nil {
				paidAt = &t
			}
		}
	} else if sale != nil {
		subID = sale.SubscriptionID
		if subID == "" {
			subID = sale.BillingAgreementID
		}
		if subID == "" {
			subID = sale.SupplementaryData.RelatedIds.SubscriptionID
		}
		if sale.CreateTime != "" {
			if t, err := parsePayPalTime(sale.CreateTime); err == nil {
				paidAt = &t
			}
		}
	}

	premiumSince := user.PremiumSince
	if premiumSince == nil {
		p := now
		premiumSince = &p
	}

	var storePlanID = user.SubscriptionPlanID
	if planID != "" {
		storePlanID = &planID
	}
	var storeSubID = user.SubscriptionID
	if subID != "" {
		storeSubID = &subID
	}

	if _, err := s.db.ExecContext(ctx, `
		UPDATE users
		SET plan = 'premium',
		    plan_grace_until = NULL,
		    premium_since = $2,
		    subscription_id = COALESCE(NULLIF($3, ''), subscription_id),
		    subscription_plan_id = $4,
		    subscription_status = $5,
		    next_billing_date = $6,
		    last_payment_at = $7
		WHERE id = $1`, userID, premiumSince, strFromPtr(storeSubID), storePlanID, status, nextBilling, paidAt); err != nil {
		return err
	}
	if err := s.recordPlanChange(ctx, userID, nil, fromPlan, PlanPremium, nil, "webhook", "paypal: "+status); err != nil {
		return err
	}
	// Renewals of an already-premium account stay silent; only brand-new
	// activations get the welcome email (P15).
	if fromPlan != PlanPremium {
		s.notify(ctx, user, NotifyActivated, nil)
	}
	return nil
}

// revokeFromWebhook downgrades a user from a cancel/expire/suspend/refund
// event. On cancel/expire the grace period runs until the end of the paid
// period (next_billing_time); refunds are immediate (nil grace).
func (s *BillingService) revokeFromWebhook(ctx context.Context, userID string, eventType string, sub *subscriptionResource, sale *saleResource, graceUntil *time.Time) error {
	user, err := s.loadUser(ctx, userID)
	if err != nil {
		return err
	}
	fromPlan := user.Plan

	status := "CANCELLED"
	switch {
	case strings.HasPrefix(strings.ToUpper(eventType), "BILLING.SUBSCRIPTION.SUSPENDED"):
		status = "SUSPENDED"
	case strings.HasPrefix(strings.ToUpper(eventType), "BILLING.SUBSCRIPTION.EXPIRED"):
		status = "EXPIRED"
	case strings.HasPrefix(strings.ToUpper(eventType), "PAYMENT."):
		status = "TERMINATED"
	}

	// Preserve the next billing date when grace runs until it (paid-through).
	var keepNextBilling *time.Time
	if graceUntil != nil && user.NextBillingDate != nil && user.NextBillingDate.Equal(*graceUntil) {
		keepNextBilling = user.NextBillingDate
	}

	if _, err := s.db.ExecContext(ctx, `
		UPDATE users
		SET plan = 'free',
		    plan_grace_until = $3,
		    subscription_status = $2,
		    next_billing_date = $4
		WHERE id = $1`, userID, status, graceUntil, keepNextBilling); err != nil {
		return err
	}
	reason := "paypal: " + strings.ToLower(status)
	if err := s.recordPlanChange(ctx, userID, nil, fromPlan, PlanFree, graceUntil, "webhook", reason); err != nil {
		return err
	}
	if graceUntil != nil {
		s.notify(ctx, user, NotifyEnterGrace, graceUntil)
	} else {
		s.notify(ctx, user, NotifyDowngrade, nil)
	}
	return nil
}

// graceFromSub computes the paid-through grace until the next billing time.
func graceFromSub(sub *subscriptionResource) *time.Time {
	if sub == nil || sub.BillingInfo.NextBillingTime == "" {
		return nil
	}
	t, err := parsePayPalTime(sub.BillingInfo.NextBillingTime)
	if err != nil {
		return nil
	}
	return &t
}

func parsePayPalTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, errors.New("empty time")
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02T15:04:05.999Z0700", s)
}

func strFromPtr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
