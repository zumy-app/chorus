import { useState } from 'react'
import { Link } from 'react-router-dom'
import { authAPI } from '../services/api'
import { detectBrowserLanguage, getNativeLanguageName } from '../services/language'

interface RegisterProps {
  onRegister: (tokens: { accessToken: string; refreshToken: string }) => void
}

export default function Register({ onRegister }: RegisterProps) {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [isLoading, setIsLoading] = useState(false)

  const detectedLang = detectBrowserLanguage()
  const nativeLangName = getNativeLanguageName(detectedLang)

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
        nativeLanguage: detectedLang,
        targetLanguages: [],
      })
      onRegister(response.tokens)
    } catch (err: any) {
      const status = err?.response?.status
      const errorMessage =
        (typeof err?.response?.data === 'string'
          ? err.response.data
          : err?.response?.data?.error) ||
        'Registration failed'

      if (status === 409 || errorMessage.toLowerCase().includes('already')) {
        try {
          const loginResponse = await authAPI.login({
            username: normalizedEmail,
            password,
          })
          onRegister(loginResponse.tokens)
          return
        } catch {
          setError('Account exists but login failed. Please try Login.')
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
      <div className="bg-white p-8 rounded-lg shadow-xl w-full max-w-md">
        <div className="text-center mb-8">
          <div className="w-16 h-16 bg-gradient-to-br from-indigo-600 to-purple-600 rounded-full flex items-center justify-center mx-auto mb-4">
            <svg className="w-8 h-8 text-white" fill="currentColor" viewBox="0 0 20 20">
              <path d="M2 5a2 2 0 012-2h12a2 2 0 012 2v10a2 2 0 01-2 2H4a2 2 0 01-2-2V5z"></path>
              <path d="M7.5 7.5a1.5 1.5 0 113 0 1.5 1.5 0 01-3 0z"></path>
            </svg>
          </div>
          <h1 className="text-3xl font-bold text-gray-800">Join Chorus</h1>
          <p className="text-gray-500 mt-2">
            We detected your language as <strong>{nativeLangName}</strong>
          </p>
        </div>

        {error && (
          <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded mb-4">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-5">
          <div>
            <label className="block text-gray-700 text-sm font-bold mb-2">
              Email
            </label>
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="you@example.com"
              className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent transition"
              required
              autoFocus
            />
          </div>

          <div>
            <label className="block text-gray-700 text-sm font-bold mb-2">
              Password
            </label>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="At least 8 characters"
              className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent transition"
              required
              minLength={8}
            />
          </div>

          <button
            type="submit"
            disabled={isLoading}
            className="w-full bg-gradient-to-r from-indigo-600 to-purple-600 text-white font-bold py-3 px-4 rounded-lg hover:opacity-90 transition disabled:opacity-50 text-lg"
          >
            {isLoading ? 'Creating account...' : 'Create Account'}
          </button>
        </form>

        <p className="text-center text-gray-600 mt-6">
          Already have an account?{' '}
          <Link to="/login" className="text-indigo-600 font-semibold hover:underline">
            Log in
          </Link>
        </p>

        <p className="text-center text-xs text-gray-400 mt-4">
          You can set up your display name and learning languages later.
        </p>
      </div>
    </div>
  )
}
