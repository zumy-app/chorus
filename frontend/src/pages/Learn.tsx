import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import BottomNav from '../components/BottomNav'
import AppHeader from '../components/AppHeader'
import { learningAPI } from '../services/api'
import { useStore } from '../store'
import type { LearningDashboard } from '@chorus/shared'

const RING_RADIUS = 42
const RING_CIRCUMFERENCE = 2 * Math.PI * RING_RADIUS

export default function Learn() {
  const { t } = useTranslation()
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

  return (
    <div className="h-screen flex flex-col bg-background">
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
                        <button className="bg-primary/10 text-primary font-label-md text-label-md px-4 py-1.5 rounded-full hover:bg-primary/20 transition-colors">
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
