import { useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { authAPI } from '../services/api'
import { detectBrowserLanguage, getNativeLanguageName } from '../services/language'
import AuthShell from '../components/AuthShell'

export default function ResetPassword() {
  const { t } = useTranslation()
  const [searchParams] = useSearchParams()
  const token = searchParams.get('token') || ''
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [error, setError] = useState('')
  const [message, setMessage] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [selectedLang, setSelectedLang] = useState(() => localStorage.getItem('preferredLanguage') || detectBrowserLanguage())

  const nativeLangName = getNativeLanguageName(selectedLang)

  const handleLanguageChange = (code: string) => {
    setSelectedLang(code)
    localStorage.setItem('preferredLanguage', code)
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setMessage('')

    if (password !== confirmPassword) {
      setError(t('auth.passwordsDoNotMatch'))
      return
    }

    setIsLoading(true)

    try {
      const response = await authAPI.resetPassword(token, password)
      setMessage(response.message)
      setPassword('')
      setConfirmPassword('')
    } catch (err: any) {
      setError(err.response?.data?.error || t('auth.resetLinkInvalid'))
    } finally {
      setIsLoading(false)
    }
  }

  const inputClasses =
    'w-full bg-surface text-on-surface pl-10 pr-4 py-3.5 rounded-xl border-none focus:ring-2 focus:ring-primary-container focus:bg-surface-container-lowest transition-colors placeholder:text-outline-variant font-body-md text-body-md shadow-sm outline-none'

  return (
    <AuthShell
      selectedLang={selectedLang}
      onLanguageChange={handleLanguageChange}
      title={t('auth.chooseNewPassword')}
      tagline={
        <>
          {t('auth.tagline')}
          <span className="block font-label-sm text-label-sm text-on-surface-variant mt-1">
            {t('auth.language', { name: nativeLangName })}
          </span>
        </>
      }
      bottom={
        <>
          <p className="font-body-sm text-body-sm text-on-surface-variant">{t('auth.remembered')}</p>
          <Link
            to="/login"
            className="font-label-md text-label-md text-primary hover:text-primary-container px-4 py-2 rounded-full hover:bg-primary-container/10 transition-colors"
          >
            {t('auth.backToLogin')}
          </Link>
        </>
      }
    >
      <div className="bg-surface-container-lowest rounded-3xl p-6 shadow-[0px_4px_12px_rgba(0,0,0,0.05)] border border-outline-variant/30">
        {!token && (
          <div className="bg-surface-container-high text-on-surface px-4 py-3 rounded-xl mb-4 font-body-sm text-body-sm">
            {t('auth.resetLinkMissing')}
          </div>
        )}
        {error && (
          <div className="bg-error-container text-on-error-container px-4 py-3 rounded-xl mb-4 font-body-sm text-body-sm">
            {error}
          </div>
        )}
        {message && (
          <div className="bg-tertiary-container text-on-tertiary-container px-4 py-3 rounded-xl mb-4 font-body-sm text-body-sm">
            {message}
          </div>
        )}

        <form onSubmit={handleSubmit} className="flex flex-col gap-5">
          {/* New password */}
          <div className="flex flex-col gap-1.5">
            <label className="font-label-sm text-label-sm text-on-surface-variant ml-1" htmlFor="reset-password">
              {t('auth.newPassword')}
            </label>
            <div className="relative flex items-center">
              <span className="material-symbols-outlined absolute left-3.5 text-outline pointer-events-none">lock</span>
              <input
                id="reset-password"
                type={showPassword ? 'text' : 'password'}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder={t('common.minCharsPlaceholder')}
                className={inputClasses}
                required
                minLength={8}
                autoFocus
              />
            </div>
          </div>

          {/* Confirm password */}
          <div className="flex flex-col gap-1.5">
            <label className="font-label-sm text-label-sm text-on-surface-variant ml-1" htmlFor="reset-confirm">
              {t('auth.confirmPassword')}
            </label>
            <div className="relative flex items-center">
              <span className="material-symbols-outlined absolute left-3.5 text-outline pointer-events-none">lock</span>
              <input
                id="reset-confirm"
                type={showPassword ? 'text' : 'password'}
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                placeholder={t('auth.reEnterPassword')}
                className={inputClasses}
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
            disabled={isLoading || !token}
            className="mt-2 w-full bg-primary-container text-on-primary-container font-label-md text-label-md py-4 rounded-xl flex items-center justify-center gap-2 hover:bg-primary hover:text-on-primary transition-colors active:scale-[0.98] duration-150 shadow-md disabled:opacity-50"
          >
            <span>{isLoading ? t('auth.resetting') : t('auth.resetPassword')}</span>
            <span className="material-symbols-outlined text-[18px]">arrow_forward</span>
          </button>
        </form>
      </div>
    </AuthShell>
  )
}
