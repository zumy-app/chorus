import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import AppHeader from '../components/AppHeader'
import BottomNav from '../components/BottomNav'
import { learningAPI } from '../services/api'
import { useStore } from '../store'
import type { LearningPath, UnitProgressSummary } from '@chorus/shared'

export default function LearningRoadmap() {
  const navigate = useNavigate()
  const user = useStore(s => s.user)
  const targetLanguage = user?.targetLanguages?.[0] ?? 'es'
  const nativeLanguage = user?.nativeLanguage ?? 'en'
  const [path, setPath] = useState<LearningPath | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(false)

  useEffect(() => {
    learningAPI
      .getPath(targetLanguage, nativeLanguage)
      .then(setPath)
      .catch(() => setError(true))
      .finally(() => setLoading(false))
  }, [targetLanguage, nativeLanguage])

  const levelGroup = (level: string) => path?.units?.filter(u => u.cefrLevel === level) || []

  return (
    <div className="h-screen flex flex-col bg-background">
      <AppHeader />
      <main className="flex-1 overflow-y-auto px-margin-mobile pt-4 pb-32 max-w-md w-full mx-auto">
        <h1 className="font-headline-lg text-headline-lg text-on-background">{'Your Roadmap'}</h1>
        <p className="font-body-md text-body-md text-on-surface-variant mb-4">{'Progress through A1 to B2'}</p>

        {loading && <p className="font-body-md text-body-md text-on-surface-variant">{'Loading roadmap...'}</p>}
        {!loading && error && <p className="font-body-md text-body-md text-error">{'Could not load your roadmap.'}</p>}

        {!loading && !error && path && (
          <div className="flex flex-col gap-6">
            {['A1', 'A2', 'B1', 'B2'].map(level => {
              const units = levelGroup(level)
              if (units.length === 0) return null
              return (
                <section key={level} className="flex flex-col gap-2">
                  <header className="flex items-center gap-2">
                    <span className="bg-primary-fixed-dim/30 text-primary font-label-md text-label-md px-3 py-1 rounded-full">{level}</span>
                    <div className="flex-1 h-px bg-surface-container-high" />
                  </header>
                  {units.map(u => (
                    <UnitRow
                      key={u.id}
                      unit={u}
                      onOpen={() => navigate(u.status === 'completed' ? '/learn/session?mode=vocabulary' : '/learn/session?mode=lesson')}
                    />
                  ))}
                </section>
              )
            })}
          </div>
        )}
      </main>
      <BottomNav />
    </div>
  )
}

function UnitRow({ unit, onOpen }: { unit: UnitProgressSummary; onOpen: () => void }) {
  const completed = unit.status === 'completed'
  const available = unit.status === 'available' || unit.status === 'in_progress'
  const locked = unit.status === 'locked'
  return (
    <button
      onClick={onOpen}
      disabled={locked}
      className={`bg-surface-container-lowest rounded-2xl p-4 shadow-[0px_4px_12px_rgba(0,0,0,0.05)] flex items-center gap-3 text-left ${
        locked ? 'opacity-50' : 'hover:border-primary-fixed-dim transition-colors'
      }`}
    >
      <span className={`w-9 h-9 rounded-full flex items-center justify-center ${completed ? 'bg-success-container/20 text-success' : available ? 'bg-primary/10 text-primary' : 'bg-surface-container-high text-outline'}`}>
        <span className="material-symbols-outlined text-[20px]">{completed ? 'check' : locked ? 'lock' : unit.ordinal}</span>
      </span>
      <div className="flex-1">
        <div className="flex items-center justify-between">
          <span className="font-headline-sm text-headline-sm text-on-surface">{unit.title}</span>
          {unit.progressPct > 0 && <span className="font-label-sm text-label-sm text-primary">{unit.progressPct}%</span>}
        </div>
        <p className="font-label-sm text-label-sm text-on-surface-variant mt-0.5">{unit.canDoStatement}</p>
      </div>
    </button>
  )
}
