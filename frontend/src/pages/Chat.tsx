import { useEffect, useState, useRef } from 'react'
import { useStore } from '../store'
import ChatList from '../components/ChatList'
import ChatArea from '../components/ChatArea'
import NewChatModal from '../components/NewChatModal'
import Settings from './Settings'
import About from './About'

interface ChatProps {
  onLogout: () => void
}

export default function Chat({ onLogout }: ChatProps) {
  const { user, loadChats } = useStore()
  const [showNewChatModal, setShowNewChatModal] = useState(false)
  const [showSettings, setShowSettings] = useState(false)
  const [showAbout, setShowAbout] = useState(false)
  const [showProfileMenu, setShowProfileMenu] = useState(false)
  const profileMenuRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    loadChats()
  }, [loadChats])

  // Close profile menu when clicking outside
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (profileMenuRef.current && !profileMenuRef.current.contains(e.target as Node)) {
        setShowProfileMenu(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  if (showAbout) {
    return (
      <div className="relative">
        <button
          onClick={() => setShowAbout(false)}
          className="fixed top-4 left-4 z-50 px-4 py-2 bg-white border border-gray-300 rounded-lg shadow-lg hover:bg-gray-50 font-semibold"
        >
          ← Back to Chat
        </button>
        <About />
      </div>
    )
  }

  return (
    <div className="h-screen flex">
      {/* Sidebar */}
      <div className="w-80 border-r border-gray-200 flex flex-col bg-white">
        <div className="p-4 border-b border-gray-200">
          <div className="flex items-center justify-between mb-4">
            <h1 className="text-2xl font-bold bg-gradient-to-r from-indigo-600 to-purple-600 bg-clip-text text-transparent">
              Chorus
            </h1>
            {/* Profile Dropdown */}
            <div className="relative" ref={profileMenuRef}>
              <button
                onClick={() => setShowProfileMenu(!showProfileMenu)}
                className="w-9 h-9 rounded-full bg-gradient-to-br from-indigo-600 to-purple-600 text-white font-bold text-sm flex items-center justify-center hover:opacity-90 transition"
              >
                {user?.displayName?.charAt(0).toUpperCase() || '?'}
              </button>

              {showProfileMenu && (
                <div className="absolute right-0 mt-2 w-56 bg-white rounded-lg shadow-xl border border-gray-200 z-50">
                  <div className="p-4 border-b border-gray-100">
                    <p className="font-semibold text-gray-900 truncate">{user?.displayName}</p>
                    <p className="text-xs text-gray-500 truncate">{user?.email}</p>
                    <div className="flex items-center gap-1 mt-1">
                      <span className="text-xs bg-gray-100 px-2 py-0.5 rounded-full text-gray-600">
                        {user?.nativeLanguage?.toUpperCase()}
                      </span>
                      {user?.targetLanguages && user.targetLanguages.length > 0 && (
                        <span className="text-xs text-gray-400">
                          → {user.targetLanguages.map(l => l.toUpperCase()).join(', ')}
                        </span>
                      )}
                    </div>
                  </div>
                  <div className="py-1">
                    <button
                      onClick={() => { setShowSettings(true); setShowProfileMenu(false) }}
                      className="w-full px-4 py-2.5 text-left text-sm text-gray-700 hover:bg-gray-50 flex items-center gap-3"
                    >
                      <span>⚙️</span> Settings
                    </button>
                    <button
                      onClick={() => { setShowAbout(true); setShowProfileMenu(false) }}
                      className="w-full px-4 py-2.5 text-left text-sm text-gray-700 hover:bg-gray-50 flex items-center gap-3"
                    >
                      <span>ℹ️</span> About Chorus
                    </button>
                    <hr className="my-1" />
                    <button
                      onClick={() => { setShowProfileMenu(false); onLogout() }}
                      className="w-full px-4 py-2.5 text-left text-sm text-red-600 hover:bg-red-50 flex items-center gap-3"
                    >
                      <span>🚪</span> Sign Out
                    </button>
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>

        <div className="p-4">
          <button
            onClick={() => setShowNewChatModal(true)}
            className="w-full bg-gradient-to-r from-indigo-600 to-purple-600 text-white py-2.5 px-4 rounded-lg hover:opacity-90 transition font-semibold"
          >
            + New Chat
          </button>
        </div>

        <ChatList />
      </div>

      {/* Main Chat Area */}
      <div className="flex-1 flex flex-col bg-gray-50">
        <ChatArea />
      </div>

      {showNewChatModal && (
        <NewChatModal onClose={() => setShowNewChatModal(false)} />
      )}

      {showSettings && (
        <Settings onClose={() => setShowSettings(false)} />
      )}
    </div>
  )
}
