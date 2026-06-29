import { useState } from 'react'
import { useStore } from '../store'
import { authAPI } from '../services/api'
import { SUPPORTED_LANGUAGES } from '../services/language'

interface SettingsProps {
  onClose: () => void
}

export default function Settings({ onClose }: SettingsProps) {
  const { user, updateUser } = useStore()
  const [displayName, setDisplayName] = useState(user?.displayName || '')
  const [nativeLanguage, setNativeLanguage] = useState(user?.nativeLanguage || 'en')
  const [targetLanguages, setTargetLanguages] = useState<string[]>(user?.targetLanguages || [])
  const [isLoading, setIsLoading] = useState(false)
  const [message, setMessage] = useState('')

  const handleSave = async () => {
    setIsLoading(true)
    setMessage('')
    try {
      await authAPI.updateMe({ displayName, nativeLanguage, targetLanguages })
      updateUser({ displayName, nativeLanguage, targetLanguages })
      setMessage('Settings saved successfully!')
      setTimeout(() => onClose(), 1500)
    } catch (err) {
      setMessage('Failed to save settings')
    } finally {
      setIsLoading(false)
    }
  }

  const toggleTargetLanguage = (code: string) => {
    if (code === nativeLanguage) return
    setTargetLanguages(prev =>
      prev.includes(code) ? prev.filter(l => l !== code) : [...prev, code]
    )
  }

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl w-full max-w-lg max-h-[90vh] overflow-y-auto">
        <div className="p-6 border-b border-gray-200 flex items-center justify-between">
          <h2 className="text-2xl font-bold text-gray-900">Settings</h2>
          <button onClick={onClose} className="text-gray-500 hover:text-gray-700 text-2xl">×</button>
        </div>

        <div className="p-6 space-y-6">
          {message && (
            <div className={`px-4 py-3 rounded-lg text-sm ${
              message.includes('success') ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700'
            }`}>
              {message}
            </div>
          )}

          {/* Display Name */}
          <div>
            <label className="block text-sm font-bold text-gray-700 mb-2">Display Name</label>
            <input
              type="text"
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500"
            />
          </div>

          {/* Email (read-only) */}
          <div>
            <label className="block text-sm font-bold text-gray-700 mb-2">Email</label>
            <input
              type="email"
              value={user?.email || ''}
              readOnly
              className="w-full px-4 py-3 border border-gray-200 rounded-lg bg-gray-50 text-gray-500"
            />
          </div>

          {/* Native Language */}
          <div>
            <label className="block text-sm font-bold text-gray-700 mb-2">Native Language</label>
            <select
              value={nativeLanguage}
              onChange={(e) => setNativeLanguage(e.target.value)}
              className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500"
            >
              {SUPPORTED_LANGUAGES.map((lang) => (
                <option key={lang.code} value={lang.code}>
                  {lang.flag} {lang.name} ({lang.nativeName})
                </option>
              ))}
            </select>
          </div>

          {/* Target Languages */}
          <div>
            <label className="block text-sm font-bold text-gray-700 mb-2">
              Languages You Want to Learn
            </label>
            <div className="grid grid-cols-2 gap-2">
              {SUPPORTED_LANGUAGES.filter(l => l.code !== nativeLanguage).map((lang) => (
                <label
                  key={lang.code}
                  className={`flex items-center space-x-3 p-3 rounded-lg border cursor-pointer transition ${
                    targetLanguages.includes(lang.code)
                      ? 'border-indigo-500 bg-indigo-50'
                      : 'border-gray-200 hover:bg-gray-50'
                  }`}
                >
                  <input
                    type="checkbox"
                    checked={targetLanguages.includes(lang.code)}
                    onChange={() => toggleTargetLanguage(lang.code)}
                    className="rounded text-indigo-600 focus:ring-indigo-500"
                  />
                  <span className="text-sm">{lang.flag} {lang.nativeName}</span>
                </label>
              ))}
            </div>
          </div>
        </div>

        <div className="p-6 border-t border-gray-200 flex justify-end space-x-3">
          <button
            onClick={onClose}
            className="px-6 py-2 border border-gray-300 rounded-lg text-gray-700 hover:bg-gray-50"
          >
            Cancel
          </button>
          <button
            onClick={handleSave}
            disabled={isLoading}
            className="px-6 py-2 bg-gradient-to-r from-indigo-600 to-purple-600 text-white rounded-lg hover:opacity-90 disabled:opacity-50"
          >
            {isLoading ? 'Saving...' : 'Save Settings'}
          </button>
        </div>
      </div>
    </div>
  )
}
