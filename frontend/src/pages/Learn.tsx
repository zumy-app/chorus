import { useTranslation } from 'react-i18next'
import BottomNav from '../components/BottomNav'
import AppHeader from '../components/AppHeader'

const RING_RADIUS = 42
const RING_CIRCUMFERENCE = 2 * Math.PI * RING_RADIUS
const GOAL_PCT = 75

export default function Learn() {
  const { t } = useTranslation()

  return (
    <div className="h-screen flex flex-col bg-background">
      <AppHeader />
      <main className="flex-1 overflow-y-auto px-margin-mobile pt-4 pb-32 max-w-md w-full mx-auto">
        <header className="flex flex-col gap-unit">
          <h1 className="font-headline-lg text-headline-lg text-on-background">{t('learn.title')}</h1>
          <p className="font-body-md text-body-md text-on-surface-variant">{t('learn.subtitle')}</p>
        </header>

        {/* Bento Grid Dashboard */}
        <div className="grid grid-cols-2 gap-stack-md mt-6">
          {/* Daily Goal Ring */}
          <div className="col-span-2 bg-surface-container-lowest rounded-[1.5rem] p-6 shadow-[0px_4px_12px_rgba(0,0,0,0.05)] flex items-center justify-between relative overflow-hidden">
            <div className="absolute inset-0 bg-gradient-to-br from-primary-fixed-dim/20 to-transparent opacity-50 pointer-events-none" />
            <div className="flex flex-col gap-stack-sm relative z-10">
              <h2 className="font-headline-sm text-headline-sm text-on-surface">{t('learn.dailyGoal')}</h2>
              <p className="font-body-sm text-body-sm text-on-surface-variant">{t('learn.almostThere')}</p>
              <div className="mt-2 inline-flex items-center gap-2 bg-secondary-fixed-dim/30 px-3 py-1.5 rounded-full w-max">
                <span className="material-symbols-outlined text-secondary text-[18px]">local_fire_department</span>
                <span className="font-label-md text-label-md text-on-surface-variant">{t('learn.dayStreak', { count: 12 })}</span>
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
                  strokeDashoffset={RING_CIRCUMFERENCE * (1 - GOAL_PCT / 100)}
                  className="text-primary"
                />
              </svg>
              <div className="absolute inset-0 flex items-center justify-center flex-col">
                <span className="font-headline-md text-headline-md text-primary">{GOAL_PCT}%</span>
              </div>
            </div>
          </div>

          {/* Messages Translated */}
          <div className="bg-surface-container-lowest rounded-[1.25rem] p-5 shadow-[0px_4px_12px_rgba(0,0,0,0.05)] flex flex-col justify-between gap-stack-md relative overflow-hidden">
            <div className="w-10 h-10 rounded-full bg-tertiary-container/10 flex items-center justify-center text-tertiary-container">
              <span className="material-symbols-outlined text-[20px]">forum</span>
            </div>
            <div>
              <div className="font-headline-lg text-headline-lg text-on-surface">1.2k</div>
              <div className="font-label-sm text-label-sm text-on-surface-variant uppercase tracking-wider mt-1">{t('learn.messagesTranslated')}</div>
            </div>
          </div>

          {/* New Words Learned */}
          <div className="bg-surface-container-lowest rounded-[1.25rem] p-5 shadow-[0px_4px_12px_rgba(0,0,0,0.05)] flex flex-col justify-between gap-stack-md relative overflow-hidden">
            <div className="w-10 h-10 rounded-full bg-primary-container/10 flex items-center justify-center text-primary">
              <span className="material-symbols-outlined text-[20px]">menu_book</span>
            </div>
            <div>
              <div className="font-headline-lg text-headline-lg text-on-surface">342</div>
              <div className="font-label-sm text-label-sm text-on-surface-variant uppercase tracking-wider mt-1">{t('learn.newWordsLearned')}</div>
            </div>
          </div>

          {/* Grammar Mastered */}
          <div className="col-span-2 bg-surface-container-lowest rounded-[1.25rem] p-5 shadow-[0px_4px_12px_rgba(0,0,0,0.05)] flex items-center justify-between relative overflow-hidden">
            <div className="flex items-center gap-stack-md">
              <div className="w-12 h-12 rounded-full bg-secondary-container/10 flex items-center justify-center text-secondary">
                <span className="material-symbols-outlined text-[20px]">rule</span>
              </div>
              <div>
                <div className="font-label-sm text-label-sm text-on-surface-variant uppercase tracking-wider">{t('learn.grammarMastered')}</div>
                <div className="font-headline-sm text-headline-sm text-on-surface mt-0.5">{t('learn.concepts', { count: 48 })}</div>
              </div>
            </div>
            <span className="material-symbols-outlined text-outline-variant">chevron_right</span>
          </div>
        </div>

        {/* Recommended Activities */}
        <section className="flex flex-col gap-stack-md mt-6">
          <header className="flex items-center justify-between">
            <h3 className="font-headline-sm text-headline-sm text-on-surface">{t('learn.recommendedActivities')}</h3>
            <button className="font-label-md text-label-md text-primary hover:underline">{t('learn.viewAll')}</button>
          </header>
          <div className="flex flex-col gap-stack-sm">
            {/* Activity 1: High priority */}
            <div className="bg-surface-container-lowest rounded-xl p-4 shadow-[0px_4px_12px_rgba(0,0,0,0.05)] border border-transparent hover:border-primary-fixed-dim transition-colors flex flex-col gap-stack-sm cursor-pointer">
              <div className="flex items-start justify-between gap-2">
                <div className="flex items-center gap-2">
                  <span className="material-symbols-outlined text-error">error</span>
                  <span className="font-label-md text-label-md text-on-surface">{t('learn.pastTenseVerbs')}</span>
                </div>
                <span className="bg-error-container text-on-error-container font-label-sm text-label-sm px-2 py-0.5 rounded-full">{t('learn.highPriority')}</span>
              </div>
              <p className="font-body-sm text-body-sm text-on-surface-variant">{t('learn.pastTenseDesc')}</p>
              <div className="mt-1 flex items-center justify-between">
                <span className="font-label-sm text-label-sm text-outline">{t('learn.minExercise', { count: 5 })}</span>
                <button className="bg-primary/10 text-primary font-label-md text-label-md px-4 py-1.5 rounded-full hover:bg-primary/20 transition-colors">{t('learn.practice')}</button>
              </div>
            </div>

            {/* Activity 2: Vocabulary review */}
            <div className="bg-surface-container-lowest rounded-xl p-4 shadow-[0px_4px_12px_rgba(0,0,0,0.05)] border border-transparent hover:border-primary-fixed-dim transition-colors flex flex-col gap-stack-sm cursor-pointer relative overflow-hidden">
              <div className="absolute inset-y-0 left-0 w-1 bg-secondary-fixed" />
              <div className="flex items-start justify-between gap-2 pl-2">
                <div className="flex items-center gap-2">
                  <span className="material-symbols-outlined text-secondary">tips_and_updates</span>
                  <span className="font-label-md text-label-md text-on-surface">{t('learn.vocabReview')}</span>
                </div>
              </div>
              <p className="font-body-sm text-body-sm text-on-surface-variant pl-2">{t('learn.vocabReviewDesc')}</p>
              <div className="mt-1 flex items-center justify-between pl-2">
                <span className="font-label-sm text-label-sm text-outline">{t('learn.minReview', { count: 3 })}</span>
                <button className="bg-primary text-on-primary font-label-md text-label-md px-4 py-1.5 rounded-full shadow-sm hover:opacity-90 transition-opacity">{t('learn.start')}</button>
              </div>
            </div>
          </div>
        </section>
      </main>
      <BottomNav />
    </div>
  )
}