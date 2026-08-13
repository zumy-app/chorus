import { useTranslation } from 'react-i18next'
import { useStore } from '../store'

interface PlanBadgeProps {
  size?: 'sm' | 'md'
}

// PlanBadge renders the user's current plan. It renders nothing until
// entitlements are loaded, and hides entirely on self-hosted deployments
// where monetization surfaces are suppressed.
export default function PlanBadge({ size = 'sm' }: PlanBadgeProps) {
  const { t } = useTranslation()
  const entitlements = useStore((s) => s.entitlements)

  if (!entitlements) return null
  if (entitlements.selfHost) return null

  const label = (
    entitlements.effectivePlan === 'unlimited'
      ? t('plan.unlimited')
      : entitlements.effectivePlan === 'premium'
        ? t('plan.premium')
        : t('plan.free')
  )

  const palette = {
    free: 'bg-gray-100 text-gray-600 border-gray-200',
    premium: 'bg-gradient-to-r from-amber-400 to-yellow-300 text-amber-900 border-amber-300',
    unlimited: 'bg-emerald-100 text-emerald-700 border-emerald-200',
  }[entitlements.effectivePlan] ?? 'bg-gray-100 text-gray-600 border-gray-200'

  const sizeCls = size === 'md' ? 'px-3 py-1 text-sm' : 'px-2 py-0.5 text-xs'

  return (
    <span
      className={`inline-flex items-center gap-1 rounded-full border font-semibold ${palette} ${sizeCls}`}
      title={entitlements.effectivePlan === 'unlimited' ? t('plan.selfHosted') : undefined}
    >
      {entitlements.effectivePlan === 'premium' ? '✦' : ''}
      {label}
    </span>
  )
}