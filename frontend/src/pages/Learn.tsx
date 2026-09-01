import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import BottomNav from '../components/BottomNav'
import AppHeader from '../components/AppHeader'
import { learningAPI } from '../services/api'
import { useStore } from '../store'
import type { LearningDashboard, MonthlyActivityPoint } from '@chorus/shared'

const RING_RADIUS = 42
const RING_CIRCUMFERENCE = 2 * Math.PI * RING_RADIUS

export default function Learn() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const user = useStore(s => s.user)
  const [dashboard, setDashboard] = useState<LearningDashboard | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(false)

  const targetLanguage = user?.targetLanguages?.[0] ?? 'es'
  const nativeLanguage = user?.nativeLanguage ?? 'en'

  useEffect(() => {
    let active = true
    setLoading(true)
    setError(false)
    learningAPI
      .getDashboard(targetLanguage, nativeLanguage)
      .then(d => {
        if (active) setDashboard(d)
      })
      .catch(() => {
        if (active) setError(true)
      })
      .finally(() => {
        if (active) setLoading(false)
      })
    return () => {
      active = false
    }
  }, [targetLanguage, nativeLanguage])

  const supportTier = dashboard?.capability.supportTier

  const startSession = (mode: string) => navigate(`/learn/session?mode=${mode}`)

  return (    <div className="h-screen flex flex-col bg-background">
      <AppHeader />
      <main className="flex-1 overflow-y-auto px-margin-mobile pt-4 pb-32 max-w-md w-full mx-auto">
        <header className="flex flex-col gap-unit">
          <h1 className="font-headline-lg text-headline-lg text-on-background">{t('learn.title')}</h1>
          <p className="font-body-md text-body-md text-on-surface-variant">{t('learn.subtitle')}</p>
        </header>

        {loading && <p className="font-body-md text-body-md text-on-surface-variant mt-6">{t('learn.loading')}</p>}

        {!loading && error && (
          <div className="mt-6 bg-surface-container-lowest rounded-xl p-6 flex flex-col gap-stack-md items-start">
            <p className="font-body-md text-body-md text-on-surface-variant">{t('learn.error')}</p>
            <button
              onClick={() => {
                setError(false)
                setLoading(true)
                learningAPI
                  .getDashboard(targetLanguage, nativeLanguage)
                  .then(setDashboard)
                  .catch(() => setError(true))
                  .finally(() => setLoading(false))
              }}
              className="bg-primary text-on-primary font-label-md text-label-md px-4 py-2 rounded-full"
            >
              {t('learn.retry')}
            </button>
          </div>
        )}

        {!loading && !error && supportTier === 'disabled' && (
          <div className="mt-6 bg-surface-container-lowest rounded-xl p-6 flex flex-col gap-stack-sm">
            <h2 className="font-headline-sm text-headline-sm text-on-surface">{t('learn.learningUnavailableTitle')}</h2>
            <p className="font-body-sm text-body-sm text-on-surface-variant">{t('learn.learningUnavailableBody')}</p>
          </div>
        )}

        {!loading && !error && dashboard?.profile.placementStatus === 'not_started' && supportTier === 'full_course' && (
          <div className="mt-6 bg-surface-container-lowest rounded-2xl p-6 shadow-[0px_4px_12px_rgba(0,0,0,0.05)] flex flex-col gap-3">
            <h2 className="font-headline-md text-headline-md text-on-surface">{'Find your starting level'}</h2>
            <p className="font-body-sm text-body-sm text-on-surface-variant">{'Take a short placement test or start from the beginning.'}</p>
            <div className="flex gap-2">
              <button onClick={() => navigate('/learn/placement')} className="bg-primary text-on-primary font-label-md text-label-md px-4 py-2 rounded-full">
                {'Start test'}
              </button>
              <button
                onClick={() => learningAPI.skipPlacement(targetLanguage, nativeLanguage).then(() => window.location.reload())}
                className="bg-surface-container-high text-on-surface-variant font-label-md text-label-md px-4 py-2 rounded-full"
              >
                {'Start from scratch'}
              </button>
            </div>
          </div>
        )}

        {!loading && !error && supportTier === 'full_course' && (
          <div className="grid grid-cols-4 gap-2 mt-6">
            {[
              { label: 'Drills', icon: 'bolt', onClick: () => startSession('quick_drill') },
              { label: 'Vocabulary', icon: 'menu_book', onClick: () => navigate('/learn/vocabulary') },
              { label: 'Scenarios', icon: 'record_voice_over', onClick: () => navigate('/learn/scenarios') },
              { label: 'Real Talk', icon: 'forum', onClick: () => navigate('/learn/real-talk') },
            ].map(a => (
              <button key={a.label} onClick={a.onClick} className="bg-surface-container-lowest rounded-2xl p-3 flex flex-col items-center gap-1 shadow-[0px_4px_12px_rgba(0,0,0,0.05)]">
                <span className="material-symbols-outlined text-primary">{a.icon}</span>
                <span className="font-label-sm text-label-sm text-on-surface-variant">{a.label}</span>
              </button>
            ))}
          </div>
        )}

        {!loading && !error && supportTier === 'vocab_only' && (
          <div className="mt-6 bg-surface-container-lowest rounded-xl p-6 flex flex-col gap-stack-sm">
            <h2 className="font-headline-sm text-headline-sm text-on-surface">{t('learn.courseComingSoonTitle')}</h2>
            <p className="font-body-sm text-body-sm text-on-surface-variant">{t('learn.courseComingSoonBody')}</p>
          </div>
        )}

        {!loading && !error && dashboard && (
          <>
            {/* Daily goal + streak ring */}
            <div className="grid grid-cols-2 gap-stack-md mt-6">
              <div className="col-span-2 bg-surface-container-lowest rounded-[1.5rem] p-6 shadow-[0px_4px_12px_rgba(0,0,0,0.05)] flex items-center justify-between relative overflow-hidden">
                <div className="absolute inset-0 bg-gradient-to-br from-primary-fixed-dim/20 to-transparent opacity-50 pointer-events-none" />
                <div className="flex flex-col gap-stack-sm relative z-10">
                  <h2 className="font-headline-sm text-headline-sm text-on-surface">{t('learn.dailyGoal')}</h2>
                  <p className="font-body-sm text-body-sm text-on-surface-variant">
                    {dashboard.dailyGoal.completedItems} / {dashboard.dailyGoal.targetItems}
                  </p>
                  <div className="mt-2 inline-flex items-center gap-2 bg-secondary-fixed-dim/30 px-3 py-1.5 rounded-full w-max">
                    <span className="material-symbols-outlined text-secondary text-[18px]">local_fire_department</span>
                    <span className="font-label-md text-label-md text-on-surface-variant">
                      {t('learn.dayStreak', { count: dashboard.streak.days })}
                    </span>
                  </div>
                </div>
                <div className="relative w-24 h-24 flex-shrink-0 z-10 flex items-center justify-center">
                  <svg className="w-full h-full transform -rotate-90" viewBox="0 0 100 100">
                    <circle cx="50" cy="50" fill="transparent" r={RING_RADIUS} stroke="currentColor" strokeWidth="12" className="text-surface-container-high" />
                    <circle
                      cx="50"
                      cy="50"
                      fill="transparent"
                      r={RING_RADIUS}
                      stroke="currentColor"
                      strokeWidth="12"
                      strokeLinecap="round"
                      strokeDasharray={RING_CIRCUMFERENCE}
                      strokeDashoffset={RING_CIRCUMFERENCE * (1 - dashboard.dailyGoal.percent / 100)}
                      className="text-primary"
                    />
                  </svg>
                  <div className="absolute inset-0 flex items-center justify-center flex-col">
                    <span className="font-headline-md text-headline-md text-primary">{dashboard.dailyGoal.percent}%</span>
                  </div>
                </div>
              </div>

              {/* Fluency readiness */}
              <div className="bg-surface-container-lowest rounded-[1.25rem] p-5 shadow-[0px_4px_12px_rgba(0,0,0,0.05)] flex flex-col justify-between gap-stack-md relative overflow-hidden">
                <div className="w-10 h-10 rounded-full bg-primary-container/10 flex items-center justify-center text-primary">
                  <span className="material-symbols-outlined text-[20px]">speed</span>
                </div>
                <div>
                  <div className="font-headline-lg text-headline-lg text-on-surface">{dashboard.fluency.readinessScore}</div>
                  <div className="font-label-sm text-label-sm text-on-surface-variant uppercase tracking-wider mt-1">{t('learn.fluency')}</div>
                  <div className="font-label-sm text-label-sm text-primary mt-1">{dashboard.fluency.label}</div>
                </div>
              </div>

              {/* Vocabulary */}
              <div className="bg-surface-container-lowest rounded-[1.25rem] p-5 shadow-[0px_4px_12px_rgba(0,0,0,0.05)] flex flex-col justify-between gap-stack-md relative overflow-hidden">
                <div className="w-10 h-10 rounded-full bg-tertiary-container/10 flex items-center justify-center text-tertiary-container">
                  <span className="material-symbols-outlined text-[20px]">menu_book</span>
                </div>
                <div>
                  <div className="font-headline-lg text-headline-lg text-on-surface">{dashboard.vocabulary.total}</div>
                  <div className="font-label-sm text-label-sm text-on-surface-variant uppercase tracking-wider mt-1">{t('learn.vocabulary')}</div>
                  <div className="font-label-sm text-label-sm text-on-surface-variant mt-1">
                    {t('learn.dueWords')}: {dashboard.vocabulary.dueToday}
                  </div>
                </div>
              </div>

              {/* Grammar weakness */}
              <div className="col-span-2 bg-surface-container-lowest rounded-[1.25rem] p-5 shadow-[0px_4px_12px_rgba(0,0,0,0.05)] flex items-center justify-between relative overflow-hidden">
                <div className="flex items-center gap-stack-md">
                  <div className="w-12 h-12 rounded-full bg-secondary-container/10 flex items-center justify-center text-secondary">
                    <span className="material-symbols-outlined text-[20px]">rule</span>
                  </div>
                  <div>
                    <div className="font-label-sm text-label-sm text-on-surface-variant uppercase tracking-wider">{t('learn.grammar')}</div>
                    <div className="font-headline-sm text-headline-sm text-on-surface mt-0.5">
                      {dashboard.grammar.weakestPointTitle || '—'}
                    </div>
                    {dashboard.grammar.dueToday > 0 && (
                      <div className="font-label-sm text-label-sm text-outline mt-0.5">{t('learn.dueCount', { count: dashboard.grammar.dueToday })}</div>
                    )}
                  </div>
                </div>
              </div>

              {supportTier === 'full_course' && dashboard.currentUnit && (
                <div className="col-span-2 bg-surface-container-lowest rounded-[1.25rem] p-5 shadow-[0px_4px_12px_rgba(0,0,0,0.05)] flex flex-col gap-stack-sm">
                  <div className="font-label-sm text-label-sm text-on-surface-variant uppercase tracking-wider">{t('learn.currentUnit')}</div>
                  <div className="flex items-center justify-between">
                    <div className="font-headline-sm text-headline-sm text-on-surface">{dashboard.currentUnit.title}</div>
                    <div className="font-label-md text-label-md text-primary">{dashboard.currentUnit.cefrLevel}</div>
                  </div>
                  <div className="w-full h-2 rounded-full bg-surface-container-high overflow-hidden">
                    <div className="h-full bg-primary rounded-full" style={{ width: `${dashboard.currentUnit.progressPct}%` }} />
                  </div>
                  {dashboard.nextLesson && (
                    <div className="font-body-sm text-body-sm text-on-surface-variant">
                      {t('learn.nextLesson')}: {dashboard.nextLesson.title}
                    </div>
                  )}
                </div>
              )}
            </div>

            {/* Monthly activity (FR-31): words learned / sentences understood per month */}
            <MonthlyActivityCard activity={dashboard.monthlyActivity ?? []} />

            {/* Recommended activities */}
            {dashboard.recommendedActivities.length > 0 && (
              <section className="flex flex-col gap-stack-md mt-6">
                <header className="flex items-center justify-between">
                  <h3 className="font-headline-sm text-headline-sm text-on-surface">{t('learn.recommendedActivities')}</h3>
                </header>
                <div className="flex flex-col gap-stack-sm">
                  {dashboard.recommendedActivities.map(activity => (
                    <div
                      key={activity.id}
                      className="bg-surface-container-lowest rounded-xl p-4 shadow-[0px_4px_12px_rgba(0,0,0,0.05)] border border-transparent hover:border-primary-fixed-dim transition-colors flex flex-col gap-stack-sm cursor-pointer"
                    >
                      <div className="flex items-start justify-between gap-2">
                        <div className="flex items-center gap-2">
                          <span className={`material-symbols-outlined ${activity.priority === 'high' ? 'text-error' : 'text-secondary'}`}>
                            {activity.priority === 'high' ? 'error' : 'tips_and_updates'}
                          </span>
                          <span className="font-label-md text-label-md text-on-surface">{activity.title}</span>
                        </div>
                        {activity.priority === 'high' && (
                          <span className="bg-error-container text-on-error-container font-label-sm text-label-sm px-2 py-0.5 rounded-full">
                            {t('learn.highPriority')}
                          </span>
                        )}
                      </div>
                      <p className="font-body-sm text-body-sm text-on-surface-variant">{activity.description}</p>
                      <div className="mt-1 flex items-center justify-between">
                        <span className="font-label-sm text-label-sm text-outline">{t('learn.minExercise', { count: activity.estimatedMinutes })}</span>
                        <button
                          onClick={() => {
                            if (activity.action === 'open_scenarios') navigate('/learn/scenarios')
                            else if (activity.type === 'lesson') startSession('daily')
                            else startSession(activity.id === 'vocabulary' ? 'vocabulary' : activity.type)
                          }}
                          className="bg-primary/10 text-primary font-label-md text-label-md px-4 py-1.5 rounded-full hover:bg-primary/20 transition-colors"
                        >
                          {t('learn.start')}
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              </section>
            )}
          </>
        )}
      </main>
      <BottomNav />
    </div>
  )
}

function MonthlyActivityCard({ activity }: { activity: MonthlyActivityPoint[] }) {
  const { t } = useTranslation()
  // Default to the most recent bucket (ascending order), but always allow
  // browsing back through the 12 returned months.
  const [idx, setIdx] = useState(() => Math.max(0, activity.length - 1))
  const clamped = Math.min(idx, Math.max(0, activity.length - 1))
  const point = activity[clamped]

  const prev = () => setIdx(i => Math.max(0, Math.min(i, activity.length - 1) - 1))
  const next = () => setIdx(i => Math.min(activity.length - 1, Math.min(i, activity.length - 1) + 1))

  return (
    <div data-testid="learn-monthly" className="bg-surface-container-lowest rounded-[1.25rem] p-5 shadow-[0px_4px_12px_rgba(0,0,0,0.05)] flex flex-col gap-stack-md mt-6 relative overflow-hidden">
      <div className="flex items-center justify-between">
        <div className="w-10 h-10 rounded-full bg-primary-container/10 flex items-center justify-center text-primary">
          <span className="material-symbols-outlined text-[20px]">insights</span>
        </div>
        <div className="flex items-center gap-1">
          <button
            aria-label={t('learn.previousMonth')}
            onClick={prev}
            disabled={clamped <= 0}
            className="w-8 h-8 rounded-full flex items-center justify-center text-on-surface-variant hover:bg-surface-container-high disabled:opacity-30 disabled:pointer-events-none"
          >
            <span className="material-symbols-outlined text-[20px]">chevron_left</span>
          </button>
          <button
            aria-label={t('learn.nextMonth')}
            onClick={next}
            disabled={clamped >= activity.length - 1}
            className="w-8 h-8 rounded-full flex items-center justify-center text-on-surface-variant hover:bg-surface-container-high disabled:opacity-30 disabled:pointer-events-none"
          >
            <span className="material-symbols-outlined text-[20px]">chevron_right</span>
          </button>
        </div>
      </div>
      <div>
        <div className="font-label-sm text-label-sm text-on-surface-variant uppercase tracking-wider">
          {t('learn.monthlyActivity')}
        </div>
        <div className="font-headline-md text-headline-md text-on-surface mt-1">
          {point ? formatMonth(point.month) : t('learn.noMonthlyActivity')}
        </div>
      </div>

      {point ? (
        <div className="flex items-center gap-stack-md">
          <div className="flex-1 bg-secondary-container/10 rounded-xl p-4">
            <div className="font-headline-lg text-headline-lg text-secondary">{point.wordsLearned}</div>
            <div className="font-label-sm text-label-sm text-on-surface-variant mt-1">
              {t('learn.wordsThisMonth')}
            </div>
          </div>
          <div className="flex-1 bg-tertiary-container/10 rounded-xl p-4">
            <div className="font-headline-lg text-headline-lg text-tertiary">{point.sentencesUnderstood}</div>
            <div className="font-label-sm text-label-sm text-on-surface-variant mt-1">
              {t('learn.sentencesThisMonth')}
            </div>
          </div>
        </div>
      ) : (
        <div className="rounded-xl p-4 bg-surface-container-high/40">
          <p className="font-body-sm text-body-sm text-on-surface-variant">{t('learn.noMonthlyActivity')}</p>
        </div>
      )}
    </div>
  )
}

const MONTH_NAMES = [
  'January', 'February', 'March', 'April', 'May', 'June',
  'July', 'August', 'September', 'October', 'November', 'December',
]

function formatMonth(month: string): string {
  const [y, m] = month.split('-').map(Number)
  if (!y || !m || m < 1 || m > 12) return month
  return `${MONTH_NAMES[m - 1]} ${y}`
}
