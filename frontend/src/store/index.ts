import { create } from 'zustand'
import type { User, Chat, Message } from '../types'
import { chatAPI, keyAPI, messageAPI } from '../services/api'
import { wsService } from '../services/websocket'
import {
  decryptMessage,
  encryptMessage,
  ensureDeviceKeys,
  generateChatKey,
  getStoredChatKey,
  storeChatKey,
  unwrapChatKey,
  wrapChatKeyForDevice,
} from '../services/crypto'

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
      const decrypted = await decryptMessagesForDisplay(chatId, messages, get().user)
      set((state) => ({
        messages: {
          ...state.messages,
          [chatId]: decrypted.reverse(),
        },
      }))
    } catch (error) {
      console.error('Failed to load messages:', error)
    }
  },

  addMessage: (message) => {
    const applyMessage = (displayMessage: Message) => {
      set((state) => {
        const chatMessages = state.messages[displayMessage.chatId] || []
        // Avoid duplicates
        if (chatMessages.some(m => m.id === displayMessage.id)) {
          return state
        }
        return {
          messages: {
            ...state.messages,
            [displayMessage.chatId]: [...chatMessages, displayMessage],
          },
        }
      })
      // Also update the chat's last message and reorder
      get().updateChatLastMessage(displayMessage.chatId, displayMessage)
    }
    if (!message.ciphertext) {
      applyMessage(message)
      return
    }
    decryptMessageForDisplay(message, get().user).then(applyMessage)
  },

  updateMessage: (message) => {
    const applyMessage = (displayMessage: Message) => {
      set((state) => {
        const chatMessages = state.messages[displayMessage.chatId] || []
        const index = chatMessages.findIndex((m) => m.id === displayMessage.id)
        if (index !== -1) {
          const newMessages = [...chatMessages]
          newMessages[index] = { ...newMessages[index], ...displayMessage }
          return {
            messages: {
              ...state.messages,
              [displayMessage.chatId]: newMessages,
            },
          }
        }
        return state
      })
    }
    if (!message.ciphertext) {
      applyMessage(message)
      return
    }
    decryptMessageForDisplay(message, get().user).then(applyMessage)
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
      const user = get().user
      if (!user) throw new Error('Cannot send encrypted messages before login')
      await ensureRegisteredDevice(user)
      if (!await getStoredChatKey(chatId)) {
        await ensureChatKeyForRead(chatId, user)
      }
      if (!await getStoredChatKey(chatId)) {
        await generateChatKey(chatId)
      }
      const encryptedPayload = await encryptMessage(chatId, text)
      const message = await messageAPI.sendMessage(chatId, encryptedPayload)
      get().addMessage({ ...message, text })
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
      const user = get().user
      let recipientKeys
      let pendingChatKey: CryptoKey | undefined
      if (user) {
        const localDevice = await ensureRegisteredDevice(user)
        pendingChatKey = await generateChatKey(`pending:${crypto.randomUUID()}`)
        const participantIDs = Array.from(new Set([user.id, ...participants]))
        const deviceGroups = await Promise.all(participantIDs.map(async (participantID) => {
          if (participantID === user.id) {
            return [{ ...localDevice, userId: user.id }]
          }
          try {
            return await keyAPI.getUserDeviceKeys(participantID)
          } catch (error) {
            console.warn('Failed to fetch recipient device keys:', error)
            return []
          }
        }))
        recipientKeys = (await Promise.all(deviceGroups.flat().map((device) =>
          wrapChatKeyForDevice({
            chatId: '',
            userId: device.userId || user.id,
            device,
            chatKey: pendingChatKey as CryptoKey,
          })
        ))).filter((envelope) => envelope.ciphertext)
      }

      const chat = await chatAPI.createChat({ type, participants, name, recipientKeys })
      if (pendingChatKey) {
        await storeChatKey(chat.id, pendingChatKey)
      }
      set((state) => {
        // If the chat already exists (backend returns existing direct chats),
        // move it to the top instead of adding a duplicate
        const existingIndex = state.chats.findIndex(c => c.id === chat.id)
        if (existingIndex !== -1) {
          const updatedChats = [...state.chats]
          const [existing] = updatedChats.splice(existingIndex, 1)
          return { chats: [existing, ...updatedChats] }
        }
        return { chats: [chat, ...state.chats] }
      })
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

async function ensureRegisteredDevice(user: User) {
  const device = await ensureDeviceKeys(user.id)
  try {
    await keyAPI.registerDeviceKeys({
      deviceId: device.deviceId,
      deviceName: device.deviceName,
      deviceType: device.deviceType,
      identityPublicKey: device.identityPublicKey,
      signedPreKey: device.signedPreKey,
      signedPreKeySignature: device.signedPreKeySignature,
      oneTimePreKeys: device.oneTimePreKeys,
    })
  } catch (error) {
    console.warn('Failed to register device keys:', error)
  }
  return device
}

async function ensureChatKeyForRead(chatId: string, user: User | null) {
  if (!user || await getStoredChatKey(chatId)) return
  const device = await ensureDeviceKeys(user.id)
  try {
    const envelope = await keyAPI.getChatRecipientKey(chatId, device.deviceId)
    await unwrapChatKey(chatId, envelope)
  } catch (error) {
    console.warn('No decryptable chat key for this device:', error)
  }
}

async function decryptMessageForDisplay(message: Message, user: User | null) {
  if (message.ciphertext) {
    await ensureChatKeyForRead(message.chatId, user)
  }
  return decryptMessage(message)
}

async function decryptMessagesForDisplay(chatId: string, messages: Message[], user: User | null) {
  if (messages.some((message) => message.ciphertext)) {
    await ensureChatKeyForRead(chatId, user)
  }
  return Promise.all(messages.map((message) => decryptMessage(message)))
}

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

// Re-fetch active chat messages on WebSocket reconnect
// This ensures missed message_updated events are recovered
wsService.onReconnect(() => {
  const store = useStore.getState()
  if (store.activeChat) {
    store.loadMessages(store.activeChat.id)
  }
  store.loadChats()
})
