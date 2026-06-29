import { useEffect, useRef, useState } from 'react'
import { useStore } from '../store'
import MessageBubble from './MessageBubble'
import ChatLanguageModal from './ChatLanguageModal'
import { wsService } from '../services/websocket'

export default function ChatArea() {
  const { activeChat, messages, user, sendMessage } = useStore()
  const [inputText, setInputText] = useState('')
  const [isTyping, setIsTyping] = useState(false)
  const [showLangSettings, setShowLangSettings] = useState(false)
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const typingTimeoutRef = useRef<number>()

  const chatMessages = activeChat ? messages[activeChat.id] || [] : []

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [chatMessages])

  const handleSend = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!inputText.trim() || !activeChat) return

    try {
      await sendMessage(activeChat.id, inputText)
      setInputText('')
      wsService.sendTyping(activeChat.id, false)
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
          <h2 className="text-2xl font-bold mb-2 text-gray-800">Welcome to Chorus</h2>
          <p className="text-gray-500">Select a chat or create a new one to start messaging</p>
        </div>
      </div>
    )
  }

  const otherParticipant = activeChat.type === 'direct'
    ? activeChat.participants?.find(p => p.user?.id !== user?.id)?.user
    : null

  const chatName = activeChat.type === 'group'
    ? activeChat.name || 'Unnamed Group'
    : otherParticipant?.displayName || 'Unknown User'

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
              <span>{activeChat.participants?.length || 0} members</span>
            )}
          </div>
        </div>
        <button
          onClick={() => setShowLangSettings(true)}
          className="px-3 py-2 text-sm border border-gray-300 rounded-lg hover:bg-gray-50 flex items-center gap-2 transition"
          title="Language Settings"
        >
          <span>🌐</span>
          <span className="hidden sm:inline">Language</span>
        </button>
      </div>

      {/* Messages Area */}
      <div className="flex-1 overflow-y-auto p-4 space-y-4">
        {chatMessages.length === 0 ? (
          <div className="text-center text-gray-500 mt-8">
            No messages yet. Start the conversation!
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
            placeholder="Type a message..."
            className="flex-1 px-4 py-2 border border-gray-300 rounded-lg resize-none focus:outline-none focus:ring-2 focus:ring-primary"
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
            disabled={!inputText.trim()}
            className="bg-primary text-white px-6 py-2 rounded-lg hover:bg-primary/90 transition disabled:opacity-50 disabled:cursor-not-allowed"
          >
            Send
          </button>
        </form>
      </div>

      {showLangSettings && (
        <ChatLanguageModal onClose={() => setShowLangSettings(false)} />
      )}
    </>
  )
}
