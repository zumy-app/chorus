import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'
import { useStore } from '../store'
import MessageBubble from './MessageBubble'
import ChatLanguageModal from './ChatLanguageModal'
import ReportModal from './ReportModal'
import { moderationAPI } from '../services/api'
import { wsService } from '../services/websocket'

export default function ChatArea() {
  const { t } = useTranslation()
  const { activeChat, messages, user, entitlements, sendMessage } = useStore()
  const [inputText, setInputText] = useState('')
  const [isTyping, setIsTyping] = useState(false)
  const [showLangSettings, setShowLangSettings] = useState(false)
  const [showChatMenu, setShowChatMenu] = useState(false)
  const [confirmBlock, setConfirmBlock] = useState(false)
  const [showReportModal, setShowReportModal] = useState(false)
  const [actionNotice, setActionNotice] = useState('')
  const [actionError, setActionError] = useState('')
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const typingTimeoutRef = useRef<number>()
  const chatMenuRef = useRef<HTMLDivElement>(null)

  const chatMessages = activeChat ? messages[activeChat.id] || [] : []

  // The translation word limit mirrored from the server entitlements
  // (free = 280, premium = 1,000, self-hosted = unlimited). Any message that
  // exceeds it won't be translated, so we surface that instantly instead of
  // waiting for a round-trip + a server-side "premium needed" notification.
  const wordLimit = entitlements?.features?.translationWordLimit ?? null
  // Whitespace-delimited tokens — matches the backend's WordCount (strings.Fields).
  const wordCount = inputText.trim().split(/\s+/).filter(Boolean).length
  const isOverLimit = wordLimit != null && wordCount > wordLimit

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [chatMessages])

  // Close the header actions menu when clicking outside.
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (chatMenuRef.current && !chatMenuRef.current.contains(e.target as Node)) {
        setShowChatMenu(false)
        setConfirmBlock(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  const getOtherParticipant = () =>
    activeChat && activeChat.type === 'direct'
      ? activeChat.participants?.find((p) => p.user?.id !== user?.id)?.user ?? null
      : null

  const handleBlock = async () => {
    const other = getOtherParticipant()
    if (!other) return
    if (!confirmBlock) {
      setConfirmBlock(true)
      return
    }
    try {
      await moderationAPI.block(other.id)
      setActionNotice(t('report.blocked', { name: other.displayName }))
      setActionError('')
      setConfirmBlock(false)
      setShowChatMenu(false)
      setTimeout(() => setActionNotice(''), 2500)
    } catch (err: any) {
      setActionError(err?.response?.data?.error || t('report.blockError'))
    }
  }

  const handleSend = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!inputText.trim() || !activeChat) return
    if (isOverLimit) return

    const text = inputText
    setInputText('')
    wsService.sendTyping(activeChat.id, false)

    try {
      await sendMessage(activeChat.id, text)
    } catch (error) {
      console.error('Failed to send message:', error)
    }
  }

  const handleInputChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    setInputText(e.target.value)

    if (!activeChat) return

    if (!isTyping) {
      setIsTyping(true)
      wsService.sendTyping(activeChat.id, true)
    }

    if (typingTimeoutRef.current) {
      clearTimeout(typingTimeoutRef.current)
    }

    typingTimeoutRef.current = (setTimeout(() => {
      setIsTyping(false)
      wsService.sendTyping(activeChat.id, false)
    }, 2000) as unknown) as number
  }

  if (!activeChat) {
    return (
      <div className="flex-1 flex items-center justify-center text-gray-500">
        <div className="text-center">
          <div className="w-20 h-20 bg-gradient-to-br from-indigo-100 to-purple-100 rounded-full flex items-center justify-center mx-auto mb-6">
            <svg className="w-10 h-10 text-indigo-400" fill="currentColor" viewBox="0 0 20 20">
              <path d="M2 5a2 2 0 012-2h12a2 2 0 012 2v10a2 2 0 01-2 2H4a2 2 0 01-2-2V5z"></path>
              <path d="M7.5 7.5a1.5 1.5 0 113 0 1.5 1.5 0 01-3 0z"></path>
            </svg>
          </div>
          <h2 className="text-2xl font-bold mb-2 text-gray-800">{t('chat.welcomeTitle')}</h2>
          <p className="text-gray-500">{t('chat.welcomeSubtitle')}</p>
        </div>
      </div>
    )
  }

  const otherParticipant = getOtherParticipant()

  const chatName = activeChat.type === 'group'
    ? activeChat.name || t('chat.unnamedGroup')
    : otherParticipant?.displayName || t('chat.unknownUser')

  return (
    <>
      {/* Chat Header */}
      <div className="bg-white border-b border-gray-200 px-4 py-3 flex items-center justify-between">
        <div>
          <h2 className="text-xl font-semibold text-gray-900">{chatName}</h2>
          <div className="flex items-center gap-2 text-sm text-gray-500">
            {activeChat.type === 'direct' && otherParticipant && (
              <span>🌍 {otherParticipant.nativeLanguage?.toUpperCase()}</span>
            )}
            {activeChat.type === 'group' && (
              <span>{t('common.members', { count: activeChat.participants?.length || 0 })}</span>
            )}
          </div>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => setShowLangSettings(true)}
            className="px-3 py-2 text-sm border border-gray-300 rounded-lg hover:bg-gray-50 flex items-center gap-2 transition"
            title={t('chat.languageSettings')}
          >
            <span>🌐</span>
            <span className="hidden sm:inline">{t('chat.language')}</span>
          </button>

          {otherParticipant && (
            <div className="relative" ref={chatMenuRef}>
              <button
                onClick={() => { setShowChatMenu(!showChatMenu); setConfirmBlock(false) }}
                className="w-9 h-9 flex items-center justify-center border border-gray-300 rounded-lg hover:bg-gray-50 transition"
                title={t('report.moreActions')}
                aria-label={t('report.moreActions')}
              >
                <span className="text-gray-600 text-lg leading-none">⋯</span>
              </button>

              {showChatMenu && (
                <div className="absolute right-0 mt-2 w-56 bg-white rounded-lg shadow-xl border border-gray-200 z-50">
                  <div className="py-1">
                    <button
                      onClick={handleBlock}
                      className={`w-full px-4 py-2.5 text-left text-sm flex items-center gap-3 ${
                        confirmBlock ? 'text-white bg-red-600 hover:bg-red-700' : 'text-red-600 hover:bg-red-50'
                      }`}
                    >
                      <span>🚫</span>
                      {confirmBlock ? t('report.confirmBlock', { name: otherParticipant.displayName }) : t('report.blockUser', { name: otherParticipant.displayName })}
                    </button>
                    <button
                      onClick={() => { setShowReportModal(true); setShowChatMenu(false); setConfirmBlock(false) }}
                      className="w-full px-4 py-2.5 text-left text-sm text-gray-700 hover:bg-gray-50 flex items-center gap-3"
                    >
                      <span>🚩</span> {t('report.reportUser')}
                    </button>
                  </div>
                  {actionError && (
                    <div className="px-4 py-2 text-xs text-red-600 border-t border-gray-100">{actionError}</div>
                  )}
                </div>
              )}
            </div>
          )}
        </div>
      </div>

      {actionNotice && (
        <div className="bg-green-50 border-b border-green-200 px-4 py-2 text-sm text-green-700">{actionNotice}</div>
      )}

      {/* Messages Area */}
      <div className="flex-1 overflow-y-auto p-4 space-y-4">
        {chatMessages.length === 0 ? (
          <div className="text-center text-gray-500 mt-8">
            {t('chat.noMessagesYet')}
          </div>
        ) : (
          chatMessages.map((message) => (
            <MessageBubble
              key={message.id}
              message={message}
              isOwn={message.senderId === user?.id}
              nativeLanguage={user?.nativeLanguage || 'en'}
              targetLanguage={user?.targetLanguages?.[0]}
            />
          ))
        )}
        <div ref={messagesEndRef} />
      </div>

      {/* Input Area */}
      <div className="bg-white border-t border-gray-200 p-4">
        <form onSubmit={handleSend} className="flex items-end space-x-2">
          <textarea
            value={inputText}
            onChange={handleInputChange}
            placeholder={t('chat.typeMessage')}
            className={`flex-1 px-4 py-2 border rounded-lg resize-none focus:outline-none focus:ring-2 focus:ring-primary ${
              isOverLimit ? 'border-amber-400 bg-amber-50' : 'border-gray-300'
            }`}
            rows={1}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault()
                handleSend(e)
              }
            }}
          />
          <button
            type="submit"
            disabled={!inputText.trim() || isOverLimit}
            title={isOverLimit ? t('plan.wordLimitNotice', { limit: wordLimit }) : undefined}
            className="bg-primary text-white px-6 py-2 rounded-lg hover:bg-primary/90 transition disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {t('common.send')}
          </button>
        </form>
        <div className="flex flex-row items-center justify-between gap-2 mt-2 min-h-[1.5rem]">
          <div className="flex items-center gap-2 flex-wrap">
            {isOverLimit ? (
              <>
                <span className="text-xs text-amber-700 font-medium">
                  {t('plan.wordLimitNotice', { limit: wordLimit })}
                </span>
                <Link
                  to="/pricing"
                  className="text-xs px-3 py-1 bg-gradient-to-r from-indigo-600 to-purple-600 text-white rounded-lg font-semibold hover:opacity-90 transition"
                >
                  {t('plan.upgrade')}
                </Link>
              </>
            ) : null}
          </div>
          {wordLimit != null && (
            <span className={`text-xs whitespace-nowrap ${isOverLimit ? 'text-amber-700 font-semibold' : 'text-gray-400'}`}>
              {wordCount.toLocaleString()} / {wordLimit.toLocaleString()}
            </span>
          )}
        </div>
      </div>

      {showLangSettings && (
        <ChatLanguageModal onClose={() => setShowLangSettings(false)} />
      )}

      {showReportModal && otherParticipant && (
        <ReportModal
          targetType="user"
          targetUserId={otherParticipant.id}
          reportedUserName={otherParticipant.displayName}
          onClose={() => setShowReportModal(false)}
        />
      )}
    </>
  )
}
