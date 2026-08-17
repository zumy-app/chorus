import { useEffect, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { authAPI } from '../services/api'
import { detectBrowserLanguage, getNativeLanguageName } from '../services/language'
import AuthShell from '../components/AuthShell'

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
  const [showPassword, setShowPassword] = useState(false)
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
    <AuthShell
      selectedLang={selectedLang}
      onLanguageChange={handleLanguageChange}
      title={t('auth.joinChorus')}
      tagline={
        <>
          {t('auth.yourLanguage')} <strong className="text-primary">{nativeLangName}</strong>
        </>
      }
      bottom={
        <>
          <p className="font-body-sm text-body-sm text-on-surface-variant">{t('auth.alreadyHaveAccount')}</p>
          <Link
            to="/login"
            className="font-label-md text-label-md text-primary hover:text-primary-container px-4 py-2 rounded-full hover:bg-primary-container/10 transition-colors"
          >
            {t('auth.logIn')}
          </Link>
        </>
      }
    >
      <div className="bg-surface-container-lowest rounded-3xl p-6 shadow-[0px_4px_12px_rgba(0,0,0,0.05)] border border-outline-variant/30">
        {!inviteToken && (
          <div className="bg-surface-container-high text-on-surface px-4 py-3 rounded-xl mb-4 font-body-sm text-body-sm">
            {t('auth.inviteOnly')}{' '}
            <Link to="/waitlist" className="font-semibold text-primary underline">
              {t('auth.joinWaitlist')}
            </Link>
            .
          </div>
        )}
        {error && (
          <div className="bg-error-container text-on-error-container px-4 py-3 rounded-xl mb-4 font-body-sm text-body-sm">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="flex flex-col gap-5">
          {/* Email field */}
          <div className="flex flex-col gap-1.5">
            <label className="font-label-sm text-label-sm text-on-surface-variant ml-1" htmlFor="register-email">
              {t('common.email')}
            </label>
            <div className="relative flex items-center">
              <span className="material-symbols-outlined absolute left-3.5 text-outline pointer-events-none">mail</span>
              <input
                id="register-email"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder={t('common.emailPlaceholder')}
                className="w-full bg-surface text-on-surface pl-10 pr-4 py-3.5 rounded-xl border-none focus:ring-2 focus:ring-primary-container focus:bg-surface-container-lowest transition-colors placeholder:text-outline-variant font-body-md text-body-md shadow-sm outline-none readOnly:bg-surface-container-high"
                required
                autoFocus
                readOnly={Boolean(invitedEmail)}
              />
            </div>
            {inviteLoading ? (
              <p className="ml-1 font-label-sm text-label-sm text-on-surface-variant">{t('common.loading')}</p>
            ) : invitedEmail ? (
              <p className="ml-1 flex items-center gap-1 font-label-sm text-label-sm text-tertiary">
                <span className="material-symbols-outlined text-[16px]" style={{ fontVariationSettings: "'FILL' 1" }}>check_circle</span>
                {t('auth.invitedAs', { email: invitedEmail })}
              </p>
            ) : null}
          </div>

          {/* Password field */}
          <div className="flex flex-col gap-1.5">
            <label className="font-label-sm text-label-sm text-on-surface-variant ml-1" htmlFor="register-password">
              {t('common.password')}
            </label>
            <div className="relative flex items-center">
              <span className="material-symbols-outlined absolute left-3.5 text-outline pointer-events-none">lock</span>
              <input
                id="register-password"
                type={showPassword ? 'text' : 'password'}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder={t('common.minCharsPlaceholder')}
                className="w-full bg-surface text-on-surface pl-10 pr-12 py-3.5 rounded-xl border-none focus:ring-2 focus:ring-primary-container focus:bg-surface-container-lowest transition-colors placeholder:text-outline-variant font-body-md text-body-md shadow-sm outline-none"
                required
                minLength={8}
              />
              <button
                type="button"
                aria-label={showPassword ? t('auth.hidePassword') : t('auth.showPassword')}
                onClick={() => setShowPassword(!showPassword)}
                className="absolute right-3.5 text-outline hover:text-on-surface transition-colors"
              >
                <span className="material-symbols-outlined">{showPassword ? 'visibility' : 'visibility_off'}</span>
              </button>
            </div>
          </div>

          {/* Primary action button */}
          <button
            type="submit"
            disabled={isLoading || !inviteToken}
            className="mt-2 w-full bg-primary-container text-on-primary-container font-label-md text-label-md py-4 rounded-xl flex items-center justify-center gap-2 hover:bg-primary hover:text-on-primary transition-colors active:scale-[0.98] duration-150 shadow-md disabled:opacity-50"
          >
            <span>{isLoading ? t('auth.creatingAccount') : t('auth.createAccount')}</span>
            <span className="material-symbols-outlined text-[18px]">arrow_forward</span>
          </button>
        </form>

        <p className="text-center font-label-sm text-label-sm text-on-surface-variant mt-4">
          {t('auth.setupLater')}
        </p>
      </div>
    </AuthShell>
  )
}
