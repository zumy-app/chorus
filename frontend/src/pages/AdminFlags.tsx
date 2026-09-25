import { useCallback, useEffect, useState } from 'react'
import { adminAPI } from '../services/api'
import { apiErrorMessage, type FeatureFlagDefinition } from '@chorus/shared'

// Admin Flags console — internal tool, English-only by design (not part of
// the localized product surface). Lets the admin:
//   1. flip audience tiers per flag (admin-only / beta / stable),
//   2. set or delete per-user overrides,
//   3. preview the resolved flag map exactly as any user sees it.
// Route-guarded to admins in App.tsx; the API enforces RequireRole(admin).
export default function AdminFlags() {
  const [flags, setFlags] = useState<FeatureFlagDefinition[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const [savingKey, setSavingKey] = useState<string | null>(null)

  // Preview-as-user state.
  const [previewId, setPreviewId] = useState('')
  const [previewFlags, setPreviewFlags] = useState<Record<string, boolean> | null>(null)
  const [previewLoading, setPreviewLoading] = useState(false)

  // Override editor state.
  const [overrideKey, setOverrideKey] = useState('')
  const [overrideUser, setOverrideUser] = useState('')
  const [overrideOn, setOverrideOn] = useState(true)
  const [userQuery, setUserQuery] = useState('')
  const [userHits, setUserHits] = useState<{ id: string; email: string; displayName: string }[]>([])

  const load = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      setFlags(await adminAPI.listFlags())
    } catch (e) {
      setError(apiErrorMessage(e, 'Failed to load flags'))
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { load() }, [load])

  const flash = (msg: string) => {
    setNotice(msg)
    window.setTimeout(() => setNotice(''), 4000)
  }

  const saveTiers = async (key: string, tiers: { adminOnly: boolean; betaAccess: boolean; stable: boolean }) => {
    setSavingKey(key)
    setError('')
    try {
      await adminAPI.updateFlagTiers(key, tiers)
      setFlags(prev => prev.map(f => (f.key === key ? { ...f, ...tiers } : f)))
      flash(`Saved tiers for ${key}`)
    } catch (e) {
      setError(apiErrorMessage(e, `Failed to save ${key}`))
    } finally {
      setSavingKey(null)
    }
  }

  const runPreview = async () => {
    const id = previewId.trim()
    if (!id) return
    setPreviewLoading(true)
    setError('')
    try {
      setPreviewFlags(await adminAPI.previewUserFlags(id))
    } catch (e) {
      setError(apiErrorMessage(e, 'Preview failed — check the user id'))
      setPreviewFlags(null)
    } finally {
      setPreviewLoading(false)
    }
  }

  const lookupUsers = async () => {
    const q = userQuery.trim()
    if (!q) return
    try {
      const { users } = await adminAPI.listUsers({ q, limit: 10 })
      setUserHits(users.map(u => ({ id: u.id, email: u.email, displayName: u.displayName })))
    } catch (e) {
      setError(apiErrorMessage(e, 'User lookup failed'))
    }
  }

  const saveOverride = async () => {
    if (!overrideKey || !overrideUser.trim()) return
    setError('')
    try {
      await adminAPI.setFlagOverride(overrideKey, overrideUser.trim(), overrideOn)
      flash(`Override set: ${overrideKey} = ${overrideOn ? 'on' : 'off'} for ${overrideUser.trim()}`)
    } catch (e) {
      setError(apiErrorMessage(e, 'Failed to set override'))
    }
  }

  const removeOverride = async () => {
    if (!overrideKey || !overrideUser.trim()) return
    setError('')
    try {
      await adminAPI.deleteFlagOverride(overrideKey, overrideUser.trim())
      flash(`Override removed: ${overrideKey} for ${overrideUser.trim()}`)
    } catch (e) {
      setError(apiErrorMessage(e, 'Failed to delete override'))
    }
  }

  return (
    <div className="min-h-screen bg-background text-on-surface p-6 max-w-5xl mx-auto flex flex-col gap-8" data-testid="admin-flags-page">
      <header>
        <h1 className="font-headline-lg text-headline-lg">Feature flags</h1>
        <p className="font-body-sm text-body-sm text-on-surface-variant mt-1">
          Admin-only console. Tier changes apply instantly (30s client cache). Stable = general users see it.
        </p>
      </header>

      {error && <div className="bg-error-container text-on-error-container rounded-xl px-4 py-3 text-sm">{error}</div>}
      {notice && <div className="bg-tertiary-container text-on-tertiary-container rounded-xl px-4 py-3 text-sm">{notice}</div>}

      <section>
        <h2 className="font-headline-sm text-headline-sm mb-3">Flag tiers</h2>
        {loading ? (
          <p className="text-on-surface-variant">Loading flags…</p>
        ) : (
          <div className="flex flex-col gap-2">
            {flags.map(f => (
              <FlagRow
                key={f.key}
                flag={f}
                saving={savingKey === f.key}
                onSave={(tiers) => saveTiers(f.key, tiers)}
              />
            ))}
          </div>
        )}
      </section>

      <section>
        <h2 className="font-headline-sm text-headline-sm mb-3">Preview as user</h2>
        <p className="font-body-sm text-body-sm text-on-surface-variant mb-2">
          Resolves exactly as a general user — this is what that account sees.
        </p>
        <div className="flex gap-2">
          <input
            value={previewId}
            onChange={e => setPreviewId(e.target.value)}
            placeholder="User id"
            data-testid="preview-user-id"
            className="flex-1 bg-surface-container-lowest border border-outline-variant/30 rounded-xl px-4 py-2.5"
          />
          <button onClick={runPreview} disabled={previewLoading} className="bg-primary text-on-primary px-5 py-2.5 rounded-full">
            {previewLoading ? '…' : 'Preview'}
          </button>
        </div>
        {previewFlags && (
          <div data-testid="preview-grid" className="mt-3 grid grid-cols-2 md:grid-cols-3 gap-2">
            {Object.entries(previewFlags).map(([k, v]) => (
              <div key={k} className="flex items-center justify-between bg-surface-container-lowest rounded-xl px-3 py-2 text-sm">
                <span className="truncate">{k}</span>
                <span className={v ? 'text-tertiary font-semibold' : 'text-on-surface-variant'}>{v ? 'ON' : 'off'}</span>
              </div>
            ))}
          </div>
        )}
      </section>

      <section>
        <h2 className="font-headline-sm text-headline-sm mb-3">Per-user override</h2>
        <div className="flex gap-2 mb-2">
          <input
            value={userQuery}
            onChange={e => setUserQuery(e.target.value)}
            placeholder="Find user by email"
            className="flex-1 bg-surface-container-lowest border border-outline-variant/30 rounded-xl px-4 py-2.5"
          />
          <button onClick={lookupUsers} className="bg-surface-container-high px-5 py-2.5 rounded-full">Find</button>
        </div>
        {userHits.length > 0 && (
          <div className="flex flex-col gap-1 mb-3">
            {userHits.map(u => (
              <button key={u.id} onClick={() => { setOverrideUser(u.id); setUserQuery(u.email) }} className="text-left bg-surface-container-lowest rounded-xl px-3 py-2 text-sm hover:border-primary">
                {u.displayName} <span className="text-on-surface-variant">· {u.email}</span>
              </button>
            ))}
          </div>
        )}
        <div className="flex flex-col md:flex-row gap-2">
          <select
            value={overrideKey}
            onChange={e => setOverrideKey(e.target.value)}
            className="flex-1 bg-surface-container-lowest border border-outline-variant/30 rounded-xl px-4 py-2.5"
          >
            <option value="">Select flag…</option>
            {flags.map(f => (
              <option key={f.key} value={f.key}>{f.key}</option>
            ))}
          </select>
          <input
            value={overrideUser}
            onChange={e => setOverrideUser(e.target.value)}
            placeholder="User id"
            data-testid="override-user-id"
            className="flex-1 bg-surface-container-lowest border border-outline-variant/30 rounded-xl px-4 py-2.5"
          />
          <button
            onClick={() => setOverrideOn(v => !v)}
            className={`px-5 py-2.5 rounded-full ${overrideOn ? 'bg-tertiary-container text-on-tertiary-container' : 'bg-surface-container-high'}`}
          >
            {overrideOn ? 'ON' : 'OFF'}
          </button>
          <button onClick={saveOverride} className="bg-primary text-on-primary px-5 py-2.5 rounded-full">Set</button>
          <button onClick={removeOverride} className="bg-surface-container-high px-5 py-2.5 rounded-full">Delete</button>
        </div>
      </section>
    </div>
  )
}

