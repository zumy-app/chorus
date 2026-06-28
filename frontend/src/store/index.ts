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
  updateChatLastMessage: (chatId: string, message: Message) => void
  sendMessage: (chatId: string, text: string) => Promise<void>
  createChat: (type: 'direct' | 'group', participants: string[], name?: string) => Promise<Chat>
  updateUser: (updates: Partial<User>) => void
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
