import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'
import { useStore } from '../store'
import MessageBubble from './MessageBubble'
import DeepDiveSheet from './DeepDiveSheet'
import ChatLanguageModal from './ChatLanguageModal'
import ReportModal from './ReportModal'
import EmojiPicker from './EmojiPicker'
import { moderationAPI } from '../services/api'
import { wsService } from '../services/websocket'
import { formatDistanceToNow } from 'date-fns'

export default function ChatArea() {
  const { t } = useTranslation()
  const { activeChat, messages, user, entitlements, sendMessage, typingUsers, presence, fetchPresence } = useStore()
  const [inputText, setInputText] = useState('')
  const [isTyping, setIsTyping] = useState(false)
  const [showLangSettings, setShowLangSettings] = useState(false)
  const [showChatMenu, setShowChatMenu] = useState(false)
  const [confirmBlock, setConfirmBlock] = useState(false)
  const [showReportModal, setShowReportModal] = useState(false)
  const [showEmojiPicker, setShowEmojiPicker] = useState(false)
  const [actionNotice, setActionNotice] = useState('')
  const [actionError, setActionError] = useState('')
  const [translateAsType, setTranslateAsType] = useState(
    () => localStorage.getItem('translateAsType') === '1'
  )
  const [deepDiveMessage, setDeepDiveMessage] = useState<null | { id: string; text: string; sender?: any; analysis?: any }>(null)
  const [deepDiveOpen, setDeepDiveOpen] = useState(false)
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const typingTimeoutRef = useRef<number>()
  const chatMenuRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLTextAreaElement>(null)

  const chatMessages = activeChat ? messages[activeChat.id] || [] : []
  const targetLang = user?.targetLanguages?.[0]?.toUpperCase()

  // Keep the other participants' presence fresh whenever the active chat (or
  // the current user) changes. Group chats include every participant; direct
  // chats resolve to the one other person.
  useEffect(() => {
    if (!activeChat) return
    const ids = (activeChat.participants || [])
      .filter((p) => p.user?.id !== user?.id)
      .map((p) => p.user?.id)
      .filter((id): id is string => Boolean(id))
    if (ids.length) fetchPresence(ids)
  }, [activeChat?.id, activeChat?.participants, user?.id])

  const handleToggleTranslate = () => {
    setTranslateAsType((prev) => {
      const next = !prev
      localStorage.setItem('translateAsType', next ? '1' : '0')
      return next
    })
  }

  const handleDeepDive = (message: { id: string; text: string; sender?: any; analysis?: any }) => {
    setDeepDiveMessage(message)
    setDeepDiveOpen(true)
  }

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

  // Insert an emoji at the current cursor position in the composer, keeping the
  // caret right after it so the user can keep typing. Emoji are plain characters
  // in the message text — they are never modified, so they pass through
  // translation unchanged (FR-21).
  const insertEmoji = (emoji: string) => {
    const el = inputRef.current
    const start = el?.selectionStart ?? inputText.length
    const end = el?.selectionEnd ?? start
    const next = inputText.slice(0, start) + emoji + inputText.slice(end)
    setInputText(next)
    if (el && typeof requestAnimationFrame !== 'undefined') {
      const pos = start + emoji.length
      requestAnimationFrame(() => {
        el.focus()
        try {
          el.setSelectionRange(pos, pos)
        } catch {
          // jsdom / some browsers may not expose selectionRange; the value is set regardless.
        }
      })
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

  const participantPresence = otherParticipant ? presence[otherParticipant.id] : null
  const isOtherOnline = participantPresence?.status === 'online'
  const isOtherAway = participantPresence?.status === 'away'

  // Display names of everyone (except us) currently typing in the active chat.
  const typingNames = (activeChat
    ? (activeChat.participants || [])
        .filter((p) => p.user?.id && p.user?.id !== user?.id && typingUsers[activeChat.id]?.[p.user.id])
        .map((p) => p.user?.displayName)
        .filter((name): name is string => Boolean(name))
    : []) as string[]
  const otherTyping = typingNames.length > 0

  const presenceLabel = activeChat.type === 'direct' && otherParticipant
    ? isOtherOnline
      ? t('chat.online')
      : isOtherAway
        ? t('chat.away')
        : participantPresence?.lastSeen
          ? t('chat.lastSeen', { time: formatDistanceToNow(new Date(participantPresence.lastSeen), { addSuffix: true }) })
          : t('chat.offline')
    : ''

  return (
    <>
      {/* Chat Header */}
      <div className="bg-surface border-b border-outline-variant px-4 py-3 flex items-center justify-between gap-3">
        <div className="flex items-center gap-3 min-w-0">
          <div className="w-10 h-10 rounded-full bg-primary-container text-on-primary-container flex items-center justify-center font-bold text-sm shrink-0">
            {chatName.charAt(0).toUpperCase()}
          </div>
          <div className="min-w-0">
            <h2 className="font-headline-sm text-headline-sm text-primary truncate">{chatName}</h2>
            <div className="flex items-center gap-2">
              {activeChat.type === 'direct' && otherParticipant && (
                <span
                  className={`font-label-sm text-label-sm flex items-center gap-1 ${
                    isOtherOnline ? 'text-tertiary-container' : isOtherAway ? 'text-secondary' : 'text-on-surface-variant'
                  }`}
                >
                  <span
                    className={`w-2 h-2 rounded-full ${
                      isOtherOnline ? 'bg-tertiary-container' : isOtherAway ? 'bg-secondary' : 'bg-on-surface-variant/40'
                    }`}
                  />
                  {presenceLabel}
                </span>
              )}
              {activeChat.type === 'group' && (
                <span className="font-label-sm text-label-sm text-on-surface-variant">
                  {otherTyping
                    ? t('chat.someoneTyping')
                    : t('common.members', { count: activeChat.participants?.length || 0 })}
                </span>
              )}
            </div>
          </div>
        </div>
        <div className="flex items-center gap-1 shrink-0">
          <button
            onClick={() => setShowLangSettings(true)}
            className="w-10 h-10 flex items-center justify-center text-primary hover:bg-surface-variant/20 rounded-full transition active:scale-95"
            title={t('chat.languageSettings')}
            aria-label={t('chat.languageSettings')}
          >
            <span className="material-symbols-outlined">translate</span>
          </button>

          {otherParticipant && (
            <div className="relative" ref={chatMenuRef}>
              <button
                onClick={() => { setShowChatMenu(!showChatMenu); setConfirmBlock(false) }}
                className="w-10 h-10 flex items-center justify-center text-on-surface-variant hover:bg-surface-variant/20 rounded-full transition active:scale-95"
                title={t('report.moreActions')}
                aria-label={t('report.moreActions')}
              >
                <span className="material-symbols-outlined">more_vert</span>
              </button>

              {showChatMenu && (
                <div className="absolute right-0 mt-2 w-56 bg-surface-container-lowest rounded-xl shadow-[0px_8px_24px_rgba(0,0,0,0.12)] border border-outline-variant/30 z-50 overflow-hidden">
                  <div className="py-1">
                    <button
                      onClick={handleBlock}
                      className={`w-full px-4 py-2.5 text-left text-sm flex items-center gap-3 ${
                        confirmBlock ? 'text-white bg-error hover:bg-error' : 'text-error hover:bg-error-container'
                      }`}
                    >
                      <span>🚫</span>
                      {confirmBlock ? t('report.confirmBlock', { name: otherParticipant.displayName }) : t('report.blockUser', { name: otherParticipant.displayName })}
                    </button>
                    <button
                      onClick={() => { setShowReportModal(true); setShowChatMenu(false); setConfirmBlock(false) }}
                      className="w-full px-4 py-2.5 text-left text-sm text-on-surface hover:bg-surface-container-low flex items-center gap-3"
                    >
                      <span>🚩</span> {t('report.reportUser')}
                    </button>
                  </div>
                  {actionError && (
                    <div className="px-4 py-2 text-xs text-error border-t border-outline-variant/30">{actionError}</div>
                  )}
                </div>
              )}
            </div>
          )}
        </div>
      </div>

      {actionNotice && (
        <div className="bg-tertiary-container text-on-tertiary-container border-b border-outline-variant px-4 py-2 text-sm">{actionNotice}</div>
      )}

      {/* Sparky FAB */}
      <button
        onClick={() => { setDeepDiveMessage(null); setDeepDiveOpen(true) }}
        title={t('grammar.sparkyHint')}
        aria-label={t('grammar.sparkyHint')}
        className="absolute bottom-36 right-4 z-40 w-14 h-14 bg-secondary text-white rounded-full flex items-center justify-center shadow-[0_4px_12px_rgba(132,85,239,0.3)] hover:bg-secondary-container transition-colors active:scale-95"
      >
        <span className="material-symbols-outlined text-[28px]">robot_2</span>
        <span className="absolute top-1 right-1 w-3 h-3 bg-tertiary-fixed rounded-full border-2 border-surface animate-pulse" />
      </button>

      {/* Messages Area */}
      <div className="flex-1 overflow-y-auto p-4 space-y-4 bg-background">
        {chatMessages.length === 0 ? (
          <div className="text-center text-on-surface-variant mt-8">
            {t('chat.noMessagesYet')}
          </div>
        ) : (
          <>
            <div className="flex justify-center">
              <span className="font-label-sm text-label-sm text-on-surface-variant bg-surface-container-high px-3 py-1 rounded-full">
                {t('chat.today')}
              </span>
            </div>
            {chatMessages.map((message) => (
              <MessageBubble
                key={message.id}
                message={message}
                isOwn={message.senderId === user?.id}
                nativeLanguage={user?.nativeLanguage || 'en'}
                targetLanguage={user?.targetLanguages?.[0]}
                onDeepDive={handleDeepDive}
              />
            ))}
          </>
        )}
        <div ref={messagesEndRef} />
      </div>

      {/* Input Area */}
      <div className="bg-surface border-t border-outline-variant px-4 pt-3 pb-4">
        {/* Translate as I type toggle */}
        <div className="flex items-center justify-between mb-3 px-1">
          <button
            type="button"
            onClick={handleToggleTranslate}
            aria-pressed={translateAsType}
            className="flex items-center gap-2 text-on-surface-variant hover:text-primary transition"
          >
            <span className="material-symbols-outlined text-[20px]">g_translate</span>
            <span className="font-label-md text-label-md">{t('chat.translateAsType')}</span>
          </button>
          <button
            type="button"
            role="switch"
            aria-checked={translateAsType}
            onClick={handleToggleTranslate}
            className={`relative inline-flex items-center h-5 w-10 rounded-full transition-colors ${
              translateAsType ? 'bg-primary' : 'bg-surface-variant border border-outline-variant'
            }`}
          >
            <span
              className={`inline-block h-4 w-4 bg-white rounded-full shadow transition-transform ${
                translateAsType ? 'translate-x-[22px]' : 'translate-x-[2px]'
              }`}
            />
          </button>
        </div>

        {/* Other participants typing */}
        {activeChat.type === 'group' ? (
          otherTyping && (
            <div className="mb-3 px-1 flex items-center gap-2 text-on-surface-variant">
              <TypingDots />
              <span className="font-label-sm text-label-sm">
                {typingNames.length === 1
                  ? t('chat.isTyping', { name: typingNames[0] })
                  : t('chat.someoneTyping')}
              </span>
            </div>
          )
        ) : (
          otherTyping && typingNames[0] && (
            <div className="mb-3 px-1 flex items-center gap-2 text-on-surface-variant">
              <TypingDots />
              <span className="font-label-sm text-label-sm">{t('chat.isTyping', { name: typingNames[0] })}</span>
            </div>
          )
        )}

        {/* Live translation preview while typing */}
        {translateAsType && inputText.trim() && (
          <div className="mb-3 rounded-xl border border-secondary/30 bg-secondary-fixed/10 px-3 py-2">
            <div className="text-xs text-secondary flex items-center gap-1 mb-1">
              <span className="material-symbols-outlined text-[14px]">auto_awesome</span>
              {t('chat.liveTranslationPreview', { lang: targetLang })}
            </div>
            <div className="text-sm italic text-on-surface-variant break-words">{inputText}</div>
          </div>
        )}

        <form onSubmit={handleSend} className="flex items-end gap-2">
          <button
            type="button"
            title={t('chat.settings')}
            className="w-10 h-10 flex items-center justify-center text-primary hover:bg-surface-variant/20 rounded-full transition active:scale-95 shrink-0"
          >
            <span className="material-symbols-outlined text-[22px]">add_circle</span>
          </button>
          <div className="relative shrink-0">
            <button
              type="button"
              onClick={() => setShowEmojiPicker((v) => !v)}
              aria-label={t('chat.emoji')}
              aria-expanded={showEmojiPicker}
              title={t('chat.emoji')}
              className="w-10 h-10 flex items-center justify-center text-on-surface-variant hover:bg-surface-variant/20 rounded-full transition active:scale-95 shrink-0"
            >
              <span className="material-symbols-outlined text-[22px]">emoji_emotions</span>
            </button>
            {showEmojiPicker && (
              <EmojiPicker
                onSelect={(emoji) => insertEmoji(emoji)}
                onClose={() => setShowEmojiPicker(false)}
              />
            )}
          </div>
          <textarea
            ref={inputRef}
            value={inputText}
            onChange={handleInputChange}
            placeholder={t('chat.typeMessage')}
            className={`flex-1 px-4 py-3 rounded-[1.5rem] bg-surface-container-low border resize-none focus:outline-none focus:ring-2 focus:ring-primary ${
              isOverLimit ? 'border-error bg-error-container/20' : 'border-outline-variant'
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
            type="button"
            title={t('chat.settings')}
            className="w-10 h-10 flex items-center justify-center text-on-surface-variant hover:bg-surface-variant/20 rounded-full transition active:scale-95 shrink-0"
          >
            <span className="material-symbols-outlined text-[22px]">mic</span>
          </button>
          <button
            type="submit"
            disabled={!inputText.trim() || isOverLimit}
            title={isOverLimit ? t('plan.wordLimitNotice', { limit: wordLimit }) : undefined}
            className="w-10 h-10 bg-primary text-on-primary rounded-full flex items-center justify-center hover:bg-primary/90 transition disabled:opacity-40 disabled:cursor-not-allowed shrink-0 active:scale-95"
            aria-label={t('common.send')}
          >
            <span className="material-symbols-outlined text-[22px]">send</span>
          </button>
        </form>
        <div className="flex flex-row items-center justify-between gap-2 mt-2 min-h-[1.5rem]">
          <div className="flex items-center gap-2 flex-wrap">
            {isOverLimit ? (
              <>
                <span className="text-xs text-error font-medium">
                  {t('plan.wordLimitNotice', { limit: wordLimit })}
                </span>
                <Link
                  to="/pricing"
                  className="text-xs px-3 py-1 bg-primary-container text-on-primary-container rounded-lg font-semibold hover:bg-primary hover:text-on-primary transition"
                >
                  {t('plan.upgrade')}
                </Link>
              </>
            ) : null}
          </div>
          {wordLimit != null && (
            <span className={`text-xs whitespace-nowrap ${isOverLimit ? 'text-error font-semibold' : 'text-on-surface-variant'}`}>
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

      {deepDiveOpen && (
        <DeepDiveSheet message={deepDiveMessage} onClose={() => setDeepDiveOpen(false)} />
      )}
    </>
  )
}

// Animated three-dot typing indicator (ChatArea: FR-9 presence + typing).
function TypingDots() {
  return (
    <span className="flex items-center gap-[3px]" aria-hidden="true">
      {[0, 150, 300].map((delay) => (
        <span
          key={delay}
          className="w-1 h-1 rounded-full bg-current animate-bounce"
          style={{ animationDelay: `${delay}ms` }}
        />
      ))}
    </span>
  )
}
