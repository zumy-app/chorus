import { FormEvent, useState } from 'react'
import { Link } from 'react-router-dom'
import { SUPPORTED_LANGUAGES } from '../services/language'
import { waitlistAPI } from '../services/api'

const REASONS = [
  'I want to learn a new language',
  "I have a family member/friend who doesn't speak my language",
  'For work',
  'For travel',
]

export default function Waitlist() {
  const [email, setEmail] = useState('')
  const [spokenLanguage, setSpokenLanguage] = useState('en')
  const [targetLanguages, setTargetLanguages] = useState<string[]>([])
  const [reasons, setReasons] = useState<string[]>([])
  const [queuePosition, setQueuePosition] = useState<number | null>(null)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const toggle = (value: string, set: React.Dispatch<React.SetStateAction<string[]>>) =>
    set(current => current.includes(value) ? current.filter(item => item !== value) : [...current, value])

  const submit = async (event: FormEvent) => {
    event.preventDefault()
    setError('')
    if (!targetLanguages.length || !reasons.length) {
      setError('Choose at least one language to learn and one reason.')
      return
    }
    setLoading(true)
    try {
      const result = await waitlistAPI.join({ email, spokenLanguage, targetLanguages, reasons })
      setQueuePosition(result.entry.queuePosition)
    } catch (err: any) {
      setError(err?.response?.data?.error || 'Unable to join the waitlist.')
    } finally { setLoading(false) }
  }

  if (queuePosition !== null) return (
    <main className="min-h-screen grid place-items-center bg-gradient-to-br from-primary to-secondary p-6">
      <section className="max-w-md rounded-xl bg-white p-8 text-center shadow-xl">
        <h1 className="text-3xl font-bold text-gray-900">You’re on the list</h1>
        <p className="mt-4 text-gray-600">Your waitlist number is <strong>#{queuePosition}</strong>. We’ll email you when an invitation is ready.</p>
        <Link to="/" className="mt-6 inline-block font-semibold text-indigo-600">Back to Chorus</Link>
      </section>
    </main>
  )

  return (
    <main className="min-h-screen bg-gradient-to-br from-primary to-secondary py-10 px-4">
      <form onSubmit={submit} className="mx-auto max-w-xl rounded-xl bg-white p-8 shadow-xl space-y-6">
        <div><h1 className="text-3xl font-bold text-gray-900">Join the waitlist</h1><p className="mt-2 text-gray-600">Tell us what you want to learn. We’ll invite you as Chorus opens up.</p></div>
        {error && <p className="rounded bg-red-100 p-3 text-red-700">{error}</p>}
        <label className="block font-semibold text-gray-700">Email<input required type="email" value={email} onChange={e => setEmail(e.target.value)} className="mt-2 w-full rounded border p-3" /></label>
        <label className="block font-semibold text-gray-700">I speak<select value={spokenLanguage} onChange={e => setSpokenLanguage(e.target.value)} className="mt-2 w-full rounded border p-3">{SUPPORTED_LANGUAGES.map(language => <option key={language.code} value={language.code}>{language.name}</option>)}</select></label>
        <fieldset><legend className="font-semibold text-gray-700">I want to learn</legend><div className="mt-2 grid grid-cols-2 gap-2">{SUPPORTED_LANGUAGES.map(language => <label key={language.code} className="flex gap-2"><input type="checkbox" checked={targetLanguages.includes(language.code)} onChange={() => toggle(language.code, setTargetLanguages)} />{language.name}</label>)}</div></fieldset>
        <fieldset><legend className="font-semibold text-gray-700">Reason for using Chorus.talk</legend><div className="mt-2 space-y-2">{REASONS.map(reason => <label key={reason} className="flex gap-2"><input type="checkbox" checked={reasons.includes(reason)} onChange={() => toggle(reason, setReasons)} />{reason}</label>)}</div></fieldset>
        <button disabled={loading} className="w-full rounded-lg bg-indigo-600 py-3 font-bold text-white disabled:opacity-50">{loading ? 'Joining…' : 'Join the waitlist'}</button>
        <p className="text-center text-sm text-gray-600">Already have an account? <Link to="/login" className="font-semibold text-indigo-600">Log in</Link></p>
      </form>
    </main>
  )
}
