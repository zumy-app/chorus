import { FormEvent, useState } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import LanguagePicker from '../components/LanguagePicker'
import { waitlistAPI } from '../services/api'

const SPEAK_TOP_CODES = ['en', 'es', 'fr', 'de', 'hi', 'zh', 'ar', 'pt']
const LEARN_TOP_CODES = ['es', 'fr', 'de', 'it', 'ja', 'zh', 'ko', 'en']

// Reason values are sent to the backend verbatim; localized labels are resolved
// at render time (reasonFriends/reasonLearn/reasonWork/reasonTravel).
const REASONS: { value: string; labelKey: string }[] = [
  { value: 'Connect with friends or family', labelKey: 'waitlist.reasonFriends' },
  { value: 'Learn a new language', labelKey: 'waitlist.reasonLearn' },
  { value: 'For work', labelKey: 'waitlist.reasonWork' },
  { value: 'For travel', labelKey: 'waitlist.reasonTravel' },
]

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

export default function Waitlist() {
  const { t } = useTranslation()
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
      setError(t('waitlist.invalidEmail'))
      return
    }
    if (!spokenLanguages.length) {
      setError(t('waitlist.chooseSpoken'))
      return
    }
    if (!targetLanguages.length) {
      setError(t('waitlist.chooseLearn'))
      return
    }
    if (!reasons.length) {
      setError(t('waitlist.chooseReason'))
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
      setError(err?.response?.data?.error || t('waitlist.joinFailed'))
    } finally { setLoading(false) }
  }

  if (queuePosition !== null) return (
    <main className="min-h-screen grid place-items-center bg-gradient-to-br from-primary to-secondary p-6">
      <section className="max-w-md rounded-xl bg-white p-8 text-center shadow-xl">
        <h1 className="text-3xl font-bold text-gray-900">{alreadyJoined ? t('waitlist.welcomeBack') : t('waitlist.youAreOnList')}</h1>
        <p className="mt-3 rounded-lg bg-indigo-50 p-3 text-sm text-indigo-800">
          {alreadyJoined
            ? t('waitlist.alreadyOnList', { position: queuePosition })
            : t('waitlist.yourNumber', { position: queuePosition })}
        </p>
        {serverMessage && <p className="mt-3 text-sm text-gray-600">{serverMessage}</p>}
        <p className="mt-3 text-sm text-gray-600">
          {t('waitlist.inboxHint', { what: alreadyJoined ? t('waitlist.anUpdate') : t('waitlist.aConfirmation') })}
        </p>
        <a
          href="https://discord.gg/7DVwM6jsS"
          target="_blank"
          rel="noreferrer"
          className="mt-4 inline-block rounded-full bg-[#5865F2] px-5 py-2 text-sm font-semibold text-white"
        >
          {t('waitlist.joinDiscord')}
        </a>
        <Link to="/" className="mt-6 block font-semibold text-indigo-600">{t('waitlist.backToChorus')}</Link>
      </section>
    </main>
  )

  return (
    <main className="min-h-screen bg-gradient-to-br from-primary to-secondary py-10 px-4">
      <form onSubmit={submit} noValidate className="mx-auto max-w-xl rounded-xl bg-white p-8 shadow-xl space-y-6">
        <div>
          <h1 className="text-3xl font-bold text-gray-900">{t('waitlist.joinTheWaitlist')}</h1>
          <p className="mt-2 text-gray-600">{t('waitlist.intro')}</p>
        </div>
        {error && <p className="rounded bg-red-100 p-3 text-red-700">{error}</p>}

        <label className="block font-semibold text-gray-700">
          {t('common.email')}
          <input
            required
            type="email"
            value={email}
            onChange={e => setEmail(e.target.value)}
            placeholder={t('common.emailPlaceholder')}
            className="mt-2 w-full rounded-lg border border-gray-300 px-4 py-3 focus:border-primary focus:outline-none"
          />
        </label>

        <LanguagePicker
          label={t('waitlist.iSpeak')}
          hint={t('waitlist.iSpeakHint')}
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
          label={t('waitlist.iWantToLearn')}
          hint={t('waitlist.iWantToLearnHint')}
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
          <legend className="font-semibold text-gray-700">{t('waitlist.reason')}</legend>
          <div className="mt-2 grid gap-2 sm:grid-cols-2">
            {REASONS.map(reason => (
              <label
                key={reason.value}
                className={`flex cursor-pointer items-center gap-2 rounded-lg border p-3 text-sm transition ${
                  reasons.includes(reason.value)
                    ? 'border-primary bg-indigo-50 text-gray-900'
                    : 'border-gray-300 text-gray-700 hover:border-primary'
                }`}
              >
                <input
                  type="checkbox"
                  checked={reasons.includes(reason.value)}
                  onChange={() => toggle(reason.value, setReasons)}
                  className="rounded text-indigo-600 focus:ring-indigo-500"
                />
                {t(reason.labelKey)}
              </label>
            ))}
          </div>
        </fieldset>

        <label className="block">
          <span className="font-semibold text-gray-700">{t('waitlist.additionalComments')} <span className="font-normal text-gray-400">{t('waitlist.optional')}</span></span>
          <textarea
            value={comments}
            onChange={e => setComments(e.target.value)}
            rows={3}
            maxLength={500}
            placeholder={t('waitlist.commentsPlaceholder')}
            className="mt-2 w-full resize-none rounded-lg border border-gray-300 px-4 py-3 focus:border-primary focus:outline-none"
          />
        </label>

        <button disabled={loading} className="w-full rounded-lg bg-indigo-600 py-3 font-bold text-white disabled:opacity-50">
          {loading ? t('waitlist.joining') : t('waitlist.joinTheWaitlist')}
        </button>
        <p className="text-center text-sm text-gray-600">
          {t('waitlist.alreadyHaveAccount')}{' '}<Link to="/login" className="font-semibold text-indigo-600">{t('waitlist.logIn')}</Link>
        </p>
      </form>
    </main>
  )
}