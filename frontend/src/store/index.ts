import { create } from 'zustand'
import type { User, Chat, Message } from '../types'
import { chatAPI, messageAPI } from '../services/api'
import { wsService } from '../services/websocket'

// --- Slug helpers ---

/** Generate a human-readable URL slug for a chat. */
export function getChatSlug(chat: Chat, currentUserId?: string): string {
  if (chat.type === 'direct') {
    const other = chat.participants?.find(p => p.user?.id !== currentUserId)?.user
    if (other?.username) return `@${other.username}`
    if (other?.displayName) return `@${other.displayName.replace(/\s+/g, '-').toLowerCase()}`
    return `dm-${chat.id.slice(0, 8)}`
  }
  // Group chat
  if (chat.name) {
    return `group/${chat.name.replace(/[^a-zA-Z0-9]+/g, '-').replace(/^-+|-+$/g, '').toLowerCase() || 'unnamed'}`
  }
  return `group/${chat.id.slice(0, 8)}`
}

/** Find a chat by its slug. Returns null if not found. */
export function findChatBySlug(chats: Chat[], slug: string, currentUserId?: string): Chat | null {
  // Direct chat: /chat/@username
  if (slug.startsWith('@')) {
    const identifier = slug.slice(1).toLowerCase()
    return chats.find(c => {
      if (c.type !== 'direct') return false
      const other = c.participants?.find(p => p.user?.id !== currentUserId)?.user
      if (!other) return false
      return other.username?.toLowerCase() === identifier ||
             other.displayName?.toLowerCase().replace(/\s+/g, '-') === identifier
    }) || null
  }
  // Group chat: /chat/group/some-name
  if (slug.startsWith('group/')) {
    const namePart = slug.slice(6)
    return chats.find(c =>
      c.type === 'group' &&
      c.name?.toLowerCase().replace(/[^a-zA-Z0-9]+/g, '-').replace(/^-+|-+$/g, '') === namePart
    ) || null
  }
  // Fallback: try raw chat ID (old bookmark compatibility)
  return chats.find(c => c.id === slug) || null
}

// --- Store ---

interface AppState {
  user: User | null
  chats: Chat[]
  activeChat: Chat | null
  messages: Record<string, Message[]>
  
  // Actions
  setUser: (user: User | null) => void
  loadChats: () => Promise<void>
  setActiveChat: (chat: Chat | null) => void
  loadMessages: (chatId: string) => Promise<void>
  addMessage: (message: Message) => void
  updateMessage: (message: Message) => void
  updateChatLastMessage: (chatId: string, message: Message) => void
  sendMessage: (chatId: string, text: string) => Promise<void>
  createChat: (type: 'direct' | 'group', participants: string[], name?: string) => Promise<Chat>
  updateUser: (updates: Partial<User>) => void
  // Slug-based navigation
  navigateToSlug: (slug: string) => boolean
}

export const useStore = create<AppState>((set, get) => ({
  user: null,
  chats: [],
  activeChat: null,
  messages: {},

  setUser: (user) => set({ user }),

  loadChats: async () => {
    try {
      const chats = await chatAPI.getChats()
      set({ chats })
    } catch (error) {
      console.error('Failed to load chats:', error)
    }
  },

  setActiveChat: (chat) => {
    set({ activeChat: chat })
    if (chat) {
      get().loadMessages(chat.id)
    }
  },

  loadMessages: async (chatId) => {
    try {
      const messages = await messageAPI.getMessages(chatId)
      set((state) => ({
        messages: {
          ...state.messages,
          [chatId]: messages.reverse(),
        },
      }))
    } catch (error) {
      console.error('Failed to load messages:', error)
    }
  },

  addMessage: (message) => {
    set((state) => {
      const chatMessages = state.messages[message.chatId] || []
      // Avoid duplicates
      if (chatMessages.some(m => m.id === message.id)) {
        return state
      }
      return {
        messages: {
          ...state.messages,
          [message.chatId]: [...chatMessages, message],
        },
      }
    })
    // Also update the chat's last message and reorder
    get().updateChatLastMessage(message.chatId, message)
  },

  updateMessage: (message) => {
    set((state) => {
      const chatMessages = state.messages[message.chatId] || []
      const index = chatMessages.findIndex((m) => m.id === message.id)
      if (index !== -1) {
        const newMessages = [...chatMessages]
        newMessages[index] = { ...newMessages[index], ...message }
        return {
          messages: {
            ...state.messages,
            [message.chatId]: newMessages,
          },
        }
      }
      return state
    })
  },

  updateChatLastMessage: (chatId, message) => {
    set((state) => {
      const chatIndex = state.chats.findIndex(c => c.id === chatId)
      if (chatIndex === -1) return state
      
      const updatedChats = [...state.chats]
      updatedChats[chatIndex] = {
        ...updatedChats[chatIndex],
        lastMessage: message,
      }
      
      // Move chat to top of list
      const chat = updatedChats.splice(chatIndex, 1)[0]
      updatedChats.unshift(chat)
      
      return { chats: updatedChats }
    })
  },

  sendMessage: async (chatId, text) => {
    try {
      const message = await messageAPI.sendMessage(chatId, { text })
      get().addMessage(message)
    } catch (error) {
      console.error('Failed to send message:', error)
      throw error
    }
  },

  navigateToSlug: (slug: string) => {
    const { chats, user, setActiveChat } = get()
    const chat = findChatBySlug(chats, slug, user?.id)
    if (chat) {
      setActiveChat(chat)
      return true
    }
    return false
  },

  createChat: async (type, participants, name) => {
    try {
      const chat = await chatAPI.createChat({ type, participants, name })
      set((state) => ({
        chats: [chat, ...state.chats],
      }))
      return chat
    } catch (error) {
      console.error('Failed to create chat:', error)
      throw error
    }
  },

  updateUser: (updates) => {
    set((state) => ({
      user: state.user ? { ...state.user, ...updates } : null,
    }))
  },
}))

// Setup WebSocket listeners
wsService.onMessage((message) => {
  const store = useStore.getState()
  
  switch (message.type) {
    case 'new_message':
      store.addMessage(message.data)
      break
    case 'message_updated':
      store.updateMessage(message.data)
      break
    case 'chat_updated':
      store.loadChats()
      break
  }
})
