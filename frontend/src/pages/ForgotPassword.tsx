import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { authAPI } from '../services/api'
import { detectBrowserLanguage, getNativeLanguageName } from '../services/language'
import AuthShell from '../components/AuthShell'

export default function ForgotPassword() {
  const { t } = useTranslation()
  const [email, setEmail] = useState('')
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
    setIsLoading(true)

    try {
      const response = await authAPI.forgotPassword(email.trim().toLowerCase())
      setMessage(response.message)
      setEmail('')
    } catch (err: any) {
      setError(err.response?.data?.error || t('auth.error'))
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <AuthShell
      selectedLang={selectedLang}
      onLanguageChange={handleLanguageChange}
      title={t('auth.forgotPasswordTitle')}
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
          {/* Email field */}
          <div className="flex flex-col gap-1.5">
            <label className="font-label-sm text-label-sm text-on-surface-variant ml-1" htmlFor="forgot-email">
              {t('common.email')}
            </label>
            <div className="relative flex items-center">
              <span className="material-symbols-outlined absolute left-3.5 text-outline pointer-events-none">mail</span>
              <input
                id="forgot-email"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder={t('common.emailPlaceholder')}
                className="w-full bg-surface text-on-surface pl-10 pr-4 py-3.5 rounded-xl border-none focus:ring-2 focus:ring-primary-container focus:bg-surface-container-lowest transition-colors placeholder:text-outline-variant font-body-md text-body-md shadow-sm outline-none"
                required
                autoFocus
              />
            </div>
          </div>

          {/* Primary action button */}
          <button
            type="submit"
            disabled={isLoading}
            className="mt-2 w-full bg-primary-container text-on-primary-container font-label-md text-label-md py-4 rounded-xl flex items-center justify-center gap-2 hover:bg-primary hover:text-on-primary transition-colors active:scale-[0.98] duration-150 shadow-md disabled:opacity-50"
          >
            <span>{isLoading ? t('common.sending') : t('auth.sendResetLink')}</span>
            <span className="material-symbols-outlined text-[18px]">arrow_forward</span>
          </button>
        </form>
      </div>
    </AuthShell>
  )
}
