package services

import (
	"strings"
	"testing"
	"time"

	"github.com/chorus/messenger/internal/models"
)

func TestEntitlementServiceResolve_Free(t *testing.T) {
	s := NewEntitlementService(false)
	u := &models.User{Plan: PlanFree}

	e := s.ResolveNow(u)

	if e.EffectivePlan != EffectiveFree {
		t.Fatalf("expected free, got %s", e.EffectivePlan)
	}
	if !e.ShowAds {
		t.Fatal("expected ShowAds on the free plan")
	}
	if e.SelfHost {
		t.Fatal("expected SelfHost=false on hosted")
	}
	// No per-day quotas anywhere; every legacy daily limit must be unlimited.
	assertAllLimitsUnlimited(t, e.Limits)
	// Free = 280 word cap, manual (non-auto) grammar.
	if w := *e.Features.TranslationWordLimit; w != TranslationWordLimitFree {
		t.Fatalf("expected free word limit %d, got %d", TranslationWordLimitFree, w)
	}
	if e.Features.AutoGrammar {
		t.Fatal("expected no auto grammar on free")
	}
}

func TestEntitlementServiceResolve_Premium(t *testing.T) {
	s := NewEntitlementService(false)
	u := &models.User{Plan: PlanPremium}

	e := s.ResolveNow(u)

	if e.EffectivePlan != EffectivePremium {
		t.Fatalf("expected premium, got %s", e.EffectivePlan)
	}
	if e.ShowAds {
		t.Fatal("expected ShowAds=false on premium")
	}
	assertAllLimitsUnlimited(t, e.Limits)
	// Premium = 1000 word cap, auto grammar.
	if w := *e.Features.TranslationWordLimit; w != TranslationWordLimitPremium {
		t.Fatalf("expected premium word limit %d, got %d", TranslationWordLimitPremium, w)
	}
	if !e.Features.AutoGrammar {
		t.Fatal("expected auto grammar on premium")
	}
}

func TestEntitlementServiceResolve_GraceWindow(t *testing.T) {
	s := NewEntitlementService(false)
	grace := time.Now().Add(30 * 24 * time.Hour)
	u := &models.User{Plan: PlanFree, PlanGraceUntil: &grace}

	e := s.ResolveNow(u)

	if e.EffectivePlan != EffectivePremium {
		t.Fatalf("expected premium during grace, got %s", e.EffectivePlan)
	}
	if e.ShowAds {
		t.Fatal("expected ShowAds=false during grace")
	}
	if e.Plan != PlanFree {
		t.Fatalf("expected stored plan to remain free, got %s", e.Plan)
	}
	if e.PlanGraceUntil == nil || !e.PlanGraceUntil.Equal(grace) {
		t.Fatal("expected graced deadline to be exposed")
	}
	if w := *e.Features.TranslationWordLimit; w != TranslationWordLimitPremium {
		t.Fatalf("expected premium word limit during grace, got %d", w)
	}
}

func TestEntitlementServiceResolve_GraceExpired(t *testing.T) {
	s := NewEntitlementService(false)
	grace := time.Now().Add(-1 * time.Hour)
	u := &models.User{Plan: PlanFree, PlanGraceUntil: &grace}

	e := s.ResolveNow(u)

	if e.EffectivePlan != EffectiveFree {
		t.Fatalf("expected free after grace expiry, got %s", e.EffectivePlan)
	}
	if !e.ShowAds {
		t.Fatal("expected ShowAds after grace expiry")
	}
}

func TestEntitlementServiceResolve_SelfHost(t *testing.T) {
	s := NewEntitlementService(true)
	u := &models.User{Plan: PlanFree}

	e := s.ResolveNow(u)

	if e.EffectivePlan != EffectiveUnlimited {
		t.Fatalf("expected unlimited, got %s", e.EffectivePlan)
	}
	if e.ShowAds {
		t.Fatal("expected ShowAds=false on self-host")
	}
	if !e.SelfHost {
		t.Fatal("expected SelfHost=true")
	}
	assertAllLimitsUnlimited(t, e.Limits)
	if e.Features.TranslationWordLimit != nil {
		t.Fatal("expected unlimited word limit on self-host")
	}
	if e.Features.TranslationCharLimit != nil {
		t.Fatal("expected unlimited char limit on self-host")
	}
}

func TestEntitlementServiceResolve_NilUser(t *testing.T) {
	s := NewEntitlementService(false)
	e := s.ResolveNow(nil)

	if e.EffectivePlan != EffectiveFree {
		t.Fatalf("expected free for nil user, got %s", e.EffectivePlan)
	}
	if w := *e.Features.TranslationWordLimit; w != TranslationWordLimitFree {
		t.Fatalf("expected free word limit for nil user, got %d", w)
	}
}

