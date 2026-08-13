import { Link } from 'react-router-dom'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useStore } from '../store'
import { detectBrowserLanguage } from '../services/language'
import LanguageSelector from '../components/LanguageSelector'
import { MONTHLY_PRICE, YEARLY_PRICE, YEARLY_LIST_PRICE, PAYPAL_MONTHLY_URL, PAYPAL_YEARLY_URL } from '../config/subscriptions'

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
    <div className="min-h-screen bg-gradient-to-b from-white to-gray-50" lang={selectedLang}>
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
          <div className="flex items-center gap-4">
            <ul className="hidden md:flex gap-8 items-center">
              <li><Link to="/#features" className="text-gray-700 hover:text-indigo-600 transition">{t('nav.features')}</Link></li>
              <li><Link to="/#how-it-works" className="text-gray-700 hover:text-indigo-600 transition">{t('nav.how')}</Link></li>
              <li><Link to="/#languages" className="text-gray-700 hover:text-indigo-600 transition">{t('nav.languages')}</Link></li>
              <li><Link to="/pricing" className="text-indigo-600 font-semibold transition">{t('nav.pricing')}</Link></li>
            </ul>
            <LanguageSelector
              currentLang={selectedLang}
              onLanguageChange={handleLanguageChange}
              variant="navbar"
            />
            {user ? (
              <Link to="/chat" className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition whitespace-nowrap">
                {t('nav.openApp')}
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
      <section className="pt-32 pb-16 px-6">
        <div className="max-w-6xl mx-auto text-center">
          <h1 className="text-4xl md:text-5xl font-bold mb-4">{t('pricing.title')}</h1>
          <p className="text-xl text-gray-600 mb-4">{t('pricing.subtitle')}</p>
          <div className="inline-flex items-center gap-2 bg-indigo-100 text-indigo-700 text-sm font-semibold px-4 py-2 rounded-full">
            ✨ {t('plan.nudgeBody')}
          </div>
        </div>
      </section>

      {/* Pricing Cards */}
      <section className="px-6 pb-20">
        <div className="max-w-6xl mx-auto grid grid-cols-1 md:grid-cols-3 gap-8 items-start">
          {/* Free */}
          <div className="bg-white rounded-2xl shadow p-8 flex flex-col">
            <h3 className="text-2xl font-bold mb-1">{t('pricing.freeName')}</h3>
            <p className="text-4xl font-bold mb-6">{t('pricing.freePrice')}<span className="text-base font-normal text-gray-500">/{t('pricing.freePer')}</span></p>
            <ul className="mb-8 space-y-3 flex-1">
              {[t('pricing.freeFeature1'), t('pricing.freeFeature2'), t('pricing.freeFeature3'), t('pricing.freeFeature4')].map((f, i) => (
                <li key={i} className="flex items-start gap-2 text-gray-600">
                  <span className="text-green-500 mt-0.5">✓</span>{f}
                </li>
              ))}
            </ul>
            {user ? (
              <span className="w-full px-4 py-3 border border-gray-300 rounded-lg text-gray-500 font-semibold text-center cursor-default">
                {t('plan.currentPlan')}
              </span>
            ) : (
              <Link to="/register" className="w-full px-4 py-3 border border-gray-300 rounded-lg text-gray-700 font-semibold text-center hover:border-indigo-600 hover:text-indigo-600 transition">
                {t('landing.heroCta')}
              </Link>
            )}
          </div>

          {/* Premium */}
          <div className="bg-gradient-to-br from-indigo-600 to-purple-600 rounded-2xl p-8 text-white shadow-2xl flex flex-col md:-my-4">
            <h3 className="text-2xl font-bold mb-4">✦ {t('pricing.premiumName')}</h3>
            <div className="flex gap-1 bg-white/10 rounded-lg p-1 mb-4">
              <button
                onClick={() => setBilling('monthly')}
                className={`flex-1 px-3 py-1.5 rounded-md text-sm font-semibold transition ${billing === 'monthly' ? 'bg-white text-indigo-600' : 'text-indigo-100 hover:text-white'}`}
              >
                {t('plan.billingMonthly')}
              </button>
              <button
                onClick={() => setBilling('annual')}
                className={`flex-1 px-3 py-1.5 rounded-md text-sm font-semibold transition ${billing === 'annual' ? 'bg-white text-indigo-600' : 'text-indigo-100 hover:text-white'}`}
              >
                {t('plan.billingAnnual')}
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
                  <span className="text-base font-normal text-indigo-200">/{t('pricing.premiumPer')}</span>
                </p>
              )}
            </div>
            <ul className="mb-8 space-y-3 flex-1">
              {[t('pricing.premiumFeature1'), t('pricing.premiumFeature2'), t('pricing.premiumFeature3'), t('pricing.premiumFeature4')].map((f, i) => (
                <li key={i} className="flex items-start gap-2">
                  <span className="text-green-300 mt-0.5">✓</span>{f}
                </li>
              ))}
            </ul>
            {isPremium ? (
              <span className="w-full px-4 py-3 bg-white/20 text-white rounded-lg font-bold text-center cursor-default">
                {t('plan.currentPlan')}
              </span>
            ) : isFree ? (
              <a
                href={billing === 'annual' ? PAYPAL_YEARLY_URL : PAYPAL_MONTHLY_URL}
                target="_blank"
                rel="noreferrer"
                className="w-full px-4 py-3 bg-white text-indigo-600 rounded-lg font-bold text-center hover:bg-gray-100 transition block"
              >
                {t('pricing.premiumCta')}
              </a>
            ) : (
              <Link to="/register" className="w-full px-4 py-3 bg-white text-indigo-600 rounded-lg font-bold text-center hover:bg-gray-100 transition">
                {t('auth.joinChorus')}
              </Link>
            )}
            <p className="text-xs text-indigo-100 mt-3 text-center opacity-90">{t('pricing.purchaseNote')}</p>
          </div>

          {/* Enterprise */}
          <div className="bg-white rounded-2xl shadow p-8 flex flex-col">
            <h3 className="text-2xl font-bold mb-1">{t('pricing.enterpriseName')}</h3>
            <p className="text-gray-500 mb-6">{t('pricing.enterpriseDesc')}</p>
            <ul className="mb-8 space-y-3 flex-1">
              {[t('pricing.enterpriseFeature1'), t('pricing.enterpriseFeature2'), t('pricing.enterpriseFeature3')].map((f, i) => (
                <li key={i} className="flex items-start gap-2 text-gray-600">
                  <span className="text-green-500 mt-0.5">✓</span>{f}
                </li>
              ))}
            </ul>
            <a href="mailto:hello@chorus.talk?subject=Enterprise%20Enquiry" className="w-full px-4 py-3 border border-gray-300 rounded-lg text-gray-700 font-semibold text-center hover:border-indigo-600 hover:text-indigo-600 transition">
              {t('pricing.enterpriseCta')}
            </a>
          </div>
        </div>
      </section>

      {/* Comparison table */}
      <section className="px-6 pb-20">
        <div className="max-w-4xl mx-auto">
          <h2 className="text-3xl md:text-4xl font-bold text-center mb-2">{t('pricing.compareTitle')}</h2>
          <p className="text-center text-gray-600 mb-10">{t('pricing.compareSubtitle')}</p>
          <div className="bg-white rounded-2xl shadow overflow-hidden">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-gray-200 bg-gray-50 text-left">
                  <th className="px-6 py-4 text-gray-900 font-semibold">{t('pricing.feature')}</th>
                  <th className="px-6 py-4 text-center text-gray-900 font-semibold">{t('pricing.freeName')}</th>
                  <th className="px-6 py-4 text-center text-indigo-600 font-semibold">✦ {t('pricing.premiumName')}</th>
                </tr>
              </thead>
              <tbody>
                {featureRows.map((row, i) => (
                  <tr key={i} className={`border-b border-gray-100 text-gray-600 ${i % 2 ? 'bg-gray-50/50' : ''}`}>
                    <td className="px-6 py-4">{row.label}</td>
                    <td className="px-6 py-4 text-center">{row.free}</td>
                    <td className="px-6 py-4 text-center font-semibold text-indigo-600">{row.premium}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <p className="text-center text-sm text-gray-500 mt-6">{t('pricing.note')}</p>
        </div>
      </section>

      {/* Footer */}
      <footer className="bg-gray-900 text-white py-12 px-6">
        <div className="max-w-6xl mx-auto text-center">
          <p className="text-gray-400 mb-4">{t('pricing.title')} · Chorus</p>
          <Link to="/" className="inline-block text-indigo-400 hover:text-indigo-300 transition">{t('waitlist.backToChorus')}</Link>
        </div>
      </footer>
    </div>
  )
}