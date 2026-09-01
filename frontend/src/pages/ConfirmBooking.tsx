import { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import AppHeader from '../components/AppHeader'
import BottomNav from '../components/BottomNav'
import { teacherAPI } from '../services/api'
import type { TutorProfile } from '@chorus/shared'

export default function ConfirmBooking() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [tutor, setTutor] = useState<TutorProfile | null>(null)
  const [loading, setLoading] = useState(true)
  const [booking, setBooking] = useState(false)
  const [msg, setMsg] = useState('')
  const [err, setErr] = useState('')

  useEffect(() => {
    if (!id) return
    teacherAPI.getProfile(id).then(setTutor).catch(() => setErr('Tutor not found')).finally(() => setLoading(false))
  }, [id])

  const tomorrow = new Date(Date.now() + 24 * 60 * 60 * 1000)
  const dateLabel = tomorrow.toLocaleDateString(undefined, { weekday: 'long', month: 'short', day: 'numeric' })
  const timeLabel = '10:00 AM - 10:30 AM'

  const confirm = async () => {
    if (!id) return
    setBooking(true)
    setMsg('')
    setErr('')
    try {
      const start = new Date(tomorrow)
      start.setHours(10, 0, 0, 0)
      const end = new Date(start.getTime() + 30 * 60 * 1000)
      await teacherAPI.book(id, { startTime: start.toISOString(), endTime: end.toISOString(), isTrial: true })
      setMsg('Trial booked! Check your bookings.')
      setTimeout(() => navigate('/trial-credits'), 1200)
    } catch (e: any) {
      setErr(e?.response?.data?.error || e.message)
    }
    setBooking(false)
  }

  if (loading) return <div className="h-screen flex flex-col bg-background"><AppHeader /><main className="flex-1 flex items-center justify-center"><p className="text-sm text-gray-500">Loading...</p></main><BottomNav /></div>
  if (!tutor) return <div className="h-screen flex flex-col bg-background"><AppHeader /><main className="flex-1 flex items-center justify-center flex-col gap-2"><p className="text-sm text-gray-500">{err || 'Tutor not found'}</p><button onClick={() => navigate(-1)} className="text-primary text-sm">Back</button></main><BottomNav /></div>

  return (
    <div className="h-screen flex flex-col bg-background">
      <header className="flex items-center gap-3 px-4 h-16 border-b border-outline-variant/30">
        <button onClick={() => navigate(-1)} className="w-8 h-8 rounded-full flex items-center justify-center text-on-surface-variant hover:bg-surface-container">←</button>
        <h1 className="font-headline-sm text-headline-sm text-on-surface flex-1 text-center pr-8">Confirm Booking</h1>
      </header>
      <main className="flex-1 overflow-y-auto px-4 py-6 space-y-6 pb-32 max-w-md w-full mx-auto">
        <div className="flex flex-col items-center text-center gap-2">
          <div className="w-12 h-12 rounded-full bg-tertiary-container/20 flex items-center justify-center text-tertiary text-xl">✓</div>
          <h2 className="text-xl font-bold">Great choice!</h2>
          <p className="text-sm text-gray-500">Review your trial session details below.</p>
        </div>
        <div className="border rounded-xl p-4 flex gap-4 items-center bg-surface-container-lowest">
          <div className="w-12 h-12 rounded-full bg-secondary-container flex items-center justify-center font-bold text-on-secondary-container">{(tutor.displayName || id || 'T').slice(0, 1).toUpperCase()}</div>
          <div>
            <p className="text-xs uppercase tracking-wider text-gray-500">Your Tutor</p>
            <p className="font-bold">{tutor.displayName}</p>
            <p className="text-xs text-primary">{(tutor.languages || []).slice(0, 1).join('') || 'Spanish'} · Native</p>
          </div>
        </div>
        <div className="border rounded-xl p-4 space-y-3 bg-surface-container-lowest">
          <div className="flex gap-3 items-center">
            <span className="w-8 h-8 rounded-full bg-primary/10 flex items-center justify-center text-primary text-sm">📅</span>
            <div><p className="text-xs text-gray-500">Date</p><p className="text-sm font-medium">{dateLabel}</p></div>
          </div>
          <div className="h-px bg-outline-variant/30" />
          <div className="flex gap-3 items-center">
            <span className="w-8 h-8 rounded-full bg-primary/10 flex items-center justify-center text-primary text-sm">⏰</span>
            <div><p className="text-xs text-gray-500">Time</p><p className="text-sm font-medium">{timeLabel} <span className="text-xs text-gray-500">(local)</span></p></div>
          </div>
        </div>
        <div className="bg-surface-container-low rounded-xl p-4 space-y-2">
          <h3 className="text-xs uppercase tracking-wider font-semibold text-gray-500">Payment Summary</h3>
          <div className="flex justify-between text-sm text-gray-600"><span>Trial Session</span><span>1 Credit</span></div>
          <div className="flex justify-between text-sm text-gray-600"><span>Credits Applied</span><span>-1 Credit</span></div>
          <div className="h-px bg-outline-variant/50 my-2" />
          <div className="flex justify-between font-bold"><span>Total</span><span>$0.00</span></div>
        </div>
        <div className="flex gap-2 text-xs text-gray-500 bg-surface p-3 rounded-lg">
          <span>ℹ</span><p><strong>Cancellation Policy:</strong> You can reschedule or cancel for free up to 24 hours before your trial.</p>
        </div>
        {msg && <p className="text-sm text-center text-primary">{msg}</p>}
        {err && <p className="text-sm text-center text-error">{err}</p>}
      </main>
      <div className="fixed bottom-16 md:bottom-0 left-0 w-full bg-surface-container-lowest p-4 shadow-[0_-8px_24px_rgba(0,0,0,0.05)]">
        <button onClick={confirm} disabled={booking} className="w-full max-w-md mx-auto block bg-primary text-white py-3 rounded-full text-sm font-semibold disabled:opacity-50" data-testid="confirm-booking">{booking ? 'Confirming...' : 'Confirm Booking'}</button>
      </div>
      <BottomNav />
    </div>
  )
}
