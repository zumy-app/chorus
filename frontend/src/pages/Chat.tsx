import { useEffect, useState, useRef } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useStore, getChatSlug } from '../store'
import ChatList from '../components/ChatList'
import ChatArea from '../components/ChatArea'
import NewChatModal from '../components/NewChatModal'
import SearchMessages from '../components/SearchMessages'
import Vocabulary from '../components/Vocabulary'
import LanguageSelector from '../components/LanguageSelector'
import { authAPI } from '../services/api'
import Settings from './Settings'
import About from './About'
import PlanBadge from '../components/PlanBadge'
import BottomNav from '../components/BottomNav'

interface ChatProps {
  onLogout: () => void
}

export default function Chat({ onLogout }: ChatProps) {
  const { t } = useTranslation()
  const { slug } = useParams<{ slug: string }>()
  const navigate = useNavigate()
  const { user, isModerator, loadChats, activeChat, chats, setActiveChat, navigateToSlug, updateUser, entitlements } = useStore()
  const [showNewChatModal, setShowNewChatModal] = useState(false)
  const [showSettings, setShowSettings] = useState(false)
  const [showAbout, setShowAbout] = useState(false)
  const [showSearch, setShowSearch] = useState(false)
  const [showVocabulary, setShowVocabulary] = useState(false)
  const [showProfileMenu, setShowProfileMenu] = useState(false)
  const [showMobileChat, setShowMobileChat] = useState(false)
  const [searchQuery, setSearchQuery] = useState('')
  const profileMenuRef = useRef<HTMLDivElement>(null)
  const [, setSlugError] = useState(false)

  useEffect(() => {
    loadChats()
  }, [loadChats])

  // Deep link: when slug is in the URL and chats have loaded, resolve it to a chat
  useEffect(() => {
    if (slug && chats.length > 0) {
      const found = navigateToSlug(slug)
      if (!found) {
        // For direct chats, the other user might have just created an account
        // and we haven't loaded their chat yet. Show a fallback message.
        setSlugError(true)
      } else {
        setSlugError(false)
      }
    }
  }, [slug, chats, navigateToSlug])

  // Sync URL when activeChat changes (e.g. from clicking a chat in the list)
  useEffect(() => {
    if (activeChat && user) {
      const currentSlug = getChatSlug(activeChat, user.id)
      const expectedPath = `/chat/${currentSlug}`
      if (window.location.pathname !== expectedPath) {
        navigate(expectedPath, { replace: true })
      }
    }
  }, [activeChat, user, navigate])

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
    navigate('/chat', { replace: true })
  }

  const handleLanguageChange = async (code: string) => {
    localStorage.setItem('preferredLanguage', code)
    if (user) {
      updateUser({ nativeLanguage: code })
      try {
        await authAPI.updateMe({ nativeLanguage: code })
      } catch (err) {
        console.error('Failed to persist language preference:', err)
      }
    }
  }

  if (!activeChat && slug) {
    return (
      <>
        <div className="h-screen flex flex-col bg-gray-50">
        <header className="bg-white border-b border-gray-200 px-4 py-2.5 flex items-center shrink-0">
          <button onClick={() => { setSlugError(false); navigate('/chat', { replace: true }) }}
                  className="p-2 hover:bg-gray-100 rounded-lg transition mr-2">
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
            </svg>
          </button>
          <h1 className="text-xl font-bold bg-gradient-to-r from-indigo-600 to-purple-600 bg-clip-text text-transparent">Chorus</h1>
        </header>
        <div className="flex-1 flex items-center justify-center">
          <div className="text-center max-w-md p-8">
            <p className="text-5xl mb-4">🔗</p>
            <h2 className="text-xl font-bold text-gray-900 mb-2">{t('chat.chatNotFound')}</h2>
            <p className="text-gray-500 mb-6">{t('chat.chatNotFoundDesc')}</p>
            <button onClick={() => navigate('/chat', { replace: true })}
                    className="px-6 py-2 bg-gradient-to-r from-indigo-600 to-purple-600 text-white rounded-lg hover:opacity-90 transition font-semibold">
              {t('chat.goToChats')}
            </button>
          </div>
        </div>
      </div>
        <BottomNav />
      </>
    )
  }

  if (showAbout) {
    return (
      <div className="relative">
        <button
          onClick={() => setShowAbout(false)}
          className="fixed top-4 left-4 z-50 px-4 py-2 bg-white border border-gray-300 rounded-lg shadow-lg hover:bg-gray-50 font-semibold"
        >
          {t('chat.backToChat')}
        </button>
        <About />
      </div>
    )
  }

  return (
    <div className="h-screen flex flex-col bg-gray-50">
      {/* ===== GLOBAL TOP HEADER BAR ===== */}
      <header className="bg-surface border-b border-outline-variant px-margin-mobile py-2.5 flex items-center justify-between shrink-0 z-40">
        {/* Left: Logo */}
        <div className="flex items-center gap-3">
          <div className="w-8 h-8 bg-primary-container rounded-full flex items-center justify-center">
            <span className="material-symbols-outlined text-on-primary-container text-[18px]" style={{ fontVariationSettings: "'FILL' 1" }}>
              forum
            </span>
          </div>
          <h1 className="font-headline-md text-headline-md font-bold text-primary tracking-tight">
            Chorus
          </h1>
        </div>

        {/* Right: Language selector + Profile */}
        <div className="flex items-center gap-2">
          <PlanBadge />
          <LanguageSelector
            currentLang={user?.nativeLanguage}
            onLanguageChange={handleLanguageChange}
            variant="compact"
          />
          <div className="relative" ref={profileMenuRef}>
            <button
              onClick={() => setShowProfileMenu(!showProfileMenu)}
              className="w-9 h-9 rounded-full bg-gradient-to-br from-indigo-600 to-purple-600 text-white font-bold text-sm flex items-center justify-center hover:opacity-90 transition"
              title={user?.displayName || t('chat.profile')}
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
                    <span>📚</span> {t('chat.vocabulary')}
                  </button>
                  <button
                    onClick={() => { setShowSettings(true); setShowProfileMenu(false) }}
                    className="w-full px-4 py-2.5 text-left text-sm text-gray-700 hover:bg-gray-50 flex items-center gap-3"
                  >
                    <span>⚙️</span> {t('chat.settings')}
                  </button>
                  <button
                    onClick={() => { setShowAbout(true); setShowProfileMenu(false) }}
                    className="w-full px-4 py-2.5 text-left text-sm text-gray-700 hover:bg-gray-50 flex items-center gap-3"
                  >
                    <span>ℹ️</span> {t('chat.about')}
                  </button>
                  {entitlements?.effectivePlan === 'free' && (
                    <button
                      onClick={() => { setShowProfileMenu(false); navigate('/pricing') }}
                      className="w-full px-4 py-2.5 text-left text-sm text-amber-600 hover:bg-amber-50 flex items-center gap-3"
                    >
                      <span>✨</span> {t('plan.upgrade')}
                    </button>
                  )}
                  {isModerator && (
                    <button
                      onClick={() => { setShowProfileMenu(false); navigate('/admin') }}
                      className="w-full px-4 py-2.5 text-left text-sm text-indigo-600 hover:bg-indigo-50 flex items-center gap-3"
                    >
                      <span>🛠️</span> {t('chat.adminControls')}
                    </button>
                  )}
                  <hr className="my-1" />
                  <button
                    onClick={() => { setShowProfileMenu(false); onLogout() }}
                    className="w-full px-4 py-2.5 text-left text-sm text-red-600 hover:bg-red-50 flex items-center gap-3"
                  >
                    <span>🚪</span> {t('chat.signOut')}
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
          w-full md:w-80 md:min-w-[320px] border-r border-outline-variant flex flex-col bg-background shrink-0
          ${showMobileChat ? 'hidden md:flex' : 'flex'}
        `}>
          {/* Search Bar */}
          <div className="p-4 pb-2">
            <div className="relative w-full elevation-1 rounded-xl overflow-hidden bg-surface-container-lowest">
              <div className="flex items-center px-4 py-3 gap-3 focus-within:ring-2 focus-within:ring-primary rounded-xl transition-shadow duration-200">
                <span className="material-symbols-outlined text-outline">search</span>
                <input
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  placeholder={t('chat.searchPlaceholder')}
                  className="flex-1 bg-transparent border-none p-0 font-body-md text-body-md text-on-surface placeholder:text-outline focus:ring-0 focus:outline-none"
                />
                <button
                  aria-label={t('chat.searchMessages')}
                  onClick={() => setShowSearch(true)}
                  className="text-primary hover:bg-surface-container-low rounded-full p-1 transition"
                >
                  <span className="material-symbols-outlined text-[20px]">mic</span>
                </button>
              </div>
            </div>
          </div>

          {/* Learning Insights Bento Grid */}
          <div className="px-4 pb-4 grid grid-cols-2 gap-3 w-full">
            <button className="bg-primary-container text-on-primary-container rounded-xl p-4 flex flex-col gap-2 items-start justify-between elevation-1 hover:brightness-105 transition-all duration-200 active:scale-95 text-left h-28">
              <span className="material-symbols-outlined">psychology</span>
              <div>
                <h3 className="font-label-md text-label-md">{t('chat.dailyReview')}</h3>
                <p className="font-body-sm text-body-sm text-on-primary-container/80 line-clamp-1">{t('chat.dailyReviewDesc')}</p>
              </div>
            </button>
            <button
              onClick={() => navigate('/learn/scenarios')}
              className="bg-surface-container-high text-on-surface rounded-xl p-4 flex flex-col gap-2 items-start justify-between elevation-1 hover:bg-surface-container-highest transition-all duration-200 active:scale-95 text-left h-28 insight-glow relative overflow-hidden"
            >
              <div className="absolute top-0 right-0 w-16 h-16 bg-secondary/10 rounded-bl-full" />
              <span className="material-symbols-outlined text-secondary">forum</span>
              <div>
                <h3 className="font-label-md text-label-md">{t('chat.practicePrompt')}</h3>
                <p className="font-body-sm text-body-sm text-on-surface-variant line-clamp-1">{t('chat.practicePromptDesc')}</p>
              </div>
            </button>
          </div>

          {/* Active Conversations */}
          <h2 className="font-label-md text-label-md text-on-surface-variant px-6 mb-2 uppercase tracking-wider shrink-0">
            {t('chat.activeConversations')}
          </h2>

          <ChatList searchQuery={searchQuery} />
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
                aria-label={t('chat.backToChats')}
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
                      ? activeChat.name || t('chat.unnamedGroup')
                      : otherParticipant?.displayName || t('chat.unknownUser')
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

      {!showMobileChat && (
        <button
          aria-label={t('chat.newChat')}
          onClick={() => setShowNewChatModal(true)}
          className="fixed right-margin-mobile bottom-[100px] md:right-auto md:bottom-6 md:left-72 z-40 w-14 h-14 bg-primary text-on-primary rounded-2xl flex items-center justify-center shadow-[0px_8px_24px_rgba(0,74,198,0.25)] hover:bg-primary/90 transition-transform duration-300 active:scale-90"
        >
          <span className="material-symbols-outlined text-[28px]" style={{ fontVariationSettings: "'FILL' 1" }}>edit_square</span>
        </button>
      )}

      {!showMobileChat && <BottomNav />}
    </div>
  )
}
