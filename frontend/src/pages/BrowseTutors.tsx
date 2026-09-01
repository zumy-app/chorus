import { useEffect, useState, useCallback } from 'react'
import { Link } from 'react-router-dom'
import AppHeader from '../components/AppHeader'
import BottomNav from '../components/BottomNav'
import { teacherAPI } from '../services/api'
import type { TutorProfile } from '@chorus/shared'

export default function BrowseTutors() {
  const [q, setQ] = useState('')
  const [tutors, setTutors] = useState<TutorProfile[]>([])
  const [loading, setLoading] = useState(true)
  const [msg, setMsg] = useState('')

  const load = useCallback(async () => {
    setLoading(true)
    setMsg('')
    try {
      const res = await teacherAPI.browse({ search: q || undefined, limit: 20 })
      setTutors(res.tutors)
    } catch (e: any) {
      setMsg(e?.response?.data?.error || 'Failed to load tutors')
    } finally { setLoading(false) }
  }, [q])

  useEffect(() => { load() }, [load])

  return (
    <div className="h-screen flex flex-col bg-background">
      <AppHeader />
      <main className="flex-1 overflow-y-auto px-4 py-6 space-y-6 pb-32 max-w-md w-full mx-auto">
        <div className="flex justify-between items-center">
          <h2 className="text-xl font-bold">Tutors</h2>
          <Link to="/become-teacher" className="text-sm text-primary font-medium">Become a teacher</Link>
        </div>
        <div className="flex gap-2">
          <input value={q} onChange={e => setQ(e.target.value)} onKeyDown={e => e.key === 'Enter' && load()} placeholder="Find a tutor or language..." className="flex-1 border rounded-full px-4 py-2 text-sm" data-testid="tutor-search" />
          <button onClick={load} className="bg-primary text-white rounded-full px-4 text-sm font-medium">Search</button>
        </div>
        {msg && <p className="text-sm text-error text-center">{msg}</p>}
        {loading ? <p className="text-sm text-center text-gray-500">Loading...</p> : tutors.length === 0 ? (
          <div className="border rounded-xl p-6 text-center space-y-2">
            <p className="font-medium">No tutors yet</p>
            <p className="text-sm text-gray-500">Try a different search.</p>
          </div>
        ) : (
          <div className="space-y-3">
            {tutors.map(t => (
              <Link key={t.userId} to={`/tutors/${t.userId}`} className="block border rounded-xl p-4 hover:border-primary transition-colors">
                <div className="flex gap-3">
                  <div className="w-12 h-12 rounded-full bg-secondary-container flex items-center justify-center text-on-secondary-container font-bold">{(t.displayName || t.userId).slice(0,1).toUpperCase()}</div>
                  <div className="flex-1">
                    <div className="flex justify-between items-center">
                      <p className="font-semibold text-sm">{t.displayName || t.userId}</p>
                      <span className="text-xs bg-surface-container rounded-full px-2 py-0.5">★ {(t.ratingAvg ?? 5).toFixed(1)}</span>
                    </div>
                    <p className="text-xs text-gray-500">{(t.languages||[]).join(' • ') || 'Tutor'} {t.verified ? '· Verified' : ''}</p>
                    <p className="text-xs text-primary font-medium mt-1">${Math.round((t.rateCents??1800)/100)}/session</p>
                  </div>
                </div>
              </Link>
            ))}
            <div className="flex gap-2 justify-center pt-2">
              <Link to="/trial-credits" className="text-sm text-primary">Trial credits</Link>
              <span className="text-gray-300">·</span>
              <Link to="/teacher/dashboard" className="text-sm text-primary">Teacher dashboard</Link>
              <span className="text-gray-300">·</span>
              <Link to="/teacher/payouts" className="text-sm text-primary">Payouts</Link>
            </div>
          </div>
        )}
      </main>
      <BottomNav />
    </div>
  )
}
