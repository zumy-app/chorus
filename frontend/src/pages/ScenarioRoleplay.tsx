import { useCallback, useEffect, useRef, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import AppHeader from '../components/AppHeader'
import BottomNav from '../components/BottomNav'
import { learningAPI } from '../services/api'
import { useStore } from '../store'
import type { ScenarioRun, ScenarioTurn, ScenarioChunk } from '@chorus/shared'

export default function ScenarioRoleplay() {
  const { scenarioId } = useParams()
  const navigate = useNavigate()
  const user = useStore(s => s.user)
  const targetLanguage = user?.targetLanguages?.[0] ?? 'es'
  const nativeLanguage = user?.nativeLanguage ?? 'en'

  const [run, setRun] = useState<ScenarioRun | null>(null)
  const [turns, setTurns] = useState<ScenarioTurn[]>([])
  const [message, setMessage] = useState('')
  const [chunks, setChunks] = useState<ScenarioChunk[]>([])
  const [loading, setLoading] = useState(true)
  const [sending, setSending] = useState(false)
  const [done, setDone] = useState(false)
  const [summary, setSummary] = useState<any>(null)
  const [showHints, setShowHints] = useState(false)
  const scrollRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!scenarioId) return
    setLoading(true)
    learningAPI
      .startScenario(scenarioId, targetLanguage, nativeLanguage)
      .then(res => {
        setRun(res.run)
        setChunks(res.aiResponse.suggestedChunks || [])
        setTurns([{ id: 'ai-0', runId: res.run.id, ordinal: 0, speaker: 'ai', text: res.aiResponse.aiMessage, translation: res.aiResponse.translation, phaseOrdinal: 1, createdAt: new Date().toISOString() }])
      })
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [scenarioId, targetLanguage, nativeLanguage])

  useEffect(() => {
    scrollRef.current?.scrollTo({ top: scrollRef.current.scrollHeight, behavior: 'smooth' })
  }, [turns])

  const send = useCallback(async () => {
    if (!message.trim() || !run) return
    setSending(true)
    const reply = await learningAPI.sendScenarioMessage(run.id, message.trim())
    setTurns(ts => [
      ...ts,
      { id: `u-${Date.now()}`, runId: run.id, ordinal: ts.length, speaker: 'user', text: message.trim(), translation: '', phaseOrdinal: run.currentPhaseOrdinal, createdAt: new Date().toISOString() },
      { id: `ai-${Date.now()}`, runId: run.id, ordinal: ts.length + 1, speaker: 'ai', text: reply.aiMessage, translation: reply.translation, phaseOrdinal: reply.nextPhaseOrdinal || run.currentPhaseOrdinal, createdAt: new Date().toISOString() },
    ])
    setMessage('')
    setChunks(reply.suggestedChunks || [])
    if (reply.runCompleted) {
      setDone(true)
      setSummary(reply.summary)
    }
    setSending(false)
  }, [message, run])

  const useChunk = useCallback((chunk: ScenarioChunk) => {
    setMessage(chunk.text)
    setShowHints(false)
  }, [])

  return (
    <div className="h-screen flex flex-col bg-background">
      <AppHeader />
      <main className="flex-1 overflow-y-auto px-margin-mobile pt-4 pb-32 max-w-md w-full mx-auto flex flex-col">
        <div className="flex items-center justify-between mb-3">
          <button onClick={() => navigate('/learn/scenarios')} className="text-on-surface-variant font-label-md text-label-md">← Scenarios</button>
          {run && <span className="font-label-sm text-label-sm text-on-surface-variant">{run.scaffoldLevel} · Phase {run.currentPhaseOrdinal}</span>}
        </div>

        {loading && <p className="font-body-md text-body-md text-on-surface-variant mt-6">{'Starting roleplay...'}</p>}

        {done && (
          <div className="mt-6 bg-surface-container-lowest rounded-2xl p-6 flex flex-col items-center gap-3 text-center">
            <span className="material-symbols-outlined text-success text-[44px]">celebration</span>
            <h2 className="font-headline-lg text-headline-lg text-on-surface">{'Scenario complete!'}</h2>
            <p className="font-body-md text-body-md text-on-surface-variant">
              {summary ? `Score ${summary.score} · +${summary.xpAwarded} XP · ${summary.vocabularyAdded} new words` : 'Great conversation!'}
            </p>
            <button onClick={() => navigate('/learn')} className="bg-primary text-on-primary font-label-md text-label-md px-6 py-2.5 rounded-full mt-2">{'Back to Learn'}</button>
          </div>
        )}

        {!done && (
          <>
            <div ref={scrollRef} className="flex-1 flex flex-col gap-3 overflow-y-auto pr-1">
              {turns.map(t => (
                <div key={t.id} className={`max-w-[85%] px-4 py-3 rounded-2xl ${t.speaker === 'ai' ? 'bg-surface-container-lowest self-start' : 'bg-primary text-on-primary self-end'}`}>
                  <p className="font-body-md text-body-md">{t.text}</p>
                  {t.speaker === 'ai' && t.translation && (
                    <p className="font-label-sm text-label-sm text-outline mt-1">{t.translation}</p>
                  )}
                </div>
              ))}
            </div>

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
              <button onClick={() => run && learningAPI.requestScenarioHint(run.id).then(setChunks)} className="text-on-surface-variant">
                <span className="material-symbols-outlined">lightbulb</span>
              </button>
              <input
                value={message}
                onChange={e => setMessage(e.target.value)}
                onKeyDown={e => e.key === 'Enter' && send()}
                placeholder="Escribe en español..."
                className="flex-1 bg-surface-container rounded-full px-4 py-3 font-body-md text-body-md text-on-surface outline-none"
              />
              <button onClick={send} disabled={sending || !message.trim()} className="bg-primary text-on-primary rounded-full px-4 py-3 disabled:opacity-40">
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
