import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import AppHeader from '../components/AppHeader'
import BottomNav from '../components/BottomNav'
import { learningAPI } from '../services/api'
import { useStore } from '../store'
import type { ScenarioScript } from '@chorus/shared'

export default function Scenarios() {
  const navigate = useNavigate()
  const user = useStore(s => s.user)
  const targetLanguage = user?.targetLanguages?.[0] ?? 'es'
  const nativeLanguage = user?.nativeLanguage ?? 'en'
  const [scenarios, setScenarios] = useState<ScenarioScript[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    learningAPI
      .getScenarios(targetLanguage, nativeLanguage)
      .then(setScenarios)
      .catch(() => setScenarios([]))
      .finally(() => setLoading(false))
  }, [targetLanguage, nativeLanguage])

  return (
    <div className="h-screen flex flex-col bg-background">
      <AppHeader />
      <main className="flex-1 overflow-y-auto px-margin-mobile pt-4 pb-32 max-w-md w-full mx-auto">
        <header className="flex items-center justify-between">
          <div>
            <h1 className="font-headline-lg text-headline-lg text-on-background">{'Real-World Scenarios'}</h1>
            <p className="font-body-md text-body-md text-on-surface-variant">{'Practice real conversations with AI'}</p>
          </div>
        </header>

        {loading && <p className="font-body-md text-body-md text-on-surface-variant mt-6">{'Loading scenarios...'}</p>}
        {!loading && scenarios.length === 0 && (
          <div className="mt-6 bg-surface-container-lowest rounded-xl p-6 text-center">
            <p className="font-body-md text-body-md text-on-surface-variant">{'No scenarios available yet.'}</p>
          </div>
        )}

        <div className="flex flex-col gap-3 mt-4">
          {scenarios.map(sc => {
            const done = sc.metadata?.completed === true
            return (
              <button
                key={sc.id}
                onClick={() => navigate(`/learn/scenarios/${sc.id}`)}
                className="bg-surface-container-lowest rounded-2xl p-5 shadow-[0px_4px_12px_rgba(0,0,0,0.05)] flex flex-col gap-2 text-left hover:border-primary-fixed-dim transition-colors"
              >
                <div className="flex items-center justify-between">
                  <span className="font-headline-sm text-headline-sm text-on-surface">{sc.title}</span>
                  {done && <span className="material-symbols-outlined text-success">check_circle</span>}
                </div>
                <span className="font-label-sm text-label-sm text-on-surface-variant">{sc.canDoStatement}</span>
                <div className="flex items-center gap-2 mt-1">
                  <span className="bg-secondary-fixed-dim/30 text-secondary font-label-xs text-label-sm px-2 py-0.5 rounded-full">{sc.cefrLevel}</span>
                  <span className="font-label-sm text-label-sm text-outline">{`${sc.estimatedMinutes} min`}</span>
                  <span className="font-label-sm text-label-sm text-outline">· {sc.domain}</span>
                </div>
              </button>
            )
          })}
        </div>
      </main>
      <BottomNav />
    </div>
  )
}
