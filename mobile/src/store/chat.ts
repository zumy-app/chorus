import { create } from 'zustand'

import { messageApi } from '../services/api'
import { websocketManager } from '../services/websocket'
import type { Chat, Message, MessageStatusEvent, TypingEvent, WebSocketEvent } from '../types'

interface ChatState {
  currentUserId: string | null
  activeChatId: string | null
  chatsById: Record<string, Chat>
  chatIds: string[]
  messagesById: Record<string, Message>
  messageIdsByChatId: Record<string, string[]>
  hasMoreByChatId: Record<string, boolean>
  unreadCountByChatId: Record<string, number>
  typingUserIdsByChatId: Record<string, string[]>
  setCurrentUserId(userId: string | null): void
  setActiveChatId(chatId: string | null): void
  applySnapshot(chats: Chat[]): void
  applyMessagePage(chatId: string, messages: Message[], hasMore: boolean): void
  sendMessage(chatId: string, text: string, replyToId?: string): Promise<void>
  handleWebSocketEvent(event: WebSocketEvent): void
  reset(): void
}

const emptyState = {
  currentUserId: null,
  activeChatId: null,
  chatsById: {},
  chatIds: [],
  messagesById: {},
  messageIdsByChatId: {},
  hasMoreByChatId: {},
  unreadCountByChatId: {},
  typingUserIdsByChatId: {},
}

function orderedMessageIds(messages: Record<string, Message>, ids: string[]): string[] {
  return [...new Set(ids)].sort((left, right) => messages[left].timestamp.localeCompare(messages[right].timestamp))
}

function isMessage(value: unknown): value is Message {
  return Boolean(value) && typeof value === 'object' && typeof (value as Message).id === 'string' && typeof (value as Message).chatId === 'string'
}

function mergeMessage(state: Pick<ChatState, 'messagesById' | 'messageIdsByChatId'>, message: Message): Pick<ChatState, 'messagesById' | 'messageIdsByChatId'> {
  const messagesById = { ...state.messagesById, [message.id]: { ...state.messagesById[message.id], ...message } }
  const existingIds = state.messageIdsByChatId[message.chatId] ?? []
  const messageIdsByChatId = {
    ...state.messageIdsByChatId,
    [message.chatId]: orderedMessageIds(messagesById, [...existingIds, message.id]),
  }
  return { messagesById, messageIdsByChatId }
}

export const useChatStore = create<ChatState>((set, get) => ({
  ...emptyState,

  setCurrentUserId: (currentUserId) => set({ currentUserId }),

  setActiveChatId: (activeChatId) =>
    set((state) => ({
      activeChatId,
      unreadCountByChatId: activeChatId ? { ...state.unreadCountByChatId, [activeChatId]: 0 } : state.unreadCountByChatId,
    })),

  applySnapshot: (chats) =>
    set((state) => {
      const chatsById = { ...state.chatsById }
      const unreadCountByChatId = { ...state.unreadCountByChatId }
      for (const chat of chats) {
        chatsById[chat.id] = { ...chatsById[chat.id], ...chat }
        unreadCountByChatId[chat.id] = chat.unreadCount ?? unreadCountByChatId[chat.id] ?? 0
      }
      return { chatsById, chatIds: chats.map((chat) => chat.id), unreadCountByChatId }
    }),

  applyMessagePage: (chatId, messages, hasMore) =>
    set((state) => {
      let next = { messagesById: state.messagesById, messageIdsByChatId: state.messageIdsByChatId }
      for (const message of messages) next = mergeMessage(next, message)
      return { ...next, hasMoreByChatId: { ...state.hasMoreByChatId, [chatId]: hasMore } }
    }),

  sendMessage: async (chatId, text, replyToId) => {
    const trimmed = text.trim()
    if (!trimmed) return
    const senderId = get().currentUserId
    if (!senderId) throw new Error('Cannot send a message without an authenticated user.')

    const optimisticId = `local-${globalThis.crypto?.randomUUID?.() ?? Date.now()}`
    const optimistic: Message = {
      id: optimisticId,
      chatId,
      senderId,
      text: trimmed,
      originalLanguage: '',
      deliveryStatus: 'pending',
      replyToId,
      timestamp: new Date().toISOString(),
    }
    set((state) => mergeMessage(state, optimistic))

    try {
      const message = await messageApi.send(chatId, { text: trimmed, replyToId })
      set((state) => {
        const { [optimisticId]: _removed, ...messagesById } = state.messagesById
        const ids = (state.messageIdsByChatId[chatId] ?? []).filter((id) => id !== optimisticId)
        return mergeMessage({ messagesById, messageIdsByChatId: { ...state.messageIdsByChatId, [chatId]: ids } }, message)
      })
    } catch (error) {
      set((state) => ({
        messagesById: {
          ...state.messagesById,
          [optimisticId]: { ...state.messagesById[optimisticId], deliveryStatus: 'failed' },
        },
      }))
      throw error
    }
  },

  handleWebSocketEvent: (event) => {
    if (event.type === 'new_message' || event.type === 'message_updated') {
      if (!isMessage(event.data)) return
      const message: Message = event.data
      set((state) => {
        const isKnownMessage = Boolean(state.messagesById[message.id])
        const matchingOptimisticId = Object.values(state.messagesById).find(
          (candidate) =>
            candidate.id.startsWith('local-') &&
            candidate.chatId === message.chatId &&
            candidate.senderId === message.senderId &&
            candidate.text === message.text,
        )?.id
        const messagesById = { ...state.messagesById }
        const ids = [...(state.messageIdsByChatId[message.chatId] ?? [])]
        if (matchingOptimisticId) {
          delete messagesById[matchingOptimisticId]
          ids.splice(ids.indexOf(matchingOptimisticId), 1)
        }
        const next = mergeMessage({ messagesById, messageIdsByChatId: { ...state.messageIdsByChatId, [message.chatId]: ids } }, message)
        const isIncoming = !isKnownMessage && message.senderId !== state.currentUserId && state.activeChatId !== message.chatId
        return {
          ...next,
          unreadCountByChatId: isIncoming
            ? { ...state.unreadCountByChatId, [message.chatId]: (state.unreadCountByChatId[message.chatId] ?? 0) + 1 }
            : state.unreadCountByChatId,
        }
      })
      return
    }

    if (event.type === 'user_typing') {
      const typing = event.data as TypingEvent
      if (!typing?.chatId || !typing.userId || typing.userId === get().currentUserId) return
      set((state) => {
        const current = state.typingUserIdsByChatId[typing.chatId] ?? []
        const users = typing.isTyping ? [...new Set([...current, typing.userId])] : current.filter((id) => id !== typing.userId)
        return { typingUserIdsByChatId: { ...state.typingUserIdsByChatId, [typing.chatId]: users } }
      })
      return
    }

    if (event.type === 'message_delivered' || event.type === 'message_read') {
      const status = event.data as MessageStatusEvent
      if (!status?.messageId) return
      set((state) => {
        const message = state.messagesById[status.messageId]
        if (!message) return state
        return { messagesById: { ...state.messagesById, [message.id]: { ...message, deliveryStatus: 'delivered' } } }
      })
    }
  },

  reset: () => set(emptyState),
}))

websocketManager.subscribe?.((event) => useChatStore.getState().handleWebSocketEvent(event))
