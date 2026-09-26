import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import AppHeader from '../components/AppHeader'
import BottomNav from '../components/BottomNav'
import { teacherAPI } from '../services/api'
import type { TutorProfile } from '@chorus/shared'

export default function TrialCredits() {
  const navigate = useNavigate()
  const [data, setData] = useState<any>(null)
  const [loading, setLoading] = useState(true)
  const [err, setErr] = useState('')
  const [tutors, setTutors] = useState<TutorProfile[]>([])

  useEffect(() => {
    let active = true
    fetch('/api/v1/teachers/trial-credits/dashboard', { headers: { Authorization: `Bearer ${localStorage.getItem('accessToken')}` } })
      .then(r => r.json())
      .then(j => { if (active) setData(j.dashboard ?? j) })
      .catch(e => active && setErr(e.message))
      .finally(() => active && setLoading(false))
    teacherAPI.browse({ limit: 2 }).then(r => active && setTutors(r.tutors.slice(0, 2))).catch(() => {})
    return () => { active = false }
  }, [])

  const credits = data?.credits ?? data?.trialCredits?.credits ?? 0
  const nextGrant = data?.nextGrantAt ? new Date(data.nextGrantAt).toLocaleDateString() : null
  const history = data?.history ?? []

  return (
    <div className="h-screen flex flex-col bg-background">
      <AppHeader />
      <main className="flex-1 overflow-y-auto px-4 py-6 space-y-6 pb-32 max-w-3xl w-full mx-auto">
        {loading ? <p className="text-sm text-gray-500">Loading...</p> : err ? <p className="text-sm text-error">{err}</p> : (
          <>
            <div className="bg-surface-container-lowest rounded-xl p-6 shadow-[0px_8px_24px_rgba(0,0,0,0.08)] text-center relative overflow-hidden">
              <div className="absolute -right-12 -top-12 w-48 h-48 bg-primary/5 rounded-full blur-3xl" />
              <div className="relative z-10 flex flex-col items-center">
                <span className="material-symbols-outlined text-4xl text-primary mb-2" style={{ fontVariationSettings: "'FILL' 1" }}>star</span>
                <h2 className="text-sm text-on-surface-variant">Trial Credits</h2>
                <div className="text-[48px] leading-none font-bold tracking-tight">{credits}</div>
                <p className="text-sm text-outline mt-1">Available to use right now</p>
                {nextGrant && <p className="text-xs text-gray-500 mt-1">Next credit: {nextGrant}</p>}
                <button onClick={() => navigate('/tutors')} className="bg-primary text-white w-full py-3 rounded-xl text-sm font-semibold mt-6 flex items-center justify-center gap-2">
                  <span className="material-symbols-outlined text-lg">search</span> Find a Tutor
                </button>
              </div>
            </div>

            <section>
              <h3 className="font-semibold text-on-surface mb-3">How Trials Work</h3>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                <div className="bg-surface-container-lowest rounded-xl p-4 flex gap-4 shadow-sm">
                  <div className="w-10 h-10 rounded-full bg-surface-container flex items-center justify-center text-primary shrink-0">⏱</div>
                  <div><h4 className="text-sm font-semibold">20 Minutes</h4><p className="text-xs text-gray-500">A focused, 1-on-1 session to assess your level and teaching style fit.</p></div>
                </div>
                <div className="bg-surface-container-lowest rounded-xl p-4 flex gap-4 shadow-sm">
                  <div className="w-10 h-10 rounded-full bg-secondary-container/20 flex items-center justify-center text-secondary shrink-0">🤝</div>
                  <div><h4 className="text-sm font-semibold">Meet & Greet</h4><p className="text-xs text-gray-500">No pressure. Just a casual chat to see if the tutor is the right match.</p></div>
                </div>
              </div>
            </section>

            <section>
              <div className="flex justify-between items-end mb-3">
                <h3 className="font-semibold">Recommended for Trials</h3>
                <Link to="/tutors" className="text-primary text-sm">See all</Link>
              </div>
              <div className="space-y-3">
                {tutors.length === 0 ? <p className="text-sm text-gray-500">No tutors yet.</p> : tutors.map(t => (
                  <div key={t.userId} onClick={() => navigate(`/tutors/${t.userId}`)} className="bg-surface-container-lowest rounded-xl p-4 flex gap-4 cursor-pointer hover:bg-surface-container-low shadow-sm">
                    <div className="w-12 h-12 rounded-xl bg-surface-container flex items-center justify-center font-bold">{(t.displayName || t.userId).slice(0, 1).toUpperCase()}</div>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2"><p className="font-semibold text-sm truncate">{t.displayName}</p><span className="text-xs bg-surface-container px-2 py-0.5 rounded">★ {(t.ratingAvg ?? 4.9).toFixed(1)}</span></div>
                      <p className="text-xs text-gray-500 truncate">{(t.languages || []).slice(0, 2).join(' • ')} • Beginner to Advanced</p>
                    </div>
                    <button onClick={(e) => { e.stopPropagation(); navigate(`/tutors/${t.userId}/confirm`) }} className="bg-primary/10 text-primary px-3 py-1.5 rounded-lg text-xs font-semibold shrink-0">Book Trial</button>
                  </div>
                ))}
              </div>
            </section>

            <div className="border rounded-xl p-4 space-y-3 bg-surface-container-lowest">
              <h3 className="font-semibold text-sm">History</h3>
              {history.length === 0 ? <p className="text-sm text-gray-500">No trial bookings yet.</p> : history.map((b:any)=> (
                <div key={b.id} className="border-t pt-2 text-sm flex justify-between">
                  <span>{new Date(b.startTime).toLocaleDateString()} · {b.isTrial ? 'Trial' : 'Paid'} · {b.status}</span>
                  <span className="text-xs text-gray-500">{b.teacherName || b.teacherUserId?.slice(0,6)}</span>
                </div>
              ))}
            </div>
          </>
        )}
      </main>
      <BottomNav />
    </div>
  )
}
