import { useEffect, useState } from 'react'
import { waitlistAPI } from '../services/api'
import type { WaitlistEntry } from '../types'

export default function AdminWaitlist() {
  const [entries, setEntries] = useState<WaitlistEntry[]>([])
  const [error, setError] = useState('')
  const [busy, setBusy] = useState<string | null>(null)
  useEffect(() => { waitlistAPI.pending().then(setEntries).catch(() => setError('You are not authorized to view the waitlist.')) }, [])
  const approve = async (id: string) => {
    setBusy(id); setError('')
    try { await waitlistAPI.approve(id); setEntries(items => items.filter(entry => entry.id !== id)) }
    catch (err: any) { setError(err?.response?.data?.error || 'Could not send invitation.') }
    finally { setBusy(null) }
  }
  return <main className="min-h-screen bg-gray-50 p-8"><section className="mx-auto max-w-4xl"><h1 className="text-3xl font-bold">Waitlist administration</h1>{error && <p className="mt-4 rounded bg-red-100 p-3 text-red-700">{error}</p>}<div className="mt-6 space-y-3">{entries.map(entry => <article key={entry.id} className="rounded-lg bg-white p-5 shadow"><div className="flex justify-between gap-4"><div><p className="font-semibold">#{entry.queuePosition} · {entry.email}</p><p className="text-sm text-gray-600">Speaks {entry.spokenLanguage}; learning {entry.targetLanguages.join(', ')}</p><p className="text-sm text-gray-600">{entry.reasons.join(' · ')}</p></div><button onClick={() => approve(entry.id)} disabled={busy === entry.id} className="rounded bg-indigo-600 px-4 py-2 font-semibold text-white disabled:opacity-50">{busy === entry.id ? 'Sending…' : 'Approve & invite'}</button></div></article>)}</div></section></main>
}
