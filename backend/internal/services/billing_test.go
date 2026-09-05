package services

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/chorus/messenger/internal/models"
	"github.com/lib/pq"
)

func billingTestService(t *testing.T, db *sql.DB) *BillingService {
	t.Helper()
	return NewBillingService(db, nil, NewEntitlementService(false), "plan-monthly", "plan-yearly")
}

func userRowWithPlan(plan string) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "username", "email", "display_name", "native_language", "target_language_level", "target_languages",
		"role", "created_at", "last_active_at", "suspended_at", "deleted_at", "plan",
		"plan_grace_until", "premium_since", "subscription_id", "subscription_provider",
		"subscription_plan_id", "subscription_status", "next_billing_date", "last_payment_at",
	}).AddRow("user-1", "testuser", "test@example.com", "Test User", "en", "A1", pq.Array([]string{"es"}),
		"member", time.Now(), time.Now(), nil, nil, plan, nil, nil, nil, "paypal", nil, nil, nil, nil)
}

func TestBillingService_GrantPremium_Indefinite(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := billingTestService(t, db)
	s.now = func() time.Time { return time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC) }

	mock.ExpectQuery(`SELECT id, username, email, display_name, native_language, target_language_level, target_languages, role, created_at, last_active_at, suspended_at, deleted_at, plan, plan_grace_until, premium_since, subscription_id, subscription_provider, subscription_plan_id, subscription_status, next_billing_date, last_payment_at FROM users WHERE id = \$1`).
		WithArgs("user-1").WillReturnRows(userRowWithPlan("free"))
	mock.ExpectExec(`UPDATE users SET plan = \$1, plan_grace_until = \$2, premium_since = \$3 WHERE id = \$4`).
		WithArgs("premium", sqlmock.AnyArg(), sqlmock.AnyArg(), "user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO plan_changes \(user_id, actor_id, from_plan, to_plan, grace_until, source, reason\) VALUES \(\$1, \$2, \$3, \$4, \$5, \$6, \$7\)`).
		WithArgs("user-1", "admin-1", "free", "premium", sqlmock.AnyArg(), "admin", "manual grant").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT id, username, email, display_name, native_language, target_language_level, target_languages, role, created_at, last_active_at, suspended_at, deleted_at, plan, plan_grace_until, premium_since, subscription_id, subscription_provider, subscription_plan_id, subscription_status, next_billing_date, last_payment_at FROM users WHERE id = \$1`).
		WithArgs("user-1").WillReturnRows(userRowWithPlan("premium"))

	actor := "admin-1"
	user, err := s.GrantPremium(context.Background(), &actor, "user-1", models.GrantPlanRequest{
		Plan: PlanPremium, Mode: "indefinite", Reason: "manual grant",
	})
	if err != nil {
		t.Fatalf("GrantPremium failed: %v", err)
	}
	if user.Plan != PlanPremium {
		t.Fatalf("expected premium, got %s", user.Plan)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestBillingService_GrantPremium_Temporary(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := billingTestService(t, db)
	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }

	mock.ExpectQuery(`SELECT .* FROM users WHERE id = \$1`).WithArgs("user-1").WillReturnRows(userRowWithPlan("free"))
	mock.ExpectExec(`UPDATE users SET plan = \$1, plan_grace_until = \$2, premium_since = \$3 WHERE id = \$4`).
		WithArgs("free", sqlmock.AnyArg(), sqlmock.AnyArg(), "user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO plan_changes .*`).
		WithArgs("user-1", "admin-1", "free", "free", sqlmock.AnyArg(), "admin", "30-day trial").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT .* FROM users WHERE id = \$1`).WithArgs("user-1").WillReturnRows(userRowWithPlan("free"))

	actor := "admin-1"
	user, err := s.GrantPremium(context.Background(), &actor, "user-1", models.GrantPlanRequest{
		Plan: PlanPremium, Mode: "days", Days: 30, Reason: "30-day trial",
	})
	if err != nil {
		t.Fatalf("GrantPremium failed: %v", err)
	}
	if user.Plan != PlanFree {
		t.Fatalf("temporary grants keep stored plan free, got %s", user.Plan)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestBillingService_RevokePremium(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := billingTestService(t, db)

	mock.ExpectQuery(`SELECT .* FROM users WHERE id = \$1`).WithArgs("user-1").WillReturnRows(userRowWithPlan("premium"))
	mock.ExpectExec(`UPDATE users SET plan = 'free', plan_grace_until = \$1, next_billing_date = NULL, subscription_status = NULL WHERE id = \$2`).
		WithArgs(sqlmock.AnyArg(), "user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO plan_changes .*`).
		WithArgs("user-1", "admin-1", "premium", "free", sqlmock.AnyArg(), "admin", "violation").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT .* FROM users WHERE id = \$1`).WithArgs("user-1").WillReturnRows(userRowWithPlan("free"))

	actor := "admin-1"
	user, err := s.RevokePremium(context.Background(), &actor, "user-1", models.GrantPlanRequest{Reason: "violation"})
	if err != nil {
		t.Fatalf("RevokePremium failed: %v", err)
	}
	if user.Plan != PlanFree {
		t.Fatalf("expected free, got %s", user.Plan)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestBillingService_GetSubscription(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := billingTestService(t, db)

	// Free user: word limit + manual grammar surface, no manage link.
	mock.ExpectQuery(`SELECT .* FROM users WHERE id = \$1`).WithArgs("user-1").WillReturnRows(userRowWithPlan("free"))
	info, err := s.GetSubscription(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("GetSubscription failed: %v", err)
	}
	if info.EffectivePlan != EffectiveFree {
		t.Fatalf("expected free, got %s", info.EffectivePlan)
	}
	if info.InGrace {
		t.Fatal("expected not in grace")
	}
	if info.WordLimit == nil || *info.WordLimit != TranslationWordLimitFree {
		t.Fatalf("expected free word limit %d, got %v", TranslationWordLimitFree, info.WordLimit)
	}
	if info.AutoGrammar {
		t.Fatal("expected manual grammar on free")
	}
	if info.ManageURL != "" {
		t.Fatalf("expected no manage url, got %q", info.ManageURL)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Pure webhook mapping tests (no DB)
// ---------------------------------------------------------------------------

func eventFor(resourceType, etype, resource string) providerEvent {
	return providerEvent{
		ID:           "event-1",
		EventType:    etype,
		ResourceType: resourceType,
		Resource:     []byte(resource),
	}
}

func TestMapEvent_Grant(t *testing.T) {
	s := billingTestService(t, nil)
	res := `{"id":"I-123","custom_id":"user-1","plan_id":"plan-monthly","status":"ACTIVE","billing_info":{"next_billing_time":"2026-09-12T00:00:00Z"}}`
	action, planKey, sub, _, err := s.mapEvent(eventFor("subscription", "BILLING.SUBSCRIPTION.ACTIVATED", res))
	if err != nil {
		t.Fatal(err)
	}
	if action != "grant" || planKey != PlanKeyMonthly || sub == nil || sub.ID != "I-123" {
		t.Fatalf("bad mapping: action=%q planKey=%q sub=%+v", action, planKey, sub)
	}
}

func TestMapEvent_CancelGrace(t *testing.T) {
	s := billingTestService(t, nil)
	res := `{"id":"I-123","custom_id":"user-1","plan_id":"plan-yearly","billing_info":{"next_billing_time":"2026-12-01T00:00:00Z"}}`
	action, planKey, sub, _, err := s.mapEvent(eventFor("subscription", "BILLING.SUBSCRIPTION.CANCELLED", res))
	if err != nil {
		t.Fatal(err)
	}
	if action != "revoke" || planKey != PlanKeyAnnual {
		t.Fatalf("bad mapping: action=%q planKey=%q", action, planKey)
	}
	if grace := graceFromSub(sub); grace == nil || !grace.Equal(time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("expected grace until 2026-12-01, got %v", grace)
	}
}

func TestMapEvent_RefundImmediate(t *testing.T) {
	s := billingTestService(t, nil)
	res := `{"id":"sale-1","billing_agreement_id":"I-123","custom":"user-1","create_time":"2026-08-12T12:00:00Z"}`
	action, _, _, sale, err := s.mapEvent(eventFor("sale", "PAYMENT.SALE.REVERSED", res))
	if err != nil {
		t.Fatal(err)
	}
	if action != "revoke" || sale == nil || sale.BillingAgreementID != "I-123" {
		t.Fatalf("bad mapping: action=%q sale=%+v", action, sale)
	}
	// Refunds are immediate: no sub → grace nil.
	if grace := graceFromSub(nil); grace != nil {
		t.Fatalf("expected nil grace, got %v", grace)
	}
}

func TestMapEvent_Renew(t *testing.T) {
	s := billingTestService(t, nil)
	res := `{"id":"sale-2","subscription_id":"I-123","custom_id":"user-1","create_time":"2026-08-12T12:00:00Z"}`
	action, _, _, sale, err := s.mapEvent(eventFor("sale", "PAYMENT.SALE.COMPLETED", res))
	if err != nil {
		t.Fatal(err)
	}
	if action != "renew" || sale == nil || sale.SubscriptionID != "I-123" {
		t.Fatalf("bad mapping: action=%q sale=%+v", action, sale)
	}
}

func TestMapEvent_IgnoreUnknown(t *testing.T) {
	s := billingTestService(t, nil)
	action, _, _, _, err := s.mapEvent(eventFor("metadata", "CUSTOM.METADATA.UPDATED", `{"id":"x"}`))
	if err != nil {
		t.Fatal(err)
	}
	if action != "ignore" {
		t.Fatalf("expected ignore, got %q", action)
	}
}

func TestParsePayPalTime(t *testing.T) {
	cases := []string{
		"2026-09-12T00:00:00Z",
		"2026-09-12T00:00:00.123Z",
		"2026-09-12T12:34:56+02:00",
	}
	for _, c := range cases {
		if _, err := parsePayPalTime(c); err != nil {
			t.Errorf("parsePayPalTime(%q) failed: %v", c, err)
		}
	}
	if _, err := parsePayPalTime("not-a-time"); err == nil {
		t.Error("expected error for invalid time")
	}
}

// ---------------------------------------------------------------------------
// Premium lifecycle notifications (P15)
// ---------------------------------------------------------------------------

func TestBillingService_Notifications_AdminGrant(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := billingTestService(t, db)
	got := map[string]bool{}
	s.SetNotifier(func(_ context.Context, u *models.User, kind string, graceUntil *time.Time) {
		if u.Email != "test@example.com" {
			t.Errorf("expected notifier for test@example.com, got %s", u.Email)
		}
		if kind == NotifyEnterGrace && graceUntil == nil {
			t.Error("grace notification must carry graceUntil")
		}
		got[kind] = true
	})

	mock.ExpectQuery(`SELECT .* FROM users WHERE id = \$1`).WithArgs("user-1").WillReturnRows(userRowWithPlan("free"))
	mock.ExpectExec(`UPDATE users SET plan = \$1, plan_grace_until = \$2, premium_since = \$3 WHERE id = \$4`).
		WithArgs("premium", sqlmock.AnyArg(), sqlmock.AnyArg(), "user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO plan_changes .*`).
		WithArgs("user-1", "admin-1", "free", "premium", sqlmock.AnyArg(), "admin", "manual grant").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT .* FROM users WHERE id = \$1`).WithArgs("user-1").WillReturnRows(userRowWithPlan("premium"))

	actor := "admin-1"
	if _, err := s.GrantPremium(context.Background(), &actor, "user-1", models.GrantPlanRequest{Plan: PlanPremium, Mode: "indefinite", Reason: "manual grant"}); err != nil {
		t.Fatalf("GrantPremium failed: %v", err)
	}
	if !got[NotifyActivated] {
		t.Fatal("expected activated notification")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestBillingService_Notifications_AdminTimedGrant(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := billingTestService(t, db)
	s.now = func() time.Time { return time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC) }
	var gotKind string
	s.SetNotifier(func(_ context.Context, _ *models.User, kind string, _ *time.Time) {
		gotKind = kind
	})

	mock.ExpectQuery(`SELECT .* FROM users WHERE id = \$1`).WithArgs("user-1").WillReturnRows(userRowWithPlan("free"))
	mock.ExpectExec(`UPDATE users SET plan = \$1, plan_grace_until = \$2, premium_since = \$3 WHERE id = \$4`).
		WithArgs("free", sqlmock.AnyArg(), sqlmock.AnyArg(), "user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO plan_changes .*`).
		WithArgs("user-1", "admin-1", "free", "free", sqlmock.AnyArg(), "admin", "30-day trial").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT .* FROM users WHERE id = \$1`).WithArgs("user-1").WillReturnRows(userRowWithPlan("free"))

	actor := "admin-1"
	if _, err := s.GrantPremium(context.Background(), &actor, "user-1", models.GrantPlanRequest{Plan: PlanPremium, Mode: "days", Days: 30, Reason: "30-day trial"}); err != nil {
		t.Fatalf("GrantPremium failed: %v", err)
	}
	if gotKind != NotifyEnterGrace {
		t.Fatalf("expected grace notification, got %q", gotKind)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestBillingService_Notifications_SilentExtension(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := billingTestService(t, db)
	var notified bool
	s.SetNotifier(func(_ context.Context, _ *models.User, _ string, _ *time.Time) {
		notified = true
	})

	mock.ExpectQuery(`SELECT .* FROM users WHERE id = \$1`).WithArgs("user-1").WillReturnRows(userRowWithPlan("premium"))
	mock.ExpectExec(`UPDATE users SET plan = \$1, plan_grace_until = \$2, premium_since = \$3 WHERE id = \$4`).
		WithArgs("premium", sqlmock.AnyArg(), sqlmock.AnyArg(), "user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO plan_changes .*`).
		WithArgs("user-1", "admin-1", "premium", "premium", sqlmock.AnyArg(), "admin", "extension").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT .* FROM users WHERE id = \$1`).WithArgs("user-1").WillReturnRows(userRowWithPlan("premium"))

	actor := "admin-1"
	if _, err := s.GrantPremium(context.Background(), &actor, "user-1", models.GrantPlanRequest{Plan: PlanPremium, Mode: "indefinite", Reason: "extension"}); err != nil {
		t.Fatalf("GrantPremium failed: %v", err)
	}
	if notified {
		t.Fatal("extending an existing premium must not send a notification")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestBillingService_Notifications_RevokeGrace(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := billingTestService(t, db)
	s.now = func() time.Time { return time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC) }
	var gotKind string
	var gotGrace *time.Time
	s.SetNotifier(func(_ context.Context, _ *models.User, kind string, graceUntil *time.Time) {
		gotKind = kind
		gotGrace = graceUntil
	})

	mock.ExpectQuery(`SELECT .* FROM users WHERE id = \$1`).WithArgs("user-1").WillReturnRows(userRowWithPlan("premium"))
	mock.ExpectExec(`UPDATE users SET plan = 'free', plan_grace_until = \$1, next_billing_date = NULL, subscription_status = NULL WHERE id = \$2`).
		WithArgs(sqlmock.AnyArg(), "user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO plan_changes .*`).
		WithArgs("user-1", "admin-1", "premium", "free", sqlmock.AnyArg(), "admin", "payment dispute").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT .* FROM users WHERE id = \$1`).WithArgs("user-1").WillReturnRows(userRowWithPlan("free"))

	actor := "admin-1"
	if _, err := s.RevokePremium(context.Background(), &actor, "user-1", models.GrantPlanRequest{ClearGraceInDays: 7, Reason: "payment dispute"}); err != nil {
		t.Fatalf("RevokePremium failed: %v", err)
	}
	if gotKind != NotifyEnterGrace || gotGrace == nil {
		t.Fatalf("expected grace notification with deadline, got kind=%q grace=%v", gotKind, gotGrace)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestBillingService_Notifications_RevokeImmediate(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := billingTestService(t, db)
	var gotKind string
	s.SetNotifier(func(_ context.Context, _ *models.User, kind string, _ *time.Time) {
		gotKind = kind
	})

	mock.ExpectQuery(`SELECT .* FROM users WHERE id = \$1`).WithArgs("user-1").WillReturnRows(userRowWithPlan("premium"))
	mock.ExpectExec(`UPDATE users SET plan = 'free', plan_grace_until = \$1, next_billing_date = NULL, subscription_status = NULL WHERE id = \$2`).
		WithArgs(sqlmock.AnyArg(), "user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO plan_changes .*`).
		WithArgs("user-1", "admin-1", "premium", "free", sqlmock.AnyArg(), "admin", "refund").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT .* FROM users WHERE id = \$1`).WithArgs("user-1").WillReturnRows(userRowWithPlan("free"))

	actor := "admin-1"
	if _, err := s.RevokePremium(context.Background(), &actor, "user-1", models.GrantPlanRequest{Reason: "refund"}); err != nil {
		t.Fatalf("RevokePremium failed: %v", err)
	}
	if gotKind != NotifyDowngrade {
		t.Fatalf("expected downgrade notification, got %q", gotKind)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestBillingService_SweepExpiredGrace(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := billingTestService(t, db)
	var gotKind string
	s.SetNotifier(func(_ context.Context, u *models.User, kind string, _ *time.Time) {
		gotKind = kind
		if u.Email != "test@example.com" {
			t.Errorf("expected test@example.com, got %s", u.Email)
		}
	})

	expired := sqlmock.NewRows([]string{"id", "username", "display_name", "email"}).
		AddRow("user-1", "testuser", "Test User", "test@example.com")
	mock.ExpectQuery(`SELECT id, username, display_name, email FROM users WHERE plan = 'free' AND plan_grace_until IS NOT NULL AND plan_grace_until <= CURRENT_TIMESTAMP AND grace_notified_at IS NULL AND deleted_at IS NULL LIMIT 100`).
		WillReturnRows(expired)
	mock.ExpectExec(`UPDATE users SET grace_notified_at = CURRENT_TIMESTAMP WHERE id = \$1 AND grace_notified_at IS NULL`).
		WithArgs("user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	s.sweepExpiredGrace(context.Background())
	if gotKind != NotifyDowngrade {
		t.Fatalf("expected downgrade notification, got %q", gotKind)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestBillingService_SweepExpiredGrace_NoNotifier(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := billingTestService(t, db)

	// No notifier configured: the sweeper must not touch the database.
	s.sweepExpiredGrace(context.Background())
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected db access without notifier: %v", err)
	}
}
