import { create } from 'zustand'
import type { User, Chat, Message, Entitlements, TranslationBlocked, GrammarJob } from '../types'
import { chatAPI, messageAPI, adminAPI, authAPI, grammarAPI } from '../services/api'
import { wsService } from '../services/websocket'

// --- Slug helpers ---

/** localStorage key used to remember a grammar job across reloads. */
function grammarJobStorageKey(messageId: string): string {
  return `grammar_jobs:${messageId}`
}

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
  entitlements: Entitlements | null
  isAdmin: boolean
  isModerator: boolean
  userRole: string
  chats: Chat[]
  activeChat: Chat | null
  messages: Record<string, Message[]>
  blockedTranslations: Record<string, TranslationBlocked>
  grammarJobs: Record<string, GrammarJob>

  // Actions
  setUser: (user: User | null) => void
  setEntitlements: (entitlements: Entitlements | null) => void
  refreshEntitlements: () => Promise<void>
  setAdmin: (isAdmin: boolean) => void
  setRole: (role: string) => void
  refreshAdminStatus: () => Promise<void>
  loadChats: () => Promise<void>
  setActiveChat: (chat: Chat | null) => void
  loadMessages: (chatId: string) => Promise<void>
  addMessage: (message: Message) => void
  updateMessage: (message: Message) => void
  updateChatLastMessage: (chatId: string, message: Message) => void
  sendMessage: (chatId: string, text: string) => Promise<void>
  createChat: (type: 'direct' | 'group', participants: string[], name?: string) => Promise<Chat>
  updateUser: (updates: Partial<User>) => void
  markTranslationBlocked: (blocked: TranslationBlocked) => void
  setGrammarJob: (job: GrammarJob) => void
  resyncGrammarJob: (messageId: string) => Promise<void>
  // Slug-based navigation
  navigateToSlug: (slug: string) => boolean
}

export const useStore = create<AppState>((set, get) => ({
  user: null,
  entitlements: null,
  isAdmin: false,
  isModerator: false,
  userRole: '',
  chats: [],
  activeChat: null,
  messages: {},
  blockedTranslations: {},
  grammarJobs: {},

  setUser: (user) => set({ user }),
  setEntitlements: (entitlements) => set({ entitlements }),
  refreshEntitlements: async () => {
    try {
      const entitlements = await authAPI.getEntitlements()
      set({ entitlements })
    } catch (error) {
      console.error('Failed to load entitlements:', error)
    }
  },
  setAdmin: (isAdmin) => set({ isAdmin }),
  setRole: (role) => {
    const isAdmin = role === 'admin'
    const isModerator = isAdmin || role === 'moderator'
    set({ userRole: role, isAdmin, isModerator })
  },

  refreshAdminStatus: async () => {
    try {
      const status = await adminAPI.status()
      get().setRole(status.role)
    } catch {
      set({ isAdmin: false, isModerator: false, userRole: '' })
    }
  },

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
    const tempId = `pending-${Date.now()}`
    const optimisticMessage: Message = {
      id: tempId,
      chatId,
      senderId: get().user?.id || '',
      text,
      deliveryStatus: 'sent',
      timestamp: new Date().toISOString(),
    }

    get().addMessage(optimisticMessage)

    try {
      const message = await messageAPI.sendMessage(chatId, { text })
      set((state) => {
        const chatMessages = state.messages[chatId] || []
        const alreadyExists = chatMessages.some(m => m.id === message.id)
        if (alreadyExists) {
          return {
            messages: {
              ...state.messages,
              [chatId]: chatMessages.filter(m => m.id !== tempId),
            },
          }
        }
        return {
          messages: {
            ...state.messages,
            [chatId]: chatMessages.map(m => m.id === tempId ? message : m),
          },
        }
      })
      get().updateChatLastMessage(chatId, message)
    } catch (error) {
      set((state) => {
        const chatMessages = state.messages[chatId] || []
        return {
          messages: {
            ...state.messages,
            [chatId]: chatMessages.map(m =>
              m.id === tempId ? { ...m, deliveryStatus: 'failed' as const } : m
            ),
          },
        }
      })
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

  markTranslationBlocked: (blocked) => {
    set((state) => ({
      blockedTranslations: {
        ...state.blockedTranslations,
        [blocked.messageId]: blocked,
      },
    }))
  },

  // Track AI grammar analysis jobs, keyed by the analyzed message's id, so a
  // bubble can accurately render queued / processing / done / failed states.
  // The job id is persisted so a page reload can re-fetch the outcome.
  setGrammarJob: (job) => {
    const messageId = job.messageId
    if (!messageId) return
    try {
      if (job.status === 'done' || job.status === 'failed') {
        localStorage.removeItem(grammarJobStorageKey(messageId))
      } else {
        localStorage.setItem(grammarJobStorageKey(messageId), JSON.stringify({ jobId: job.jobId, status: job.status }))
      }
    } catch (e) {
      // storage unavailable — resync just won't run
    }
    set((state) => {
      const prev = state.grammarJobs[messageId]
      const next: GrammarJob = {
        ...(prev || {}),
        ...job,
      }
      // Transient statuses must not carry a stale completed analysis.
      if (job.status !== 'done') {
        delete next.analysis
        delete next.providerUsed
      }
      if (job.status !== 'failed') {
        delete next.error
      }
      return {
        grammarJobs: {
          ...state.grammarJobs,
          [messageId]: next,
        },
      }
    })
  },

  // On mount (or reconnect) a job that was started before a reload is no longer
  // in memory. Re-fetch it by id so the bubble shows the real outcome.
  resyncGrammarJob: async (messageId) => {
    let saved: { jobId: string; status: string } | null = null
    try {
      const raw = localStorage.getItem(grammarJobStorageKey(messageId))
      saved = raw ? JSON.parse(raw) : null
    } catch (e) {
      saved = null
    }
    if (!saved?.jobId) return
    const { setGrammarJob } = useStore.getState()
    if (useStore.getState().grammarJobs[messageId]) return
    try {
      const job = await grammarAPI.getAnalysis(saved.jobId)
      setGrammarJob({ ...job, messageId })
    } catch (e) {
      // job may be gone (cleaned up) — drop the saved reference
      try {
        localStorage.removeItem(grammarJobStorageKey(messageId))
      } catch {}
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
    case 'translation_blocked':
      store.markTranslationBlocked(message.data)
      break
    case 'grammar_analysis':
      store.setGrammarJob(message.data)
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
