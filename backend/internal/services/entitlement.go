package services

import (
	"time"

	"github.com/chorus/messenger/internal/models"
)

// Billing plans. plan is stored per user; entitlements are resolved at
// request time so gating/limits can evolve without schema changes.
const (
	PlanFree    = "free"
	PlanPremium = "premium"
)

// Effective plan values (what the client should display/act on). Self-hosted
// deployments are resolved as "unlimited" regardless of the stored plan.
const (
	EffectiveFree      = "free"
	EffectivePremium   = "premium"
	EffectiveUnlimited = "unlimited"
)

// ValidPlan reports whether p is a stored billing plan.
func ValidPlan(p string) bool {
	switch p {
	case PlanFree, PlanPremium:
		return true
	}
	return false
}

// TranslationCharLimitFree caps how long a message may be for it to be
// translated on the free plan. Longer messages are stored but not translated.
const TranslationCharLimitFree = 200

// Limits describes the usage quotas for a resolved entitlement set. A nil
// pointer means "no limit" (unlimited). These are intentionally not enforced
// yet — Phase 0 establishes the data + resolution surface only.
type Limits struct {
	// DailyLLMTranslations caps AI-powered message translations per day.
	DailyLLMTranslations *int `json:"dailyLLMTranslations"`
	// DailyLLMGrammarAnalyses caps AI grammar analyses (learn/analyze) per day.
	DailyLLMGrammarAnalyses *int `json:"dailyLLMGrammarAnalyses"`
	// DailyLLMCorrections caps AI grammar corrections per day.
	DailyLLMCorrections *int `json:"dailyLLMCorrections"`
	// DailyVoiceMessages caps up to how many voice messages can be sent per day.
	DailyVoiceMessages *int `json:"dailyVoiceMessages"`
	// VocabularyItems caps the number of saved vocabulary entries.
	VocabularyItems *int `json:"vocabularyItems"`
}

func intPtr(n int) *int { return &n }

// intLimit returns &n unless unlimited is true, in which case it returns nil.
func intLimit(unlimited bool, n int) *int {
	if unlimited {
		return nil
	}
	return intPtr(n)
}

// Entitlements is the fully-resolved result for a user: what plan they are on,
// whether an ad slot should render, and what quotas apply.
type Entitlements struct {
	// Plan is the stored billing plan ("free" / "premium").
	Plan string `json:"plan"`
	// PlanGraceUntil mirrors the stored grace deadline (may be nil).
	PlanGraceUntil *time.Time `json:"planGraceUntil,omitempty"`
	// EffectivePlan is what the client should display/act on: one of
	// free / premium / unlimited (self-hosted).
	EffectivePlan string `json:"effectivePlan"`
	// SelfHost is true for self-hosted deployments — monetization surfaces
	// (upsell, plan badges, ads) must be suppressed.
	SelfHost bool `json:"selfHost"`
	// ShowAds is true when an ad slot may render (free plan, hosted, not in
	// a grace window). It is a surface only — no ad network is integrated.
	ShowAds bool `json:"showAds"`
	// Features is the set of premium feature tiers the user is entitled to.
	Features FeatureFlags `json:"features"`
	Limits   Limits       `json:"limits"`
}

// FeatureFlags enumerates the premium feature tiers. Phase "premium feature
// tiers" — each flag drives a real product delta:
//   - AutoGrammar: ChatGPT-style automatic grammar analysis on incoming
//     messages (free plan requires a manual button press).
//   - TranslationCharLimit: how long a message may be before it is translated
//     (free = TranslationCharLimitFree, premium/unlimited = nil → any length).
//   - FasterResponses: translation jobs are queued with priority so premium
//     users' messages translate ahead of free (shorter waits).
type FeatureFlags struct {
	AutoGrammar          bool `json:"autoGrammar"`
	FasterResponses      bool `json:"fasterResponses"`
	TranslationCharLimit *int `json:"translationCharLimit,omitempty"`
}

func hasActiveGrace(u *models.User, now time.Time) bool {
	return u.PlanGraceUntil != nil && now.Before(*u.PlanGraceUntil)
}

// EntitlementService resolves per-user entitlements. It knows the deployment
// mode (hosted vs self-hosted) so monetization can be globally suppressed.
type EntitlementService struct {
	selfHost bool
}

func NewEntitlementService(selfHost bool) *EntitlementService {
	return &EntitlementService{selfHost: selfHost}
}

// premiumLimits returns the entitlement set for a premium (unlimited) user.
func premiumEntitlements(u *models.User) Entitlements {
	limit := 2000 // generous but finite default so a runaway message can't blow the LLM bill
	return Entitlements{
		Plan:           u.Plan,
		PlanGraceUntil: u.PlanGraceUntil,
		EffectivePlan:  EffectivePremium,
		ShowAds:        false,
		Features: FeatureFlags{
			AutoGrammar:          true,
			FasterResponses:      true,
			TranslationCharLimit: &limit,
		},
		Limits: Limits{
			DailyLLMTranslations:    nil,
			DailyLLMGrammarAnalyses: nil,
			DailyLLMCorrections:     nil,
			DailyVoiceMessages:      nil,
			VocabularyItems:         nil,
		},
	}
}

// freeLimitValues are the quotas for the free tier. Tuned to protect the
// biggest cost driver (LLM usage) while keeping the core product usable.
func freeLimits() Limits {
	return Limits{
		DailyLLMTranslations:    intPtr(50),
		DailyLLMGrammarAnalyses: intPtr(20),
		DailyLLMCorrections:     intPtr(20),
		DailyVoiceMessages:      intPtr(10),
		VocabularyItems:         intPtr(500),
	}
}

// Resolve computes the entitlements for a user at the given time.
func (s *EntitlementService) Resolve(u *models.User, now time.Time) Entitlements {
	if u == nil {
		limit := TranslationCharLimitFree
		return Entitlements{
			Plan:          PlanFree,
			EffectivePlan: EffectiveFree,
			ShowAds:       true,
			Features: FeatureFlags{
				AutoGrammar:          false,
				FasterResponses:      false,
				TranslationCharLimit: &limit,
			},
			Limits: freeLimits(),
		}
	}

	// Self-hosted deployments bypass monetization entirely.
	if s.selfHost {
		lim := Limits{}
		return Entitlements{
			Plan:           u.Plan,
			PlanGraceUntil: u.PlanGraceUntil,
			EffectivePlan:  EffectiveUnlimited,
			SelfHost:       true,
			ShowAds:        false,
			Features: FeatureFlags{
				AutoGrammar:          true,
				FasterResponses:      true,
				TranslationCharLimit: nil,
			},
			Limits: lim,
		}
	}

	// Grace window: accounts created before a paid cutover keep premium
	// entitlements until the explicit deadline, then drop to the stored plan.
	if hasActiveGrace(u, now) {
		e := premiumEntitlements(u)
		return e
	}

	switch u.Plan {
	case PlanPremium:
		return premiumEntitlements(u)
	default:
		limit := TranslationCharLimitFree
		return Entitlements{
			Plan:           u.Plan,
			PlanGraceUntil: u.PlanGraceUntil,
			EffectivePlan:  EffectiveFree,
			ShowAds:        true,
			Features: FeatureFlags{
				AutoGrammar:          false,
				FasterResponses:      false,
				TranslationCharLimit: &limit,
			},
			Limits: freeLimits(),
		}
	}
}

// ResolveNow computes entitlements at the current time; convenience for
// request-time resolution.
func (s *EntitlementService) ResolveNow(u *models.User) Entitlements {
	return s.Resolve(u, time.Now())
}
