import { FormEvent, useState } from 'react'
import { Link } from 'react-router-dom'
import LanguagePicker from '../components/LanguagePicker'
import { waitlistAPI } from '../services/api'

const SPEAK_TOP_CODES = ['en', 'es', 'fr', 'de', 'hi', 'zh', 'ar', 'pt']
const LEARN_TOP_CODES = ['es', 'fr', 'de', 'it', 'ja', 'zh', 'ko', 'en']

const REASONS = [
  'Connect with friends or family',
  'Learn a new language',
  'For work',
  'For travel',
]

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

export default function Waitlist() {
  const [email, setEmail] = useState('')
  const [spokenLanguages, setSpokenLanguages] = useState<string[]>([])
  const [targetLanguages, setTargetLanguages] = useState<string[]>([])
  const [reasons, setReasons] = useState<string[]>([])
  const [comments, setComments] = useState('')
  const [queuePosition, setQueuePosition] = useState<number | null>(null)
  const [alreadyJoined, setAlreadyJoined] = useState(false)
  const [serverMessage, setServerMessage] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const toggle = (value: string, set: React.Dispatch<React.SetStateAction<string[]>>) =>
    set(current => current.includes(value) ? current.filter(item => item !== value) : [...current, value])

  const submit = async (event: FormEvent) => {
    event.preventDefault()
    setError('')
    const trimmedEmail = email.trim()
    if (!EMAIL_RE.test(trimmedEmail)) {
      setError('Please enter a valid email address.')
      return
    }
    if (!spokenLanguages.length) {
      setError('Choose the language(s) you speak.')
      return
    }
    if (!targetLanguages.length) {
      setError('Choose at least one language you want to learn.')
      return
    }
    if (!reasons.length) {
      setError('Pick at least one reason for joining.')
      return
    }
    setLoading(true)
    try {
      const result = await waitlistAPI.join({
        email: trimmedEmail,
        spokenLanguages,
        targetLanguages,
        reasons,
        comments,
      })
      setQueuePosition(result.entry.queuePosition)
      setAlreadyJoined(!!result.alreadyJoined)
      setServerMessage(result.message || '')
    } catch (err: any) {
      setError(err?.response?.data?.error || 'Unable to join the waitlist.')
    } finally { setLoading(false) }
  }

  if (queuePosition !== null) return (
    <main className="min-h-screen grid place-items-center bg-gradient-to-br from-primary to-secondary p-6">
      <section className="max-w-md rounded-xl bg-white p-8 text-center shadow-xl">
        <h1 className="text-3xl font-bold text-gray-900">{alreadyJoined ? 'Welcome back!' : 'You’re on the list'}</h1>
        <p className="mt-3 rounded-lg bg-indigo-50 p-3 text-sm text-indigo-800">
          {alreadyJoined
            ? `You’re already on the waitlist — we’ve updated your preferences. Your spot in line is #${queuePosition}.`
            : `Your waitlist number is ${queuePosition}.`}
        </p>
        {serverMessage && <p className="mt-3 text-sm text-gray-600">{serverMessage}</p>}
        <p className="mt-3 text-sm text-gray-600">
          Keep an eye on your inbox — and check your <strong>spam/junk folder</strong> — for
          {alreadyJoined ? ' an update' : ' a confirmation'} from <strong>info@chorus.talk</strong>.
          We’ll email you there when your invitation is ready.
        </p>
        <a
          href="https://discord.gg/7DVwM6jsS"
          target="_blank"
          rel="noreferrer"
          className="mt-4 inline-block rounded-full bg-[#5865F2] px-5 py-2 text-sm font-semibold text-white"
        >
          Join the Chorus Discord
        </a>
        <Link to="/" className="mt-6 block font-semibold text-indigo-600">Back to Chorus</Link>
      </section>
    </main>
  )

  return (
    <main className="min-h-screen bg-gradient-to-br from-primary to-secondary py-10 px-4">
      <form onSubmit={submit} className="mx-auto max-w-xl rounded-xl bg-white p-8 shadow-xl space-y-6">
        <div>
          <h1 className="text-3xl font-bold text-gray-900">Join the waitlist</h1>
          <p className="mt-2 text-gray-600">Tell us what you want to learn. We’ll invite you as Chorus opens up.</p>
        </div>
        {error && <p className="rounded bg-red-100 p-3 text-red-700">{error}</p>}

        <label className="block font-semibold text-gray-700">
          Email
          <input
            required
            type="email"
            value={email}
            onChange={e => setEmail(e.target.value)}
            placeholder="you@example.com"
            className="mt-2 w-full rounded-lg border border-gray-300 px-4 py-3 focus:border-primary focus:outline-none"
          />
        </label>

        <LanguagePicker
          label="I speak"
          hint="Pick every language you already know."
          multiple
          selected={spokenLanguages}
          topCodes={SPEAK_TOP_CODES}
          onChange={(code, add) =>
            add
              ? setSpokenLanguages(current => current.includes(code) ? current : [...current, code])
              : setSpokenLanguages(current => current.filter(item => item !== code))
          }
        />

        <LanguagePicker
          label="I want to learn"
          hint="Pick as many as you like."
          multiple
          selected={targetLanguages}
          topCodes={LEARN_TOP_CODES}
          exclude={spokenLanguages}
          onChange={(code, add) =>
            add
              ? setTargetLanguages(current => current.includes(code) ? current : [...current, code])
              : setTargetLanguages(current => current.filter(item => item !== code))
          }
        />

        <fieldset>
          <legend className="font-semibold text-gray-700">Reason for using Chorus.talk</legend>
          <div className="mt-2 grid gap-2 sm:grid-cols-2">
            {REASONS.map(reason => (
              <label
                key={reason}
                className={`flex cursor-pointer items-center gap-2 rounded-lg border p-3 text-sm transition ${
                  reasons.includes(reason)
                    ? 'border-primary bg-indigo-50 text-gray-900'
                    : 'border-gray-300 text-gray-700 hover:border-primary'
                }`}
              >
                <input
                  type="checkbox"
                  checked={reasons.includes(reason)}
                  onChange={() => toggle(reason, setReasons)}
                  className="rounded text-indigo-600 focus:ring-indigo-500"
                />
                {reason}
              </label>
            ))}
          </div>
        </fieldset>

        <label className="block">
          <span className="font-semibold text-gray-700">Additional comments <span className="font-normal text-gray-400">(optional)</span></span>
          <textarea
            value={comments}
            onChange={e => setComments(e.target.value)}
            rows={3}
            maxLength={500}
            placeholder="Anything else you’d like us to know?"
            className="mt-2 w-full resize-none rounded-lg border border-gray-300 px-4 py-3 focus:border-primary focus:outline-none"
          />
        </label>

        <button disabled={loading} className="w-full rounded-lg bg-indigo-600 py-3 font-bold text-white disabled:opacity-50">
          {loading ? 'Joining…' : 'Join the waitlist'}
        </button>
        <p className="text-center text-sm text-gray-600">
          Already have an account? <Link to="/login" className="font-semibold text-indigo-600">Log in</Link>
        </p>
      </form>
    </main>
  )
}