func TestValidPlan(t *testing.T) {
	if !ValidPlan(PlanFree) || !ValidPlan(PlanPremium) {
		t.Fatal("expected free and premium to be valid plans")
	}
	if ValidPlan("enterprise") || ValidPlan("") {
		t.Fatal("expected unsupported plans to be invalid")
	}
}

func TestEntitlementServiceResolve_ExplicitTime(t *testing.T) {
	s := NewEntitlementService(false)
	future := time.Now().Add(24 * time.Hour)
	expired := time.Now().Add(-24 * time.Hour)

	if e := s.Resolve(&models.User{Plan: PlanFree, PlanGraceUntil: &future}, time.Now()); e.EffectivePlan != EffectivePremium {
		t.Fatalf("expected premium with future grace, got %s", e.EffectivePlan)
	}
	if e := s.Resolve(&models.User{Plan: PlanFree, PlanGraceUntil: &expired}, time.Now()); e.EffectivePlan != EffectiveFree {
		t.Fatalf("expected free with past grace, got %s", e.EffectivePlan)
	}
}

func TestEntitlementService_FeatureFlags_Free(t *testing.T) {
	s := NewEntitlementService(false)
	e := s.ResolveNow(&models.User{Plan: PlanFree})

	if e.Features.AutoGrammar {
		t.Fatal("expected no auto grammar on free")
	}
	if e.Features.FasterResponses {
		t.Fatal("expected no faster responses on free")
	}
	if e.Features.TranslationWordLimit == nil || *e.Features.TranslationWordLimit != TranslationWordLimitFree {
		t.Fatalf("expected free word limit %d, got %v", TranslationWordLimitFree, e.Features.TranslationWordLimit)
	}
	if e.Features.TranslationCharLimit == nil || *e.Features.TranslationCharLimit != TranslationCharLimitFree {
		t.Fatalf("expected free char limit %d, got %v", TranslationCharLimitFree, e.Features.TranslationCharLimit)
	}
}

func TestEntitlementService_FeatureFlags_Premium(t *testing.T) {
	s := NewEntitlementService(false)
	e := s.ResolveNow(&models.User{Plan: PlanPremium})

	if !e.Features.AutoGrammar {
		t.Fatal("expected auto grammar on premium")
	}
	if !e.Features.FasterResponses {
		t.Fatal("expected faster responses on premium")
	}
	if e.Features.TranslationWordLimit == nil || *e.Features.TranslationWordLimit != TranslationWordLimitPremium {
		t.Fatalf("expected premium word limit %d, got %v", TranslationWordLimitPremium, e.Features.TranslationWordLimit)
	}
}

func TestEntitlementService_FeatureFlags_SelfHost(t *testing.T) {
	s := NewEntitlementService(true)
	e := s.ResolveNow(&models.User{Plan: PlanFree})

	if !e.Features.AutoGrammar {
		t.Fatal("expected auto grammar on self-host")
	}
	if !e.Features.FasterResponses {
		t.Fatal("expected faster responses on self-host")
	}
	if e.Features.TranslationWordLimit != nil {
		t.Fatal("expected unlimited word limit on self-host")
	}
}

func TestEntitlementService_FeatureFlags_Grace(t *testing.T) {
	s := NewEntitlementService(false)
	grace := time.Now().Add(24 * time.Hour)
	e := s.ResolveNow(&models.User{Plan: PlanFree, PlanGraceUntil: &grace})

	if !e.Features.AutoGrammar {
		t.Fatal("expected auto grammar during grace")
	}
	if !e.Features.FasterResponses {
		t.Fatal("expected faster responses during grace")
	}
}

func TestWordCount(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"", 0},
		{"   ", 0},
		{"hello", 1},
		{"hello world", 2},
		{"  hello   brave  world  ", 3},
		{"café déjà vu", 3},
		{"日本語のテキスト", 1},
		{"one two three four", 4},
	}
	for _, c := range cases {
		if got := WordCount(c.in); got != c.want {
			t.Errorf("WordCount(%q) = %d, want %d", c.in, got, c.want)
		}
	}
	// A long message just under the free cap must pass; just over must trip.
	long := strings.Repeat("word ", 280)
	if WordCount(long) != 280 {
		t.Fatalf("expected 280 words, got %d", WordCount(long))
	}
	if WordCount(long+" extra") != 281 {
		t.Fatalf("expected 281 words, got %d", WordCount(long+" extra"))
	}
}

// assertAllLimitsUnlimited ensures no legacy per-day quota is set anywhere.
func assertAllLimitsUnlimited(t *testing.T, l Limits) {
	t.Helper()
	if l.DailyLLMTranslations != nil ||
		l.DailyLLMGrammarAnalyses != nil ||
		l.DailyLLMCorrections != nil ||
		l.DailyVoiceMessages != nil ||
		l.VocabularyItems != nil {
		t.Fatalf("expected all daily limits unlimited, got %+v", l)
	}
}
