import { useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useStore } from '../store'
import { billingAPI } from '../services/api'
import PlanBadge from '../components/PlanBadge'
import {
  MONTHLY_PRICE,
  YEARLY_PRICE,
  YEARLY_LIST_PRICE,
  PAYPAL_MONTHLY_URL,
  PAYPAL_YEARLY_URL,
} from '@chorus/shared'
import type { SubscriptionInfo } from '@chorus/shared'

// =============================================================================
// /premium — the subscription hub. Logged-in premium users manage their
// subscription (PayPal), see billing/grace details, and return here after
// checkout. Free users see the plans and start checkout through the backend
// (POST /users/me/subscription/checkout); guests are routed to registration.
// =============================================================================

function fmtDate(value?: string | null): string | undefined {
  if (!value) return undefined
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return undefined
  return d.toLocaleDateString()
}

export default function Premium() {
  const { t } = useTranslation()
  const { user, entitlements, refreshEntitlements } = useStore()
  const [billing, setBilling] = useState<'monthly' | 'annual'>('annual')
  const [subscription, setSubscription] = useState<SubscriptionInfo | null>(null)
  const [checkingOut, setCheckingOut] = useState<'monthly' | 'annual' | null>(null)

  const effectivePlan = entitlements?.effectivePlan ?? null
  const isPremium = !!user && effectivePlan === 'premium'
  const isFree = !!user && effectivePlan === 'free'
  const selfHost = !!entitlements?.selfHost

  const loadSubscription = useCallback(async () => {
    if (!user) return
    try {
      const info = await billingAPI.getMySubscription()
      setSubscription(info)
    } catch (error) {
      console.error('Failed to load subscription:', error)
      setSubscription(null)
    }
  }, [user])

  // P11: refresh subscription + entitlements on mount, so users returning from
  // PayPal checkout see their new premium status immediately.
  useEffect(() => {
    loadSubscription()
    refreshEntitlements()
  }, [loadSubscription, refreshEntitlements])

  const startCheckout = async (plan: 'monthly' | 'annual') => {
    setCheckingOut(plan)
    try {
      const resp = await billingAPI.checkout(plan, '/premium', '/premium')
      window.location.href = resp.approvalUrl
      return
    } catch (error) {
      console.warn('Checkout unavailable, opening PayPal directly:', error)
    }
    // Fall back to the provider's plan URLs when the backend checkout can't
    // start (e.g. PayPal unconfigured in dev).
    window.location.href = plan === 'annual' ? PAYPAL_YEARLY_URL : PAYPAL_MONTHLY_URL
  }

  const premiumFeatures = [
    t('pricing.premiumFeature1'),
    t('pricing.premiumFeature2'),
    t('pricing.premiumFeature3'),
    t('pricing.premiumFeature4'),
  ]

  return (
    <div className="min-h-screen bg-background text-on-surface">
      {/* Navigation */}
      <nav className="fixed top-0 w-full bg-surface/90 backdrop-blur border-b border-outline-variant/40 z-50">
        <div className="max-w-6xl mx-auto px-6 py-4 flex justify-between items-center">
          <Link to="/" className="flex items-center gap-3">
            <div className="w-10 h-10 bg-primary-container/10 rounded-full flex items-center justify-center">
              <span className="material-symbols-outlined text-primary text-2xl" style={{ fontVariationSettings: "'FILL' 1" }}>graphic_eq</span>
            </div>
            <span className="font-headline-md text-headline-md font-bold text-primary">Chorus</span>
          </Link>
          <div className="flex items-center gap-3">
            <PlanBadge size="md" />
            {user ? (
              <Link to="/chat" className="px-4 py-2 bg-primary text-on-primary rounded-lg hover:bg-on-primary-fixed-variant transition whitespace-nowrap">
                {t('premium.backToChat')}
              </Link>
            ) : (
              <Link to="/login" className="px-4 py-2 border border-outline-variant text-on-surface rounded-lg hover:border-primary hover:text-primary transition whitespace-nowrap">
                {t('nav.login')}
              </Link>
            )}
          </div>
        </div>
      </nav>

      {/* Hero */}
      <section className="pt-32 pb-12 px-6">
        <div className="max-w-6xl mx-auto text-center">
          <h1 className="text-4xl md:text-5xl font-bold mb-4">✦ {t('premium.title')}</h1>
          <p className="text-body-lg text-on-surface-variant">{t('premium.subtitle')}</p>
        </div>
      </section>

      {selfHost ? (
        <section className="px-6 pb-20">
          <div className="max-w-lg mx-auto bg-surface-container-lowest rounded-2xl shadow-[0px_4px_12px_rgba(0,0,0,0.05)] p-8 text-center">
            <p className="text-on-surface-variant">{t('plan.selfHosted')}</p>
          </div>
        </section>
      ) : isPremium ? (
        <section className="px-6 pb-20">
          <div className="max-w-2xl mx-auto space-y-6">
            {subscription?.inGrace && (
              <div className="bg-amber-50 border border-amber-200 rounded-2xl p-6">
                <p className="font-semibold text-amber-900 mb-1">
                  {t('premium.graceBanner', { date: fmtDate(subscription.graceUntil) })}
                </p>
                <p className="text-sm text-amber-800 mb-4">{t('premium.graceBannerDesc')}</p>
                <button
                  onClick={() => startCheckout('annual')}
                  disabled={checkingOut != null}
                  className="px-4 py-2 bg-gradient-to-r from-primary to-secondary text-on-primary rounded-lg font-semibold hover:opacity-90 transition disabled:opacity-50"
                >
                  {checkingOut ? t('common.loading') : t('premium.renew')}
                </button>
              </div>
            )}

            <div className="bg-surface-container-lowest rounded-2xl shadow-[0px_4px_12px_rgba(0,0,0,0.05)] p-8">
              <div className="flex items-center justify-between mb-6">
                <div>
                  <h2 className="text-2xl font-bold mb-1">{t('premium.youArePremium')}</h2>
                  {subscription?.premiumSince && (
                    <p className="text-sm text-on-surface-variant">
                      {t('premium.membershipSince', { date: fmtDate(subscription.premiumSince) })}
                    </p>
                  )}
                </div>
                <PlanBadge size="md" />
              </div>

              <dl className="divide-y divide-outline-variant/40 border-t border-b border-outline-variant/40 mb-6">
                {subscription?.nextBillingDate && (
                  <div className="flex justify-between py-3 text-sm">
                    <dt className="text-on-surface-variant">{t('premium.nextBilling')}</dt>
                    <dd className="font-semibold text-on-surface">{fmtDate(subscription.nextBillingDate)}</dd>
                  </div>
                )}
                {subscription?.status && (
                  <div className="flex justify-between py-3 text-sm">
                    <dt className="text-on-surface-variant">{t('premium.status')}</dt>
                    <dd className="font-semibold text-on-surface">{subscription.status}</dd>
                  </div>
                )}
                {subscription?.wordLimit != null && (
                  <div className="flex justify-between py-3 text-sm">
                    <dt className="text-on-surface-variant">{t('plan.messageSizeLimit')}</dt>
                    <dd className="font-semibold text-on-surface">{subscription.wordLimit.toLocaleString()} {t('plan.words')}</dd>
                  </div>
                )}
              </dl>

              {subscription?.manageUrl ? (
                <a
                  href={subscription.manageUrl}
                  target="_blank"
                  rel="noreferrer"
                  className="block w-full px-4 py-3 bg-gradient-to-r from-primary to-secondary text-on-primary rounded-lg font-bold text-center hover:opacity-90 transition"
                >
                  {t('premium.manageLink')}
                </a>
              ) : (
                <p className="text-sm text-on-surface-variant text-center">{t('premium.noManageUrl')}</p>
              )}
            </div>

            <div className="bg-surface-container-lowest rounded-2xl shadow-[0px_4px_12px_rgba(0,0,0,0.05)] p-8">
              <h3 className="text-lg font-bold mb-4">{t('pricing.compareTitle')}</h3>
              <ul className="space-y-3">
                {premiumFeatures.map((f, i) => (
                  <li key={i} className="flex items-start gap-2 text-on-surface-variant">
                    <span className="text-tertiary mt-0.5">✓</span>{f}
                  </li>
                ))}
              </ul>
              <p className="text-xs text-on-surface-variant/60 mt-6">{t('pricing.purchaseNote')}</p>
            </div>
          </div>
        </section>
      ) : (
        <section className="px-6 pb-20">
          <div className="max-w-lg mx-auto bg-gradient-to-br from-primary to-secondary rounded-2xl p-8 text-on-primary shadow-2xl">
            <h2 className="text-2xl font-bold mb-4">✦ {t('pricing.premiumName')}</h2>
            <div className="flex gap-1 bg-white/10 rounded-lg p-1 mb-4">
              <button
                onClick={() => setBilling('monthly')}
                className={`flex-1 px-3 py-1.5 rounded-md text-sm font-semibold transition ${billing === 'monthly' ? 'bg-white text-primary' : 'text-on-primary/80 hover:text-on-primary'}`}
              >
                {t('premium.monthly')}
              </button>
              <button
                onClick={() => setBilling('annual')}
                className={`flex-1 px-3 py-1.5 rounded-md text-sm font-semibold transition ${billing === 'annual' ? 'bg-white text-primary' : 'text-on-primary/80 hover:text-on-primary'}`}
              >
                {t('premium.annual')}
              </button>
            </div>
            <div className="mb-6 text-center">
              {billing === 'annual' ? (
                <>
                  <p className="text-base font-semibold text-on-primary/80 mb-1">
                    <s>{YEARLY_LIST_PRICE}</s>
                    <span className="text-sm font-normal text-on-primary/80">/{t('plan.perYear')}</span>
                  </p>
                  <p className="text-4xl font-bold mb-1">
                    {YEARLY_PRICE}
                    <span className="text-base font-normal text-on-primary/80">/{t('plan.perYear')}</span>
                  </p>
                  <p className="text-xs font-semibold text-on-primary/90">✨ {t('pricing.yearlyFreeMonths')}</p>
                </>
              ) : (
                <p className="text-4xl font-bold">
                  {MONTHLY_PRICE}
                  <span className="text-base font-normal text-on-primary/80">/{t('plan.perMonth')}</span>
                </p>
              )}
            </div>
            <ul className="mb-8 space-y-3">
              {premiumFeatures.map((f, i) => (
                <li key={i} className="flex items-start gap-2">
                  <span className="text-tertiary-fixed mt-0.5">✓</span>{f}
                </li>
              ))}
            </ul>
            {isFree ? (
              <button
                onClick={() => startCheckout(billing)}
                disabled={checkingOut != null}
                className="w-full px-4 py-3 bg-white text-primary rounded-lg font-bold text-center hover:bg-gray-100 transition disabled:opacity-60"
              >
                {checkingOut ? t('common.loading') : t('pricing.premiumCta')}
              </button>
            ) : (
              <Link to="/register" className="block w-full px-4 py-3 bg-white text-primary rounded-lg font-bold text-center hover:bg-gray-100 transition">
                {t('auth.joinChorus')}
              </Link>
            )}
            <p className="text-xs text-on-primary/80 mt-3 text-center opacity-90">{t('pricing.purchaseNote')}</p>
          </div>
        </section>
      )}

      {/* Footer */}
      <footer className="bg-inverse-surface text-white py-12 px-6">
        <div className="max-w-6xl mx-auto text-center">
          <p className="text-gray-400 mb-4">{t('premium.title')} · Chorus</p>
          {user ? (
            <Link to="/chat" className="inline-block text-primary-fixed hover:text-white transition">
              {t('premium.backToChat')}
            </Link>
          ) : (
            <Link to="/" className="inline-block text-primary-fixed hover:text-white transition">
              {t('waitlist.backToChorus')}
            </Link>
          )}
        </div>
      </footer>
    </div>
  )
}