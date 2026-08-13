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
} from '../config/subscriptions'
import type { SubscriptionInfo } from '../types'

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
    <div className="min-h-screen bg-gradient-to-b from-white to-gray-50">
      {/* Navigation */}
      <nav className="fixed top-0 w-full bg-white/95 backdrop-blur border-b border-gray-200 z-50">
        <div className="max-w-6xl mx-auto px-6 py-4 flex justify-between items-center">
          <Link to="/" className="flex items-center gap-3">
            <div className="w-10 h-10 bg-gradient-to-br from-indigo-600 to-purple-600 rounded-full flex items-center justify-center">
              <svg className="w-6 h-6 text-white" fill="currentColor" viewBox="0 0 20 20">
                <path d="M2 5a2 2 0 012-2h12a2 2 0 012 2v10a2 2 0 01-2 2H4a2 2 0 01-2-2V5z"></path>
                <path d="M7.5 7.5a1.5 1.5 0 113 0 1.5 1.5 0 01-3 0z"></path>
              </svg>
            </div>
            <span className="text-2xl font-bold bg-gradient-to-r from-indigo-600 to-purple-600 bg-clip-text text-transparent">Chorus</span>
          </Link>
          <div className="flex items-center gap-3">
            <PlanBadge size="md" />
            {user ? (
              <Link to="/chat" className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition whitespace-nowrap">
                {t('premium.backToChat')}
              </Link>
            ) : (
              <Link to="/login" className="px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:border-indigo-600 hover:text-indigo-600 transition whitespace-nowrap">
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
          <p className="text-xl text-gray-600">{t('premium.subtitle')}</p>
        </div>
      </section>

      {selfHost ? (
        <section className="px-6 pb-20">
          <div className="max-w-lg mx-auto bg-white rounded-2xl shadow p-8 text-center">
            <p className="text-gray-600">{t('plan.selfHosted')}</p>
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
                  className="px-4 py-2 bg-gradient-to-r from-indigo-600 to-purple-600 text-white rounded-lg font-semibold hover:opacity-90 transition disabled:opacity-50"
                >
                  {checkingOut ? t('common.loading') : t('premium.renew')}
                </button>
              </div>
            )}

            <div className="bg-white rounded-2xl shadow p-8">
              <div className="flex items-center justify-between mb-6">
                <div>
                  <h2 className="text-2xl font-bold mb-1">{t('premium.youArePremium')}</h2>
                  {subscription?.premiumSince && (
                    <p className="text-sm text-gray-500">
                      {t('premium.membershipSince', { date: fmtDate(subscription.premiumSince) })}
                    </p>
                  )}
                </div>
                <PlanBadge size="md" />
              </div>

              <dl className="divide-y divide-gray-100 border-t border-b border-gray-100 mb-6">
                {subscription?.nextBillingDate && (
                  <div className="flex justify-between py-3 text-sm">
                    <dt className="text-gray-500">{t('premium.nextBilling')}</dt>
                    <dd className="font-semibold text-gray-900">{fmtDate(subscription.nextBillingDate)}</dd>
                  </div>
                )}
                {subscription?.status && (
                  <div className="flex justify-between py-3 text-sm">
                    <dt className="text-gray-500">{t('premium.status')}</dt>
                    <dd className="font-semibold text-gray-900">{subscription.status}</dd>
                  </div>
                )}
                {subscription?.wordLimit != null && (
                  <div className="flex justify-between py-3 text-sm">
                    <dt className="text-gray-500">{t('pricing.premiumFeature3')}</dt>
                    <dd className="font-semibold text-gray-900">{subscription.wordLimit.toLocaleString()}</dd>
                  </div>
                )}
              </dl>

              {subscription?.manageUrl ? (
                <a
                  href={subscription.manageUrl}
                  target="_blank"
                  rel="noreferrer"
                  className="block w-full px-4 py-3 bg-gradient-to-r from-indigo-600 to-purple-600 text-white rounded-lg font-bold text-center hover:opacity-90 transition"
                >
                  {t('premium.manageLink')}
                </a>
              ) : (
                <p className="text-sm text-gray-500 text-center">{t('premium.noManageUrl')}</p>
              )}
            </div>

            <div className="bg-white rounded-2xl shadow p-8">
              <h3 className="text-lg font-bold mb-4">{t('pricing.compareTitle')}</h3>
              <ul className="space-y-3">
                {premiumFeatures.map((f, i) => (
                  <li key={i} className="flex items-start gap-2 text-gray-700">
                    <span className="text-green-500 mt-0.5">✓</span>{f}
                  </li>
                ))}
              </ul>
              <p className="text-xs text-gray-400 mt-6">{t('pricing.purchaseNote')}</p>
            </div>
          </div>
        </section>
      ) : (
        <section className="px-6 pb-20">
          <div className="max-w-lg mx-auto bg-gradient-to-br from-indigo-600 to-purple-600 rounded-2xl p-8 text-white shadow-2xl">
            <h2 className="text-2xl font-bold mb-4">✦ {t('pricing.premiumName')}</h2>
            <div className="flex gap-1 bg-white/10 rounded-lg p-1 mb-4">
              <button
                onClick={() => setBilling('monthly')}
                className={`flex-1 px-3 py-1.5 rounded-md text-sm font-semibold transition ${billing === 'monthly' ? 'bg-white text-indigo-600' : 'text-indigo-100 hover:text-white'}`}
              >
                {t('premium.monthly')}
              </button>
              <button
                onClick={() => setBilling('annual')}
                className={`flex-1 px-3 py-1.5 rounded-md text-sm font-semibold transition ${billing === 'annual' ? 'bg-white text-indigo-600' : 'text-indigo-100 hover:text-white'}`}
              >
                {t('premium.annual')}
              </button>
            </div>
            <div className="mb-6 text-center">
              {billing === 'annual' ? (
                <>
                  <p className="text-base font-semibold text-indigo-200/80 mb-1">
                    <s>{YEARLY_LIST_PRICE}</s>
                    <span className="text-sm font-normal text-indigo-200">/{t('plan.perYear')}</span>
                  </p>
                  <p className="text-4xl font-bold mb-1">
                    {YEARLY_PRICE}
                    <span className="text-base font-normal text-indigo-200">/{t('plan.perYear')}</span>
                  </p>
                  <p className="text-xs font-semibold text-indigo-50">✨ {t('pricing.yearlyFreeMonths')}</p>
                </>
              ) : (
                <p className="text-4xl font-bold">
                  {MONTHLY_PRICE}
                  <span className="text-base font-normal text-indigo-200">/{t('plan.perMonth')}</span>
                </p>
              )}
            </div>
            <ul className="mb-8 space-y-3">
              {premiumFeatures.map((f, i) => (
                <li key={i} className="flex items-start gap-2">
                  <span className="text-green-300 mt-0.5">✓</span>{f}
                </li>
              ))}
            </ul>
            {isFree ? (
              <button
                onClick={() => startCheckout(billing)}
                disabled={checkingOut != null}
                className="w-full px-4 py-3 bg-white text-indigo-600 rounded-lg font-bold text-center hover:bg-gray-100 transition disabled:opacity-60"
              >
                {checkingOut ? t('common.loading') : t('pricing.premiumCta')}
              </button>
            ) : (
              <Link to="/register" className="block w-full px-4 py-3 bg-white text-indigo-600 rounded-lg font-bold text-center hover:bg-gray-100 transition">
                {t('auth.joinChorus')}
              </Link>
            )}
            <p className="text-xs text-indigo-100 mt-3 text-center opacity-90">{t('pricing.purchaseNote')}</p>
          </div>
        </section>
      )}

      {/* Footer */}
      <footer className="bg-gray-900 text-white py-12 px-6">
        <div className="max-w-6xl mx-auto text-center">
          <p className="text-gray-400 mb-4">{t('premium.title')} · Chorus</p>
          {user ? (
            <Link to="/chat" className="inline-block text-indigo-400 hover:text-indigo-300 transition">
              {t('premium.backToChat')}
            </Link>
          ) : (
            <Link to="/" className="inline-block text-indigo-400 hover:text-indigo-300 transition">
              {t('waitlist.backToChorus')}
            </Link>
          )}
        </div>
      </footer>
    </div>
  )
}