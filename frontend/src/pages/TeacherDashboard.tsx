import { useEffect, useState } from 'react'
import AppHeader from '../components/AppHeader'
import BottomNav from '../components/BottomNav'

export default function TeacherDashboard() {
  const [dash, setDash] = useState<any>(null)
  const [loading, setLoading] = useState(true)
  const [err, setErr] = useState('')

  useEffect(() => {
    let active = true
    fetch('/api/v1/teachers/dashboard', { headers: { Authorization: `Bearer ${localStorage.getItem('accessToken')}` } })
      .then(r => r.json().then(j => ({ ok: r.ok, j })))
      .then(({ ok, j }) => {
        if (!active) return
        if (!ok) throw new Error(j.error || 'Failed')
        setDash(j.dashboard)
      }).catch(e => active && setErr(e.message))
      .finally(() => active && setLoading(false))
    return () => { active = false }
  }, [])

  const pct = dash?.checklist?.completionPct ?? dash?.checklist?.percent ?? 0

  return (
    <div className="h-screen flex flex-col bg-background">
      <AppHeader />
      <main className="flex-1 overflow-y-auto px-4 py-6 space-y-6 pb-32 max-w-5xl w-full mx-auto">
        <h2 className="text-xl font-bold">Teacher Dashboard</h2>
        {loading ? <p className="text-sm text-gray-500">Loading...</p> : err ? <p className="text-sm text-error">{err} — <a href="/become-teacher" className="text-primary underline">Apply</a></p> : dash ? (
          <>
            <div className="bg-white p-6 rounded-2xl shadow border border-outline-variant/20 flex flex-col md:flex-row justify-between gap-4">
              <div><h3 className="text-lg font-bold">Welcome back!</h3><p className="text-sm text-gray-500">Here&apos;s what&apos;s happening with your classes today.</p></div>
              <div className="flex items-center gap-2 bg-surface-container-low px-4 py-2 rounded-full border"><span className="text-xs uppercase text-gray-500">Status</span><span className="text-sm font-medium">Accepting New Students</span></div>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-12 gap-6">
              <div className="md:col-span-8 bg-white p-6 rounded-2xl shadow border">
                <div className="flex justify-between mb-4"><h3 className="font-semibold flex gap-2">💰 Earnings Overview</h3><span className="text-xs bg-surface-container px-2 py-1 rounded">This Month</span></div>
                <div className="grid grid-cols-3 gap-4">
                  <div className="bg-surface-bright p-3 rounded-xl border"><p className="text-xs text-gray-500">Total Earned</p><p className="text-lg font-bold text-primary">${((dash.earnings?.totalGrossCents ?? dash.earnings?.totalGross ?? 0)/100).toFixed(2)}</p></div>
                  <div className="bg-surface-bright p-3 rounded-xl border"><p className="text-xs text-gray-500">Pending Payout</p><p className="text-lg font-bold text-secondary">${((dash.earnings?.pendingCents ?? dash.earnings?.pendingGrossCents ?? 0)/100).toFixed(2)}</p></div>
                  <div className="bg-surface-bright p-3 rounded-xl border"><p className="text-xs text-gray-500">Platform Fee ({dash.earnings?.platformFeePct ?? 15}%)</p><p className="text-lg font-bold text-outline">-${(((dash.earnings?.totalGrossCents ?? 0) - (dash.earnings?.totalNetCents ?? 0))/100).toFixed(2)}</p></div>
                </div>
              </div>
              <div className="md:col-span-4 bg-gradient-to-br from-primary to-secondary p-6 rounded-2xl text-white flex flex-col justify-between">
                <div><h3 className="font-semibold flex gap-2">✓ Premium Program</h3><p className="text-sm opacity-90 mt-2">You are enrolled in the inclusive premium sessions program.</p></div>
                <a href="/teacher/payouts" className="bg-white text-primary py-2 px-4 rounded-lg text-center text-sm font-semibold mt-4">Manage Premium Settings</a>
              </div>
              <div className="md:col-span-6 bg-white p-6 rounded-2xl shadow border">
                <div className="flex justify-between mb-4"><h3 className="font-semibold">Availability</h3><a href="/teacher/payouts" className="text-primary text-xs">Edit Schedule</a></div>
                {(dash.upcomingAvailability ?? dash.availability ?? []).length === 0 ? <p className="text-sm text-gray-500">No availability set.</p> :
                  (dash.upcomingAvailability ?? dash.availability ?? []).slice(0, 3).map((a:any, i:number) => (
                    <div key={i} className="flex justify-between p-3 bg-surface rounded-xl border mb-2"><span className="text-sm">{new Date(a.startTime).toLocaleDateString()} {new Date(a.startTime).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })} - {new Date(a.endTime).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</span><span className="text-xs text-gray-500">{a.status || 'Available'}</span></div>
                  ))}
                {(dash.upcoming ?? []).length > 0 && <div className="mt-3"><h4 className="text-xs font-semibold uppercase text-gray-500">Upcoming Sessions</h4>{dash.upcoming.slice(0, 2).map((b:any)=><div key={b.id} className="text-sm border-t pt-2 mt-2">{new Date(b.startTime).toLocaleString()} — {b.studentName || b.studentUserId?.slice(0,6)} {b.isTrial?'(trial)':''}</div>)}</div>}
              </div>
              <div className="md:col-span-6 bg-white p-6 rounded-2xl shadow border">
                <div className="flex justify-between mb-4"><h3 className="font-semibold">Recent Students</h3><span className="text-primary text-xs">View All</span></div>
                {(dash.students ?? []).length === 0 ? <p className="text-sm text-gray-500">No students yet.</p> :
                  (dash.students ?? []).slice(0, 3).map((s:any,i:number)=><div key={i} className="flex justify-between items-center border-t pt-3 mt-2"><span className="text-sm">{s.displayName || s.studentName || s.userId?.slice(0,6)}</span><span className="text-xs text-gray-500">{s.bookingsCount ?? ''} bookings</span></div>)}
              </div>
              <div className="md:col-span-12 bg-gradient-to-r from-surface to-surface-container-low p-6 rounded-2xl shadow border border-primary/20">
                <h3 className="font-semibold mb-2">Profile Completion — {pct}%</h3>
                <div className="w-full bg-surface-variant rounded-full h-2 mb-3"><div className="bg-primary h-2 rounded-full" style={{ width: `${pct}%` }} /></div>
                <div className="grid grid-cols-2 gap-3">
                  {[
                    { label: 'Basic Bio', done: dash.checklist?.hasBio },
                    { label: 'Upload Video Intro', done: dash.checklist?.hasVideo },
                    { label: 'Add Certifications', done: dash.checklist?.hasCertificate },
                  ].map(it => (
                    <div key={it.label} className={`p-3 rounded-lg border bg-white flex gap-2 ${it.done ? 'opacity-60' : 'border-primary/40'}`}><span>{it.done ? '✓' : '○'}</span><span className={`text-sm ${it.done ? 'line-through text-gray-500' : 'text-primary'}`}>{it.label}</span></div>
                  ))}
                </div>
              </div>
            </div>

            <div className="flex gap-2">
              <a href="/teacher/payouts" className="flex-1 bg-primary text-white rounded-full py-3 text-center text-sm font-medium">Payouts</a>
              <a href="/tutors" className="flex-1 border border-primary text-primary rounded-full py-3 text-center text-sm font-medium">Browse tutors</a>
            </div>
          </>
        ) : <p className="text-sm text-gray-500">No dashboard data.</p>}
      </main>
      <BottomNav />
    </div>
  )
}
