import { useEffect, useState } from 'react'
import AppHeader from '../components/AppHeader'
import BottomNav from '../components/BottomNav'
import { payoutsAPI } from '../services/api'
import type { PayoutOverview, PayoutMethod, PayoutRecord } from '@chorus/shared'

export default function Payouts() {
  const [overview, setOverview] = useState<PayoutOverview | null>(null)
  const [methods, setMethods] = useState<PayoutMethod[]>([])
  const [history, setHistory] = useState<PayoutRecord[]>([])
  const [loading, setLoading] = useState(true)
  const [err, setErr] = useState('')
  const [label, setLabel] = useState('')
  const [details, setDetails] = useState('')
  const [type, setType] = useState<'paypal' | 'bank'>('paypal')
  const [withdrawAmt, setWithdrawAmt] = useState('10')

  const load = async () => {
    setLoading(true); setErr('')
    try {
      const [ov, ms, hs] = await Promise.all([payoutsAPI.overview(), payoutsAPI.methods(), payoutsAPI.history({ limit: 10 })])
      setOverview(ov); setMethods(ms); setHistory(hs.payouts)
    } catch (e:any) { setErr(e?.response?.data?.error || e.message) }
    setLoading(false)
  }
  useEffect(()=>{ load() }, [])

  const addMethod = async () => {
    try { await payoutsAPI.addMethod({ type, label: label || type, details: details || label }); setLabel(''); setDetails(''); load() } catch(e:any){ setErr(e?.response?.data?.error||e.message) }
  }
  const withdraw = async () => {
    try { await payoutsAPI.withdraw({ amountCents: Math.round(parseFloat(withdrawAmt)*100) }); load() } catch(e:any){ setErr(e?.response?.data?.error||e.message) }
  }

  return (
    <div className="h-screen flex flex-col bg-background">
      <AppHeader />
      <main className="flex-1 overflow-y-auto px-4 py-6 space-y-6 pb-32 max-w-5xl w-full mx-auto">
        <div className="flex items-center gap-2"><a href="/teacher/dashboard" className="text-primary text-sm">← Back to Earnings Overview</a></div>
        <h2 className="text-2xl font-bold">Payout Settings &amp; History</h2>
        <p className="text-sm text-gray-500">Manage your connected bank accounts and review your past withdrawals.</p>
        {err && <p className="text-sm text-error bg-error-container p-2 rounded">{err}</p>}
        {loading ? <p className="text-sm text-gray-500">Loading...</p> : (
          <>
            <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
              <div className="bg-surface-container-lowest rounded-xl p-6 shadow border flex flex-col justify-between">
                <div><p className="text-xs uppercase text-gray-500">Total Lifetime Earnings</p><p className="text-3xl font-bold text-primary mt-1">${((overview?.lifetimeGross ?? 0)/100).toFixed(2)}</p></div>
                <div className="mt-4 bg-surface p-3 rounded-lg flex justify-between border"><span className="text-xs text-gray-500">Available for payout</span><span className="text-sm font-bold">${((overview?.availableCents ?? 0)/100).toFixed(2)}</span></div>
                <button onClick={() => document.getElementById('withdraw')?.scrollIntoView()} className="w-full mt-4 bg-primary text-white py-3 rounded-full text-sm font-semibold">Withdraw Funds →</button>
              </div>
              <div className="lg:col-span-2 bg-surface-container-lowest rounded-xl p-6 shadow border">
                <div className="flex justify-between mb-4"><h3 className="font-semibold">Payout Methods</h3><span className="text-primary text-sm">Add Method</span></div>
                {methods.length===0 ? <p className="text-xs text-gray-500">No methods yet. Add PayPal or bank.</p> : methods.map(m=>(
                  <div key={m.id} className="flex justify-between items-center border rounded-lg p-3 mb-2 text-sm bg-surface-container-low">
                    <span>{m.type} · {m.label} {m.isDefault?'· Default':''}</span>
                    <button onClick={() => payoutsAPI.removeMethod(m.id).then(load)} className="text-xs text-error">Remove</button>
                  </div>
                ))}
                <div className="flex gap-2 mt-3">
                  <select value={type} onChange={e=>setType(e.target.value as any)} className="border rounded px-2 py-1 text-sm"><option value="paypal">PayPal</option><option value="bank">Bank</option></select>
                  <input value={label} onChange={e=>setLabel(e.target.value)} placeholder="Label" className="flex-1 border rounded px-2 py-1 text-sm" />
                  <input value={details} onChange={e=>setDetails(e.target.value)} placeholder="Email / IBAN" className="flex-1 border rounded px-2 py-1 text-sm" />
                  <button onClick={addMethod} className="bg-primary text-white rounded px-3 text-sm">Add</button>
                </div>
              </div>
              <div className="lg:col-span-3 bg-surface-container-lowest rounded-xl p-6 shadow border">
                <h3 className="font-semibold mb-4">This Month's Breakdown</h3>
                <div className="space-y-3">
                  <div className="flex justify-between border-b pb-2"><span className="text-sm text-gray-500">Gross Earnings</span><span className="text-sm font-semibold">${((overview?.lifetimeGross ?? 0)/100).toFixed(2)}</span></div>
                  <div className="flex justify-between border-b pb-2"><span className="text-sm text-gray-500">Platform Fee {overview?.platformFeePct ?? 15}%</span><span className="text-sm text-error">-${(((overview?.lifetimeGross ?? 0)-(overview?.lifetimeNet ?? 0))/100).toFixed(2)}</span></div>
                  <div className="flex justify-between"><span className="text-sm font-bold">Net Income</span><span className="text-sm font-bold text-tertiary">${((overview?.lifetimeNet ?? 0)/100).toFixed(2)}</span></div>
                </div>
              </div>
              <div className="lg:col-span-3 grid md:grid-cols-2 gap-6">
                <div id="withdraw" className="border rounded-xl p-4 space-y-3 bg-surface-container-lowest">
                  <h3 className="font-semibold text-sm">Withdraw</h3>
                  <div className="flex gap-2">
                    <input type="number" value={withdrawAmt} onChange={e=>setWithdrawAmt(e.target.value)} className="flex-1 border rounded px-2 py-1 text-sm" />
                    <button onClick={withdraw} className="bg-primary text-white rounded px-4 text-sm font-medium">Withdraw</button>
                  </div>
                  {overview && <p className="text-xs text-gray-500">Available: ${(overview.availableCents/100).toFixed(2)} · Pending: ${(overview.pendingCents/100).toFixed(2)}</p>}
                </div>
                <div className="border-l-4 border-l-secondary bg-surface-bright rounded-xl p-4 shadow">
                  <h3 className="text-sm font-semibold text-secondary">✦ Performance Insight</h3>
                  <p className="text-sm mt-2">You are on track to earn <strong>12% more</strong> than last month based on your current booking rate.</p>
                  <div className="flex gap-4 mt-4"><div className="flex-1 bg-surface-container-low p-3 rounded-lg"><p className="text-xs text-gray-500">Hours Taught</p><p className="font-bold text-primary">42h</p></div><div className="flex-1 bg-surface-container-low p-3 rounded-lg"><p className="text-xs text-gray-500">Active Students</p><p className="font-bold text-primary">{overview?.activeStudents ?? 0}</p></div></div>
                </div>
              </div>
            </div>
            <div className="border rounded-xl p-4 space-y-2 bg-surface-container-lowest">
              <h3 className="font-semibold text-sm">Payout History</h3>
              {history.length===0 ? <p className="text-xs text-gray-500">No payouts yet.</p> : history.map(p=>(
                <div key={p.id} className="border-t pt-2 text-sm flex justify-between">
                  <span>${(p.amountCents/100).toFixed(2)} · {p.status} · {new Date(p.createdAt).toLocaleDateString()}</span>
                  <span className="text-xs text-gray-500">{p.reference}</span>
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
