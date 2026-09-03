import { useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useStore, getChatSlug } from '../store'
import { formatDistanceToNow } from 'date-fns'

interface ChatListProps {
  searchQuery?: string
}

export default function ChatList({ searchQuery = '' }: ChatListProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { chats, activeChat, user, setActiveChat, presence } = useStore()

  const filteredChats = useMemo(() => {
    const q = searchQuery.trim().toLowerCase()
    if (!q) return chats
    return chats.filter((chat) => {
      const other = chat.type === 'direct'
        ? chat.participants?.find(p => p.user?.id !== user?.id)?.user
        : null
      const name = chat.type === 'group'
        ? (chat.name || '')
        : (other?.displayName || '')
      const preview = chat.lastMessage?.text || ''
      return `${name} ${preview}`.toLowerCase().includes(q)
    })
  }, [chats, searchQuery, user?.id])

  if (filteredChats.length === 0) {
    return (
      <div className="flex-1 flex items-center justify-center text-on-surface-variant p-4 text-center font-body-sm text-body-sm">
        {searchQuery ? t('chat.noMatches') : t('chat.noChatsYet')}
      </div>
    )
  }

  const handleSelectChat = (chat: any) => {
    setActiveChat(chat)
    const slug = getChatSlug(chat, user?.id)
    navigate(`/chat/${slug}`, { replace: true })
  }

  return (
    <div className="flex-1 overflow-y-auto pb-24 md:pb-0 no-scrollbar">
      {filteredChats.map((chat) => {
        const isActive = activeChat?.id === chat.id
        const otherParticipant = chat.type === 'direct'
          ? chat.participants?.find(p => p.user?.id !== user?.id)?.user
          : null

        const chatName = chat.type === 'group'
          ? chat.name || t('chat.unnamedGroup')
          : otherParticipant?.displayName || t('chat.unknownUser')

        const langCode = otherParticipant?.targetLanguages?.[0]
          || otherParticipant?.nativeLanguage
          || user?.targetLanguages?.[0]

        const otherPresence = chat.type === 'direct' && otherParticipant
          ? presence[otherParticipant.id]
          : null
        const isOnline = otherPresence?.status === 'online'
        const isAway = otherPresence?.status === 'away'

        const isUnread = Boolean(chat.unreadCount && chat.unreadCount > 0)

        return (
          <div key={chat.id}>
            <button
              data-testid="chat-list-item"
              onClick={() => handleSelectChat(chat)}
              className={`w-full flex items-center p-3 rounded-xl hover:bg-surface-container-low transition-colors duration-200 active:bg-surface-container gap-4 text-left relative cursor-pointer ${
                isActive ? 'bg-surface-container' : ''
              }`}
            >
              {/* Avatar with language badge */}
              <div className="relative flex-shrink-0">
                <div className={`w-14 h-14 ${chat.type === 'group' ? 'rounded-xl' : 'rounded-full'} overflow-hidden bg-surface-variant elevation-1 flex items-center justify-center`}>
                  <span className="material-symbols-outlined text-[28px] text-outline-variant">person</span>
                </div>
                {langCode && (
                  <div className="absolute -bottom-1 -right-1 bg-surface rounded-full p-[2px]">
                    <div className="w-4 h-4 bg-primary-container rounded-full flex items-center justify-center text-on-primary-container border border-surface shadow-sm">
                      <span className="text-[9px] font-bold leading-none">
                        {langCode.slice(0, 2).toUpperCase()}
                      </span>
                    </div>
                  </div>
                )}
                {chat.type === 'direct' && (
                  <span
                    className={`absolute -top-0.5 -right-0.5 w-3 h-3 rounded-full border-2 border-surface ${
                      isOnline ? 'bg-tertiary-container' : isAway ? 'bg-secondary' : 'bg-on-surface-variant/40'
                    }`}
                    aria-label={isOnline ? t('chat.online') : isAway ? t('chat.away') : t('chat.offline')}
                  />
                )}
              </div>

              {/* Chat details */}
              <div className="flex-1 min-w-0 flex flex-col gap-1 justify-center">
                <div className="flex justify-between items-baseline gap-2">
                  <h3 className={`font-headline-sm text-headline-sm truncate ${isActive ? 'text-primary' : 'text-on-surface'}`}>
                    {chatName}
                  </h3>
                  {chat.lastMessage && (
                    <span className={`font-label-sm text-label-sm flex-shrink-0 ${isUnread ? 'text-primary' : 'text-on-surface-variant'}`}>
                      {formatDistanceToNow(new Date(chat.lastMessage.timestamp), { addSuffix: true })}
                    </span>
                  )}
                </div>
                <div className="flex items-center gap-2">
                  <p className={`font-body-sm text-body-sm truncate flex-1 ${isUnread ? 'text-on-surface font-semibold' : 'text-on-surface-variant'}`}>
                    {chat.lastMessage?.text || t('chat.noMessagesYet')}
                  </p>
                  {isUnread && (
                    <span className={`${chat.unreadCount! > 1 ? 'bg-primary text-on-primary text-[10px] font-bold px-2 py-0.5 rounded-full flex-shrink-0' : 'w-2.5 h-2.5 bg-primary rounded-full flex-shrink-0'}`}>
                      {chat.unreadCount! > 1 ? chat.unreadCount : ''}
                    </span>
                  )}
                </div>
                <div className="flex items-center gap-2 mt-1">
                  {chat.unreadCount && chat.unreadCount > 0 && (
                    <span className="inline-flex items-center gap-1 bg-orange-50 text-orange-700 px-2 py-0.5 rounded-full font-label-sm text-label-sm">
                      <span className="material-symbols-outlined text-[12px]">local_fire_department</span>
                      4 Day Streak
                    </span>
                  )}
                  {langCode && (
                    <span className="font-label-sm text-label-sm text-outline capitalize">
                      {chat.type === 'group' ? t('chat.groupChat') : t('chat.learning', { lang: langCode })}
                    </span>
                  )}
                </div>
              </div>
            </button>
            <div className="w-[calc(100%-4rem)] h-px bg-surface-container-high ml-16" />
          </div>
        )
      })}
    </div>
  )
}