function FlagRow({
  flag,
  saving,
  onSave,
}: {
  flag: FeatureFlagDefinition
  saving: boolean
  onSave: (tiers: { adminOnly: boolean; betaAccess: boolean; stable: boolean }) => void
}) {
  const [adminOnly, setAdminOnly] = useState(flag.adminOnly)
  const [betaAccess, setBetaAccess] = useState(flag.betaAccess)
  const [stable, setStable] = useState(flag.stable)
  const dirty = adminOnly !== flag.adminOnly || betaAccess !== flag.betaAccess || stable !== flag.stable

  return (
    <div data-testid={`flag-row-${flag.key}`} className="bg-surface-container-lowest rounded-xl px-4 py-3 flex flex-col md:flex-row md:items-center gap-2">
      <div className="flex-1 min-w-0">
        <div className="font-label-md text-label-md">{flag.key}</div>
        <div className="font-body-sm text-body-sm text-on-surface-variant truncate">{flag.description}</div>
      </div>
      <label className="flex items-center gap-1 text-sm">
        <input type="checkbox" checked={adminOnly} onChange={e => setAdminOnly(e.target.checked)} /> admin
      </label>
      <label className="flex items-center gap-1 text-sm">
        <input type="checkbox" checked={betaAccess} onChange={e => setBetaAccess(e.target.checked)} /> beta
      </label>
      <label className="flex items-center gap-1 text-sm">
        <input type="checkbox" checked={stable} onChange={e => setStable(e.target.checked)} /> stable
      </label>
      <button
        onClick={() => onSave({ adminOnly, betaAccess, stable })}
        disabled={!dirty || saving}
        className="bg-primary text-on-primary px-4 py-1.5 rounded-full text-sm disabled:opacity-40"
      >
        {saving ? '…' : 'Save'}
      </button>
    </div>
  )
}
