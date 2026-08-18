import { beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('../services/api', () => ({
  messageApi: { send: vi.fn() },
}))
vi.mock('../services/session', () => ({
  getAccessToken: vi.fn(),
}))

import { messageApi } from '../services/api'
import { useChatStore } from './chat'

const chat = { id: 'chat-1', type: 'direct' as const, name: 'Practice' }
const firstMessage = {
  id: 'message-1',
  chatId: 'chat-1',
  senderId: 'other-user',
  text: 'Hello',
  originalLanguage: 'en',
  deliveryStatus: 'sent' as const,
  timestamp: '2026-08-18T10:00:00.000Z',
}

describe('chat store', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useChatStore.getState().reset()
    useChatStore.getState().setCurrentUserId('me')
  })

  it('normalizes snapshots and preserves message order while merging pages', () => {
    useChatStore.getState().applySnapshot([chat])
    useChatStore.getState().applyMessagePage('chat-1', [firstMessage], true)
    useChatStore.getState().applyMessagePage(
      'chat-1',
      [{ ...firstMessage, id: 'message-0', timestamp: '2026-08-18T09:00:00.000Z' }],
      false,
    )

    const state = useChatStore.getState()
    expect(state.chatIds).toEqual(['chat-1'])
    expect(state.messageIdsByChatId['chat-1']).toEqual(['message-0', 'message-1'])
    expect(state.hasMoreByChatId['chat-1']).toBe(false)
  })

  it('reconciles an optimistic message when its websocket echo arrives', async () => {
    vi.mocked(messageApi.send).mockResolvedValue({ ...firstMessage, id: 'server-message', senderId: 'me' })

    await useChatStore.getState().sendMessage('chat-1', 'Hi')
    const optimisticId = useChatStore.getState().messageIdsByChatId['chat-1'][0]
    expect(useChatStore.getState().messagesById[optimisticId].deliveryStatus).toBe('sent')

    useChatStore.getState().handleWebSocketEvent({
      type: 'new_message',
      data: { ...firstMessage, id: 'server-message', senderId: 'me', text: 'Hi' },
    })

    expect(useChatStore.getState().messageIdsByChatId['chat-1']).toEqual(['server-message'])
    expect(useChatStore.getState().messagesById['server-message'].text).toBe('Hi')
  })

  it('deduplicates messages, updates typing, and increments unread messages from others', () => {
    const event = { type: 'new_message' as const, data: firstMessage }
    useChatStore.getState().handleWebSocketEvent(event)
    useChatStore.getState().handleWebSocketEvent(event)
    useChatStore.getState().handleWebSocketEvent({
      type: 'user_typing',
      data: { chatId: 'chat-1', userId: 'other-user', isTyping: true },
    })

    const state = useChatStore.getState()
    expect(state.messageIdsByChatId['chat-1']).toEqual(['message-1'])
    expect(state.unreadCountByChatId['chat-1']).toBe(1)
    expect(state.typingUserIdsByChatId['chat-1']).toEqual(['other-user'])
  })
})
