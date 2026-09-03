import { useEffect, useState } from 'react'
import { otpAPI } from '../services/api'
import type { PhoneStatus } from '@chorus/shared'

export default function TwoFactorSettings() {
  const [status, setStatus] = useState<PhoneStatus | null>(null)
  const [phone, setPhone] = useState('')
  const [code, setCode] = useState('')
  const [loading, setLoading] = useState(false)
  const [msg, setMsg] = useState('')

  const load = async () => {
    try { setStatus(await otpAPI.getPhoneStatus()) } catch {}
  }
  useEffect(() => { load() }, [])

  const requestOTP = async () => {
    setLoading(true); setMsg('')
    try { const r = await otpAPI.requestOTP(phone); setMsg(`Code sent to ${r.phoneMasked}`) } catch (e: any) { setMsg(e.response?.data?.error || 'Failed to send code') }
    finally { setLoading(false) }
  }
  const verify = async () => {
    setLoading(true); setMsg('')
    try { await otpAPI.verifyPhone(phone, code); setMsg('Phone verified'); setCode(''); await load() } catch (e: any) { setMsg(e.response?.data?.error || 'Invalid code') }
    finally { setLoading(false) }
  }
  const toggle2FA = async () => {
    if (!status) return
    setLoading(true); setMsg('')
    try { const s = await otpAPI.setTwoFactor(!status.twoFactorEnabled); setStatus(s); setMsg(s.twoFactorEnabled ? '2FA enabled' : '2FA disabled') } catch (e: any) { setMsg(e.response?.data?.error || 'Failed to update 2FA') }
    finally { setLoading(false) }
  }

  return (
    <div className="space-y-4">
      <h3 className="font-label-md text-label-md text-on-surface">Two-factor authentication</h3>
      {status && (
        <div className="text-sm text-on-surface-variant space-y-1">
          <p>Phone: {status.phoneMasked || 'not set'} {status.phoneVerified ? '✓ verified' : ''}</p>
          <p>2FA: {status.twoFactorEnabled ? 'enabled' : 'disabled'}</p>
        </div>
      )}
      <div className="flex gap-2">
        <input value={phone} onChange={e=>setPhone(e.target.value)} placeholder="+14155551234" className="flex-1 px-3 py-2 border rounded-lg text-sm" />
        <button onClick={requestOTP} disabled={loading || !phone} className="px-4 py-2 bg-primary text-on-primary rounded-lg text-sm disabled:opacity-50">Send code</button>
      </div>
      <div className="flex gap-2">
        <input value={code} onChange={e=>setCode(e.target.value)} placeholder="123456" maxLength={6} className="flex-1 px-3 py-2 border rounded-lg text-sm" />
        <button onClick={verify} disabled={loading || code.length!==6} className="px-4 py-2 bg-primary-container text-on-primary-container rounded-lg text-sm disabled:opacity-50">Verify</button>
      </div>
      <button onClick={toggle2FA} disabled={loading || !status?.phoneVerified} className="w-full py-2 border rounded-lg text-sm disabled:opacity-50">
        {status?.twoFactorEnabled ? 'Disable 2FA' : 'Enable 2FA'}
      </button>
      {!status?.phoneVerified && <p className="text-xs text-outline">Verify your phone before enabling 2FA.</p>}
      {msg && <p className="text-sm text-on-surface-variant">{msg}</p>}
    </div>
  )
}
