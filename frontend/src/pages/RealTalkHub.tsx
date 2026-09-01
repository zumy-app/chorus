import { useCallback, useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import AppHeader from '../components/AppHeader'
import BottomNav from '../components/BottomNav'
import StudySandbox from '../components/StudySandbox'
import { learningAPI } from '../services/api'
import { useStore } from '../store'
import type { RealTalkPrompt } from '@chorus/shared'

const CATEGORIES = ['All', 'Icebreakers', 'Deep Dives', 'Task-Based'] as const
type Tab = typeof CATEGORIES[number]

export default function RealTalkHub() {
  const navigate = useNavigate()
  const user = useStore(s => s.user)
  const targetLanguage = user?.targetLanguages?.[0] ?? 'es'
  const nativeLanguage = user?.nativeLanguage ?? 'en'
  const [prompts, setPrompts] = useState<RealTalkPrompt[]>([])
  const [loading, setLoading] = useState(true)
  const [tab, setTab] = useState<Tab>('All')
  const [dashboard, setDashboard] = useState<any>(null)
  const [usedId, setUsedId] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const [p, d] = await Promise.all([
        learningAPI.getRealTalkPrompts(targetLanguage, nativeLanguage).catch(() => [] as RealTalkPrompt[]),
        learningAPI.getDashboard(targetLanguage, nativeLanguage).catch(() => null),
      ])
      setPrompts(p)
      setDashboard(d)
    } finally {
      setLoading(false)
    }
  }, [targetLanguage, nativeLanguage])

  useEffect(() => { load() }, [load])

  const filtered = useMemo(() => {
    if (tab === 'All') return prompts
    return prompts.filter(p => p.category === tab)
  }, [prompts, tab])

  const currentUnit = dashboard?.currentUnit
  const useInChat = useCallback(async (prompt: RealTalkPrompt) => {
    setUsedId(prompt.id)
    try { await learningAPI.markRealTalkUsed(prompt.id) } catch {}
    localStorage.setItem('realTalkDraft', prompt.text)
    localStorage.setItem('realTalkPromptId', prompt.id)
    navigate('/chat')
  }, [navigate])

  return (
    <div className="h-screen flex flex-col bg-background">
      <AppHeader />
      <main className="flex-1 overflow-y-auto px-margin-mobile pt-4 pb-32 max-w-md w-full mx-auto flex flex-col gap-4">
        <header>
          <h1 className="font-headline-lg text-headline-lg text-on-surface">Real Talk Starters</h1>
          <p className="font-body-sm text-body-sm text-on-surface-variant">Practice real conversations — tap to drop a prompt into chat</p>
        </header>

        {currentUnit && (
          <div className="bg-surface-container-high rounded-xl p-4 border-l-4 border-primary relative overflow-hidden shadow-sm ring-1 ring-primary/10">
            <div className="flex items-center gap-2 mb-1">
              <span className="bg-primary/10 text-primary font-label-sm text-label-sm px-2 py-1 rounded-full">{currentUnit.cefrLevel}</span>
              <span className="text-on-surface-variant font-label-sm text-label-sm uppercase tracking-wider">Current Goal</span>
            </div>
            <h3 className="font-headline-sm text-headline-sm text-primary">{currentUnit.title}</h3>
            <p className="font-body-sm text-body-sm text-on-surface-variant flex items-start gap-2 mt-1">
              <span className="material-symbols-outlined text-tertiary-container text-lg">check_circle</span>
              {currentUnit.canDoStatement}
            </p>
          </div>
        )}

        <div className="flex overflow-x-auto gap-2 pb-1 -mx-margin-mobile px-margin-mobile">
          {CATEGORIES.map(c => (
            <button
              key={c}
              onClick={() => setTab(c)}
              className={`whitespace-nowrap px-5 py-2 rounded-full font-label-md text-label-md transition ${tab === c ? 'bg-primary text-on-primary shadow-sm' : 'bg-surface-container text-on-surface-variant hover:bg-surface-container-high'}`}
            >
              {c}
            </button>
          ))}
        </div>

        {loading && <p className="font-body-md text-body-md text-on-surface-variant mt-2">Loading prompts…</p>}
        {!loading && filtered.length === 0 && (
          <div className="mt-2 bg-surface-container-lowest rounded-xl p-6 text-center">
            <p className="font-body-md text-body-md text-on-surface-variant">No prompts in this category yet.</p>
            <button onClick={() => setTab('All')} className="mt-3 text-primary font-label-md text-label-md">Show all</button>
          </div>
        )}

        <div className="flex flex-col gap-3">
          {filtered.map(prompt => (
            <div key={prompt.id} className="bg-surface-container-lowest rounded-xl p-4 shadow-sm border border-surface-variant flex flex-col gap-3 hover:shadow-md transition-shadow">
              <div className="flex items-start justify-between gap-2">
                <span className="bg-secondary-fixed text-on-secondary-fixed font-label-sm text-label-sm px-2 py-1 rounded-full flex items-center gap-1">
                  <span className="material-symbols-outlined text-[14px]">psychology</span> {prompt.category}
                </span>
                {usedId === prompt.id && <span className="font-label-sm text-label-sm text-tertiary flex items-center gap-1"><span className="material-symbols-outlined text-[14px]">check</span> Used</span>}
              </div>
              <p className="font-body-lg text-body-lg text-on-surface leading-relaxed">“{prompt.text}”</p>
              <div className="flex items-center justify-between pt-2 border-t border-surface-variant/50">
                <span className="font-body-sm text-body-sm text-outline flex items-center gap-1"><span className="material-symbols-outlined text-[16px]">lightbulb</span> Try using target language</span>
                <button onClick={() => useInChat(prompt)} className="bg-primary text-on-primary font-label-md text-label-md px-6 py-2 rounded-full hover:bg-primary/90 transition active:scale-95 flex items-center gap-2">
                  Use in Chat <span className="material-symbols-outlined text-[18px]">send</span>
                </button>
              </div>
            </div>
          ))}
        </div>

        <StudySandbox />
      </main>
      <BottomNav />
    </div>
  )
}
