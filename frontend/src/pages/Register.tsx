import { useEffect, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { authAPI } from '../services/api'
import { detectBrowserLanguage, getNativeLanguageName } from '../services/language'
import LanguageSelector from '../components/LanguageSelector'

interface RegisterProps {
  onRegister: (tokens: { accessToken: string; refreshToken: string }) => void
}

export default function Register({ onRegister }: RegisterProps) {
  const { t } = useTranslation()
  const [searchParams] = useSearchParams()
  const inviteToken = searchParams.get('invite') || ''
  const [email, setEmail] = useState('')
  const [invitedEmail, setInvitedEmail] = useState('')
  const [inviteLoading, setInviteLoading] = useState(Boolean(inviteToken))
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [selectedLang, setSelectedLang] = useState(() => localStorage.getItem('preferredLanguage') || detectBrowserLanguage())

  const nativeLangName = getNativeLanguageName(selectedLang)

  // Prefill the email bound to the invitation so users never type it twice.
  useEffect(() => {
    if (!inviteToken) return
    let active = true
    authAPI.inviteEmail(inviteToken)
      .then(inviteEmail => {
        if (active) {
          setEmail(inviteEmail)
          setInvitedEmail(inviteEmail)
        }
      })
      .catch(() => {
        if (active) setError(t('auth.inviteInvalid'))
      })
      .finally(() => {
        if (active) setInviteLoading(false)
      })
    return () => { active = false }
  }, [inviteToken, t])

  const handleLanguageChange = (code: string) => {
    setSelectedLang(code)
    localStorage.setItem('preferredLanguage', code)
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setIsLoading(true)
    const normalizedEmail = email.trim().toLowerCase()

    try {
      const response = await authAPI.register({
        email: normalizedEmail,
        password,
        username: normalizedEmail,
        displayName: normalizedEmail.split('@')[0],
        nativeLanguage: selectedLang,
        targetLanguages: [],
        inviteToken,
      })
      onRegister(response.tokens)
    } catch (err: any) {
      const status = err?.response?.status
      const errorMessage =
        (typeof err?.response?.data === 'string'
          ? err.response.data
          : err?.response?.data?.error) ||
        t('auth.registrationFailed')

      if (status === 409 || errorMessage.toLowerCase().includes('already')) {
        try {
          const loginResponse = await authAPI.login({
            username: normalizedEmail,
            password,
          })
          onRegister(loginResponse.tokens)
          return
        } catch {
          setError(t('auth.accountExistsLoginFailed'))
          return
        }
      }

      setError(errorMessage)
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-primary to-secondary py-8">
      {/* Language selector top-right */}
      <div className="fixed top-4 right-4 z-50">
        <LanguageSelector
          currentLang={selectedLang}
          onLanguageChange={handleLanguageChange}
          variant="navbar"
        />
      </div>

      <div className="bg-white p-8 rounded-lg shadow-xl w-full max-w-md">
        <div className="text-center mb-8">
          <div className="w-16 h-16 bg-gradient-to-br from-indigo-600 to-purple-600 rounded-full flex items-center justify-center mx-auto mb-4">
            <svg className="w-8 h-8 text-white" fill="currentColor" viewBox="0 0 20 20">
              <path d="M2 5a2 2 0 012-2h12a2 2 0 012 2v10a2 2 0 01-2 2H4a2 2 0 01-2-2V5z"></path>
              <path d="M7.5 7.5a1.5 1.5 0 113 0 1.5 1.5 0 01-3 0z"></path>
            </svg>
          </div>
          <h1 className="text-3xl font-bold text-gray-800">{t('auth.joinChorus')}</h1>
          <p className="text-gray-500 mt-2">
            {t('auth.yourLanguage')} <strong>{nativeLangName}</strong>
          </p>
        </div>

        {!inviteToken && (
          <div className="bg-amber-100 border border-amber-300 text-amber-800 px-4 py-3 rounded mb-4">
            {t('auth.inviteOnly')}{' '}<Link to="/waitlist" className="font-semibold underline">{t('auth.joinWaitlist')}</Link>.
          </div>
        )}
        {error && (
          <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded mb-4">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-5">
          <div>
            <label className="block text-gray-700 text-sm font-bold mb-2">
              {t('common.email')}
            </label>
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder={t('common.emailPlaceholder')}
              className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent transition readOnly:bg-gray-100"
              required
              autoFocus
              readOnly={Boolean(invitedEmail)}
            />
            {inviteLoading ? (
              <p className="mt-1 text-xs text-gray-500">{t('common.loading')}</p>
            ) : invitedEmail ? (
              <p className="mt-1 flex items-center gap-1 text-xs text-green-600">
                <span>✓</span> {t('auth.invitedAs', { email: invitedEmail })}
              </p>
            ) : null}
          </div>

          <div>
            <label className="block text-gray-700 text-sm font-bold mb-2">
              {t('common.password')}
            </label>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder={t('common.minCharsPlaceholder')}
              className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent transition"
              required
              minLength={8}
            />
          </div>

          <button
            type="submit"
            disabled={isLoading || !inviteToken}
            className="w-full bg-gradient-to-r from-indigo-600 to-purple-600 text-white font-bold py-3 px-4 rounded-lg hover:opacity-90 transition disabled:opacity-50 text-lg"
          >
            {isLoading ? t('auth.creatingAccount') : t('auth.createAccount')}
          </button>
        </form>

        <p className="text-center text-gray-600 mt-6">
          {t('auth.alreadyHaveAccount')}{' '}
          <Link to="/login" className="text-indigo-600 font-semibold hover:underline">
            {t('auth.logIn')}
          </Link>
        </p>

        <p className="text-center text-xs text-gray-400 mt-4">
          {t('auth.setupLater')}
        </p>
      </div>
    </div>
  )
}
