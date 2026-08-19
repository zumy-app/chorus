import { FormEvent, useRef, useState } from 'react'
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

type FieldErrors = { email?: string; spoken?: string; learn?: string; reasons?: string }

export default function Waitlist() {
  const { t } = useTranslation()
  const [email, setEmail] = useState('')
  const [spokenLanguages, setSpokenLanguages] = useState<string[]>([])
  const [targetLanguages, setTargetLanguages] = useState<string[]>([])
  const [targetLanguageLevel, setTargetLanguageLevel] = useState('A1')
  const [reasons, setReasons] = useState<string[]>([])
  const [comments, setComments] = useState('')
  const [queuePosition, setQueuePosition] = useState<number | null>(null)
  const [alreadyJoined, setAlreadyJoined] = useState(false)
  const [serverMessage, setServerMessage] = useState('')
  const [error, setError] = useState('')
  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({})
  const [loading, setLoading] = useState(false)
  const toggle = (value: string, set: React.Dispatch<React.SetStateAction<string[]>>) =>
    set(current => current.includes(value) ? current.filter(item => item !== value) : [...current, value])

  const emailRef = useRef<HTMLInputElement>(null)
  const spokenRef = useRef<HTMLDivElement>(null)
  const learnRef = useRef<HTMLDivElement>(null)
  const reasonsRef = useRef<HTMLDivElement>(null)

  const clearFieldError = (field: keyof FieldErrors) =>
    setFieldErrors(current => (current[field] ? { ...current, [field]: undefined } : current))

  const scrollToFirstError = (errs: FieldErrors) => {
    const fields: Array<[string | undefined, React.RefObject<HTMLElement> | null]> = [
      [errs.email, emailRef],
      [errs.spoken, spokenRef],
      [errs.learn, learnRef],
      [errs.reasons, reasonsRef],
    ]
    for (const [message, ref] of fields) {
      if (message && ref?.current) {
        ref.current.scrollIntoView({ behavior: 'smooth', block: 'center' })
        if (ref === emailRef) ref.current.focus()
        return
      }
    }
  }

  const submit = async (event: FormEvent) => {
    event.preventDefault()
    setError('')
    const trimmedEmail = email.trim()
    const errs: FieldErrors = {}
    if (!EMAIL_RE.test(trimmedEmail)) errs.email = t('waitlist.invalidEmail')
    if (!spokenLanguages.length) errs.spoken = t('waitlist.chooseSpoken')
    if (!reasons.length) errs.reasons = t('waitlist.chooseReason')
    if (Object.keys(errs).length > 0) {
      setFieldErrors(errs)
      scrollToFirstError(errs)
      return
    }
    setLoading(true)
    try {
      const result = await waitlistAPI.join({
        email: trimmedEmail,
        spokenLanguages,
        targetLanguages,
        targetLanguageLevel,
        reasons,
        comments,
      })
      setQueuePosition(result.entry.queuePosition)
      setAlreadyJoined(!!result.alreadyJoined)
      setServerMessage(result.message || '')
    } catch (err: any) {
      setError(err?.response?.data?.error || t('waitlist.joinFailed'))
      window.scrollTo({ top: 0, behavior: 'smooth' })
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
        <p className="mt-3 rounded-lg bg-amber-50 px-3 py-2 text-sm text-amber-800">
          {t('waitlist.spamAndAddressBook')}
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

        <label className="block font-semibold text-gray-700">
          {t('common.email')}
          <input
            ref={emailRef}
            required
            type="email"
            value={email}
            onChange={e => { setEmail(e.target.value); clearFieldError('email') }}
            placeholder={t('common.emailPlaceholder')}
            className={`mt-2 w-full rounded-lg border px-4 py-3 focus:outline-none ${
              fieldErrors.email
                ? 'border-red-400 focus:border-red-500'
                : 'border-gray-300 focus:border-primary'
            }`}
          />
        </label>
        {fieldErrors.email && <p className="-mt-4 text-sm font-medium text-red-600">{fieldErrors.email}</p>}

        <div ref={spokenRef}>
          <LanguagePicker
            label={t('waitlist.iSpeak')}
            hint={t('waitlist.iSpeakHint')}
            multiple
            selected={spokenLanguages}
            topCodes={SPEAK_TOP_CODES}
            onChange={(code, add) => {
              clearFieldError('spoken')
              add
                ? setSpokenLanguages(current => current.includes(code) ? current : [...current, code])
                : setSpokenLanguages(current => current.filter(item => item !== code))
            }}
          />
          {fieldErrors.spoken && <p className="mt-2 text-sm font-medium text-red-600">{fieldErrors.spoken}</p>}
        </div>

        <div ref={learnRef}>
          <LanguagePicker
            label={<>{t('waitlist.iWantToLearn')} <span className="font-normal text-gray-400">{t('waitlist.optional')}</span></>}
            hint={t('waitlist.iWantToLearnHint')}
            multiple
            selected={targetLanguages}
            topCodes={LEARN_TOP_CODES}
            exclude={spokenLanguages}
            onChange={(code, add) => {
              clearFieldError('learn')
              add
                ? setTargetLanguages(current => current.includes(code) ? current : [...current, code])
                : setTargetLanguages(current => current.filter(item => item !== code))
            }}
          />
          {fieldErrors.learn && <p className="mt-2 text-sm font-medium text-red-600">{fieldErrors.learn}</p>}
        </div>

        <label className="block">
          <span className="font-semibold text-gray-700">{t('waitlist.targetLanguageLevel', 'Target Language Level')}</span>
          <select
            value={targetLanguageLevel}
            onChange={(e) => setTargetLanguageLevel(e.target.value)}
            className="mt-2 w-full rounded-lg border border-gray-300 px-4 py-3 focus:border-primary focus:outline-none"
          >
            <option value="A1">A1 (Beginner)</option>
            <option value="A2">A2 (Elementary)</option>
            <option value="B1">B1 (Intermediate)</option>
            <option value="B2">B2 (Upper Intermediate)</option>
            <option value="C1">C1 (Advanced)</option>
            <option value="C2">C2 (Mastery)</option>
          </select>
        </label>

        <fieldset>
          <legend className="font-semibold text-gray-700">{t('waitlist.reason')}</legend>
          <div ref={reasonsRef} className="mt-2 grid gap-2 sm:grid-cols-2">
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
                  onChange={() => {
                    clearFieldError('reasons')
                    toggle(reason.value, setReasons)
                  }}
                  className="rounded text-indigo-600 focus:ring-indigo-500"
                />
                {t(reason.labelKey)}
              </label>
            ))}
          </div>
          {fieldErrors.reasons && <p className="mt-2 text-sm font-medium text-red-600">{fieldErrors.reasons}</p>}
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

        {error && <p className="rounded bg-red-100 p-3 text-red-700">{error}</p>}

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