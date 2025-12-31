import { create } from 'zustand'
import type { User, Chat, Message } from '../types'
import { chatAPI, messageAPI } from '../services/api'
import { wsService } from '../services/websocket'

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
  sendMessage: (chatId: string, text: string) => Promise<void>
  createChat: (type: 'direct' | 'group', participants: string[], name?: string) => Promise<Chat>
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
      return {
        messages: {
          ...state.messages,
          [message.chatId]: [...chatMessages, message],
        },
      }
    })
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

  sendMessage: async (chatId, text) => {
    try {
      const message = await messageAPI.sendMessage(chatId, { text })
      get().addMessage(message)
    } catch (error) {
      console.error('Failed to send message:', error)
      throw error
    }
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
      // Reload chats
      store.loadChats()
      break
  }
})
