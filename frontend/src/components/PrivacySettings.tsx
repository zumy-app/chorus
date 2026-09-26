import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { settingsAPI } from '../services/api'
import type { PrivacyVisibility } from '@chorus/shared'

const OPTIONS: PrivacyVisibility[] = ['everyone', 'contacts', 'nobody']

export default function PrivacySettings() {
  const { t } = useTranslation()
  const [lastSeen, setLastSeen] = useState<PrivacyVisibility>('everyone')
  const [profilePhoto, setProfilePhoto] = useState<PrivacyVisibility>('everyone')
  const [contacts, setContacts] = useState<PrivacyVisibility>('everyone')
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState<PrivacyVisibility | null>(null)
  const [msg, setMsg] = useState('')

  useEffect(() => {
    let mounted = true
    settingsAPI.getSettings().then(s => {
      if (!mounted) return
      setLastSeen(s.lastSeenVisibility ?? 'everyone')
      setProfilePhoto(s.profilePhotoVisibility ?? 'everyone')
      setContacts(s.contactsVisibility ?? 'everyone')
    }).catch(() => {}).finally(() => { if (mounted) setLoading(false) })
    return () => { mounted = false }
  }, [])

  const save = async (field: 'lastSeenVisibility' | 'profilePhotoVisibility' | 'contactsVisibility', value: PrivacyVisibility) => {
    setSaving(value as any)
    setMsg('')
    try {
      const updated = await settingsAPI.updateSettings({ [field]: value } as any)
      setLastSeen(updated.lastSeenVisibility)
      setProfilePhoto(updated.profilePhotoVisibility)
      setContacts(updated.contactsVisibility)
      setMsg(t('settings.privacySaved'))
      setTimeout(() => setMsg(''), 2000)
    } catch {
      setMsg(t('settings.privacyFailed'))
    } finally {
      setSaving(null)
    }
  }

  const labelFor = (v: PrivacyVisibility) => {
    if (v === 'everyone') return t('settings.everyone')
    if (v === 'contacts') return t('settings.myContacts')
    return t('settings.nobody')
  }

  const Row = ({ icon, title, value, field, setter }: { icon: string, title: string, value: PrivacyVisibility, field: 'lastSeenVisibility' | 'profilePhotoVisibility' | 'contactsVisibility', setter: (v: PrivacyVisibility)=>void }) => (
    <div className="flex items-center justify-between py-3">
      <div className="flex items-center gap-3">
        <span className="material-symbols-outlined text-outline text-[20px]">{icon}</span>
        <div>
          <p className="font-body-md text-body-md text-on-surface">{title}</p>
          <p className="font-label-sm text-label-sm text-outline">{labelFor(value)}</p>
        </div>
      </div>
      <select
        value={value}
        onChange={e => { const v = e.target.value as PrivacyVisibility; setter(v); save(field, v) }}
        disabled={loading || saving !== null}
        className="px-3 py-1.5 border border-outline-variant rounded-lg bg-white text-sm focus:outline-none focus:ring-2 focus:ring-primary"
        aria-label={title}
      >
        {OPTIONS.map(o => <option key={o} value={o}>{labelFor(o)}</option>)}
      </select>
    </div>
  )

  return (
    <div>
      <div className="flex items-center gap-3 mb-2">
        <div className="w-10 h-10 rounded-full bg-tertiary-container/20 flex items-center justify-center text-tertiary">
          <span className="material-symbols-outlined">visibility</span>
        </div>
        <div>
          <h3 className="font-body-md font-semibold text-on-surface">{t('settings.privacy')}</h3>
          <p className="font-body-sm text-body-sm text-on-surface-variant">{t('settings.privacyDesc')}</p>
        </div>
      </div>
      {msg && <p className={`text-xs mb-2 ${msg === t('settings.privacySaved') ? 'text-green-600' : 'text-error'}`}>{msg}</p>}
      <div className="divide-y divide-outline-variant/40">
        <Row icon="schedule" title={t('settings.lastSeen')} value={lastSeen} field="lastSeenVisibility" setter={setLastSeen} />
        <Row icon="account_circle" title={t('settings.profilePhoto')} value={profilePhoto} field="profilePhotoVisibility" setter={setProfilePhoto} />
        <Row icon="contacts" title={t('settings.contacts')} value={contacts} field="contactsVisibility" setter={setContacts} />
      </div>
    </div>
  )
}
