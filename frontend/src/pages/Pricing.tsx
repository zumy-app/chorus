import { Link } from 'react-router-dom'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useStore } from '../store'
import { detectBrowserLanguage } from '../services/language'
import LanguageSelector from '../components/LanguageSelector'
import { MONTHLY_PRICE, YEARLY_PRICE, YEARLY_LIST_PRICE, PAYPAL_MONTHLY_URL, PAYPAL_YEARLY_URL } from '@chorus/shared'

// =============================================================================
// /pricing — the premium plan landing page. Presents plans and a feature
// comparison with a monthly (US$7.99/mo) / annual (US$79.90/yr) billing toggle.
// Logged-in free users check out directly via PayPal Subscriptions (plan IDs in
// config/subscriptions.ts); guests sign up first. Renewal/cancellation is
// managed on PayPal — the backend grants entitlements from its webhook.
// =============================================================================

interface FeatureRow {
  label: string
  free: string
  premium: string
}

export default function Pricing() {
  const { t } = useTranslation()
  const [selectedLang, setSelectedLang] = useState(() => localStorage.getItem('preferredLanguage') || detectBrowserLanguage())
  const [billing, setBilling] = useState<'monthly' | 'annual'>('annual')
  const user = useStore((s) => s.user)
  const entitlements = useStore((s) => s.entitlements)

  const isPremium = !!user && entitlements?.effectivePlan !== 'free'
  const isFree = !!user && !isPremium

  const handleLanguageChange = (code: string) => {
    setSelectedLang(code)
    localStorage.setItem('preferredLanguage', code)
  }

  const featureRows: FeatureRow[] = [
    { label: t('pricing.freeFeature2'), free: '✓', premium: '✓' },
    { label: t('pricing.premiumFeature1'), free: '✗', premium: '✓' },
    { label: t('pricing.premiumFeature2'), free: '✗', premium: '✓' },
    { label: t('pricing.premiumFeature3'), free: '✗', premium: '✓' },
    { label: t('pricing.freeFeature4'), free: '✓', premium: '✓' },
    { label: t('pricing.premiumFeature4'), free: '✗', premium: '✓' },
  ]

  return (
    <div className="min-h-screen bg-background text-on-surface" lang={selectedLang}>
      {/* Navigation */}
      <nav className="fixed top-0 w-full bg-surface/90 backdrop-blur border-b border-outline-variant/40 z-50">
        <div className="max-w-6xl mx-auto px-6 py-4 flex justify-between items-center">
          <Link to="/" className="flex items-center gap-3">
            <div className="w-10 h-10 bg-primary-container/10 rounded-full flex items-center justify-center">
              <span className="material-symbols-outlined text-primary text-2xl" style={{ fontVariationSettings: "'FILL' 1" }}>graphic_eq</span>
            </div>
            <span className="font-headline-md text-headline-md font-bold text-primary">Chorus</span>
          </Link>
          <div className="flex items-center gap-4">
            <ul className="hidden md:flex gap-8 items-center">
              <li><Link to="/#features" className="text-on-surface-variant hover:text-primary transition">{t('nav.features')}</Link></li>
              <li><Link to="/#how-it-works" className="text-on-surface-variant hover:text-primary transition">{t('nav.how')}</Link></li>
              <li><Link to="/#languages" className="text-on-surface-variant hover:text-primary transition">{t('nav.languages')}</Link></li>
              <li><Link to="/pricing" className="text-primary font-semibold transition">{t('nav.pricing')}</Link></li>
            </ul>
            <LanguageSelector
              currentLang={selectedLang}
              onLanguageChange={handleLanguageChange}
              variant="navbar"
            />
            {user ? (
              <Link to="/chat" className="px-4 py-2 bg-primary text-on-primary rounded-lg hover:bg-on-primary-fixed-variant transition whitespace-nowrap">
                {t('nav.openApp')}
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
      <section className="pt-32 pb-16 px-6">
        <div className="max-w-6xl mx-auto text-center">
          <h1 className="font-headline-md text-4xl md:text-5xl font-bold mb-4">{t('pricing.title')}</h1>
          <p className="text-body-lg text-on-surface-variant mb-4">{t('pricing.subtitle')}</p>
          <div className="inline-flex items-center gap-2 bg-primary-fixed text-primary text-sm font-semibold px-4 py-2 rounded-full">
            ✨ {t('plan.nudgeBody')}
          </div>
        </div>
      </section>

      {/* Pricing Cards */}
      <section className="px-6 pb-20">
        <div className="max-w-6xl mx-auto grid grid-cols-1 md:grid-cols-3 gap-8 items-start">
          {/* Free */}
          <div className="bg-surface-container-lowest rounded-2xl shadow-[0px_4px_12px_rgba(0,0,0,0.05)] p-8 flex flex-col">
            <h3 className="text-2xl font-bold mb-1">{t('pricing.freeName')}</h3>
            <p className="text-4xl font-bold mb-6">{t('pricing.freePrice')}<span className="text-base font-normal text-on-surface-variant">/{t('pricing.freePer')}</span></p>
            <ul className="mb-8 space-y-3 flex-1">
              {[t('pricing.freeFeature1'), t('pricing.freeFeature2'), t('pricing.freeFeature3'), t('pricing.freeFeature4')].map((f, i) => (
                <li key={i} className="flex items-start gap-2 text-on-surface-variant">
                  <span className="text-tertiary mt-0.5">✓</span>{f}
                </li>
              ))}
            </ul>
            {user ? (
              <span className="w-full px-4 py-3 border border-outline-variant rounded-lg text-on-surface-variant font-semibold text-center cursor-default">
                {t('plan.currentPlan')}
              </span>
            ) : (
              <Link to="/register" className="w-full px-4 py-3 border border-outline-variant rounded-lg text-on-surface font-semibold text-center hover:border-primary hover:text-primary transition">
                {t('landing.heroCta')}
              </Link>
            )}
          </div>

          {/* Premium */}
          <div className="bg-gradient-to-br from-primary to-secondary rounded-2xl p-8 text-on-primary shadow-2xl flex flex-col md:-my-4">
            <h3 className="text-2xl font-bold mb-4">✦ {t('pricing.premiumName')}</h3>
            <div className="flex gap-1 bg-white/10 rounded-lg p-1 mb-4">
              <button
                onClick={() => setBilling('monthly')}
                className={`flex-1 px-3 py-1.5 rounded-md text-sm font-semibold transition ${billing === 'monthly' ? 'bg-white text-primary' : 'text-on-primary/80 hover:text-on-primary'}`}
              >
                {t('plan.billingMonthly')}
              </button>
              <button
                onClick={() => setBilling('annual')}
                className={`flex-1 px-3 py-1.5 rounded-md text-sm font-semibold transition ${billing === 'annual' ? 'bg-white text-primary' : 'text-on-primary/80 hover:text-on-primary'}`}
              >
                {t('plan.billingAnnual')}
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
                  <span className="text-base font-normal text-on-primary/80">/{t('pricing.premiumPer')}</span>
                </p>
              )}
            </div>
            <ul className="mb-8 space-y-3 flex-1">
              {[t('pricing.premiumFeature1'), t('pricing.premiumFeature2'), t('pricing.premiumFeature3'), t('pricing.premiumFeature4')].map((f, i) => (
                <li key={i} className="flex items-start gap-2">
                  <span className="text-tertiary-fixed mt-0.5">✓</span>{f}
                </li>
              ))}
            </ul>
            {isPremium ? (
              <span className="w-full px-4 py-3 bg-white/20 text-on-primary rounded-lg font-bold text-center cursor-default">
                {t('plan.currentPlan')}
              </span>
            ) : isFree ? (
              <a
                href={billing === 'annual' ? PAYPAL_YEARLY_URL : PAYPAL_MONTHLY_URL}
                target="_blank"
                rel="noreferrer"
                className="w-full px-4 py-3 bg-white text-primary rounded-lg font-bold text-center hover:bg-gray-100 transition block"
              >
                {t('pricing.premiumCta')}
              </a>
            ) : (
              <Link to="/register" className="w-full px-4 py-3 bg-white text-primary rounded-lg font-bold text-center hover:bg-gray-100 transition">
                {t('auth.joinChorus')}
              </Link>
            )}
            <p className="text-xs text-on-primary/80 mt-3 text-center opacity-90">{t('pricing.purchaseNote')}</p>
          </div>

          {/* Enterprise */}
          <div className="bg-surface-container-lowest rounded-2xl shadow-[0px_4px_12px_rgba(0,0,0,0.05)] p-8 flex flex-col">
            <h3 className="text-2xl font-bold mb-1">{t('pricing.enterpriseName')}</h3>
            <p className="text-on-surface-variant mb-6">{t('pricing.enterpriseDesc')}</p>
            <ul className="mb-8 space-y-3 flex-1">
              {[t('pricing.enterpriseFeature1'), t('pricing.enterpriseFeature2'), t('pricing.enterpriseFeature3')].map((f, i) => (
                <li key={i} className="flex items-start gap-2 text-on-surface-variant">
                  <span className="text-tertiary mt-0.5">✓</span>{f}
                </li>
              ))}
            </ul>
            <a href="mailto:hello@chorus.talk?subject=Enterprise%20Enquiry" className="w-full px-4 py-3 border border-outline-variant rounded-lg text-on-surface font-semibold text-center hover:border-primary hover:text-primary transition">
              {t('pricing.enterpriseCta')}
            </a>
          </div>
        </div>
      </section>

      {/* Comparison table */}
      <section className="px-6 pb-20">
        <div className="max-w-4xl mx-auto">
          <h2 className="text-3xl md:text-4xl font-bold text-center mb-2">{t('pricing.compareTitle')}</h2>
          <p className="text-center text-on-surface-variant mb-10">{t('pricing.compareSubtitle')}</p>
          <div className="bg-surface-container-lowest rounded-2xl shadow-[0px_4px_12px_rgba(0,0,0,0.05)] overflow-hidden">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-outline-variant bg-surface-container-low text-left">
                  <th className="px-6 py-4 text-on-surface font-semibold">{t('pricing.feature')}</th>
                  <th className="px-6 py-4 text-center text-on-surface font-semibold">{t('pricing.freeName')}</th>
                  <th className="px-6 py-4 text-center text-primary font-semibold">✦ {t('pricing.premiumName')}</th>
                </tr>
              </thead>
              <tbody>
                {featureRows.map((row, i) => (
                  <tr key={i} className={`border-b border-outline-variant/40 text-on-surface-variant ${i % 2 ? 'bg-surface-container-low/50' : ''}`}>
                    <td className="px-6 py-4">{row.label}</td>
                    <td className="px-6 py-4 text-center">{row.free}</td>
                    <td className="px-6 py-4 text-center font-semibold text-primary">{row.premium}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <p className="text-center text-sm text-on-surface-variant mt-6">{t('pricing.note')}</p>
        </div>
      </section>

      {/* Footer */}
      <footer className="bg-inverse-surface text-white py-12 px-6">
        <div className="max-w-6xl mx-auto text-center">
          <p className="text-gray-400 mb-4">{t('pricing.title')} · Chorus</p>
          <Link to="/" className="inline-block text-primary-fixed hover:text-white transition">{t('waitlist.backToChorus')}</Link>
        </div>
      </footer>
    </div>
  )
}