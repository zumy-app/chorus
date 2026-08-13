package services

import (
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
	if e.Limits.DailyLLMTranslations == nil || *e.Limits.DailyLLMTranslations != 50 {
		t.Fatalf("expected 50 daily translations, got %v", e.Limits.DailyLLMTranslations)
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
	if e.Limits.DailyLLMTranslations != nil {
		t.Fatal("expected unlimited translations on premium")
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
	if e.Limits.DailyLLMTranslations != nil {
		t.Fatal("expected unlimited limits on self-host")
	}
}

func TestEntitlementServiceResolve_NilUser(t *testing.T) {
	s := NewEntitlementService(false)
	e := s.ResolveNow(nil)

	if e.EffectivePlan != EffectiveFree {
		t.Fatalf("expected free for nil user, got %s", e.EffectivePlan)
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

	graceFuture := &future
	gracePast := &expired

	if e := s.Resolve(&models.User{Plan: PlanFree, PlanGraceUntil: graceFuture}, time.Now()); e.EffectivePlan != EffectivePremium {
		t.Fatalf("expected premium with future grace, got %s", e.EffectivePlan)
	}
	if e := s.Resolve(&models.User{Plan: PlanFree, PlanGraceUntil: gracePast}, time.Now()); e.EffectivePlan != EffectiveFree {
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
	if e.Features.TranslationCharLimit == nil || *e.Features.TranslationCharLimit <= TranslationCharLimitFree {
		t.Fatalf("expected premium char limit above free, got %v", e.Features.TranslationCharLimit)
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
	if e.Features.TranslationCharLimit != nil {
		t.Fatal("expected unlimited char limit on self-host")
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
