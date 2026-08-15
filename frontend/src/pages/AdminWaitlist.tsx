import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { adminAPI } from '../services/api'
import { getLanguageName } from '../services/language'
import { useStore } from '../store'
import type { WaitlistEntry, EmailOutboxEntry, AdminStats, User, TranslationJob, ProviderHealth, PremiumUserRow, PremiumAnalytics, PlanChange, GrantPlanRequest, Report, ReportStats } from '@chorus/shared'

const names = (codes: string[]) => codes.map(code => getLanguageName(code)).join(', ') || '—'

// Tabs list per role: moderators can manage users + translations; admins add
// waitlist, email outbox, global stats and premium management. Moderation
// reports are visible to both moderators and admins.
const ADMIN_TABS = ['users', 'translations', 'reports'] as const
const ALL_TABS = ['waitlist', 'users', 'translations', 'emails', 'stats', 'premium', 'reports'] as const
type Tab = (typeof ALL_TABS)[number]
type StatusFilter = 'pending' | 'approved' | 'declined' | 'all'

const tabLabels: Record<Tab, string> = {
  waitlist: 'tabWaitlist',
  users: 'tabUsers',
  translations: 'tabTranslations',
  emails: 'tabEmails',
  stats: 'tabStats',
  premium: 'tabPremium',
  reports: 'tabReports',
}

const statusBadge: Record<string, string> = {
  pending: 'bg-amber-100 text-amber-700',
  approved: 'bg-green-100 text-green-700',
  declined: 'bg-red-100 text-red-700',
  sent: 'bg-green-100 text-green-700',
  failed: 'bg-red-100 text-red-700',
  processing: 'bg-blue-100 text-blue-700',
  done: 'bg-green-100 text-green-700',
}

