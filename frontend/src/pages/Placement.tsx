import { useCallback, useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import AppHeader from '../components/AppHeader'
import BottomNav from '../components/BottomNav'
import { learningAPI } from '../services/api'
import { useStore } from '../store'
import type { PlacementQuestion, PlacementResult, StartPlacementResponse } from '@chorus/shared'

export default function Placement() {
  const navigate = useNavigate()
  const user = useStore(s => s.user)
  const targetLanguage = user?.targetLanguages?.[0] ?? 'es'
  const nativeLanguage = user?.nativeLanguage ?? 'en'

  const [attemptId, setAttemptId] = useState<string | null>(null)
  const [question, setQuestion] = useState<PlacementQuestion | null>(null)
  const [total, setTotal] = useState(0)
  const [answered, setAnswered] = useState(0)
  const [result, setResult] = useState<PlacementResult | null>(null)
  const [loading, setLoading] = useState(true)
  const [answer, setAnswer] = useState<string | null>(null)
  const [typed, setTyped] = useState('')
  const [built, setBuilt] = useState<string[]>([])

  const prompt: any = question
    ? typeof question.prompt === 'object' && question.prompt !== null
      ? (question.prompt as any)
      : { text: String(question.prompt ?? '') }
    : {}

  const resetAnswer = () => {
    setAnswer(null)
    setTyped('')
    setBuilt([])
  }

  useEffect(() => {
    if (question?.itemType === 'gap_fill_type') setAnswer(typed.trim() || null)
  }, [typed, question?.itemType])
  useEffect(() => {
    setAnswer(built.length ? built.join(' ') : null)
  }, [built])

  const submit = useCallback(async () => {
    if (!attemptId || !answer) return
    setLoading(true)
    try {
      const res = await learningAPI.answerPlacement(attemptId, answer)
      resetAnswer()
      // If a PlacementResult came back, the test is complete.
      if ('estimatedCefr' in res && (res as PlacementResult).estimatedCefr) {
        setResult(res as PlacementResult)
      } else {
        const next = res as StartPlacementResponse
        setQuestion(next.question)
        setAnswered(answered + 1)
        setAttemptId(next.attemptId)
      }
    } catch {
      // progress response
    } finally {
      setLoading(false)
    }
  }, [attemptId, answer, answered])

  const skip = useCallback(async () => {
    setLoading(true)
    try {
      const res = await learningAPI.skipPlacement(targetLanguage, nativeLanguage)
      setResult(res)
    } catch {
      setResult({ attemptId: '', estimatedCefr: 'A1', readinessScore: 0, activeUnitId: '' })
    } finally {
      setLoading(false)
    }
  }, [targetLanguage, nativeLanguage])

  const isReconstruction = question?.itemType === 'sentence_reconstruction'
  const isGapFill = question?.itemType === 'gap_fill_type'
  const isPassage = question?.itemType === 'passage_mcq'
  const hasChoices = (question?.choices?.length ?? 0) > 0

  useEffect(() => {
    setLoading(true)
    learningAPI
      .startPlacement(targetLanguage, nativeLanguage)
      .then(res => {
        setAttemptId(res.attemptId)
        setQuestion(res.question)
        setTotal(res.totalQuestions)
      })
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [targetLanguage, nativeLanguage])

  return (
    <div className="h-screen flex flex-col bg-background">
      <AppHeader />
      <main className="flex-1 overflow-y-auto px-margin-mobile pt-4 pb-32 max-w-md w-full mx-auto">
        {loading && !question && !result && <p className="font-body-md text-body-md text-on-surface-variant mt-6">{'Loading placement...'}</p>}

        {result && (
          <div className="mt-10 flex flex-col items-center gap-4 text-center">
            <div className="w-20 h-20 rounded-full bg-primary-container/20 flex items-center justify-center">
              <span className="material-symbols-outlined text-primary text-[40px]">flag</span>
            </div>
            <h2 className="font-headline-lg text-headline-lg text-on-surface">{'You are ready to learn!'}</h2>
            <p className="font-body-md text-body-md text-on-surface-variant">
              {'Your starting level is '}
              <span className="font-headline-sm text-headline-sm text-primary">{result.estimatedCefr}</span>
              {result.readinessScore > 0 && ` — readiness ${result.readinessScore}/1000`}
            </p>
            <p className="font-body-sm text-body-sm text-on-surface-variant">{'Your learning path has been personalized.'}</p>
            <button onClick={() => navigate('/learn')} className="bg-primary text-on-primary font-label-md text-label-md px-6 py-3 rounded-full mt-2">
              {'Start Learning'}
            </button>
          </div>
        )}

        {!result && question && (
          <div className="flex flex-col gap-4">
            <div className="flex items-center justify-between">
              <span className="font-label-sm text-label-sm text-on-surface-variant">{`Question ${answered + 1} of ${total}`}</span>
              <button onClick={skip} className="font-label-sm text-label-sm text-outline">{'Skip test'}</button>
            </div>
            <div className="w-full h-2 rounded-full bg-surface-container-high overflow-hidden">
              <div className="h-full bg-primary rounded-full" style={{ width: `${(answered / total) * 100}%` }} />
            </div>
            <div className="bg-surface-container-lowest rounded-[1.5rem] p-6 shadow-[0px_4px_12px_rgba(0,0,0,0.05)] flex flex-col gap-5">
              <div className="flex items-center gap-2">
                <span className="bg-secondary-fixed-dim/30 text-secondary font-label-sm text-label-sm px-3 py-1 rounded-full">{question.cefrLevel}</span>
                <span className="font-label-sm text-label-sm text-on-surface-variant">{question.module ? `${question.module} · ` : ''}{question.itemType}</span>
              </div>

              {isPassage && prompt.passage && (
                <>
                  <p className="font-body-md text-body-md text-on-surface leading-relaxed">{prompt.passage}</p>
                  <h2 className="font-headline-md text-headline-md text-on-surface">{prompt.question}</h2>
                </>
              )}

              {isGapFill && prompt.sentence_with_blank && (
                <>
                  <h2 className="font-headline-md text-headline-md text-on-surface">{prompt.sentence_with_blank}</h2>
                  <input
                    value={typed}
                    onChange={e => setTyped(e.target.value)}
                    onKeyDown={e => { if (e.key === 'Enter') submit() }}
                    autoFocus
                    autoCapitalize="none"
                    placeholder="Type the missing word"
                    className="bg-surface-container rounded-xl px-4 py-3 font-body-lg text-body-lg text-on-surface outline-none"
                  />
                  {prompt.translation && <p className="font-body-sm text-body-sm text-on-surface-variant">{prompt.translation}</p>}
                </>
              )}

              {isReconstruction && (
                <>
                  <h2 className="font-headline-md text-headline-md text-on-surface">{prompt.instruction ?? 'Order the words to make a correct sentence.'}</h2>
                  <div className="bg-surface-container rounded-xl px-4 py-3 min-h-14 flex flex-wrap gap-2 items-center">
                    {built.length === 0 && <span className="font-body-md text-body-md text-outline">Tap the words in order</span>}
                    {built.map((w, i) => (
                      <button key={`${w}-${i}`} onClick={() => setBuilt(built.filter((_, idx) => idx !== i))} className="bg-primary text-on-primary font-body-md text-body-md px-3 py-1 rounded-full">
                        {w}
                      </button>
                    ))}
                  </div>
                  <div className="flex flex-wrap gap-2">
                    {(prompt.words ?? []).map((w: string, i: number) => {
                      const used = built.filter(b => b === w).length
                      const total = (prompt.words ?? []).filter((x: string) => x === w).length
                      const exhausted = used >= total
                      return (
                        <button key={`${w}-${i}`} disabled={exhausted} onClick={() => setBuilt([...built, w])} className={`font-body-md text-body-md px-3 py-1 rounded-full ${exhausted ? 'opacity-30 bg-surface-container-high text-on-surface-variant' : 'bg-surface-container-high text-on-surface'}`}>
                          {w}
                        </button>
                      )
                    })}
                  </div>
                </>
              )}

              {!isPassage && !isGapFill && !isReconstruction && (
                <>
                  {prompt.sentence && (
                    <p className="font-body-lg text-body-lg text-on-surface-variant">
                      {String(prompt.sentence).split(prompt.target_word ?? '\u0000').map((part: string, i: number, arr: string[]) => (
                        <span key={i}>
                          {part}
                          {i < arr.length - 1 && <span className="text-primary font-bold">{prompt.target_word}</span>}
                        </span>
                      ))}
                    </p>
                  )}
                  <h2 className="font-headline-md text-headline-md text-on-surface">{prompt.context_translation ?? prompt.text ?? ''}</h2>
                </>
              )}

              {hasChoices && !isGapFill && !isReconstruction && (
                <div className="flex flex-col gap-2">
                  {(question.choices || []).map((choice, i) => (
                    <button
                      key={i}
                      onClick={() => setAnswer(choice)}
                      className={`rounded-xl px-4 py-3 font-body-md text-body-md text-left transition-colors ${
                        answer === choice ? 'bg-primary text-on-primary' : 'bg-surface-container text-on-surface'
                      }`}
                    >
                      {choice}
                    </button>
                  ))}
                </div>
              )}

              <button
                onClick={submit}
                disabled={!answer || loading}
                className="bg-primary text-on-primary font-label-md text-label-md px-4 py-2.5 rounded-full disabled:opacity-40"
              >
                {'Check'}
              </button>
            </div>
          </div>
        )}
      </main>
      <BottomNav />
    </div>
  )
}
