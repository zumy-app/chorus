import { useEffect, useRef, useState } from 'react'
import { useStore } from '../store'
import MessageBubble from './MessageBubble'
import { wsService } from '../services/websocket'

export default function ChatArea() {
  const { activeChat, messages, user, sendMessage } = useStore()
  const [inputText, setInputText] = useState('')
  const [isTyping, setIsTyping] = useState(false)
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

    // Send typing indicator
    if (!isTyping) {
      setIsTyping(true)
      wsService.sendTyping(activeChat.id, true)
    }

    // Clear previous timeout
    if (typingTimeoutRef.current) {
      clearTimeout(typingTimeoutRef.current)
    }

    // Stop typing after 2 seconds of inactivity
    typingTimeoutRef.current = setTimeout(() => {
      setIsTyping(false)
      wsService.sendTyping(activeChat.id, false)
    }, 2000)
  }

  if (!activeChat) {
    return (
      <div className="flex-1 flex items-center justify-center text-gray-500">
        <div className="text-center">
          <h2 className="text-2xl font-bold mb-2">Welcome to Chorus</h2>
          <p>Select a chat or create a new one to start messaging</p>
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
      <div className="bg-white border-b border-gray-200 p-4">
        <h2 className="text-xl font-semibold text-gray-900">{chatName}</h2>
        {activeChat.type === 'group' && (
          <p className="text-sm text-gray-500">
            {activeChat.participants?.length || 0} members
          </p>
        )}
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
              userLanguage={user?.targetLanguages?.[0] || user?.nativeLanguage || 'en'}
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
    </>
  )
}
