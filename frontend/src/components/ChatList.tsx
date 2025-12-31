import { useStore } from '../store'
import { formatDistanceToNow } from 'date-fns'

export default function ChatList() {
  const { chats, activeChat, setActiveChat } = useStore()

  if (chats.length === 0) {
    return (
      <div className="flex-1 flex items-center justify-center text-gray-500 p-4 text-center">
        No chats yet. Create a new chat to get started!
      </div>
    )
  }

  return (
    <div className="flex-1 overflow-y-auto">
      {chats.map((chat) => {
        const isActive = activeChat?.id === chat.id
        const otherParticipant = chat.type === 'direct' 
          ? chat.participants?.find(p => p.user?.id !== useStore.getState().user?.id)?.user
          : null

        const chatName = chat.type === 'group' 
          ? chat.name || 'Unnamed Group'
          : otherParticipant?.displayName || 'Unknown User'

        return (
          <div
            key={chat.id}
            onClick={() => setActiveChat(chat)}
            className={`p-4 border-b border-gray-200 cursor-pointer hover:bg-gray-50 transition ${
              isActive ? 'bg-primary/10 border-l-4 border-l-primary' : ''
            }`}
          >
            <div className="flex items-start justify-between">
              <div className="flex-1 min-w-0">
                <div className="font-semibold text-gray-900 truncate">
                  {chatName}
                </div>
                {chat.type === 'group' && (
                  <div className="text-xs text-gray-500">
                    {chat.participants?.length || 0} members
                  </div>
                )}
                {chat.lastMessage && (
                  <div className="text-sm text-gray-600 truncate mt-1">
                    {chat.lastMessage.text}
                  </div>
                )}
              </div>
              {chat.lastMessage && (
                <div className="text-xs text-gray-500 ml-2">
                  {formatDistanceToNow(new Date(chat.lastMessage.timestamp), {
                    addSuffix: true,
                  })}
                </div>
              )}
            </div>
            {chat.unreadCount && chat.unreadCount > 0 && (
              <div className="mt-2">
                <span className="bg-primary text-white text-xs px-2 py-1 rounded-full">
                  {chat.unreadCount}
                </span>
              </div>
            )}
          </div>
        )
      })}
    </div>
  )
}
