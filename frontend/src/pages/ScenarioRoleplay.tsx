import { useCallback, useEffect, useRef, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import AppHeader from '../components/AppHeader'
import BottomNav from '../components/BottomNav'
import { learningAPI } from '../services/api'
import { useStore } from '../store'
import type { ScenarioRun, ScenarioTurn, ScenarioChunk, ScenarioScript, ScenarioError } from '@chorus/shared'

// Target-language composer placeholders (FR-34: roleplay prompts the learner in
// the target language). Falls back to a neutral label for unlisted languages.
const PLACEHOLDERS: Record<string, string> = {
  es: 'Escribe en español...',
  fr: 'Écrivez en français...',
  de: 'Schreiben Sie auf Deutsch...',
  it: 'Scrivi in italiano...',
  pt: 'Escreva em português...',
  ru: 'Напишите по-русски...',
  zh: '用中文写...',
  ja: '日本語で書いてください...',
  ko: '한국어로 작성하세요...',
  ar: 'اكتب بالعربية...',
  hi: 'हिंदी में लिखें...',
  ur: 'اردو میں لکھیں...',
  bn: 'বাংলায় লিখুন...',
}

export default function ScenarioRoleplay() {
  const { scenarioId } = useParams()
  const navigate = useNavigate()
  const user = useStore(s => s.user)
  const targetLanguage = user?.targetLanguages?.[0] ?? 'es'
  const nativeLanguage = user?.nativeLanguage ?? 'en'

  const [scenario, setScenario] = useState<ScenarioScript | null>(null)
  const [run, setRun] = useState<ScenarioRun | null>(null)
  const [turns, setTurns] = useState<ScenarioTurn[]>([])
  const [message, setMessage] = useState('')
  const [chunks, setChunks] = useState<ScenarioChunk[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(false)
  const [sending, setSending] = useState(false)
  const [done, setDone] = useState(false)
  const [summary, setSummary] = useState<any>(null)
  const [showHints, setShowHints] = useState(false)
  const [corrections, setCorrections] = useState<ScenarioError[]>([])
  const [phaseComplete, setPhaseComplete] = useState(false)
  const scrollRef = useRef<HTMLDivElement>(null)

  const currentPhase = run?.currentPhase
  const phaseOrdinal = run?.currentPhaseOrdinal ?? 1
  const totalPhases = scenario?.phases?.length ?? 0
  const placeholder = PLACEHOLDERS[targetLanguage] ?? `Write in ${targetLanguage}...`

  const load = useCallback(async () => {
    if (!scenarioId) return
    setLoading(true)
    setError(false)
    setCorrections([])
    try {
      // Fetch script metadata (for the header: title, AI role, phase count) and
      // open/continue a run in parallel; the script is optional so a missing
      // card never blocks the roleplay itself.
      const [script, started] = await Promise.all([
        learningAPI.getScenario(scenarioId).catch(() => null),
        learningAPI.startScenario(scenarioId, targetLanguage, nativeLanguage),
      ])
      if (script) setScenario(script)
      setRun(started.run)
      setChunks(started.aiResponse.suggestedChunks || [])
      setTurns([{ id: 'ai-0', runId: started.run.id, ordinal: 0, speaker: 'ai', text: started.aiResponse.aiMessage, translation: started.aiResponse.translation, phaseOrdinal: 1, createdAt: new Date().toISOString() }])
    } catch {
      setError(true)
    } finally {
      setLoading(false)
    }
  }, [scenarioId, targetLanguage, nativeLanguage])

  useEffect(() => {
    load()
  }, [load])

  useEffect(() => {
    scrollRef.current?.scrollTo?.({ top: scrollRef.current.scrollHeight, behavior: 'smooth' })
  }, [turns])

  const send = useCallback(async () => {
    const text = message.trim()
    if (!text || !run || sending) return
    setMessage('')

    // Optimistic: show the user's bubble immediately before the AI reply.
    setTurns(ts => [
      ...ts,
      { id: `u-${Date.now()}`, runId: run.id, ordinal: ts.length, speaker: 'user', text, translation: '', phaseOrdinal: run.currentPhaseOrdinal, createdAt: new Date().toISOString() },
    ])
    setSending(true)
    setCorrections([])
    try {
      const reply = await learningAPI.sendScenarioMessage(run.id, text)
      setTurns(ts => [
        ...ts,
        { id: `ai-${Date.now()}`, runId: run.id, ordinal: ts.length, speaker: 'ai', text: reply.aiMessage, translation: reply.translation, phaseOrdinal: reply.nextPhaseOrdinal || run.currentPhaseOrdinal, createdAt: new Date().toISOString() },
      ])
      setChunks(reply.suggestedChunks || [])
      setCorrections(reply.errors || [])
      setPhaseComplete(reply.phaseComplete)
      if (reply.runCompleted) {
        setDone(true)
        setSummary(reply.summary)
      }
    } catch {
      setMessage(text)
      alert('Could not send your message. Check the server and try again.')
    } finally {
      setSending(false)
    }
  }, [message, run, sending])

  const useChunk = useCallback((chunk: ScenarioChunk) => {
    setMessage(chunk.text)
    setShowHints(false)
  }, [])

  const finish = useCallback(async () => {
    if (!run || sending) return
    setSending(true)
    try {
      const reply = await learningAPI.completeScenario(run.id)
      setDone(true)
      setSummary(reply.summary)
    } catch {
      alert('Could not finish the scenario. Check the server and try again.')
    } finally {
      setSending(false)
    }
  }, [run, sending])

  const retry = () => {
    setLoading(false)
    setError(false)
    load()
  }

  const back = () => navigate('/learn/scenarios')

  return (
    <div className="h-screen flex flex-col bg-background">
      <AppHeader />
      <main className="flex-1 overflow-y-auto px-margin-mobile pt-4 pb-32 max-w-md w-full mx-auto flex flex-col">
        <div className="flex items-center justify-between mb-3">
          <button onClick={back} className="text-on-surface-variant font-label-md text-label-md">← Scenarios</button>
          {!done && (
            <button
              onClick={finish}
              disabled={!run || sending}
              className="font-label-sm text-label-sm text-primary disabled:opacity-40 flex items-center gap-1"
            >
              <span className="material-symbols-outlined text-[16px]">flag</span> {'Finish'}
            </button>
          )}
        </div>

        {loading && <p className="font-body-md text-body-md text-on-surface-variant mt-6">{'Starting roleplay...'}</p>}

        {!loading && error && (
          <div className="mt-6 bg-surface-container-lowest rounded-xl p-6 flex flex-col items-center gap-3 text-center">
            <span className="material-symbols-outlined text-error text-[40px]">error</span>
            <p className="font-body-md text-body-md text-on-surface-variant">{'Could not start the roleplay. Check the server and try again.'}</p>
            <button onClick={retry} className="bg-primary text-on-primary font-label-md text-label-md px-6 py-2.5 rounded-full">{'Try again'}</button>
          </div>
        )}

        {done && (
          <div className="mt-6 bg-surface-container-lowest rounded-2xl p-6 flex flex-col items-center gap-3 text-center">
            <span className="material-symbols-outlined text-success text-[44px]">celebration</span>
            <h2 className="font-headline-lg text-headline-lg text-on-surface">{'Scenario complete!'}</h2>
            <p className="font-body-md text-body-md text-on-surface-variant">
              {summary ? `Score ${summary.score} · +${summary.xpAwarded} XP · ${summary.vocabularyAdded} new words` : 'Great conversation!'}
            </p>
            <div className="flex gap-2 mt-2">
              <button onClick={back} className="bg-surface-container-high text-on-surface-variant font-label-md text-label-md px-6 py-2.5 rounded-full">{'More scenarios'}</button>
              <button onClick={() => navigate('/learn')} className="bg-primary text-on-primary font-label-md text-label-md px-6 py-2.5 rounded-full">{'Back to Learn'}</button>
            </div>
          </div>
        )}

        {!done && !loading && !error && run && (
          <>
            {/* Scenario intro card */}
            <div className="mb-3 bg-surface-container-lowest rounded-2xl p-4 shadow-[0px_4px_12px_rgba(0,0,0,0.05)] flex flex-col gap-2">
              <div className="flex items-center justify-between gap-2">
                <h2 className="font-headline-sm text-headline-sm text-on-surface truncate">{scenario?.title ?? 'Roleplay'}</h2>
                <span className="bg-secondary-fixed-dim/30 text-secondary font-label-xs text-label-sm px-2 py-0.5 rounded-full whitespace-nowrap">
                  {scenario?.cefrLevel ?? ''}
                </span>
              </div>
              {scenario?.canDoStatement && (
                <p className="font-body-sm text-body-sm text-on-surface-variant">{scenario.canDoStatement}</p>
              )}
              <div className="flex items-center gap-2 flex-wrap">
                {scenario?.aiRoleName && (
                  <span className="flex items-center gap-1 font-label-sm text-label-sm text-primary">
                    <span className="material-symbols-outlined text-[16px]">smart_toy</span>
                    {scenario.aiRoleName}
                  </span>
                )}
                <span className="font-label-sm text-label-sm text-on-surface-variant">
                  Phase {phaseOrdinal}{totalPhases ? ` of ${totalPhases}` : ''} · {run.scaffoldLevel}
                </span>
              </div>
            </div>

            {/* Current phase goal hint */}
            {currentPhase?.learnerGoal && (
              <div className="mb-3 bg-primary-fixed-dim/10 border border-primary-fixed-dim rounded-xl px-4 py-3 flex items-start gap-2">
                <span className="material-symbols-outlined text-primary text-[18px] mt-0.5">target</span>
                <div>
                  {currentPhase.title && <p className="font-label-sm text-label-sm text-primary">{currentPhase.title}</p>}
                  <p className="font-body-sm text-body-sm text-on-surface-variant">{currentPhase.learnerGoal}</p>
                </div>
              </div>
            )}

            <div ref={scrollRef} className="flex-1 flex flex-col gap-3 overflow-y-auto pr-1">
              {turns.map((t, i) => {
                const isLastTurn = i === turns.length - 1
                return (
                  <div key={t.id} className="flex flex-col gap-2">
                    <div className={`max-w-[85%] px-4 py-3 rounded-2xl ${t.speaker === 'ai' ? 'bg-surface-container-lowest self-start border-l-2 border-secondary-container' : 'bg-primary text-on-primary self-end'}`}>
                      <p className="font-body-md text-body-md">{t.text}</p>
                      {t.speaker === 'ai' && t.translation && (
                        <p className="font-label-sm text-label-sm text-outline mt-1 italic">{t.translation}</p>
                      )}
                    </div>
                    {t.speaker === 'ai' && isLastTurn && phaseComplete && (
                      <span className="self-start flex items-center gap-1 font-label-sm text-label-sm text-success px-2 py-0.5 rounded-full bg-success/10">
                        <span className="material-symbols-outlined text-[14px]">check_circle</span> {'Phase complete!'}
                      </span>
                    )}
                  </div>
                )
              })}
              {sending && (
                <div className="max-w-[85%] px-4 py-3 rounded-2xl bg-surface-container-lowest self-start border-l-2 border-secondary-container">
                  <p className="font-label-sm text-label-sm text-secondary italic">Sparky is typing…</p>
                </div>
              )}
            </div>

            {/* Gentle corrections from the last AI reply */}
            {corrections.length > 0 && (
              <div className="mt-3 bg-error-container/30 border border-error-container rounded-xl p-4 flex flex-col gap-2">
                <p className="font-label-sm text-label-sm text-error flex items-center gap-1">
                  <span className="material-symbols-outlined text-[16px]">school</span> Quick tips
                </p>
                {corrections.map((c, i) => (
                  <div key={i} className="flex flex-col gap-0.5">
                    {c.span && <p className="font-body-sm text-body-sm text-on-surface line-through decoration-error/60">{c.span}</p>}
                    {c.correction && <p className="font-body-sm text-body-sm text-on-surface font-medium">{'→ '}{c.correction}</p>}
                    {c.explanation && <p className="font-label-sm text-label-sm text-on-surface-variant">{c.explanation}</p>}
                  </div>
                ))}
              </div>
            )}

            {chunks.length > 0 && (
              <div className="mt-3">
                {!showHints ? (
                  <button onClick={() => setShowHints(true)} className="font-label-sm text-label-sm text-primary flex items-center gap-1">
                    <span className="material-symbols-outlined text-[16px]">lightbulb</span> {'Show suggestions'}
                  </button>
                ) : (
                  <div className="flex flex-wrap gap-2">
                    {chunks.map((c, i) => (
                      <button key={i} onClick={() => useChunk(c)} className="bg-surface-container rounded-full px-3 py-1.5 font-label-sm text-label-sm text-on-surface">
                        {c.text}
                      </button>
                    ))}
                  </div>
                )}
              </div>
            )}

            <div className="flex items-center gap-2 mt-3">
              <button onClick={() => run && learningAPI.requestScenarioHint(run.id).then(next => {
                setChunks(next || [])
                setShowHints(true)
              })} className="text-on-surface-variant" aria-label="Hint" title="Hint">
                <span className="material-symbols-outlined">lightbulb</span>
              </button>
              <input
                value={message}
                onChange={e => setMessage(e.target.value)}
                onKeyDown={e => e.key === 'Enter' && send()}
                placeholder={placeholder}
                aria-label="Your reply"
                className="flex-1 bg-surface-container rounded-full px-4 py-3 font-body-md text-body-md text-on-surface outline-none"
              />
              <button onClick={send} disabled={sending || !message.trim()} className="bg-primary text-on-primary rounded-full px-4 py-3 disabled:opacity-40" aria-label="Send">
                <span className="material-symbols-outlined">send</span>
              </button>
            </div>
          </>
        )}
      </main>
      <BottomNav />
    </div>
  )
}
