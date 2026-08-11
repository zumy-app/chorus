import { useEffect, useState } from 'react'
import { adminAPI } from '../services/api'
import { getLanguageName } from '../services/language'
import type { WaitlistEntry, EmailOutboxEntry, AdminStats } from '../types'

const names = (codes: string[]) => codes.map(code => getLanguageName(code)).join(', ') || '—'

type Tab = 'waitlist' | 'emails' | 'stats'
type StatusFilter = 'pending' | 'approved' | 'declined' | 'all'

const statusBadge: Record<string, string> = {
  pending: 'bg-amber-100 text-amber-700',
  approved: 'bg-green-100 text-green-700',
  declined: 'bg-red-100 text-red-700',
  sent: 'bg-green-100 text-green-700',
  failed: 'bg-red-100 text-red-700',
}

export default function AdminWaitlist() {
  const [tab, setTab] = useState<Tab>('waitlist')
  const [entries, setEntries] = useState<WaitlistEntry[]>([])
  const [emails, setEmails] = useState<EmailOutboxEntry[]>([])
  const [stats, setStats] = useState<AdminStats | null>(null)
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('pending')
  const [q, setQ] = useState('')
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const [busy, setBusy] = useState<string | null>(null)

  const loadWaitlist = async (status: StatusFilter = statusFilter, query = q) => {
    setError('')
    try {
      setEntries(await adminAPI.listWaitlist(status, query))
    } catch (err: any) {
      setError(err?.response?.data?.error || 'You are not authorized to view the waitlist.')
    }
  }

  const loadEmails = async (status?: string) => {
    setError('')
    try {
      setEmails(await adminAPI.emails(status))
    } catch (err: any) {
      setError(err?.response?.data?.error || 'You are not authorized to view the email log.')
    }
  }

  const loadStats = async () => {
    setError('')
    try {
      setStats(await adminAPI.stats())
    } catch (err: any) {
      setError(err?.response?.data?.error || 'You are not authorized to view stats.')
    }
  }

  useEffect(() => {
    loadWaitlist()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  useEffect(() => {
    if (tab === 'waitlist') loadWaitlist()
    else if (tab === 'emails') loadEmails()
    else loadStats()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tab])

  const act = async (fn: () => Promise<{ message: string }>, id: string, reload: () => Promise<void>) => {
    setBusy(id); setError(''); setNotice('')
    try {
      const { message } = await fn()
      setNotice(message)
      await reload()
      if (tab === 'emails') await loadEmails()
    } catch (err: any) {
      setError(err?.response?.data?.error || 'Something went wrong.')
    } finally { setBusy(null) }
  }

  const searchNow = () => loadWaitlist(statusFilter, q)

  return (
    <main className="min-h-screen bg-gray-50">
      <div className="mx-auto max-w-5xl p-6">
        <header className="mb-6 flex flex-wrap items-center justify-between gap-4">
          <h1 className="text-3xl font-bold">Admin console</h1>
          <nav className="flex gap-2">
            {(['waitlist', 'emails', 'stats'] as Tab[]).map(name => (
              <button
                key={name}
                onClick={() => setTab(name)}
                className={`rounded-lg px-4 py-2 font-semibold capitalize transition ${
                  tab === name ? 'bg-indigo-600 text-white' : 'bg-white text-gray-600 hover:bg-indigo-50'
                }`}
              >
                {name}
              </button>
            ))}
          </nav>
        </header>

        {error && <p className="mb-4 rounded bg-red-100 p-3 text-red-700">{error}</p>}
        {notice && <p className="mb-4 rounded bg-green-100 p-3 text-green-700">{notice}</p>}

        {tab === 'waitlist' && (
          <section className="space-y-4">
            <div className="flex flex-wrap items-center gap-3 rounded-lg bg-white p-3 shadow">
              <select
                value={statusFilter}
                onChange={e => {
                  const value = e.target.value as StatusFilter
                  setStatusFilter(value)
                  loadWaitlist(value, q)
                }}
                className="rounded border border-gray-300 px-3 py-2 text-sm"
              >
                <option value="pending">Pending</option>
                <option value="approved">Approved</option>
                <option value="declined">Declined</option>
                <option value="all">All</option>
              </select>
              <input
                value={q}
                onChange={e => setQ(e.target.value)}
                onKeyDown={e => e.key === 'Enter' && searchNow()}
                placeholder="Search email…"
                className="min-w-0 flex-1 rounded border border-gray-300 px-3 py-2 text-sm"
              />
              <button
                onClick={searchNow}
                className="rounded bg-gray-800 px-4 py-2 text-sm font-semibold text-white hover:bg-gray-900"
              >
                Search
              </button>
            </div>

            {entries.length === 0 ? (
              <p className="rounded-lg border border-dashed border-gray-300 bg-white p-8 text-center text-gray-500">
                No waitlist entries{statusFilter !== 'all' ? ` with status “${statusFilter}”` : ''}.
              </p>
            ) : (
              entries.map(entry => (
                <article key={entry.id} className="rounded-lg bg-white p-5 shadow">
                  <div className="flex items-start justify-between gap-4">
                    <div className="min-w-0 space-y-1">
                      <p className="flex flex-wrap items-center gap-2 font-semibold">
                        #{entry.queuePosition} · {entry.email}
                        <span className={`rounded-full px-2 py-0.5 text-xs font-semibold ${statusBadge[entry.status]}`}>
                          {entry.status}
                        </span>
                      </p>
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
                      <p className="text-xs text-gray-400">
                        Joined {new Date(entry.createdAt).toLocaleString()}
                        {entry.approvedAt && ` · Approved ${new Date(entry.approvedAt).toLocaleString()}`}
                      </p>
                    </div>
                    <div className="flex shrink-0 flex-col gap-2">
                      {entry.status === 'pending' && (
                        <>
                          <button
                            onClick={() => act(() => adminAPI.approve(entry.id), entry.id, () => loadWaitlist())}
                            disabled={busy === entry.id}
                            className="rounded bg-indigo-600 px-4 py-2 font-semibold text-white hover:bg-indigo-700 disabled:opacity-50"
                          >
                            {busy === entry.id ? 'Sending…' : 'Approve & send invite'}
                          </button>
                          <button
                            onClick={() => act(() => adminAPI.decline(entry.id), entry.id, () => loadWaitlist())}
                            disabled={busy === entry.id}
                            className="rounded border border-red-300 px-4 py-2 font-semibold text-red-600 hover:bg-red-50 disabled:opacity-50"
                          >
                            Decline
                          </button>
                        </>
                      )}
                      {entry.status === 'approved' && (
                        <button
                          onClick={() => act(() => adminAPI.resendInvite(entry.id), entry.id, () => loadWaitlist())}
                          disabled={busy === entry.id}
                          className="rounded bg-gray-800 px-4 py-2 font-semibold text-white hover:bg-gray-900 disabled:opacity-50"
                        >
                          {busy === entry.id ? 'Sending…' : 'Resend invite'}
                        </button>
                      )}
                    </div>
                  </div>
                </article>
              ))
            )}
          </section>
        )}

        {tab === 'emails' && (
          <section className="space-y-4">
            <div className="flex flex-wrap items-center gap-3 rounded-lg bg-white p-3 shadow">
              <select
                value=""
                onChange={e => e.target.value && loadEmails(e.target.value)}
                className="rounded border border-gray-300 px-3 py-2 text-sm"
              >
                <option value="" disabled>Filter status…</option>
                <option value="pending">Pending</option>
                <option value="sent">Sent</option>
                <option value="failed">Failed</option>
              </select>
              <button
                onClick={() => loadEmails()}
                className="rounded bg-gray-800 px-4 py-2 text-sm font-semibold text-white hover:bg-gray-900"
              >
                Refresh
              </button>
            </div>

            {emails.length === 0 ? (
              <p className="rounded-lg border border-dashed border-gray-300 bg-white p-8 text-center text-gray-500">
                No emails in the outbox.
              </p>
            ) : (
              <div className="overflow-hidden rounded-lg bg-white shadow">
                <table className="w-full text-left text-sm">
                  <thead className="bg-gray-100 text-gray-600">
                    <tr>
                      <th className="px-4 py-3">Recipient</th>
                      <th className="px-4 py-3">Subject</th>
                      <th className="px-4 py-3">Status</th>
                      <th className="px-4 py-3">Attempts</th>
                      <th className="px-4 py-3">Sent</th>
                      <th className="px-4 py-3">Last error</th>
                      <th className="px-4 py-3"></th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-100">
                    {emails.map(email => (
                      <tr key={email.id}>
                        <td className="px-4 py-3 font-medium">{email.recipient}</td>
                        <td className="max-w-xs truncate px-4 py-3 text-gray-600">{email.subject}</td>
                        <td className="px-4 py-3">
                          <span className={`rounded-full px-2 py-0.5 text-xs font-semibold ${statusBadge[email.status]}`}>
                            {email.status}
                          </span>
                        </td>
                        <td className="px-4 py-3 text-gray-600">{email.attempts}</td>
                        <td className="px-4 py-3 text-gray-600">
                          {email.sentAt ? new Date(email.sentAt).toLocaleString() : '—'}
                        </td>
                        <td className="max-w-xs truncate px-4 py-3 text-red-600">{email.lastError || '—'}</td>
                        <td className="px-4 py-3 text-right">
                          {email.status !== 'sent' && (
                            <button
                              onClick={() => act(() => adminAPI.retryEmail(email.id), email.id, () => loadEmails())}
                              disabled={busy === email.id}
                              className="rounded bg-indigo-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-indigo-700 disabled:opacity-50"
                            >
                              {busy === email.id ? 'Sending…' : 'Retry'}
                            </button>
                          )}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </section>
        )}

        {tab === 'stats' && (
          <section className="space-y-4">
            {!stats ? (
              <p className="rounded-lg border border-dashed border-gray-300 bg-white p-8 text-center text-gray-500">
                Loading stats…
              </p>
            ) : (
              <>
                <div className="grid grid-cols-2 gap-4 md:grid-cols-3">
                  <StatCard label="Total users" value={stats.totalUsers} />
                  <StatCard label="Waitlist pending" value={stats.waitlistPending} tone="amber" />
                  <StatCard label="Waitlist approved" value={stats.waitlistApproved} tone="green" />
                  <StatCard label="Waitlist declined" value={stats.waitlistDeclined} tone="red" />
                  <StatCard label="Emails pending" value={stats.emailsPending} tone="amber" />
                  <StatCard label="Emails sent" value={stats.emailsSent} tone="green" />
                  <StatCard label="Emails failed" value={stats.emailsFailed} tone="red" />
                </div>
                <div className="flex justify-end">
                  <button
                    onClick={loadStats}
                    className="rounded bg-gray-800 px-4 py-2 text-sm font-semibold text-white hover:bg-gray-900"
                  >
                    Refresh
                  </button>
                </div>
              </>
            )}
          </section>
        )}
      </div>
    </main>
  )
}

function StatCard({ label, value, tone }: { label: string; value: number; tone?: 'amber' | 'green' | 'red' }) {
  const tones: Record<string, string> = {
    amber: 'text-amber-600',
    green: 'text-green-600',
    red: 'text-red-600',
  }
  return (
    <div className="rounded-lg bg-white p-5 shadow">
      <p className="text-sm font-semibold text-gray-500">{label}</p>
      <p className={`mt-1 text-3xl font-bold ${tone ? tones[tone] : 'text-gray-900'}`}>{value}</p>
    </div>
  )
}
