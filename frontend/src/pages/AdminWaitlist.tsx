import { useEffect, useState } from 'react'
import { waitlistAPI } from '../services/api'
import { getLanguageName } from '../services/language'
import type { WaitlistEntry } from '../types'

const names = (codes: string[]) => codes.map(code => getLanguageName(code)).join(', ') || '—'

export default function AdminWaitlist() {
  const [entries, setEntries] = useState<WaitlistEntry[]>([])
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const [busy, setBusy] = useState<string | null>(null)
  useEffect(() => {
    waitlistAPI.pending().then(setEntries).catch(() => setError('You are not authorized to view the waitlist.'))
  }, [])

  const approve = async (id: string) => {
    setBusy(id); setError(''); setNotice('')
    try {
      const { message } = await waitlistAPI.approve(id)
      setNotice(message)
      setEntries(items => items.filter(entry => entry.id !== id))
    } catch (err: any) {
      setError(err?.response?.data?.error || 'Could not send the invitation.')
    } finally { setBusy(null) }
  }

  return (
    <main className="min-h-screen bg-gray-50 p-8">
      <section className="mx-auto max-w-4xl">
        <div className="flex items-center justify-between">
          <h1 className="text-3xl font-bold">Waitlist administration</h1>
          <span className="rounded-full bg-indigo-100 px-3 py-1 text-sm font-semibold text-indigo-700">
            {entries.length} pending
          </span>
        </div>
        <p className="mt-2 text-gray-600">
          When you are ready to invite more users, click <strong>Approve &amp; send invite</strong>. Chorus emails
          them a sign-up link that lets them create their account.
        </p>
        {error && <p className="mt-4 rounded bg-red-100 p-3 text-red-700">{error}</p>}
        {notice && <p className="mt-4 rounded bg-green-100 p-3 text-green-700">{notice}</p>}
        {entries.length === 0 && !error && (
          <p className="mt-8 rounded-lg border border-dashed border-gray-300 p-8 text-center text-gray-500">
            No pending waitlist entries.
          </p>
        )}
        <div className="mt-6 space-y-3">
          {entries.map(entry => (
            <article key={entry.id} className="rounded-lg bg-white p-5 shadow">
              <div className="flex items-start justify-between gap-4">
                <div className="min-w-0 space-y-1">
                  <p className="font-semibold">#{entry.queuePosition} · {entry.email}</p>
                  <p className="text-sm text-gray-600">
                    Speaks: <strong>{names(entry.spokenLanguages)}</strong>
                  </p>
                  <p className="text-sm text-gray-600">
                    Learning: <strong>{names(entry.targetLanguages)}</strong>
                  </p>
                  <p className="text-sm text-gray-600">{entry.reasons.join(' · ')}</p>
                  {entry.comments && (
                    <p className="rounded bg-gray-50 p-2 text-sm italic text-gray-500">“{entry.comments}”</p>
                  )}
                </div>
                <button
                  onClick={() => approve(entry.id)}
                  disabled={busy === entry.id}
                  className="shrink-0 rounded bg-indigo-600 px-4 py-2 font-semibold text-white hover:bg-indigo-700 disabled:opacity-50"
                >
                  {busy === entry.id ? 'Sending…' : 'Approve & send invite'}
                </button>
              </div>
            </article>
          ))}
        </div>
      </section>
    </main>
  )
}