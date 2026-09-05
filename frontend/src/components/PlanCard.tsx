import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'
import { useStore } from '../store'
import type { PlanLimits } from '@chorus/shared'
import PlanBadge from './PlanBadge'

// Ordered limit rows: label key in i18n + the quota field from PlanLimits.
const LIMIT_ROWS: { key: string; field: keyof PlanLimits }[] = [
  { key: 'plan.limitTranslations', field: 'dailyLLMTranslations' },
  { key: 'plan.limitGrammar', field: 'dailyLLMGrammarAnalyses' },
  { key: 'plan.limitCorrections', field: 'dailyLLMCorrections' },
  { key: 'plan.limitVoice', field: 'dailyVoiceMessages' },
  { key: 'plan.limitVocabulary', field: 'vocabularyItems' },
]

function LimitRow({ label, value }: { label: string; value: number | null | undefined }) {
  const { t } = useTranslation()
  return (
    <div className="flex items-center justify-between py-1.5 text-sm">
      <span className="text-gray-600">{label}</span>
      <span className="font-semibold text-gray-900">
        {value == null ? t('plan.unlimited') : value.toLocaleString()}
      </span>
    </div>
  )
}

// PlanCard shows the current plan, its quotas, and — only for free hosted
// users — a tasteful upsell nudge toward Premium. It renders nothing on
// self-hosted deployments.
export default function PlanCard() {
  const { t } = useTranslation()
  const entitlements = useStore((s) => s.entitlements)

  if (!entitlements) return null
  if (entitlements.selfHost) return null

  const isPremium = entitlements.effectivePlan !== 'free'
  const inGrace = entitlements.effectivePlan === 'premium' && entitlements.plan === 'free'
  const showUpsell = entitlements.effectivePlan === 'free'

  return (
    <div className="border border-gray-200 rounded-lg p-4 space-y-3">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-sm font-bold text-gray-900">
            {inGrace ? t('plan.premiumGraceTitle') : t('plan.currentPlan')}
          </h3>
          {inGrace && entitlements.planGraceUntil && (
            <p className="text-xs text-gray-500 mt-0.5">
              {t('plan.graceUntil', { date: new Date(entitlements.planGraceUntil).toLocaleDateString() })}
            </p>
          )}
        </div>
        <PlanBadge size="md" />
      </div>

      {isPremium ? (
        <p className="text-sm text-gray-600">{t('plan.premiumPerks')}</p>
      ) : (
        <>
          <div className="divide-y divide-gray-100">
            {LIMIT_ROWS.map((row) => (
              <LimitRow key={row.key} label={t(row.key)} value={entitlements.limits[row.field]} />
            ))}
          </div>
          {showUpsell && (
            <div className="mt-2 bg-gradient-to-r from-indigo-50 to-purple-50 border border-indigo-100 rounded-lg p-3">
              <p className="text-sm font-semibold text-gray-900 mb-1">{t('plan.nudgeTitle')}</p>
              <p className="text-xs text-gray-600 mb-3">{t('plan.nudgeBody')}</p>
              <Link
                to="/pricing"
                className="inline-block px-4 py-2 bg-gradient-to-r from-indigo-600 to-purple-600 text-white text-sm rounded-lg font-semibold hover:opacity-90 transition"
              >
                {t('plan.upgrade')}
              </Link>
            </div>
          )}
        </>
      )}
    </div>
  )
}