import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { authAPI } from '../services/api'
import { detectBrowserLanguage, getNativeLanguageName } from '../services/language'
import LanguageSelector from '../components/LanguageSelector'

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
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-indigo-600 to-purple-600">
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
              <path d="M10 2a6 6 0 00-6 6v1H3a1 1 0 00-1 1v7a1 1 0 001 1h14a1 1 0 001-1v-7a1 1 0 00-1-1h-1V8a6 6 0 00-6-6zM6 8a4 4 0 018 0v1H6V8zm6 4a1 1 0 00-2 0v1a1 1 0 002 0v-1z"></path>
            </svg>
          </div>
          <h1 className="text-3xl font-bold text-gray-800">{t('auth.forgotPasswordTitle')}</h1>
          <p className="text-gray-500 mt-1">{t('auth.tagline')}</p>
          <p className="text-xs text-gray-400 mt-1">{t('auth.language', { name: nativeLangName })}</p>
        </div>

        {error && (
          <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded mb-4">
            {error}
          </div>
        )}

        {message && (
          <div className="bg-green-100 border border-green-400 text-green-700 px-4 py-3 rounded mb-4">
            {message}
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
              className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent transition"
              required
              autoFocus
            />
          </div>

          <button
            type="submit"
            disabled={isLoading}
            className="w-full bg-gradient-to-r from-indigo-600 to-purple-600 text-white font-bold py-3 px-4 rounded-lg hover:opacity-90 transition disabled:opacity-50 text-lg"
          >
            {isLoading ? t('common.sending') : t('auth.sendResetLink')}
          </button>
        </form>

        <p className="text-center text-gray-600 mt-6">
          {t('auth.remembered')}{' '}
          <Link to="/login" className="text-indigo-600 font-semibold hover:underline">
            {t('auth.backToLogin')}
          </Link>
        </p>
      </div>
    </div>
  )
}
