import { useEffect, useState } from 'react'
import { useStore } from '../store'
import ChatList from '../components/ChatList'
import ChatArea from '../components/ChatArea'
import NewChatModal from '../components/NewChatModal'

interface ChatProps {
  onLogout: () => void
}

export default function Chat({ onLogout }: ChatProps) {
  const { user, loadChats } = useStore()
  const [showNewChatModal, setShowNewChatModal] = useState(false)

  useEffect(() => {
    loadChats()
  }, [loadChats])

  return (
    <div className="h-screen flex">
      {/* Sidebar */}
      <div className="w-80 border-r border-gray-200 flex flex-col bg-white">
        <div className="p-4 border-b border-gray-200">
          <div className="flex items-center justify-between mb-4">
            <h1 className="text-2xl font-bold text-primary">Chorus</h1>
            <button
              onClick={onLogout}
              className="text-sm text-gray-600 hover:text-gray-800"
            >
              Logout
            </button>
          </div>
          <div className="text-sm text-gray-600">
            {user?.displayName} (@{user?.username})
          </div>
          <div className="text-xs text-gray-500 mt-1">
            🌍 {user?.nativeLanguage?.toUpperCase()}
            {user?.targetLanguages && user.targetLanguages.length > 0 && (
              <> → {user.targetLanguages.map(l => l.toUpperCase()).join(', ')}</>
            )}
          </div>
        </div>

        <div className="p-4">
          <button
            onClick={() => setShowNewChatModal(true)}
            className="w-full bg-primary text-white py-2 px-4 rounded-lg hover:bg-primary/90 transition"
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
    </div>
  )
}
