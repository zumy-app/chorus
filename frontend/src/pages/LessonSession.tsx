import { useCallback, useEffect, useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import AppHeader from '../components/AppHeader'
import BottomNav from '../components/BottomNav'
import { learningAPI } from '../services/api'
import { useStore } from '../store'
import type { SessionQuestion } from '@chorus/shared'

type Mode = 'daily' | 'quick_drill' | 'vocabulary' | 'grammar' | 'lesson'

export default function LessonSession() {
  const navigate = useNavigate()
  const location = useLocation()
  const user = useStore(s => s.user)
  const targetLanguage = user?.targetLanguages?.[0] ?? 'es'
  const nativeLanguage = user?.nativeLanguage ?? 'en'

  const [sessionId, setSessionId] = useState<string | null>(null)
  const [items, setItems] = useState<SessionQuestion[]>([])
  const [index, setIndex] = useState(0)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(false)
  const [answer, setAnswer] = useState('')
  const [feedback, setFeedback] = useState<{ correct: boolean; message: string; correctAnswer?: string } | null>(null)
  const [done, setDone] = useState(false)
  const [xp, setXp] = useState(0)

  const mode = (new URLSearchParams(location.search).get('mode') as Mode) || 'daily'

  useEffect(() => {
    let active = true
    setLoading(true)
    setError(false)
    learningAPI
      .startSession({ targetLanguage, nativeLanguage, mode, source: 'learn_home' })
      .then(res => {
        if (!active) return
        setSessionId(res.session.id)
        setItems(res.items)
        if (res.items.length === 0) {
          setDone(true)
        }
      })
      .catch(() => active && setError(true))
      .finally(() => active && setLoading(false))
    return () => {
      active = false
    }
  }, [targetLanguage, nativeLanguage, mode])

  const submit = useCallback(
    async (value: string) => {
      if (!sessionId || !items[index]) return
      setFeedback(null)
      const it = items[index]
      const response = await learningAPI.answerSessionItem(sessionId, it.id, { text: value, choice: value })
      setFeedback({ correct: response.correct, message: response.feedback.message, correctAnswer: response.feedback.correctAnswer })
      setXp(x => x + (response.correct ? 10 : 0))
    },
    [sessionId, items, index]
  )

  const next = useCallback(async () => {
    setFeedback(null)
    setAnswer('')
    if (index + 1 < items.length) {
      setIndex(index + 1)
      return
    }
    if (sessionId) {
      await learningAPI.completeSession(sessionId)
    }
    setDone(true)
  }, [index, items.length, sessionId])

  const current = items[index]

  return (
    <div className="h-screen flex flex-col bg-background">
      <AppHeader />
      <main className="flex-1 overflow-y-auto px-margin-mobile pt-4 pb-32 max-w-md w-full mx-auto">
        <div className="mb-4 flex items-center justify-between">
          <button onClick={() => navigate('/learn')} className="text-on-surface-variant font-label-md text-label-md">
            ← Back
          </button>
          {items.length > 0 && (
            <span className="font-label-sm text-label-sm text-on-surface-variant">
              {Math.min(index + 1, items.length)} / {items.length}
            </span>
          )}
        </div>

        {loading && <p className="font-body-md text-body-md text-on-surface-variant mt-6">{'Preparing your practice session...'}</p>}
        {!loading && error && <p className="font-body-md text-body-md text-error mt-6">{'Could not start a session.'}</p>}

        {!loading && !error && done && (
          <div className="mt-10 flex flex-col items-center gap-4 text-center">
            <div className="w-20 h-20 rounded-full bg-primary-container/20 flex items-center justify-center">
              <span className="material-symbols-outlined text-wrap text-primary text-[40px]">check_circle</span>
            </div>
            <h2 className="font-headline-lg text-headline-lg text-on-surface">{'Session complete!'}</h2>
            <p className="font-body-md text-body-md text-on-surface-variant">
              {`You earned ${xp} XP. Your progress has been updated.`}
            </p>
            <div className="flex gap-3 mt-2">
              <button onClick={() => navigate('/learn')} className="bg-primary text-on-primary font-label-md text-label-md px-5 py-2.5 rounded-full">
                {'Back to Learn'}
              </button>
            </div>
          </div>
        )}

        {!loading && !error && !done && current && (
          <div className="bg-surface-container-lowest rounded-[1.5rem] p-6 shadow-[0px_4px_12px_rgba(0,0,0,0.05)] flex flex-col gap-5">
            <div className="flex items-center gap-2">
              <span className="bg-secondary-fixed-dim/30 text-secondary font-label-sm text-label-sm px-3 py-1 rounded-full">
                {current.activityType}
              </span>
            </div>
            <div>
              <p className="font-headline-md text-headline-md text-on-surface">{current.prompt.text}</p>
              {current.prompt.source && (
                <p className="font-body-lg text-body-lg text-on-surface-variant mt-2">{current.prompt.source}</p>
              )}
              {current.prompt.translation && (
                <p className="font-body-sm text-body-sm text-outline mt-1">{current.prompt.translation}</p>
              )}
            </div>

            {current.prompt.choices && current.prompt.choices.length > 0 ? (
              <div className="flex flex-col gap-2">
                {current.prompt.choices.map((choice, i) => (
                  <button
                    key={i}
                    onClick={() => submit(choice)}
                    disabled={!!feedback}
                    className="bg-surface-container rounded-xl px-4 py-3 font-body-md text-body-md text-on-surface text-left hover:bg-primary-fixed-dim/20 transition-colors"
                  >
                    {choice}
                  </button>
                ))}
              </div>
            ) : (
              <div className="flex flex-col gap-3">
                <div className="flex items-center gap-2">
                  <span className="material-symbols-outlined text-primary">keyboard</span>
                  <span className="font-label-sm text-label-sm text-on-surface-variant">{'Type your answer'}</span>
                </div>
                <div className="flex gap-2">
                  <input
                    value={answer}
                    onChange={e => setAnswer(e.target.value)}
                    onKeyDown={e => e.key === 'Enter' && answer.trim() && submit(answer.trim())}
                    placeholder="Escribe aquí..."
                    className="flex-1 bg-surface-container rounded-xl px-4 py-3 font-body-md text-body-md text-on-surface outline-none"
                  />
                  <button
                    onClick={() => answer.trim() && submit(answer.trim())}
                    className="bg-primary text-on-primary font-label-md text-label-md px-5 rounded-xl"
                  >
                    <span className="material-symbols-outlined">arrow_forward</span>
                  </button>
                </div>
              </div>
            )}

            {feedback && (
              <div className={`rounded-xl p-4 ${feedback.correct ? 'bg-success-container/20' : 'bg-error-container/20'}`}>
                <div className="flex items-center gap-2">
                  <span className={`material-symbols-outlined ${feedback.correct ? 'text-success' : 'text-error'}`}>
                    {feedback.correct ? 'check_circle' : 'cancel'}
                  </span>
                  <span className="font-body-md text-body-md text-on-surface">{feedback.message}</span>
                </div>
                {feedback.correctAnswer && (
                  <p className="font-label-sm text-label-sm text-on-surface-variant mt-1">
                    {'Answer:'} {feedback.correctAnswer}
                  </p>
                )}
                <button onClick={next} className="mt-3 bg-primary/10 text-primary font-label-md text-label-md px-4 py-2 rounded-full w-full">
                  {'Continue'}
                </button>
              </div>
            )}
          </div>
        )}
      </main>
      <BottomNav />
    </div>
  )
}
