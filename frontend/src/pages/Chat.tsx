import { useEffect, useState, useRef } from 'react'
import { useStore } from '../store'
import ChatList from '../components/ChatList'
import ChatArea from '../components/ChatArea'
import NewChatModal from '../components/NewChatModal'
import SearchMessages from '../components/SearchMessages'
import Vocabulary from '../components/Vocabulary'
import LanguageSelector from '../components/LanguageSelector'
import Settings from './Settings'
import About from './About'

interface ChatProps {
  onLogout: () => void
}

export default function Chat({ onLogout }: ChatProps) {
  const { user, loadChats, activeChat, setActiveChat, updateUser } = useStore()
  const [showNewChatModal, setShowNewChatModal] = useState(false)
  const [showSettings, setShowSettings] = useState(false)
  const [showAbout, setShowAbout] = useState(false)
  const [showSearch, setShowSearch] = useState(false)
  const [showVocabulary, setShowVocabulary] = useState(false)
  const [showProfileMenu, setShowProfileMenu] = useState(false)
  const [showMobileChat, setShowMobileChat] = useState(false)
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

  // When activeChat changes on mobile, show the chat view
  useEffect(() => {
    if (activeChat && window.innerWidth < 768) {
      setShowMobileChat(true)
    }
  }, [activeChat])

  const handleBackToList = () => {
    setShowMobileChat(false)
    setActiveChat(null)
  }

  const handleLanguageChange = (code: string) => {
    localStorage.setItem('preferredLanguage', code)
    if (user) {
      updateUser({ nativeLanguage: code })
    }
  }

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
    <div className="h-screen flex flex-col bg-gray-50">
      {/* ===== GLOBAL TOP HEADER BAR ===== */}
      <header className="bg-white border-b border-gray-200 px-4 py-2.5 flex items-center justify-between shrink-0 z-40">
        {/* Left: Logo */}
        <div className="flex items-center gap-3">
          <div className="w-8 h-8 bg-gradient-to-br from-indigo-600 to-purple-600 rounded-full flex items-center justify-center">
            <svg className="w-5 h-5 text-white" fill="currentColor" viewBox="0 0 20 20">
              <path d="M2 5a2 2 0 012-2h12a2 2 0 012 2v10a2 2 0 01-2 2H4a2 2 0 01-2-2V5z"></path>
              <path d="M7.5 7.5a1.5 1.5 0 113 0 1.5 1.5 0 01-3 0z"></path>
            </svg>
          </div>
          <h1 className="text-xl font-bold bg-gradient-to-r from-indigo-600 to-purple-600 bg-clip-text text-transparent">
            Chorus
          </h1>
        </div>

        {/* Right: Language selector + Profile */}
        <div className="flex items-center gap-2">
          <LanguageSelector
            currentLang={user?.nativeLanguage}
            onLanguageChange={handleLanguageChange}
            variant="compact"
          />
          <div className="relative" ref={profileMenuRef}>
            <button
              onClick={() => setShowProfileMenu(!showProfileMenu)}
              className="w-9 h-9 rounded-full bg-gradient-to-br from-indigo-600 to-purple-600 text-white font-bold text-sm flex items-center justify-center hover:opacity-90 transition"
              title={user?.displayName || 'Profile'}
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
                    onClick={() => { setShowVocabulary(true); setShowProfileMenu(false) }}
                    className="w-full px-4 py-2.5 text-left text-sm text-gray-700 hover:bg-gray-50 flex items-center gap-3"
                  >
                    <span>📚</span> Vocabulary
                  </button>
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
      </header>

      {/* ===== MAIN CONTENT ===== */}
      <div className="flex flex-1 overflow-hidden">
        {/* Sidebar - Chat List */}
        <div className={`
          w-full md:w-80 md:min-w-[320px] border-r border-gray-200 flex flex-col bg-white shrink-0
          ${showMobileChat ? 'hidden md:flex' : 'flex'}
        `}>
          {/* Action Buttons */}
          <div className="p-4 space-y-2 border-b border-gray-200">
            <button
              onClick={() => setShowNewChatModal(true)}
              className="w-full bg-gradient-to-r from-indigo-600 to-purple-600 text-white py-2.5 px-4 rounded-lg hover:opacity-90 transition font-semibold"
            >
              + New Chat
            </button>
            <button
              onClick={() => setShowSearch(true)}
              className="w-full border border-gray-300 text-gray-700 py-2.5 px-4 rounded-lg hover:bg-gray-50 transition font-medium flex items-center justify-center gap-2"
            >
              <span>🔍</span> Search Messages
            </button>
          </div>

          <ChatList />
        </div>

        {/* Main Chat Area */}
        <div className={`
          flex-1 flex flex-col bg-gray-50 min-w-0
          ${!showMobileChat ? 'hidden md:flex' : 'flex'}
        `}>
          {/* Mobile back button */}
          {activeChat && (
            <div className="md:hidden bg-white border-b border-gray-200 px-2 py-2 flex items-center gap-2 shrink-0">
              <button
                onClick={handleBackToList}
                className="p-2 hover:bg-gray-100 rounded-lg transition"
                aria-label="Back to chats"
              >
                <svg className="w-5 h-5 text-gray-700" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
                </svg>
              </button>
              <div className="flex-1 min-w-0">
                <div className="font-semibold text-gray-900 truncate text-sm">
                  {(() => {
                    const otherParticipant = activeChat.type === 'direct'
                      ? activeChat.participants?.find(p => p.user?.id !== user?.id)?.user
                      : null
                    return activeChat.type === 'group'
                      ? activeChat.name || 'Unnamed Group'
                      : otherParticipant?.displayName || 'Unknown User'
                  })()}
                </div>
              </div>
            </div>
          )}
          <ChatArea />
        </div>
      </div>

      {showNewChatModal && (
        <NewChatModal onClose={() => setShowNewChatModal(false)} />
      )}

      {showSettings && (
        <Settings onClose={() => setShowSettings(false)} />
      )}

      {showSearch && (
        <SearchMessages
          chatId={activeChat?.id}
          onClose={() => setShowSearch(false)}
        />
      )}

      {showVocabulary && (
        <Vocabulary onClose={() => setShowVocabulary(false)} />
      )}
    </div>
  )
}
