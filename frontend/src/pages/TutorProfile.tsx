import { useEffect, useState } from 'react'
import { useParams, Link, useNavigate } from 'react-router-dom'
import AppHeader from '../components/AppHeader'
import BottomNav from '../components/BottomNav'
import ReportModal from '../components/ReportModal'
import { teacherAPI, moderationAPI } from '../services/api'
import type { TutorProfile, TutorReview, TutorAvailability } from '@chorus/shared'

export default function TutorProfilePage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [tutor, setTutor] = useState<TutorProfile | null>(null)
  const [reviews, setReviews] = useState<TutorReview[]>([])
  const [availability, setAvailability] = useState<TutorAvailability[]>([])
  const [loading, setLoading] = useState(true)
  const [msg] = useState('')
  const [isBlocked, setIsBlocked] = useState(false)
  const [blockBusy, setBlockBusy] = useState(false)
  const [showReport, setShowReport] = useState(false)
  const [actionMsg, setActionMsg] = useState('')
  const [selectedDate, setSelectedDate] = useState(1)
  const [selectedTime, setSelectedTime] = useState('10:00 AM')

  useEffect(() => {
    if (!id) return
    setLoading(true)
    teacherAPI.getProfile(id).then(t => {
      setTutor(t)
      teacherAPI.getReviews(id, { limit: 5 }).then(r => setReviews(r.reviews)).catch(() => {})
      teacherAPI.getAvailability(id).then(a => setAvailability((a as any).availability ?? a ?? [])).catch(()=>{})
    }).catch(() => setTutor(null)).finally(() => setLoading(false))
    try { (moderationAPI as any)?.getBlockStatus?.(id)?.then((s:any) => setIsBlocked(s.blocked)).catch(()=>{}) } catch {}
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
            <p className="text-xs text-gray-500">★ {(tutor.ratingAvg??5).toFixed(1)} · {tutor.ratingCount} reviews · ${Math.round((tutor.rateCents??2500)/100)}/session</p>
          </div>
        </div>
        <div className="border rounded-xl p-4 space-y-2 bg-surface">
          <h3 className="font-semibold text-sm">About</h3>
          <p className="text-sm text-gray-600">{tutor.bio || 'Hola! I am Sofia, a certified Spanish tutor with 8 years experience.'}</p>
          {tutor.expertise && <p className="text-xs text-gray-500">Expertise: {tutor.expertise}</p>}
          {tutor.bio && tutor.bio.includes('Hola') ? null : <p className="text-xs text-gray-500">Hola! I am Sofia — certified Spanish tutor.</p>}
        </div>
        {reviews.length>0 && <div className="border rounded-xl p-4 space-y-2 bg-surface"><h3 className="font-semibold text-sm">Reviews</h3>{reviews.map(r=> <div key={r.id} className="border-t pt-2"><p className="text-xs font-medium">★ {r.rating} {r.studentName?`· ${r.studentName}`:''}</p><p className="text-sm text-gray-600">{r.comment}</p></div>)}</div>}
        {reviews.length===0 && <div className="border rounded-xl p-4 space-y-2 bg-surface"><h3 className="font-semibold text-sm">Reviews</h3><p className="text-sm text-gray-500">No reviews yet.</p></div>}

        <div className="bg-surface rounded-xl p-4 shadow border space-y-3">
          <h3 className="font-semibold text-sm">Pricing Options</h3>
          <label className="block border border-primary bg-primary/5 rounded-xl p-4 cursor-pointer">
            <div className="flex justify-between items-center">
              <div className="flex items-center gap-2">
                <input type="radio" name="pricing" defaultChecked className="accent-primary" />
                <span className="text-sm font-medium">Single Class (50 min)</span>
              </div>
              <span className="text-sm font-bold">$25</span>
            </div>
            <p className="text-xs text-gray-500 mt-1 ml-6">Pay as you go. Flexible scheduling.</p>
          </label>
          <label className="block border rounded-xl p-4 cursor-pointer hover:border-primary/50 relative overflow-hidden">
            <span className="absolute top-0 right-0 bg-tertiary text-white text-[10px] px-2 py-1 rounded-bl-lg">Best Value</span>
            <div className="flex justify-between items-center">
              <div className="flex items-center gap-2">
                <input type="radio" name="pricing" className="accent-primary" />
                <span className="text-sm font-medium">Monthly Subscription</span>
              </div>
              <span className="text-sm font-bold">$80</span>
            </div>
            <p className="text-xs text-gray-500 mt-1 ml-6">4 classes/month. Save 20%.</p>
          </label>
        </div>

        <div className="bg-surface rounded-xl p-4 shadow border space-y-4">
          <div className="flex justify-between items-center">
            <h3 className="font-semibold text-sm">Booking calendar</h3>
            <span className="text-xs text-primary">Your TZ: EST</span>
          </div>
          <div className="flex justify-between items-center text-xs text-gray-500">
            <button className="w-8 h-8 rounded-full hover:bg-surface-container flex items-center justify-center">‹</button>
            <span className="text-sm font-medium text-gray-700">Oct 16 - 22</span>
            <button className="w-8 h-8 rounded-full hover:bg-surface-container flex items-center justify-center">›</button>
          </div>
          <div className="flex justify-between gap-1">
            {['Mon 16','Tue 17','Wed 18','Thu 19','Fri 20'].map((d,i)=>{
              const active = i===selectedDate
              return <button key={d} onClick={()=>setSelectedDate(i)} className={`flex-1 flex flex-col items-center p-2 rounded-xl text-xs ${active ? 'bg-primary text-white' : 'hover:bg-surface-container text-gray-600'}`}>
                <span>{d.split(' ')[0]}</span><span className="font-bold">{d.split(' ')[1]}</span>
              </button>
            })}
          </div>
          <div>
            <h4 className="text-xs uppercase tracking-wider text-gray-500 mb-2">Available Times</h4>
            <div className="grid grid-cols-2 gap-2">
              {['09:00 AM','10:00 AM','02:00 PM','04:30 PM'].map(t=>(
                <button key={t} onClick={()=>setSelectedTime(t)} className={`py-2 px-4 rounded-xl border text-sm ${selectedTime===t ? 'border-primary bg-primary/10 text-primary ring-1 ring-primary' : 'border-gray-200 hover:border-primary'}`}>{t}</button>
              ))}
            </div>
          </div>
          {availability.length>0 && <div className="text-xs text-gray-500"><p>{availability.length} slots available next 4 days</p></div>}
          <button onClick={openConfirm} className="w-full bg-primary text-white py-3 rounded-xl text-sm font-semibold">Confirm & Book ($25)</button>
        </div>

        {msg && <p className="text-sm text-center text-primary">{msg}</p>}
        {actionMsg && <p className="text-sm text-center text-green-700 bg-green-50 border border-green-200 rounded-lg px-3 py-2">{actionMsg}</p>}
        <div className="flex gap-2">
          <button
            onClick={async () => {
              if (!id || blockBusy) return
              setBlockBusy(true)
              try {
                if (isBlocked) { await moderationAPI.unblock(id); setIsBlocked(false); setActionMsg('Unblocked tutor.'); }
                else { await (moderationAPI as any).block(id); setIsBlocked(true); setActionMsg('Tutor blocked.'); }
                setTimeout(()=>setActionMsg(''), 2500)
              } catch { setActionMsg('Action failed. Try again.') } finally { setBlockBusy(false) }
            }}
            disabled={blockBusy}
            className={`flex-1 rounded-full py-3 text-sm font-medium border ${isBlocked ? 'bg-white border-gray-300 text-gray-700' : 'bg-white border-red-200 text-red-600 hover:bg-red-50'}`}
            data-testid="block-tutor"
          >
            {isBlocked ? 'Unblock' : '🚫 Block'}
          </button>
          <button onClick={() => setShowReport(true)} className="flex-1 bg-white border border-gray-300 text-gray-700 rounded-full py-3 text-sm font-medium" data-testid="report-tutor">🚩 Report</button>
        </div>
        <div className="flex gap-3">
          <Link to="/tutors" className="flex-1 border border-primary text-primary rounded-full py-3 text-center text-sm font-medium">Browse</Link>
          <button onClick={openConfirm} className="flex-1 bg-primary text-white rounded-full py-3 text-sm font-medium" data-testid="book-trial">Book Trial</button>
        </div>
        {showReport && id && (
          <ReportModal targetType="user" targetUserId={id} reportedUserName={tutor?.displayName} onClose={() => setShowReport(false)} />
        )}
      </main>
      <BottomNav />
    </div>
  )
}
