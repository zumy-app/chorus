import { create } from 'zustand'
import type { User, Chat, Message, Entitlements, TranslationBlocked, GrammarJob, PresenceStatus } from '@chorus/shared'
import { chatAPI, messageAPI, adminAPI, authAPI, grammarAPI, presenceAPI } from '../services/api'
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
  typingUsers: Record<string, Record<string, boolean>>
  presence: Record<string, PresenceStatus>

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
  removeMessage: (chatId: string, messageId: string) => void
  updateChatLastMessage: (chatId: string, message: Message) => void
  sendMessage: (chatId: string, text: string, replyToId?: string) => Promise<void>
  sendAttachment: (chatId: string, file: File, opts?: { caption?: string; type?: string }) => Promise<void>
  sendLocation: (chatId: string, latitude: number, longitude: number, label?: string, replyToId?: string) => Promise<void>
  deleteMessage: (chatId: string, messageId: string) => Promise<void>
  forwardMessage: (sourceChatId: string, messageId: string, targetChatId: string) => Promise<void>
  pinMessage: (chatId: string, messageId: string) => Promise<void>
  unpinMessage: (chatId: string, messageId: string) => Promise<void>
  createChat: (type: 'direct' | 'group', participants: string[], name?: string) => Promise<Chat>
  updateUser: (updates: Partial<User>) => void
  markTranslationBlocked: (blocked: TranslationBlocked) => void
  setGrammarJob: (job: GrammarJob) => void
  resyncGrammarJob: (messageId: string) => Promise<void>
  setTyping: (chatId: string, userId: string, isTyping: boolean) => void
  setPresence: (presence: PresenceStatus) => void
  fetchPresence: (userIds: string[]) => Promise<void>
  // Slug-based navigation
  navigateToSlug: (slug: string) => boolean
}

// Typing indicators auto-expire if a typing_stop is missed (e.g. the other
// user's tab crashed), so the UI never shows a stuck "typing…" state.
const typingExpiryTimers: Record<string, ReturnType<typeof setTimeout>> = {}

