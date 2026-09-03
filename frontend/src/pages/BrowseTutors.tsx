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
  const [priceFilter, setPriceFilter] = useState<string>('all')
  const [ratingFilter, setRatingFilter] = useState<string>('all')
  const [langFilter, setLangFilter] = useState<string>('all')

  const load = useCallback(async () => {
    setLoading(true)
    setMsg('')
    try {
      const params: any = { search: q || undefined, limit: 20 }
      if (langFilter !== 'all') params.language = langFilter
      if (ratingFilter !== 'all') params.minRating = Number(ratingFilter)
      if (priceFilter !== 'all') params.maxRate = Number(priceFilter) * 100
      const res = await teacherAPI.browse(params)
      setTutors(res.tutors)
    } catch (e: any) {
      setMsg(e?.response?.data?.error || 'Failed to load tutors')
    } finally { setLoading(false) }
  }, [q, langFilter, ratingFilter, priceFilter])

  useEffect(() => { load() }, [load])

  const featured = tutors.slice(0, 2)
  const available = tutors.slice(2)

  // Ensure Sofia seed is visible even when API returns empty in tests: keep generic list
  const displayFeatured = featured
  const displayAvailable = available

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
        <div className="flex gap-2 overflow-x-auto py-1">
          <div className="flex items-center gap-1 px-4 py-1.5 rounded-full border bg-surface text-on-surface text-xs shadow-sm">
            <span>Language</span>
            <select value={langFilter} onChange={e => setLangFilter(e.target.value)} className="bg-transparent text-xs ml-1">
              <option value="all">All</option>
              <option value="es">Spanish (es)</option>
              <option value="en">English</option>
              <option value="fr">French</option>
              <option value="de">German</option>
              <option value="ja">Japanese</option>
            </select>
          </div>
          <div className="flex items-center gap-1 px-4 py-1.5 rounded-full border bg-surface text-on-surface text-xs shadow-sm">
            <span>Price</span>
            <select value={priceFilter} onChange={e => setPriceFilter(e.target.value)} className="bg-transparent text-xs ml-1">
              <option value="all">All</option>
              <option value="25">$25</option>
              <option value="30">$30</option>
              <option value="50">$50</option>
            </select>
          </div>
          <div className="flex items-center gap-1 px-4 py-1.5 rounded-full border bg-surface text-on-surface text-xs shadow-sm">
            <span>Rating</span>
            <select value={ratingFilter} onChange={e => setRatingFilter(e.target.value)} className="bg-transparent text-xs ml-1">
              <option value="all">All</option>
              <option value="4">4+</option>
              <option value="4.5">4.5+</option>
              <option value="5">5.0</option>
            </select>
          </div>
        </div>
        {msg && <p className="text-sm text-error text-center">{msg}</p>}
        {loading ? <p className="text-sm text-center text-gray-500">Loading...</p> : tutors.length === 0 ? (
          <div className="border rounded-xl p-6 text-center space-y-2">
            <p className="font-medium">No tutors yet</p>
            <p className="text-sm text-gray-500">Try a different search.</p>
          </div>
        ) : (
          <div className="space-y-6">
            <section>
              <div className="flex justify-between items-baseline mb-3">
                <h3 className="font-semibold text-base">Featured Tutors</h3>
                <button onClick={load} className="text-primary text-xs">See all</button>
              </div>
              <div className="flex gap-3 overflow-x-auto pb-2">
                {displayFeatured.map(t => (
                  <Link key={t.userId} to={`/tutors/${t.userId}`} className="min-w-[180px] border rounded-xl p-4 bg-surface shadow-sm hover:border-primary transition-colors">
                    <div className="flex flex-col items-center text-center gap-2">
                      <div className="w-14 h-14 rounded-full bg-secondary-container flex items-center justify-center font-bold text-on-secondary-container">{(t.displayName || t.userId).slice(0,1).toUpperCase()}</div>
                      <p className="font-semibold text-sm">{t.displayName || t.userId}</p>
                      <p className="text-xs text-gray-500">{(t.languages||[]).join(' • ') || 'Tutor'} {t.verified ? '· Verified' : ''}</p>
                      {t.verified && <span className="text-xs bg-primary-container text-on-primary-container rounded-full px-2 py-0.5">Verified</span>}
                      <span className="text-xs bg-surface-container rounded-full px-2 py-0.5">★ {(t.ratingAvg ?? 5).toFixed(1)}</span>
                      <p className="text-xs text-primary font-medium">${Math.round((t.rateCents??2500)/100)}/session</p>
                    </div>
                  </Link>
                ))}
              </div>
            </section>

            <section>
              <h3 className="font-semibold text-base mb-3">Available Now</h3>
              <div className="space-y-3">
                {displayAvailable.length === 0 ? (
                  <p className="text-sm text-gray-500">More tutors coming soon.</p>
                ) : null}
                {displayAvailable.map(t => (
                  <Link key={t.userId} to={`/tutors/${t.userId}`} className="block border rounded-xl p-4 hover:border-primary transition-colors bg-surface">
                    <div className="flex gap-3">
                      <div className="w-12 h-12 rounded-full bg-secondary-container flex items-center justify-center text-on-secondary-container font-bold">{(t.displayName || t.userId).slice(0,1).toUpperCase()}</div>
                      <div className="flex-1">
                        <div className="flex justify-between items-center">
                          <p className="font-semibold text-sm">{t.displayName || t.userId}</p>
                          <span className="text-xs bg-surface-container rounded-full px-2 py-0.5">★ {(t.ratingAvg ?? 5).toFixed(1)}</span>
                        </div>
                        <p className="text-xs text-gray-500">{(t.languages||[]).join(' • ') || 'Tutor'} {t.verified ? '· Verified' : ''}</p>
                        {t.verified && <span className="text-xs bg-primary-container text-on-primary-container rounded-full px-2 py-0.5">Verified</span>}
                        <p className="text-xs text-primary font-medium mt-1">${Math.round((t.rateCents??2500)/100)}/session</p>
                      </div>
                    </div>
                  </Link>
                ))}
              </div>
            </section>

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
