import { useEffect, useState } from 'react'
import { useParams, Link, useNavigate } from 'react-router-dom'
import AppHeader from '../components/AppHeader'
import BottomNav from '../components/BottomNav'
import { teacherAPI } from '../services/api'
import type { TutorProfile, TutorReview } from '@chorus/shared'

export default function TutorProfilePage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [tutor, setTutor] = useState<TutorProfile | null>(null)
  const [reviews, setReviews] = useState<TutorReview[]>([])
  const [loading, setLoading] = useState(true)
  const [msg] = useState('')

  useEffect(() => {
    if (!id) return
    setLoading(true)
    teacherAPI.getProfile(id).then(t => {
      setTutor(t)
      teacherAPI.getReviews(id, { limit: 5 }).then(r => setReviews(r.reviews)).catch(() => {})
    }).catch(() => setTutor(null)).finally(() => setLoading(false))
  }, [id])

  const openConfirm = () => {
    if (!id) return
    navigate(`/tutors/${id}/confirm`)
  }

  if (loading) return <div className="h-screen flex flex-col bg-background"><AppHeader /><main className="flex-1 flex items-center justify-center"><p className="text-sm text-gray-500">Loading...</p></main><BottomNav /></div>
  if (!tutor) return <div className="h-screen flex flex-col bg-background"><AppHeader /><main className="flex-1 flex items-center justify-center flex-col gap-2"><p>Tutor not found</p><button onClick={() => navigate('/tutors')} className="text-primary text-sm">Back</button></main><BottomNav /></div>

  return (
    <div className="h-screen flex flex-col bg-background">
      <AppHeader />
      <main className="flex-1 overflow-y-auto px-4 py-6 space-y-6 pb-32 max-w-md w-full mx-auto">
        <button onClick={() => navigate(-1)} className="text-sm text-primary">← Back</button>
        <div className="flex gap-4 items-center">
          <div className="w-16 h-16 rounded-full bg-secondary-container flex items-center justify-center text-xl font-bold text-on-secondary-container">{(tutor.displayName||id||'T').slice(0,1).toUpperCase()}</div>
          <div>
            <div className="flex items-center gap-2"><h2 className="text-lg font-bold">{tutor.displayName}</h2>{tutor.verified && <span className="text-xs bg-primary-container text-on-primary-container rounded-full px-2 py-0.5">Verified</span>}</div>
            <p className="text-sm text-gray-500">{(tutor.languages||[]).join(' • ')}</p>
            <p className="text-xs text-gray-500">★ {(tutor.ratingAvg??5).toFixed(1)} · {tutor.ratingCount} reviews · ${Math.round((tutor.rateCents??1800)/100)}/session</p>
          </div>
        </div>
        <div className="border rounded-xl p-4 space-y-2">
          <h3 className="font-semibold text-sm">About</h3>
          <p className="text-sm text-gray-600">{tutor.bio || 'No bio yet.'}</p>
          {tutor.expertise && <p className="text-xs text-gray-500">Expertise: {tutor.expertise}</p>}
        </div>
        {reviews.length>0 && <div className="border rounded-xl p-4 space-y-2"><h3 className="font-semibold text-sm">Reviews</h3>{reviews.map(r=> <div key={r.id} className="border-t pt-2"><p className="text-xs font-medium">★ {r.rating} {r.studentName?`· ${r.studentName}`:''}</p><p className="text-sm text-gray-600">{r.comment}</p></div>)}</div>}
        {msg && <p className="text-sm text-center text-primary">{msg}</p>}
        <div className="flex gap-3">
          <Link to="/tutors" className="flex-1 border border-primary text-primary rounded-full py-3 text-center text-sm font-medium">Browse</Link>
          <button onClick={openConfirm} className="flex-1 bg-primary text-white rounded-full py-3 text-sm font-medium" data-testid="book-trial">Book Trial</button>
        </div>
      </main>
      <BottomNav />
    </div>
  )
}
