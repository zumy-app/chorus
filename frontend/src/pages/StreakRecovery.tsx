import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import AppHeader from '../components/AppHeader'
import BottomNav from '../components/BottomNav'
import { learningAPI } from '../services/api'
import { useStore } from '../store'

export default function StreakRecovery() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const user = useStore(s => s.user)
  const targetLanguage = user?.targetLanguages?.[0] ?? 'es'
  const nativeLanguage = user?.nativeLanguage ?? 'en'
  const [recovering, setRecovering] = useState(false)
  const [recovered, setRecovered] = useState(false)
  const [error, setError] = useState(false)

  const doRecover = async (mode: string) => {
    setRecovering(true)
    setError(false)
    try {
      await learningAPI.recoverStreak(targetLanguage, nativeLanguage)
      if (mode === 'scenario') navigate('/learn/scenarios')
      else navigate('/learn/session?mode=vocabulary')
      setRecovered(true)
    } catch {
      setError(true)
    } finally {
      setRecovering(false)
    }
  }

  const skip = async () => {
    try { await learningAPI.recoverStreak(targetLanguage, nativeLanguage) } catch {}
    navigate('/learn')
  }

  return (
    <div className="h-screen flex flex-col bg-background">
      <AppHeader />
      <main className="flex-1 overflow-y-auto px-margin-mobile pt-8 pb-32 max-w-lg w-full mx-auto flex flex-col items-center">
        <div className="relative mb-6">
          <div className="absolute inset-0 bg-error-container rounded-full blur-xl opacity-40 scale-150" />
          <div className="relative w-28 h-28 mx-auto bg-surface-container-lowest rounded-full flex items-center justify-center shadow-lg border-2 border-outline-variant/50">
            <span className="material-symbols-outlined text-6xl text-outline-variant">local_fire_department</span>
            <div className="absolute -bottom-2 -right-2 bg-error text-on-error font-headline-sm text-headline-sm rounded-full w-10 h-10 flex items-center justify-center shadow-md border-2 border-surface-container-lowest">14</div>
          </div>
        </div>
        <h1 className="font-headline-lg text-headline-lg text-on-surface mb-2 text-center">{t('learn.streakRecoveryTitle', { defaultValue: 'Oh no, you missed a day!' })}</h1>
        <p className="font-body-lg text-body-lg text-on-surface-variant max-w-md mx-auto text-center mb-8">{t('learn.streakRecoveryBody', { defaultValue: "Don't let your 14-day streak burn out completely. Complete one of these quick tasks right now to recover it." })}</p>

        {recovered && <p className="font-body-md text-body-md text-tertiary mb-4">{t('learn.streakRecovered', { defaultValue: 'Streak recovered!' })}</p>}
        {error && <p className="font-body-md text-body-md text-error mb-4">{t('learn.streakRecoveryError', { defaultValue: 'Could not recover streak. Try again.' })}</p>}

        <div className="w-full flex flex-col gap-stack-md">
          <button data-testid="streak-recovery-scenario" onClick={() => doRecover('scenario')} disabled={recovering} className="bg-surface-container-lowest rounded-xl p-stack-md w-full text-left border border-transparent hover:border-primary-fixed-dim flex items-start gap-4 shadow-[0px_4px_12px_rgba(0,0,0,0.05)] disabled:opacity-50">
            <div className="w-12 h-12 rounded-full bg-secondary-fixed/50 flex items-center justify-center flex-shrink-0"><span className="material-symbols-outlined">theater_comedy</span></div>
            <div className="flex-grow">
              <div className="flex items-center gap-2 mb-1"><span className="font-label-sm text-label-sm text-secondary uppercase bg-secondary-fixed/50 px-2 py-0.5 rounded-full">The Challenge</span><span className="font-label-sm text-label-sm text-on-surface-variant flex items-center gap-1"><span className="material-symbols-outlined text-[14px]">timer</span> 5 min</span></div>
              <h3 className="font-headline-sm text-headline-sm text-on-surface">Scenario Roleplay</h3>
              <p className="font-body-sm text-body-sm text-on-surface-variant">Navigate a real-world conversation: <strong className="text-on-surface">Grocery Checkout</strong>.</p>
            </div>
            <span className="material-symbols-outlined text-outline">arrow_forward_ios</span>
          </button>

          <button data-testid="streak-recovery-review" onClick={() => doRecover('vocabulary')} disabled={recovering} className="bg-surface-container-lowest rounded-xl p-stack-md w-full text-left border border-transparent hover:border-primary-fixed-dim flex items-start gap-4 shadow-[0px_4px_12px_rgba(0,0,0,0.05)] disabled:opacity-50">
            <div className="w-12 h-12 rounded-full bg-tertiary-fixed/30 flex items-center justify-center flex-shrink-0"><span className="material-symbols-outlined">style</span></div>
            <div className="flex-grow">
              <span className="font-label-sm text-label-sm text-tertiary uppercase bg-tertiary-fixed/30 px-2 py-0.5 rounded-full">The Review</span>
              <h3 className="font-headline-sm text-headline-sm text-on-surface mt-1">Clear 15 SRS Cards</h3>
              <p className="font-body-sm text-body-sm text-on-surface-variant">Quickly review vocabulary items that are due for spaced repetition.</p>
            </div>
            <span className="material-symbols-outlined text-outline">arrow_forward_ios</span>
          </button>
        </div>

        <button onClick={skip} className="font-label-md text-label-md text-outline-variant hover:text-on-surface-variant p-2 mt-8">No thanks, let it reset to 0</button>
      </main>
      <BottomNav />
    </div>
  )
}
