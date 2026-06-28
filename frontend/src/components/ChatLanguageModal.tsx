import { useState } from 'react'
import { useStore } from '../store'
import { SUPPORTED_LANGUAGES } from '../services/language'

interface ChatLanguageModalProps {
  onClose: () => void
}

export default function ChatLanguageModal({ onClose }: ChatLanguageModalProps) {
  const { activeChat, user } = useStore()
  const [myLanguage, setMyLanguage] = useState(user?.nativeLanguage || 'en')
  const [theirLanguage, setTheirLanguage] = useState(
    activeChat?.type === 'direct'
      ? activeChat.participants?.find(p => p.user?.id !== user?.id)?.user?.nativeLanguage || 'es'
      : 'en'
  )

  if (!activeChat) return null

  const otherParticipant = activeChat.type === 'direct'
    ? activeChat.participants?.find(p => p.user?.id !== user?.id)?.user
    : null

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl w-full max-w-md">
        <div className="p-6 border-b border-gray-200 flex items-center justify-between">
          <h2 className="text-xl font-bold text-gray-900">Chat Language Settings</h2>
          <button onClick={onClose} className="text-gray-500 hover:text-gray-700 text-2xl">×</button>
        </div>

        <div className="p-6 space-y-6">
          {/* My Language */}
          <div>
            <label className="block text-sm font-bold text-gray-700 mb-2">
              🌍 My Language
            </label>
            <p className="text-xs text-gray-500 mb-2">
              Messages will be translated into this language for you
            </p>
            <select
              value={myLanguage}
              onChange={(e) => setMyLanguage(e.target.value)}
              className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500"
            >
              {SUPPORTED_LANGUAGES.map((lang) => (
                <option key={lang.code} value={lang.code}>
                  {lang.flag} {lang.nativeName} ({lang.name})
                </option>
              ))}
            </select>
          </div>

          {/* Their Language */}
          {activeChat.type === 'direct' && otherParticipant && (
            <div>
              <label className="block text-sm font-bold text-gray-700 mb-2">
                🗣️ {otherParticipant.displayName}'s Language
              </label>
              <p className="text-xs text-gray-500 mb-2">
                Messages you send will be translated into this language for them
              </p>
              <select
                value={theirLanguage}
                onChange={(e) => setTheirLanguage(e.target.value)}
                className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500"
              >
                {SUPPORTED_LANGUAGES.map((lang) => (
                  <option key={lang.code} value={lang.code}>
                    {lang.flag} {lang.nativeName} ({lang.name})
                  </option>
                ))}
              </select>
            </div>
          )}

          {/* Group chat info */}
          {activeChat.type === 'group' && (
            <div className="bg-blue-50 border border-blue-200 rounded-lg p-4 text-sm text-blue-700">
              <p className="font-semibold mb-1">💡 Group Chat</p>
              <p>Each participant sees messages translated into their own language based on their profile settings.</p>
            </div>
          )}

          {/* Preview */}
          <div className="bg-gray-50 rounded-lg p-4">
            <p className="text-sm font-semibold text-gray-700 mb-2">Preview</p>
            <div className="space-y-2">
              <div className="bg-white rounded-lg p-3 border border-gray-200">
                <p className="text-xs text-gray-500 mb-1">You'll see messages in:</p>
                <p className="font-medium">{SUPPORTED_LANGUAGES.find(l => l.code === myLanguage)?.nativeName || 'English'}</p>
              </div>
              {activeChat.type === 'direct' && otherParticipant && (
                <div className="bg-white rounded-lg p-3 border border-gray-200">
                  <p className="text-xs text-gray-500 mb-1">{otherParticipant.displayName} will see messages in:</p>
                  <p className="font-medium">{SUPPORTED_LANGUAGES.find(l => l.code === theirLanguage)?.nativeName || 'English'}</p>
                </div>
              )}
            </div>
          </div>
        </div>

        <div className="p-6 border-t border-gray-200 flex justify-end">
          <button
            onClick={onClose}
            className="px-6 py-2 bg-gradient-to-r from-indigo-600 to-purple-600 text-white rounded-lg hover:opacity-90"
          >
            Done
          </button>
        </div>
      </div>
    </div>
  )
}