// Collects the participant ids of every direct chat so we can fetch presence.
function directChatParticipantIds(chats: Chat[], currentUserId?: string): string[] {
  const ids = new Set<string>()
  for (const chat of chats) {
    if (chat.type !== 'direct') continue
    for (const p of chat.participants) {
      if (p.userId !== currentUserId) ids.add(p.userId)
    }
  }
  return Array.from(ids)
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
  typingUsers: {},
  presence: {},

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
      get().fetchPresence(directChatParticipantIds(chats, get().user?.id))
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

  removeMessage: (chatId, messageId) => {
    set((state) => {
      const chatMessages = state.messages[chatId] || []
      return {
        messages: {
          ...state.messages,
          [chatId]: chatMessages.filter((m) => m.id !== messageId),
        },
      }
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

  sendAttachment: async (chatId, file, opts) => {
    const tempId = `pending-${Date.now()}`
    const optimistic: Message = {
      id: tempId,
      chatId,
      senderId: get().user?.id || '',
      text: opts?.caption || file.name,
      deliveryStatus: 'sent',
      timestamp: new Date().toISOString(),
      media: [{ id: tempId, messageId: tempId, chatId, type: 'document', fileName: file.name, fileSize: file.size, mimeType: file.type || 'application/octet-stream', url: '', createdAt: new Date().toISOString() } as any],
    }
    get().addMessage(optimistic)
    try {
      const message = await messageAPI.sendAttachment(chatId, file, file.name, opts)
      set((state) => {
        const chatMessages = state.messages[chatId] || []
        return { messages: { ...state.messages, [chatId]: chatMessages.map(m => m.id === tempId ? message : m) } }
      })
      get().updateChatLastMessage(chatId, message)
    } catch (error) {
      set((state) => {
        const chatMessages = state.messages[chatId] || []
        return { messages: { ...state.messages, [chatId]: chatMessages.filter(m => m.id !== tempId) } }
      })
      throw error
    }
  },

  sendLocation: async (chatId, latitude, longitude, label, replyToId) => {
    const tempId = `pending-${Date.now()}`
    const text = label?.trim() || 'Shared a location'
    const optimistic: Message = {
      id: tempId,
      chatId,
      senderId: get().user?.id || '',
      text,
      deliveryStatus: 'sent',
      timestamp: new Date().toISOString(),
      media: [{ id: tempId, messageId: tempId, chatId, type: 'location', fileName: 'location', fileSize: 0, mimeType: 'application/vnd.chorus.location', url: `https://www.openstreetmap.org/?mlat=${latitude}&mlon=${longitude}#map=15/${latitude}/${longitude}`, latitude, longitude, locationName: label || '', createdAt: new Date().toISOString() } as any],
    }
    get().addMessage(optimistic)
    try {
      const message = await messageAPI.sendLocation(chatId, { latitude, longitude, label, replyToId } as any)
      set((state) => {
        const chatMessages = state.messages[chatId] || []
        return { messages: { ...state.messages, [chatId]: chatMessages.map(m => m.id === tempId ? message : m) } }
      })
      get().updateChatLastMessage(chatId, message)
    } catch (error) {
      set((state) => {
        const chatMessages = state.messages[chatId] || []
        return { messages: { ...state.messages, [chatId]: chatMessages.filter(m => m.id !== tempId) } }
      })
      throw error
    }
  },

  sendMessage: async (chatId, text, replyToId) => {
    const tempId = `pending-${Date.now()}`
    const optimisticMessage: Message = {
      id: tempId,
      chatId,
      senderId: get().user?.id || '',
      text,
      replyToId: replyToId || undefined,
      deliveryStatus: 'sent',
      timestamp: new Date().toISOString(),
    }

    get().addMessage(optimisticMessage)

    try {
      const message = await messageAPI.sendMessage(chatId, { text, replyToId } as any)
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

  deleteMessage: async (chatId, messageId) => {
    await messageAPI.deleteMessage(chatId, messageId)
    get().removeMessage(chatId, messageId)
  },

  forwardMessage: async (sourceChatId, messageId, targetChatId) => {
    const fwd = await messageAPI.forwardMessage(sourceChatId, messageId, targetChatId)
    get().addMessage(fwd)
  },

  pinMessage: async (chatId, messageId) => {
    await messageAPI.pinMessage(chatId, messageId)
  },

  unpinMessage: async (chatId, messageId) => {
    await messageAPI.unpinMessage(chatId, messageId)
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

  setTyping: (chatId, userId, isTyping) => {
    const key = `${chatId}:${userId}`
    const existing = typingExpiryTimers[key]
    if (existing) {
      clearTimeout(existing)
      delete typingExpiryTimers[key]
    }
    if (isTyping) {
      typingExpiryTimers[key] = setTimeout(() => {
        set((state) => {
          const chatTyping = state.typingUsers[chatId] || {}
          if (!chatTyping[userId]) return state
          return {
            typingUsers: {
              ...state.typingUsers,
              [chatId]: { ...chatTyping, [userId]: false },
            },
          }
        })
        delete typingExpiryTimers[key]
      }, 5000)
    }
    set((state) => {
      const chatTyping = state.typingUsers[chatId] || {}
      if ((chatTyping[userId] || false) === isTyping) return state
      return {
        typingUsers: {
          ...state.typingUsers,
          [chatId]: { ...chatTyping, [userId]: isTyping },
        },
      }
    })
  },

  setPresence: (presence) => {
    if (!presence?.userId) return
    set((state) => ({
      presence: {
        ...state.presence,
        [presence.userId]: { ...state.presence[presence.userId], ...presence },
      },
    }))
  },

  fetchPresence: async (userIds) => {
    const ids = (userIds || []).filter(Boolean)
    if (ids.length === 0) return
    try {
      const result = await presenceAPI.getMultiple(ids)
      if (!result) return
      set((state) => {
        const next = { ...state.presence }
        for (const userId of ids) {
          const p = result[userId]
          if (p) next[userId] = { ...next[userId], ...p }
        }
        return { presence: next }
      })
    } catch (error) {
      // Presence is best-effort; never block chat on a failed snapshot.
    }
  },
}))

function applyReceipt(chatId: string, messageId: string, userId: string, status: string) {
  const s = useStore.getState()
  const msgs = s.messages[chatId]
  if (!msgs) return
  const idx = msgs.findIndex(m => m.id === messageId)
  if (idx === -1) return
  const msg = msgs[idx]
  const existing = msg.receipts || []
  const receiptStatus = status === 'read' ? 'read' : 'delivered'
  let found = false
  const next = existing.map(r => {
    if (userId && r.userId === userId) {
      found = true
      return { ...r, status: receiptStatus as 'sent' | 'delivered' | 'read', deliveredAt: new Date().toISOString(), readAt: receiptStatus === 'read' ? new Date().toISOString() : r.readAt }
    }
    return r
  })
  if (!found) {
    next.push({ messageId, chatId, userId: userId || '', status: receiptStatus as 'sent' | 'delivered' | 'read', deliveredAt: new Date().toISOString(), readAt: receiptStatus === 'read' ? new Date().toISOString() : undefined })
  }
  s.updateMessage({ ...msg, receipts: next } as Message)
}

wsService.onMessage((message) => {
  const store = useStore.getState()

  switch (message.type) {
    case 'new_message': {
      store.addMessage(message.data)
      const m = message.data as Message
      if (m && m.senderId !== store.user?.id && m.chatId && m.id) {
        wsService.sendReceipt(m.chatId, m.id, 'received')
        if (store.activeChat?.id === m.chatId) {
          setTimeout(() => wsService.sendReceipt(m.chatId, m.id, 'read'), 300)
          messageAPI.markAsRead(m.chatId, m.id).catch(() => {})
        }
      }
      break
    }
    case 'message_updated':
      store.updateMessage(message.data)
      break
    case 'message_deleted': {
      const d = message.data as { chatId: string; messageId: string }
      if (d?.chatId && d?.messageId) store.removeMessage(d.chatId, d.messageId)
      break
    }
    case 'message_pinned':
    case 'message_unpinned':
      break
    case 'message_delivered': {
      const d = message.data as { chatId: string; messageId: string; userId: string; status: string }
      if (d?.chatId && d?.messageId) applyReceipt(d.chatId, d.messageId, d.userId || '', 'delivered')
      break
    }
    case 'message_read': {
      const d = message.data as { chatId: string; messageId: string; userId: string; status: string }
      if (d?.chatId && d?.messageId) applyReceipt(d.chatId, d.messageId, d.userId || '', 'read')
      break
    }
    case 'translation_blocked':
      store.markTranslationBlocked(message.data)
      break
    case 'grammar_analysis':
      store.setGrammarJob(message.data)
      break
    case 'chat_updated':
      store.loadChats()
      break
    case 'user_typing': {
      const data = message.data || {}
      if (data.chatId && data.userId && data.userId !== store.user?.id) {
        store.setTyping(data.chatId, data.userId, Boolean(data.isTyping))
      }
      break
    }
    case 'user_presence':
    case 'presence_update':
      if (message.data?.userId) {
        store.setPresence(message.data)
      }
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
