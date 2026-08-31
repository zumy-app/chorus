import { useCallback, useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import AppHeader from '../components/AppHeader'
import BottomNav from '../components/BottomNav'
import { learningAPI } from '../services/api'
import { useStore } from '../store'
import type { MinedItem } from '@chorus/shared'

export default function VocabularyReview() {
  const navigate = useNavigate()
  const user = useStore(s => s.user)
  const targetLanguage = user?.targetLanguages?.[0] ?? 'es'

  const [candidates, setCandidates] = useState<MinedItem[]>([])
  const [loading, setLoading] = useState(true)

  const load = useCallback(() => {
    setLoading(true)
    learningAPI
      .getMinedItems(targetLanguage, 'auto_added')
      .then(setCandidates)
      .catch(() => setCandidates([]))
      .finally(() => setLoading(false))
  }, [targetLanguage])

  useEffect(() => {
    load()
  }, [load])

  const accept = useCallback(async (id: string) => {
    await learningAPI.acceptMinedItem(id)
    setCandidates(cs => cs.filter(c => c.id !== id))
  }, [])

  const ignore = useCallback(async (id: string) => {
    await learningAPI.ignoreMinedItem(id)
    setCandidates(cs => cs.filter(c => c.id !== id))
  }, [])

  return (
    <div className="h-screen flex flex-col bg-background">
      <AppHeader />
      <main className="flex-1 overflow-y-auto px-margin-mobile pt-4 pb-32 max-w-md w-full mx-auto">
        <header className="flex items-center justify-between">
          <div>
            <h1 className="font-headline-lg text-headline-lg text-on-background">{'Vocabulary'}</h1>
            <p className="font-body-md text-body-md text-on-surface-variant">{'Words found in your chats'}</p>
          </div>
          <button onClick={() => navigate('/learn/session?mode=vocabulary')} className="bg-primary text-on-primary font-label-md text-label-md px-4 py-2 rounded-full">
            {'Practice'}
          </button>
        </header>

        {loading && <p className="font-body-md text-body-md text-on-surface-variant mt-6">{'Loading...'}</p>}
        {!loading && candidates.length === 0 && (
          <div className="mt-6 bg-surface-container-lowest rounded-xl p-6 text-center">
            <p className="font-body-md text-body-md text-on-surface-variant">{'No vocabulary yet. Chat in Spanish to discover words.'}</p>
          </div>
        )}

        <div className="flex flex-col gap-3 mt-4">
          {candidates.map(item => (
            <div key={item.id} className="bg-surface-container-lowest rounded-2xl p-4 shadow-[0px_4px_12px_rgba(0,0,0,0.05)] flex flex-col gap-1">
              <div className="flex items-start justify-between">
                <span className="font-headline-sm text-headline-sm text-on-surface">{item.surfaceText}</span>
                <span className={`font-label-xs text-label-sm px-2 py-0.5 rounded-full ${item.routeStatus === 'bonus' ? 'bg-surface-container-high text-on-surface-variant' : 'bg-primary-fixed-dim/30 text-primary'}`}>
                  {item.routeStatus}
                </span>
              </div>
              {item.translation && <span className="font-body-md text-body-md text-on-surface-variant">{item.translation}</span>}
              {item.contextSentence && <span className="font-label-sm text-label-sm text-outline mt-1">«{item.contextSentence}»</span>}
              <div className="flex gap-2 mt-2">
                <button onClick={() => accept(item.id)} className="bg-primary text-on-primary font-label-md text-label-md px-4 py-1.5 rounded-full">
                  {'Save'}
                </button>
                <button onClick={() => ignore(item.id)} className="bg-surface-container-high text-on-surface-variant font-label-md text-label-md px-4 py-1.5 rounded-full">
                  {'Dismiss'}
                </button>
              </div>
            </div>
          ))}
        </div>
      </main>
      <BottomNav />
    </div>
  )
}
