package services

import (
	"strings"
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

// Message-size limits. There are deliberately NO per-day usage quotas (the
// product differentiator is message size + grammar automation, not message
// counts). TranslationCharLimit* are legacy values kept for the frontend UX
// until it migrates to the word-based limits (Phase 2); the server-side gate
// is the word limit.
const (
	// TranslationCharLimitFree caps how long a message may be for it to be
	// translated on the free plan (chars). Longer messages are stored but not
	// translated. Deprecated in favor of the word limits.
	TranslationCharLimitFree = 200
	// TranslationCharLimitPremium is the premium char cap (legacy value).
	TranslationCharLimitPremium = 2000
	// TranslationWordLimitFree caps free messages at 280 words.
	TranslationWordLimitFree = 280
	// TranslationWordLimitPremium caps premium messages at 1,000 words.
	TranslationWordLimitPremium = 1000
	// MessageWordLimitMax is a hard ceiling for every plan: a runaway message
	// must never be able to blow the LLM budget regardless of entitlement.
	MessageWordLimitMax = 10000
)

// WordCount returns the number of whitespace-delimited tokens in s. It is a
// simple, language-agnostic approximation of "words".
func WordCount(s string) int {
	if strings.TrimSpace(s) == "" {
		return 0
	}
	return len(strings.Fields(s))
}

// Limits is the legacy per-day quota surface. Monetization deliberately has NO
// daily usage quotas (see REQUIREMENTS.md §7 P3/P4), so every value is always
// nil (= unlimited) for every plan. The fields are kept so the JSON shape
// (`limits: {...}`) stays stable for existing clients until they migrate.
type Limits struct {
	DailyLLMTranslations    *int `json:"dailyLLMTranslations"`
	DailyLLMGrammarAnalyses *int `json:"dailyLLMGrammarAnalyses"`
	DailyLLMCorrections     *int `json:"dailyLLMCorrections"`
	DailyVoiceMessages      *int `json:"dailyVoiceMessages"`
	VocabularyItems         *int `json:"vocabularyItems"`
}

// unlimitedLimits returns a Limits set with every quota unlimited (nil).
func unlimitedLimits() Limits {
	return Limits{}
}

// Entitlements is the fully-resolved result for a user: what plan they are on,
// whether an ad slot should render, and what caps apply.
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

// FeatureFlags enumerates the premium feature tiers — the ONLY plan
// differentiators. Each flag drives a real product delta:
//   - AutoGrammar: automatic grammar analysis on incoming messages (free plan
//     requires a manual/lazy action).
//   - TranslationWordLimit: message-size cap in words (free 280, premium 1000,
//     self-host/unlimited nil → any length). Messages beyond it are stored but
//     not translated.
//   - TranslationCharLimit: legacy char-based cap, kept for frontend UX.
//   - FasterResponses: translation jobs are queued with priority so premium
//     users' messages translate ahead of free.
type FeatureFlags struct {
	AutoGrammar          bool `json:"autoGrammar"`
	FasterResponses      bool `json:"fasterResponses"`
	TranslationCharLimit *int `json:"translationCharLimit,omitempty"`
	TranslationWordLimit *int `json:"translationWordLimit,omitempty"`
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

// premiumEntitlements returns the entitlement set for a premium user.
func premiumEntitlements(u *models.User) Entitlements {
	charLimit := TranslationCharLimitPremium
	wordLimit := TranslationWordLimitPremium
	return Entitlements{
		Plan:           u.Plan,
		PlanGraceUntil: u.PlanGraceUntil,
		EffectivePlan:  EffectivePremium,
		ShowAds:        false,
		Features: FeatureFlags{
			AutoGrammar:          true,
			FasterResponses:      true,
			TranslationCharLimit: &charLimit,
			TranslationWordLimit: &wordLimit,
		},
		Limits: unlimitedLimits(),
	}
}

// freeEntitlements returns the entitlement set for the free tier (also used
// for unknown/garbage user rows). u may be nil.
func freeEntitlements(u *models.User) Entitlements {
	charLimit := TranslationCharLimitFree
	wordLimit := TranslationWordLimitFree
	e := Entitlements{
		Plan:          PlanFree,
		EffectivePlan: EffectiveFree,
		ShowAds:       true,
		Features: FeatureFlags{
			AutoGrammar:          false,
			FasterResponses:      false,
			TranslationCharLimit: &charLimit,
			TranslationWordLimit: &wordLimit,
		},
		Limits: unlimitedLimits(),
	}
	if u != nil {
		e.Plan = u.Plan
		e.PlanGraceUntil = u.PlanGraceUntil
	}
	return e
}

// selfHostEntitlements returns the entitlement set for self-hosted installs:
// everything unlimited and monetization suppressed.
func selfHostEntitlements(u *models.User) Entitlements {
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
			TranslationWordLimit: nil,
		},
		Limits: unlimitedLimits(),
	}
}

// Resolve computes the entitlements for a user at the given time.
func (s *EntitlementService) Resolve(u *models.User, now time.Time) Entitlements {
	if u == nil {
		return freeEntitlements(nil)
	}

	// Self-hosted deployments bypass monetization entirely.
	if s.selfHost {
		return selfHostEntitlements(u)
	}

	// Grace window: accounts within their paid grace period keep premium
	// entitlements until the explicit deadline, then drop to the stored plan.
	if hasActiveGrace(u, now) {
		e := premiumEntitlements(u)
		e.Plan = u.Plan
		return e
	}

	switch u.Plan {
	case PlanPremium:
		return premiumEntitlements(u)
	default:
		return freeEntitlements(u)
	}
}

// ResolveNow computes entitlements at the current time; convenience for
// request-time resolution.
func (s *EntitlementService) ResolveNow(u *models.User) Entitlements {
	return s.Resolve(u, time.Now())
}