export default function AdminWaitlist({ defaultTab }: { defaultTab?: Tab }) {
  const { t } = useTranslation()
  const params = useParams<{ tab?: string }>()

  const statusLabel = (s: string): string =>
    ({ pending: t('admin.pending'), approved: t('admin.approved'), declined: t('admin.declined'), sent: t('admin.sent'), failed: t('admin.failed'), processing: t('admin.processing'), done: t('common.done') })[s] || s
  const userRole = useStore(s => s.userRole)
  const isAdmin = userRole === 'admin'

  const tabs: readonly Tab[] = isAdmin ? ALL_TABS : ADMIN_TABS
  const [tab, setTab] = useState<Tab>(
    () => {
      const requested = params.tab as Tab | undefined
      if (requested && tabs.includes(requested)) return requested
      return defaultTab && tabs.includes(defaultTab) ? defaultTab : (tabs[0] ?? 'users')
    }
  )

  const [entries, setEntries] = useState<WaitlistEntry[]>([])
  const [emails, setEmails] = useState<EmailOutboxEntry[]>([])
  const [stats, setStats] = useState<AdminStats | null>(null)
  const [users, setUsers] = useState<{ users: User[]; total: number }>({ users: [], total: 0 })

  const [statusFilter, setStatusFilter] = useState<StatusFilter>('pending')
  const [q, setQ] = useState('')

  const [userRoleFilter, setUserRoleFilter] = useState('all')
  const [userStatusFilter, setUserStatusFilter] = useState('all')

  const [jobs, setJobs] = useState<TranslationJob[]>([])
  const [jobFilter, setJobFilter] = useState('all')
  const [providers, setProviders] = useState<ProviderHealth[]>([])

  const [premium, setPremium] = useState<{ users: PremiumUserRow[]; total: number }>({ users: [], total: 0 })
  const [analytics, setAnalytics] = useState<PremiumAnalytics | null>(null)
  const [grantTarget, setGrantTarget] = useState<PremiumUserRow | null>(null)
  const [revokeTarget, setRevokeTarget] = useState<PremiumUserRow | null>(null)
  const [history, setHistory] = useState<{ user: PremiumUserRow; entries: PlanChange[] } | null>(null)
  const [grantMode, setGrantMode] = useState<'indefinite' | 'days' | 'until'>('indefinite')
  const [grantDays, setGrantDays] = useState(30)
  const [grantUntil, setGrantUntil] = useState('')
  const [reason, setReason] = useState('')
  const [revokeGrace, setRevokeGrace] = useState(0)

  const [reports, setReports] = useState<Report[]>([])
  const [reportTotal, setReportTotal] = useState(0)
  const [reportStats, setReportStats] = useState<ReportStats | null>(null)
  const [reportFilter, setReportFilter] = useState('open')
  const [dismissing, setDismissing] = useState<string | null>(null)
  const [dismissNote, setDismissNote] = useState('')

  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const [busy, setBusy] = useState<string | null>(null)

  const loadWaitlist = async (status: StatusFilter = statusFilter, query = q) => {
    setError('')
    try {
      setEntries(await adminAPI.listWaitlist(status, query))
    } catch (err: any) {
      setError(err?.response?.data?.error || t('admin.notAuthorizedWaitlist'))
    }
  }

  const loadEmails = async (status?: string) => {
    setError('')
    try {
      setEmails(await adminAPI.emails(status))
    } catch (err: any) {
      setError(err?.response?.data?.error || t('admin.notAuthorizedEmails'))
    }
  }

  const loadStats = async () => {
    setError('')
    try {
      setStats(await adminAPI.stats())
    } catch (err: any) {
      setError(err?.response?.data?.error || t('admin.notAuthorizedStats'))
    }
  }

  const loadUsers = async () => {
    setError('')
    try {
      setUsers(await adminAPI.listUsers({
        q: q || undefined,
        role: userRoleFilter !== 'all' ? userRoleFilter : undefined,
        status: userStatusFilter !== 'all' ? userStatusFilter : undefined,
      }))
    } catch (err: any) {
      setError(err?.response?.data?.error || t('admin.notAuthorizedUsers'))
    }
  }

  const loadTranslations = async () => {
    setError('')
    try {
      const data = await adminAPI.listTranslations(jobFilter, q)
      setJobs(data.jobs)
      if (isAdmin) {
        setProviders(await adminAPI.translationHealth().catch(() => []))
      }
    } catch (err: any) {
      setError(err?.response?.data?.error || t('admin.notAuthorizedTranslations'))
    }
  }

  const loadPremium = async () => {
    setError('')
    try {
      setPremium(await adminAPI.premiumUsers(q || undefined))
      setAnalytics(await adminAPI.premiumAnalytics())
    } catch (err: any) {
      setError(err?.response?.data?.error || t('admin.notAuthorizedPremium'))
    }
  }

  const loadReports = async (status = reportFilter, query = q) => {
    setError('')
    try {
      const data = await adminAPI.listReports(status, query)
      setReports(data.reports)
      setReportTotal(data.total)
      if (isAdmin) {
        setReportStats(await adminAPI.reportStats().catch(() => null))
      }
    } catch (err: any) {
      setError(err?.response?.data?.error || t('admin.notAuthorizedReports'))
    }
  }

  useEffect(() => {
    if (tab === 'waitlist') loadWaitlist()
    else if (tab === 'emails') loadEmails()
    else if (tab === 'stats') loadStats()
    else if (tab === 'users') loadUsers()
    else if (tab === 'premium') loadPremium()
    else if (tab === 'reports') loadReports()
    else loadTranslations()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tab])

  const act = async (fn: () => Promise<Record<string, unknown>>, id: string, reload: () => Promise<void>) => {
    setBusy(id); setError(''); setNotice('')
    try {
      const { message } = await fn()
      if (message) setNotice(String(message))
      await reload()
    } catch (err: any) {
      setError(err?.response?.data?.error || t('admin.error'))
    } finally { setBusy(null) }
  }

  const searchNow = () => {
    if (tab === 'waitlist') loadWaitlist(statusFilter, q)
    else if (tab === 'users') loadUsers()
    else if (tab === 'translations') loadTranslations()
    else if (tab === 'premium') loadPremium()
    else if (tab === 'reports') loadReports()
  }

  const resolveReport = (id: string) => act(() => adminAPI.resolveReport(id), id, () => loadReports())

  const submitDismiss = async (id: string) => {
    setBusy(id); setError(''); setNotice('')
    try {
      const { message } = await adminAPI.dismissReport(id, dismissNote.trim() || undefined)
      if (message) setNotice(String(message))
      setDismissing(null); setDismissNote('')
      await loadReports()
    } catch (err: any) {
      setError(err?.response?.data?.error || t('admin.error'))
    } finally { setBusy(null) }
  }

  const applyGrant = async () => {
    if (!grantTarget) return
    setBusy(grantTarget.id); setError(''); setNotice('')
    const req: GrantPlanRequest = { plan: 'premium', mode: grantMode, reason: reason || undefined }
    if (grantMode === 'days') req.days = grantDays
    if (grantMode === 'until') req.until = new Date(grantUntil).toISOString()
    try {
      const { message } = await adminAPI.grantPlan(grantTarget.id, req)
      if (message) setNotice(String(message))
      setGrantTarget(null); setReason('')
      await Promise.all([loadPremium()])
    } catch (err: any) {
      setError(err?.response?.data?.error || t('admin.error'))
    } finally { setBusy(null) }
  }

  const applyRevoke = async () => {
    if (!revokeTarget) return
    if (!window.confirm(t('admin.confirmRevoke', { name: revokeTarget.displayName || revokeTarget.username }))) return
    setBusy(revokeTarget.id); setError(''); setNotice('')
    try {
      const { message } = await adminAPI.revokePlan(revokeTarget.id, revokeGrace, reason || undefined)
      if (message) setNotice(String(message))
      setRevokeTarget(null); setReason(''); setRevokeGrace(0)
      await Promise.all([loadPremium()])
    } catch (err: any) {
      setError(err?.response?.data?.error || t('admin.error'))
    } finally { setBusy(null) }
  }

  const openHistory = async (user: PremiumUserRow) => {
    setBusy(user.id); setError('')
    try {
      const entries = await adminAPI.planHistory(user.id)
      setHistory({ user, entries })
    } catch (err: any) {
      setError(err?.response?.data?.error || t('admin.error'))
    } finally { setBusy(null) }
  }

  const money = (v: number) => '$' + v.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })

  const planBadge = (u: PremiumUserRow) => {
    const base = u.effectivePlan === 'premium'
      ? u.inGrace ? 'bg-amber-100 text-amber-700' : 'bg-gradient-to-r from-amber-400 to-yellow-300 text-amber-900'
      : 'bg-green-100 text-green-700'
    return <span className={`rounded-full px-2 py-0.5 text-xs font-semibold ${base}`}>{t('plan.' + u.effectivePlan)}</span>
  }

  const switchRole = async (user: User, role: string) => {
    await act(() => adminAPI.setUserRole(user.id, role), user.id, loadUsers)
  }

  return (
    <main className="min-h-screen bg-gray-50">
      <div className="mx-auto max-w-5xl p-6">
        <header className="mb-6 flex flex-wrap items-center justify-between gap-4">
          <h1 className="text-3xl font-bold">{t('admin.console')}</h1>
          <nav className="flex gap-2">
            {tabs.map(name => (
              <button
                key={name}
                onClick={() => setTab(name)}
                className={`rounded-lg px-4 py-2 font-semibold capitalize transition ${
                  tab === name ? 'bg-indigo-600 text-white' : 'bg-white text-gray-600 hover:bg-indigo-50'
                }`}
              >
                {t(`admin.${tabLabels[name]}`)}
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
                <option value="pending">{t('admin.pending')}</option>
                <option value="approved">{t('admin.approved')}</option>
                <option value="declined">{t('admin.declined')}</option>
                <option value="all">{t('admin.all')}</option>
              </select>
              <input
                value={q}
                onChange={e => setQ(e.target.value)}
                onKeyDown={e => e.key === 'Enter' && searchNow()}
                placeholder={t('admin.searchEmailPlaceholder')}
                className="min-w-0 flex-1 rounded border border-gray-300 px-3 py-2 text-sm"
              />
              <button
                onClick={searchNow}
                className="rounded bg-gray-800 px-4 py-2 text-sm font-semibold text-white hover:bg-gray-900"
              >
                {t('admin.search')}
              </button>
            </div>

            {entries.length === 0 ? (
              <p className="rounded-lg border border-dashed border-gray-300 bg-white p-8 text-center text-gray-500">
                {t('admin.noWaitlist')}
              </p>
            ) : (
              entries.map(entry => (
                <article key={entry.id} className="rounded-lg bg-white p-5 shadow">
                  <div className="flex items-start justify-between gap-4">
                    <div className="min-w-0 space-y-1">
                      <p className="flex flex-wrap items-center gap-2 font-semibold">
                        #{entry.queuePosition} · {entry.email}
                        <span className={`rounded-full px-2 py-0.5 text-xs font-semibold ${statusBadge[entry.status]}`}>
                          {statusLabel(entry.status)}
                        </span>
                      </p>
                      <p className="text-sm text-gray-600">
                        {t('admin.speaks')} <strong>{names(entry.spokenLanguages)}</strong>
                      </p>
                      <p className="text-sm text-gray-600">
                        {t('admin.learning')} <strong>{names(entry.targetLanguages)}</strong>
                      </p>
                      <p className="text-sm text-gray-600">{entry.reasons.join(' · ')}</p>
                      {entry.comments && (
                        <p className="rounded bg-gray-50 p-2 text-sm italic text-gray-500">“{entry.comments}”</p>
                      )}
                      <p className="text-xs text-gray-400">
                        {t('admin.joined', { date: new Date(entry.createdAt).toLocaleString() })}
                        {entry.approvedAt && ` · ${t('admin.approved')} ${new Date(entry.approvedAt).toLocaleString()}`}
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
                            {busy === entry.id ? t('admin.sending') : t('admin.approveSendInvite')}
                          </button>
                          <button
                            onClick={() => act(() => adminAPI.decline(entry.id), entry.id, () => loadWaitlist())}
                            disabled={busy === entry.id}
                            className="rounded border border-red-300 px-4 py-2 font-semibold text-red-600 hover:bg-red-50 disabled:opacity-50"
                          >
{t('admin.decline')}
                          </button>
                        </>
                      )}
                      {entry.status === 'approved' && (
                        <button
                          onClick={() => act(() => adminAPI.resendInvite(entry.id), entry.id, () => loadWaitlist())}
                          disabled={busy === entry.id}
                          className="rounded bg-gray-800 px-4 py-2 font-semibold text-white hover:bg-gray-900 disabled:opacity-50"
                        >
                          {busy === entry.id ? t('admin.sending') : t('admin.resendInvite')}
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
                <option value="" disabled>{t('admin.filterStatus')}</option>
                <option value="pending">{t('admin.pending')}</option>
                <option value="sent">{t('admin.sent')}</option>
                <option value="failed">{t('admin.failed')}</option>
              </select>
              <button
                onClick={() => loadEmails()}
                className="rounded bg-gray-800 px-4 py-2 text-sm font-semibold text-white hover:bg-gray-900"
              >
                {t('common.refresh')}
              </button>
            </div>

            {emails.length === 0 ? (
              <p className="rounded-lg border border-dashed border-gray-300 bg-white p-8 text-center text-gray-500">
                {t('admin.noEmails')}
              </p>
            ) : (
              <div className="overflow-hidden rounded-lg bg-white shadow">
                <table className="w-full text-left text-sm">
                  <thead className="bg-gray-100 text-gray-600">
                    <tr>
                      <th className="px-4 py-3">{t('admin.recipient')}</th>
                      <th className="px-4 py-3">{t('admin.subject')}</th>
                      <th className="px-4 py-3">{t('admin.status')}</th>
                      <th className="px-4 py-3">{t('admin.attempts')}</th>
                      <th className="px-4 py-3">{t('admin.sent')}</th>
                      <th className="px-4 py-3">{t('admin.lastError')}</th>
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
                            {statusLabel(email.status)}
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
                              {busy === email.id ? t('admin.sending') : t('admin.retry')}
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
                {t('admin.loadingStats')}
              </p>
            ) : (
              <>
                <div className="grid grid-cols-2 gap-4 md:grid-cols-3">
                  <StatCard label={t('admin.totalUsers')} value={stats.totalUsers} />
                  <StatCard label={t('admin.moderators')} value={stats.moderators} />
                  <StatCard label={t('admin.admins')} value={stats.admins} />
                  <StatCard label={t('admin.suspended')} value={stats.suspendedUsers} tone="red" />
                  <StatCard label={t('admin.waitlistPending')} value={stats.waitlistPending} tone="amber" />
                  <StatCard label={t('admin.waitlistApproved')} value={stats.waitlistApproved} tone="green" />
                  <StatCard label={t('admin.waitlistDeclined')} value={stats.waitlistDeclined} tone="red" />
                  <StatCard label={t('admin.emailsPending')} value={stats.emailsPending} tone="amber" />
                  <StatCard label={t('admin.emailsSent')} value={stats.emailsSent} tone="green" />
                  <StatCard label={t('admin.emailsFailed')} value={stats.emailsFailed} tone="red" />
                  <StatCard label={t('admin.translationsQueued')} value={stats.translationsPending} tone="amber" />
                  <StatCard label={t('admin.translationsDone')} value={stats.translationsCompleted} tone="green" />
                  <StatCard label={t('admin.translationsFailed')} value={stats.translationsFailed} tone="red" />
                </div>
                <div className="flex justify-end">
                  <button
                    onClick={loadStats}
                    className="rounded bg-gray-800 px-4 py-2 text-sm font-semibold text-white hover:bg-gray-900"
                  >
                    {t('common.refresh')}
                  </button>
                </div>
              </>
            )}
          </section>
        )}

        {tab === 'users' && (
          <section className="space-y-4">
            <div className="flex flex-wrap items-center gap-3 rounded-lg bg-white p-3 shadow">
              <select
                value={userRoleFilter}
                onChange={e => { setUserRoleFilter(e.target.value); loadUsers() }}
                className="rounded border border-gray-300 px-3 py-2 text-sm"
              >
                <option value="all">{t('admin.allRoles')}</option>
                <option value="member">{t('common.member')}</option>
                <option value="moderator">{t('common.moderator')}</option>
                <option value="admin">{t('common.admin')}</option>
              </select>
              <select
                value={userStatusFilter}
                onChange={e => { setUserStatusFilter(e.target.value); loadUsers() }}
                className="rounded border border-gray-300 px-3 py-2 text-sm"
              >
                <option value="all">{t('admin.anyStatus')}</option>
                <option value="active">{t('common.active')}</option>
                <option value="suspended">{t('admin.suspended')}</option>
                <option value="deleted">{t('admin.deleted')}</option>
              </select>
              <input
                value={q}
                onChange={e => setQ(e.target.value)}
                onKeyDown={e => e.key === 'Enter' && searchNow()}
                placeholder={t('admin.searchNamePlaceholder')}
                className="min-w-0 flex-1 rounded border border-gray-300 px-3 py-2 text-sm"
              />
              <button
                onClick={searchNow}
                className="rounded bg-gray-800 px-4 py-2 text-sm font-semibold text-white hover:bg-gray-900"
              >
                {t('admin.search')}
              </button>
            </div>

            <p className="text-sm text-gray-500">{t('admin.usersCount', { count: users.total })}</p>

            <div className="overflow-hidden rounded-lg bg-white shadow">
              <table className="w-full text-left text-sm">
                <thead className="bg-gray-100 text-gray-600">
                  <tr>
                    <th className="px-4 py-3">{t('admin.user')}</th>
                    <th className="px-4 py-3">{t('admin.languages')}</th>
                    <th className="px-4 py-3">{t('admin.role')}</th>
                    <th className="px-4 py-3">{t('admin.status')}</th>
                    <th className="px-4 py-3">{t('admin.joined', { date: '' }).trim()}</th>
                    <th className="px-4 py-3"></th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100">
                  {users.users.map(user => (
                    <tr key={user.id} className={user.deletedAt ? 'opacity-50' : ''}>
                      <td className="px-4 py-3">
                        <p className="font-medium">{user.displayName || user.username}</p>
                        <p className="text-xs text-gray-500">@{user.username} · {user.email}</p>
                      </td>
                      <td className="px-4 py-3 text-gray-600">
                        <p>{t('admin.speaks')} <strong>{getLanguageName(user.nativeLanguage) || '—'}</strong></p>
                        <p className="text-xs">{t('admin.learning')} {names(user.targetLanguages)}</p>
                      </td>
                      <td className="px-4 py-3">
                        <select
                          value={user.role}
                          disabled={!isAdmin || user.id === useStore.getState().user?.id || busy === user.id}
                          onChange={e => switchRole(user, e.target.value)}
                          className={`rounded border px-2 py-1 text-xs font-semibold ${
                            user.role === 'admin' ? 'border-indigo-300 bg-indigo-50 text-indigo-700'
                            : user.role === 'moderator' ? 'border-sky-300 bg-sky-50 text-sky-700'
                            : 'border-gray-300 bg-white text-gray-700'
                          }`}
                        >
                          <option value="member">{t('common.member')}</option>
                          <option value="moderator">{t('common.moderator')}</option>
                          <option value="admin">{t('common.admin')}</option>
                        </select>
                      </td>
                      <td className="px-4 py-3">
                        {user.deletedAt ? (
                          <span className="rounded-full bg-gray-200 px-2 py-0.5 text-xs font-semibold text-gray-600">{t('admin.deleted')}</span>
                        ) : user.suspendedAt ? (
                          <span className="rounded-full bg-red-100 px-2 py-0.5 text-xs font-semibold text-red-700">{t('admin.suspended')}</span>
                        ) : (
                          <span className="rounded-full bg-green-100 px-2 py-0.5 text-xs font-semibold text-green-700">{t('common.active')}</span>
                        )}
                      </td>
                      <td className="px-4 py-3 text-gray-600">{new Date(user.createdAt).toLocaleDateString()}</td>
                      <td className="px-4 py-3 text-right">
                        {!user.deletedAt && !user.suspendedAt && (
                          <button
                            onClick={() => act(() => adminAPI.suspendUser(user.id), user.id, loadUsers)}
                            disabled={busy === user.id}
                            className="rounded border border-amber-300 px-3 py-1.5 text-xs font-semibold text-amber-700 hover:bg-amber-50 disabled:opacity-50"
                          >
                            {t('admin.suspend')}
                          </button>
                        )}
                        {user.suspendedAt && (
                          <button
                            onClick={() => act(() => adminAPI.unsuspendUser(user.id), user.id, loadUsers)}
                            disabled={busy === user.id}
                            className="rounded bg-green-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-green-700 disabled:opacity-50"
                          >
                            {t('admin.reinstate')}
                          </button>
                        )}
                        {!user.deletedAt && (
                          <button
                            onClick={() => {
                              if (window.confirm(t('admin.deleteConfirm', { name: user.displayName || user.username }))) {
                                act(() => adminAPI.deleteUser(user.id), user.id, loadUsers)
                              }
                            }}
                            disabled={busy === user.id || !isAdmin || user.id === useStore.getState().user?.id}
                            className="ml-2 rounded border border-red-300 px-3 py-1.5 text-xs font-semibold text-red-600 hover:bg-red-50 disabled:opacity-50"
                          >
{t('common.delete')}
                          </button>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </section>
        )}

        {tab === 'translations' && (
          <section className="space-y-4">
            <div className="flex flex-wrap items-center gap-3 rounded-lg bg-white p-3 shadow">
              <select
                value={jobFilter}
                onChange={e => { setJobFilter(e.target.value); }}
                className="rounded border border-gray-300 px-3 py-2 text-sm"
              >
                <option value="all">{t('admin.allStatuses')}</option>
                <option value="pending">{t('admin.pending')}</option>
                <option value="processing">{t('admin.processing')}</option>
                <option value="done">{t('common.done')}</option>
                <option value="failed">{t('admin.failed')}</option>
              </select>
              <input
                value={q}
                onChange={e => setQ(e.target.value)}
                onKeyDown={e => e.key === 'Enter' && searchNow()}
                placeholder={t('admin.searchSourcePlaceholder')}
                className="min-w-0 flex-1 rounded border border-gray-300 px-3 py-2 text-sm"
              />
              <button
                onClick={searchNow}
                className="rounded bg-gray-800 px-4 py-2 text-sm font-semibold text-white hover:bg-gray-900"
              >
                {t('common.refresh')}
              </button>
            </div>

            {isAdmin && providers.length > 0 && (
              <div className="rounded-lg bg-white p-4 shadow">
                <p className="mb-2 text-sm font-semibold text-gray-600">{t('admin.providerHealth')}</p>
                <div className="flex flex-wrap gap-3">
                  {providers.map(p => (
                    <span
                      key={p.name}
                      className={`rounded-full px-3 py-1 text-xs font-semibold ${
                        p.ready ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700'
                      }`}
                      title={p.reason}
                    >
                      {p.name} · {p.ready ? t('common.ready') : t('common.misconfigured')}
                    </span>
                  ))}
                </div>
              </div>
            )}

            {jobs.length === 0 ? (
              <p className="rounded-lg border border-dashed border-gray-300 bg-white p-8 text-center text-gray-500">
                {t('admin.noTranslationJobs')}
              </p>
            ) : (
              <div className="overflow-hidden rounded-lg bg-white shadow">
                <table className="w-full text-left text-sm">
                  <thead className="bg-gray-100 text-gray-600">
                    <tr>
                      <th className="px-4 py-3">{t('admin.sourceTarget')}</th>
                      <th className="px-4 py-3">{t('admin.textResult')}</th>
                      <th className="px-4 py-3">{t('admin.status')}</th>
                      <th className="px-4 py-3">{t('admin.attempts')}</th>
                      <th className="px-4 py-3">{t('admin.created')}</th>
                      <th className="px-4 py-3">{t('admin.lastError')}</th>
                      <th className="px-4 py-3"></th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-100">
                    {jobs.map(job => (
                      <tr key={job.id}>
                        <td className="px-4 py-3">
                          <span className="font-semibold">{getLanguageName(job.sourceLang) || job.sourceLang || 'auto'}</span>
                          {' → '}
                          <span className="font-semibold">{getLanguageName(job.targetLang) || job.targetLang}</span>
                        </td>
                        <td className="max-w-xs px-4 py-3">
                          <p className="truncate text-gray-600">{job.text}</p>
                          {job.result && <p className="truncate text-xs text-gray-500">→ {job.result}</p>}
                        </td>
                        <td className="px-4 py-3">
                          <span className={`rounded-full px-2 py-0.5 text-xs font-semibold ${statusBadge[job.status]}`}>
                            {statusLabel(job.status)}
                          </span>
                        </td>
                        <td className="px-4 py-3 text-gray-600">{job.attempts}</td>
                        <td className="px-4 py-3 text-gray-600">
                          {new Date(job.createdAt).toLocaleString()}
                          {job.completedAt && <p className="text-xs text-gray-400">{t('common.done')} {new Date(job.completedAt).toLocaleTimeString()}</p>}
                        </td>
                        <td className="max-w-xs truncate px-4 py-3 text-red-600">{job.lastError || '—'}</td>
                        <td className="px-4 py-3 text-right">
                          {job.status !== 'done' && (
                            <button
                              onClick={() => act(() => adminAPI.retryTranslation(job.id), job.id, loadTranslations)}
                              disabled={busy === job.id}
                              className="rounded bg-indigo-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-indigo-700 disabled:opacity-50"
                            >
                              {busy === job.id ? t('admin.queuing') : t('admin.retry')}
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

        {tab === 'premium' && (
          <section className="space-y-4">
            {/* Analytics */}
            {analytics && (
              <div className="grid grid-cols-2 gap-4 md:grid-cols-3">
                <StatCard label={t('admin.premiumTotal')} value={analytics.totalPremiumUsers} tone="amber" />
                <StatCard label={t('admin.premiumStored')} value={analytics.storedPremium} tone="green" />
                <StatCard label={t('admin.premiumInGrace')} value={analytics.inGrace} tone="amber" />
                <StatCard label={t('admin.premiumMonthly')} value={analytics.monthlySubscriptions} />
                <StatCard label={t('admin.premiumYearly')} value={analytics.yearlySubscriptions} />
                <StatCard label={t('admin.premiumNew')} value={analytics.newThisMonth} tone="green" />
                <StatCard label={t('admin.premiumChurned')} value={analytics.churnedThisMonth} tone="red" />
                <StatCard label={t('admin.premiumMRR')} value={money(analytics.projectedMRR)} />
                <StatCard label={t('admin.premiumRevenue')} value={money(analytics.revenueLastYear)} />
              </div>
            )}

            <div className="flex flex-wrap items-center gap-3 rounded-lg bg-white p-3 shadow">
              <input
                value={q}
                onChange={e => setQ(e.target.value)}
                onKeyDown={e => e.key === 'Enter' && searchNow()}
                placeholder={t('admin.searchNamePlaceholder')}
                className="min-w-0 flex-1 rounded border border-gray-300 px-3 py-2 text-sm"
              />
              <button onClick={searchNow} className="rounded bg-gray-800 px-4 py-2 text-sm font-semibold text-white hover:bg-gray-900">
                {t('admin.search')}
              </button>
            </div>

            <p className="text-sm text-gray-500">{t('admin.usersCount', { count: premium.total })}</p>

            {premium.users.length === 0 ? (
              <p className="rounded-lg border border-dashed border-gray-300 bg-white p-8 text-center text-gray-500">
                {t('admin.noPremiumUsers')}
              </p>
            ) : (
              <div className="overflow-hidden rounded-lg bg-white shadow">
                <table className="w-full text-left text-sm">
                  <thead className="bg-gray-100 text-gray-600">
                    <tr>
                      <th className="px-4 py-3">{t('admin.user')}</th>
                      <th className="px-4 py-3">{t('admin.premiumPlan')}</th>
                      <th className="px-4 py-3">{t('admin.premiumSince')}</th>
                      <th className="px-4 py-3">{t('admin.premiumBilling')}</th>
                      <th className="px-4 py-3">{t('admin.premiumMessages')}</th>
                      <th className="px-4 py-3"></th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-100">
                    {premium.users.map(u => (
                      <tr key={u.id}>
                        <td className="px-4 py-3">
                          <p className="font-medium">{u.displayName || u.username}</p>
                          <p className="text-xs text-gray-500">@{u.username} · {u.email}</p>
                          {u.inGrace && u.graceUntil && (
                            <p className="mt-0.5 text-xs font-medium text-amber-600">
                              {t('admin.premiumGraceNote', { date: new Date(u.graceUntil).toLocaleDateString() })}
                            </p>
                          )}
                        </td>
                        <td className="px-4 py-3">
                          <div className="flex flex-col items-start gap-1">
                            {planBadge(u)}
                            {u.subscriptionStatus && (
                              <span className="text-xs text-gray-500">{u.subscriptionStatus}</span>
                            )}
                          </div>
                        </td>
                        <td className="px-4 py-3 text-gray-600">
                          {u.premiumSince ? new Date(u.premiumSince).toLocaleDateString() : '—'}
                        </td>
                        <td className="px-4 py-3 text-gray-600">
                          {u.nextBillingDate ? new Date(u.nextBillingDate).toLocaleDateString() : '—'}
                        </td>
                        <td className="px-4 py-3 text-gray-600">{u.messagesSent.toLocaleString()}</td>
                        <td className="px-4 py-3 text-right">
                          <div className="flex flex-wrap justify-end gap-2">
                            <button
                              onClick={() => { setGrantTarget(u); setReason(''); setGrantMode('indefinite'); setGrantDays(30); setGrantUntil('') }}
                              disabled={busy === u.id}
                              className="rounded bg-indigo-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-indigo-700 disabled:opacity-50"
                            >
                              {t('admin.grantPremium')}
                            </button>
                            {u.effectivePlan === 'premium' && (
                              <button
                                onClick={() => { setRevokeTarget(u); setReason(''); setRevokeGrace(0) }}
                                disabled={busy === u.id}
                                className="rounded border border-red-300 px-3 py-1.5 text-xs font-semibold text-red-600 hover:bg-red-50 disabled:opacity-50"
                              >
                                {t('admin.revoke')}
                              </button>
                            )}
                            <button
                              onClick={() => openHistory(u)}
                              disabled={busy === u.id}
                              className="rounded border border-gray-300 px-3 py-1.5 text-xs font-semibold text-gray-700 hover:bg-gray-50 disabled:opacity-50"
                            >
                              {busy === u.id ? t('common.loading') : t('admin.planHistory')}
                            </button>
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </section>
        )}

        {tab === 'reports' && (
          <section className="space-y-4">
            {reportStats && (
              <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
                <StatCard label={t('admin.openReports')} value={reportStats.openReports} tone="red" />
                <StatCard label={t('admin.userReports')} value={reportStats.userReports} />
                <StatCard label={t('admin.messageReports')} value={reportStats.messageReports} />
                <StatCard label={t('admin.resolvedToday')} value={reportStats.resolvedToday} tone="green" />
              </div>
            )}

            <div className="flex flex-wrap items-center gap-3 rounded-lg bg-white p-3 shadow">
              <select
                value={reportFilter}
                onChange={e => {
                  setReportFilter(e.target.value)
                  loadReports(e.target.value, q)
                }}
                className="rounded border border-gray-300 px-3 py-2 text-sm"
              >
                <option value="open">{t('admin.reportOpen')}</option>
                <option value="resolved">{t('admin.reportResolved')}</option>
                <option value="dismissed">{t('admin.reportDismissed')}</option>
                <option value="all">{t('admin.all')}</option>
              </select>
              <input
                value={q}
                onChange={e => setQ(e.target.value)}
                onKeyDown={e => e.key === 'Enter' && searchNow()}
                placeholder={t('admin.searchNamePlaceholder')}
                className="min-w-0 flex-1 rounded border border-gray-300 px-3 py-2 text-sm"
              />
              <button onClick={searchNow} className="rounded bg-gray-800 px-4 py-2 text-sm font-semibold text-white hover:bg-gray-900">
                {t('admin.search')}
              </button>
            </div>

            <p className="text-sm text-gray-500">{t('admin.usersCount', { count: reportTotal })}</p>

            {reports.length === 0 ? (
              <p className="rounded-lg border border-dashed border-gray-300 bg-white p-8 text-center text-gray-500">
                {t('admin.noReports')}
              </p>
            ) : (
              <div className="overflow-hidden rounded-lg bg-white shadow">
                <table className="w-full text-left text-sm">
                  <thead className="bg-gray-100 text-gray-600">
                    <tr>
                      <th className="px-4 py-3">{t('admin.reportType')}</th>
                      <th className="px-4 py-3">{t('admin.reportedUser')}</th>
                      <th className="px-4 py-3">{t('admin.reporter')}</th>
                      <th className="px-4 py-3">{t('admin.reason')}</th>
                      <th className="px-4 py-3">{t('admin.created')}</th>
                      <th className="px-4 py-3"></th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-100">
                    {reports.map(r => (
                      <tr key={r.id} className="align-top">
                        <td className="px-4 py-3">
                          <span className={`rounded-full px-2 py-0.5 text-xs font-semibold ${
                            r.type === 'message' ? 'bg-purple-100 text-purple-700' : 'bg-blue-100 text-blue-700'
                          }`}>
                            {t(r.type === 'message' ? 'admin.reportTypeMessage' : 'admin.reportTypeUser')}
                          </span>
                        </td>
                        <td className="px-4 py-3">
                          <p className="font-medium">{r.reportedUser?.displayName || r.reportedUser?.username}</p>
                          <p className="text-xs text-gray-500">@{r.reportedUser?.username}</p>
                          {r.messageId && <p className="mt-0.5 text-[11px] text-purple-500">{r.messageId.slice(0, 8)}…</p>}
                        </td>
                        <td className="px-4 py-3 text-gray-600">
                          <p className="font-medium">{r.reporter?.displayName || r.reporter?.username}</p>
                          <p className="text-xs text-gray-500">{r.reporter?.email}</p>
                        </td>
                        <td className="px-4 py-3">
                          <p className="max-w-[220px] break-words text-gray-700">{r.reason}</p>
                          {r.resolutionNote && (
                            <p className="mt-1 text-xs italic text-gray-400">{t('admin.resolutionNote')}: {r.resolutionNote}</p>
                          )}
                          <p className="mt-1 text-xs text-gray-400">{new Date(r.createdAt).toLocaleString()}</p>
                        </td>
                        <td className="px-4 py-3 text-xs text-gray-500">
                          {r.status === 'resolved' || r.status === 'dismissed'
                            ? new Date(r.resolvedAt || r.createdAt).toLocaleDateString()
                            : '—'}
                        </td>
                        <td className="px-4 py-3 text-right">
                          {dismissing === r.id ? (
                            <div className="flex flex-col items-end gap-2">
                              <input
                                value={dismissNote}
                                onChange={e => setDismissNote(e.target.value)}
                                placeholder={t('admin.dismissNotePlaceholder')}
                                className="w-48 rounded border border-gray-300 px-2 py-1 text-xs"
                                autoFocus
                              />
                              <div className="flex gap-2">
                                <button
                                  onClick={() => submitDismiss(r.id)}
                                  disabled={busy === r.id}
                                  className="rounded bg-gray-700 px-3 py-1.5 text-xs font-semibold text-white hover:bg-gray-800 disabled:opacity-50"
                                >
                                  {busy === r.id ? t('common.loading') : t('admin.confirmDismiss')}
                                </button>
                                <button
                                  onClick={() => { setDismissing(null); setDismissNote('') }}
                                  className="rounded border border-gray-300 px-3 py-1.5 text-xs text-gray-600 hover:bg-gray-50"
                                >
                                  {t('common.cancel')}
                                </button>
                              </div>
                            </div>
                          ) : (
                            <div className="flex justify-end gap-2">
                              <button
                                onClick={() => resolveReport(r.id)}
                                disabled={busy === r.id || r.status !== 'open'}
                                className="rounded bg-green-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-green-700 disabled:opacity-40"
                              >
                                {busy === r.id ? t('common.loading') : t('admin.resolveReport')}
                              </button>
                              <button
                                onClick={() => { setDismissing(r.id); setDismissNote('') }}
                                disabled={r.status !== 'open'}
                                className="rounded border border-gray-300 px-3 py-1.5 text-xs font-semibold text-gray-700 hover:bg-gray-50 disabled:opacity-40"
                              >
                                {t('admin.dismissReport')}
                              </button>
                            </div>
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
      </div>

      {/* Grant modal */}
      {grantTarget && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4" onClick={() => setGrantTarget(null)}>
          <div className="w-full max-w-md rounded-2xl bg-white p-6 shadow-xl" onClick={e => e.stopPropagation()}>
            <h3 className="mb-1 text-lg font-bold">{t('admin.grantTitle')}</h3>
            <p className="mb-4 text-sm text-gray-500">{grantTarget.displayName || grantTarget.username}</p>
            <div className="space-y-3">
              <select
                value={grantMode}
                onChange={e => setGrantMode(e.target.value as typeof grantMode)}
                className="w-full rounded border border-gray-300 px-3 py-2 text-sm"
              >
                <option value="indefinite">{t('admin.modeIndefinite')}</option>
                <option value="days">{t('admin.modeDays')}</option>
                <option value="until">{t('admin.modeUntil')}</option>
              </select>
              {grantMode === 'days' && (
                <input
                  type="number"
                  min={1}
                  value={grantDays}
                  onChange={e => setGrantDays(Number(e.target.value))}
                  className="w-full rounded border border-gray-300 px-3 py-2 text-sm"
                />
              )}
              {grantMode === 'until' && (
                <input
                  type="datetime-local"
                  value={grantUntil}
                  onChange={e => setGrantUntil(e.target.value)}
                  className="w-full rounded border border-gray-300 px-3 py-2 text-sm"
                />
              )}
              <input
                value={reason}
                onChange={e => setReason(e.target.value)}
                placeholder={t('admin.reasonPlaceholder')}
                className="w-full rounded border border-gray-300 px-3 py-2 text-sm"
              />
            </div>
            <div className="mt-6 flex justify-end gap-2">
              <button onClick={() => setGrantTarget(null)} className="rounded border border-gray-300 px-4 py-2 text-sm font-semibold text-gray-600 hover:bg-gray-50">
                {t('common.cancel')}
              </button>
              <button
                onClick={applyGrant}
                disabled={busy === grantTarget.id || (grantMode === 'until' && !grantUntil)}
                className="rounded bg-indigo-600 px-4 py-2 text-sm font-semibold text-white hover:bg-indigo-700 disabled:opacity-50"
              >
                {busy === grantTarget.id ? t('common.saving') : t('admin.apply')}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Revoke modal */}
      {revokeTarget && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4" onClick={() => setRevokeTarget(null)}>
          <div className="w-full max-w-md rounded-2xl bg-white p-6 shadow-xl" onClick={e => e.stopPropagation()}>
            <h3 className="mb-1 text-lg font-bold">{t('admin.revokeTitle')}</h3>
            <p className="mb-4 text-sm text-gray-500">{revokeTarget.displayName || revokeTarget.username}</p>
            <div className="space-y-3">
              <label className="flex items-center justify-between gap-3 text-sm text-gray-700">
                {t('admin.graceDays')}
                <input
                  type="number"
                  min={0}
                  value={revokeGrace}
                  onChange={e => setRevokeGrace(Number(e.target.value))}
                  className="w-24 rounded border border-gray-300 px-3 py-2 text-sm"
                />
              </label>
              <input
                value={reason}
                onChange={e => setReason(e.target.value)}
                placeholder={t('admin.reasonPlaceholder')}
                className="w-full rounded border border-gray-300 px-3 py-2 text-sm"
              />
            </div>
            <div className="mt-6 flex justify-end gap-2">
              <button onClick={() => setRevokeTarget(null)} className="rounded border border-gray-300 px-4 py-2 text-sm font-semibold text-gray-600 hover:bg-gray-50">
                {t('common.cancel')}
              </button>
              <button
                onClick={applyRevoke}
                disabled={busy === revokeTarget.id}
                className="rounded bg-red-600 px-4 py-2 text-sm font-semibold text-white hover:bg-red-700 disabled:opacity-50"
              >
                {busy === revokeTarget.id ? t('common.saving') : t('admin.apply')}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Plan history modal */}
      {history && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4" onClick={() => setHistory(null)}>
          <div className="max-h-[80vh] w-full max-w-lg overflow-y-auto rounded-2xl bg-white p-6 shadow-xl" onClick={e => e.stopPropagation()}>
            <h3 className="mb-1 text-lg font-bold">{t('admin.historyTitle')}</h3>
            <p className="mb-4 text-sm text-gray-500">{history.user.displayName || history.user.username}</p>
            {history.entries.length === 0 ? (
              <p className="rounded-lg border border-dashed border-gray-300 p-6 text-center text-gray-500">{t('admin.noPlanHistory')}</p>
            ) : (
              <ul className="divide-y divide-gray-100">
                {history.entries.map(entry => (
                  <li key={entry.id} className="py-3">
                    <div className="flex items-center justify-between gap-2">
                      <p className="font-semibold text-gray-900">
                        {t('plan.' + entry.fromPlan)} → {t('plan.' + entry.toPlan)}
                      </p>
                      <span className={`rounded-full px-2 py-0.5 text-xs font-semibold ${
                        entry.toPlan === 'premium' ? 'bg-amber-100 text-amber-700' : 'bg-gray-100 text-gray-600'
                      }`}>
                        {t('admin.' + entry.source)}
                      </span>
                    </div>
                    {entry.graceUntil && (
                      <p className="text-xs text-amber-600">{t('admin.premiumGraceNote', { date: new Date(entry.graceUntil).toLocaleDateString() })}</p>
                    )}
                    {entry.reason && <p className="text-sm text-gray-500">{entry.reason}</p>}
                    <p className="mt-0.5 text-xs text-gray-400">{new Date(entry.createdAt).toLocaleString()}</p>
                  </li>
                ))}
              </ul>
            )}
            <div className="mt-6 flex justify-end">
              <button onClick={() => setHistory(null)} className="rounded border border-gray-300 px-4 py-2 text-sm font-semibold text-gray-600 hover:bg-gray-50">
                {t('common.close')}
              </button>
            </div>
          </div>
        </div>
      )}
    </main>
  )
}

function StatCard({ label, value, tone }: { label: string; value: number | string; tone?: 'amber' | 'green' | 'red' }) {
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