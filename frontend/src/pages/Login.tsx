import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { otpAPI } from '../services/api'
import { api } from '../services/api'
import { detectBrowserLanguage, getNativeLanguageName } from '../services/language'
import AuthShell from '../components/AuthShell'

interface LoginProps {
  onLogin: (tokens: { accessToken: string; refreshToken: string }) => void
}

export default function Login({ onLogin }: LoginProps) {
  const { t } = useTranslation()
  // Local-dev convenience: prefill the test account when VITE_TEST_USER_* are
  // present in frontend/.env. Falls back to empty fields otherwise.
  const testEmail = import.meta.env.VITE_TEST_USER_EMAIL as string | undefined
  const testPassword = import.meta.env.VITE_TEST_USER_PASSWORD as string | undefined
  const [email, setEmail] = useState(testEmail || '')
  const [password, setPassword] = useState(testPassword || '')
  const [showPassword, setShowPassword] = useState(false)
  const [error, setError] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [requires2FA, setRequires2FA] = useState(false)
  const [tempToken, setTempToken] = useState('')
  const [phoneMasked, setPhoneMasked] = useState('')
  const [code, setCode] = useState('')
  const [selectedLang, setSelectedLang] = useState(() => localStorage.getItem('preferredLanguage') || detectBrowserLanguage())

  const nativeLangName = getNativeLanguageName(selectedLang)

  const handleLanguageChange = (code: string) => {
    setSelectedLang(code)
    localStorage.setItem('preferredLanguage', code)
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setIsLoading(true)
    try {
      const raw = await api.post('/auth/login', { username: email.trim().toLowerCase(), password })
      if (raw.data.requires2FA) {
        setTempToken(raw.data.tempToken)
        setPhoneMasked(raw.data.phoneMasked || '')
        setRequires2FA(true)
      } else {
        onLogin(raw.data.tokens)
      }
    } catch (err: any) {
      setError(err.response?.data?.error || t('auth.loginFailed'))
    } finally {
      setIsLoading(false)
    }
  }

  const handleVerify2FA = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(''); setIsLoading(true)
    try {
      const r = await otpAPI.verify2FA(tempToken, code)
      onLogin(r.tokens)
    } catch (err: any) {
      setError(err.response?.data?.error || 'Invalid code')
    } finally { setIsLoading(false) }
  }

  return (
    <AuthShell
      selectedLang={selectedLang}
      onLanguageChange={handleLanguageChange}
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
          <p className="font-body-sm text-body-sm text-on-surface-variant">{t('auth.noAccount')}</p>
          <Link
            to="/register"
            className="font-label-md text-label-md text-primary hover:text-primary-container px-4 py-2 rounded-full hover:bg-primary-container/10 transition-colors"
          >
            {t('auth.createOne')}
          </Link>
        </>
      }
    >
      {/* Login form container */}
      <div className="bg-surface-container-lowest rounded-3xl p-6 shadow-[0px_4px_12px_rgba(0,0,0,0.05)] border border-outline-variant/30">
          {error && (
            <div className="bg-error-container text-on-error-container px-4 py-3 rounded-xl mb-4 font-body-sm text-body-sm">
              {error}
            </div>
          )}

          {requires2FA ? (
            <form onSubmit={handleVerify2FA} className="flex flex-col gap-5">
              <p className="text-sm text-on-surface-variant">Code sent to {phoneMasked}</p>
              <input value={code} onChange={e=>setCode(e.target.value)} placeholder="123456" maxLength={6} className="w-full bg-surface text-on-surface px-4 py-3.5 rounded-xl text-center tracking-widest text-lg" />
              <button type="submit" disabled={isLoading || code.length!==6} className="w-full bg-primary-container text-on-primary-container py-4 rounded-xl disabled:opacity-50">Verify</button>
              <button type="button" onClick={()=>setRequires2FA(false)} className="text-sm text-primary">Back</button>
            </form>
          ) : (
          <form onSubmit={handleSubmit} className="flex flex-col gap-5">
            {/* Email field */}
            <div className="flex flex-col gap-1.5">
              <label className="font-label-sm text-label-sm text-on-surface-variant ml-1" htmlFor="login-email">
                {t('common.email')}
              </label>
              <div className="relative flex items-center">
                <span className="material-symbols-outlined absolute left-3.5 text-outline pointer-events-none">mail</span>
                <input
                  id="login-email"
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder={t('common.emailPlaceholder')}
                  required
                  autoFocus
                  className="w-full bg-surface text-on-surface pl-10 pr-4 py-3.5 rounded-xl border-none focus:ring-2 focus:ring-primary-container focus:bg-surface-container-lowest transition-colors placeholder:text-outline-variant font-body-md text-body-md shadow-sm outline-none"
                />
              </div>
            </div>

            {/* Password field */}
            <div className="flex flex-col gap-1.5">
              <div className="flex justify-between items-end ml-1">
                <label className="font-label-sm text-label-sm text-on-surface-variant" htmlFor="login-password">
                  {t('common.password')}
                </label>
                <Link
                  to="/forgot-password"
                  className="font-label-sm text-label-sm text-primary hover:text-primary-container transition-colors"
                >
                  {t('auth.forgot')}
                </Link>
              </div>
              <div className="relative flex items-center">
                <span className="material-symbols-outlined absolute left-3.5 text-outline pointer-events-none">lock</span>
                <input
                  id="login-password"
                  type={showPassword ? 'text' : 'password'}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder={t('common.passwordPlaceholder')}
                  required
                  className="w-full bg-surface text-on-surface pl-10 pr-12 py-3.5 rounded-xl border-none focus:ring-2 focus:ring-primary-container focus:bg-surface-container-lowest transition-colors placeholder:text-outline-variant font-body-md text-body-md shadow-sm outline-none"
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
              disabled={isLoading}
              className="mt-2 w-full bg-primary-container text-on-primary-container font-label-md text-label-md py-4 rounded-xl flex items-center justify-center gap-2 hover:bg-primary hover:text-on-primary transition-colors active:scale-[0.98] duration-150 shadow-md disabled:opacity-50"
            >
              <span>{isLoading ? t('auth.loggingIn') : t('auth.logIn')}</span>
              <span className="material-symbols-outlined text-[18px]">arrow_forward</span>
            </button>
          </form>
          )}

          {/* Divider */}
          <div className="flex items-center gap-4 my-6">
            <div className="h-px bg-outline-variant/50 flex-grow" />
            <span className="font-label-sm text-label-sm text-outline">{t('auth.orContinueWith')}</span>
            <div className="h-px bg-outline-variant/50 flex-grow" />
          </div>

          {/* Social logins */}
          <div className="flex flex-col gap-3">
            <button className="w-full bg-surface border border-outline-variant/50 text-on-surface font-label-md text-label-md py-3.5 rounded-xl flex items-center justify-center gap-3 hover:bg-surface-container-low transition-colors active:scale-[0.98] duration-150 shadow-sm">
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
                <path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z" fill="#4285F4" />
                <path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" fill="#34A853" />
                <path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z" fill="#FBBC05" />
                <path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" fill="#EA4335" />
              </svg>
              <span>Google</span>
            </button>
            <button className="w-full bg-inverse-surface text-inverse-on-surface font-label-md text-label-md py-3.5 rounded-xl flex items-center justify-center gap-3 hover:bg-on-surface transition-colors active:scale-[0.98] duration-150 shadow-sm">
              <span className="material-symbols-outlined text-[20px]" style={{ fontVariationSettings: "'FILL' 1" }}>file_download</span>
              <span>Apple</span>
            </button>
          </div>
        </div>
    </AuthShell>
  )
}
