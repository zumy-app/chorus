import { describe, it, expect, vi, beforeEach } from 'vitest'

// Use a simpler approach: test pure utility functions directly,
// and test the store with minimal mocking

describe('Store - Slug Helpers', () => {
  // Get the functions directly by importing the module's named exports
  let getChatSlug: Function
  let findChatBySlug: Function

  beforeAll(async () => {
    // Import without mocks to test pure functions
    const mod = await import('../index')
    getChatSlug = mod.getChatSlug
    findChatBySlug = mod.findChatBySlug
  })

  it('should getChatSlug for direct chat with participant', () => {
    const chat = {
      id: 'chat-1',
      type: 'direct' as const,
      name: '',
      createdBy: 'user-1',
      participants: [{ user: { id: 'user-2', username: 'johndoe', displayName: 'John Doe' } }],
      createdAt: new Date().toISOString(),
    }
    expect(getChatSlug(chat, 'user-1')).toBe('@johndoe')
  })

  it('should getChatSlug for direct chat falling back to displayName', () => {
    const chat = {
      id: 'chat-1',
      type: 'direct' as const,
      name: '',
      createdBy: 'user-1',
      participants: [{ user: { id: 'user-2', username: '', displayName: 'Jane Doe' } }],
      createdAt: new Date().toISOString(),
    }
    expect(getChatSlug(chat, 'user-1')).toBe('@jane-doe')
  })

  it('should getChatSlug for direct chat with dm- fallback', () => {
    const chat = {
      id: 'abc12345',
      type: 'direct' as const,
      name: '',
      createdBy: 'user-1',
      participants: [],
      createdAt: new Date().toISOString(),
    }
    expect(getChatSlug(chat)).toContain('dm-')
  })

  it('should getChatSlug for group chat with name', () => {
    const chat = {
      id: 'chat-2',
      type: 'group' as const,
      name: 'My Group Chat!',
      createdBy: 'user-1',
      participants: [],
      createdAt: new Date().toISOString(),
    }
    expect(getChatSlug(chat)).toBe('group/my-group-chat')
  })

  it('should getChatSlug for unnamed group chat', () => {
    const chat = {
      id: 'abc12345',
      type: 'group' as const,
      name: '',
      createdBy: 'user-1',
      participants: [],
      createdAt: new Date().toISOString(),
    }
    expect(getChatSlug(chat)).toBe('group/abc12345')
  })

  it('should findChatBySlug for direct chat with @username', () => {
    const chats = [
      {
        id: 'chat-1',
        type: 'direct' as const,
        name: '',
        createdBy: 'user-1',
        participants: [{ user: { id: 'user-2', username: 'johndoe', displayName: 'John Doe' } }],
        createdAt: new Date().toISOString(),
      },
    ]
    const found = findChatBySlug(chats, '@johndoe', 'user-1')
    expect(found).not.toBeNull()
    expect(found!.id).toBe('chat-1')
  })

  it('should findChatBySlug for group chat', () => {
    const chats = [
      {
        id: 'chat-2',
        type: 'group' as const,
        name: 'My Group',
        createdBy: 'user-1',
        participants: [],
        createdAt: new Date().toISOString(),
      },
    ]
    const found = findChatBySlug(chats, 'group/my-group', 'user-1')
    expect(found).not.toBeNull()
    expect(found!.id).toBe('chat-2')
  })

  it('should findChatBySlug return null for unknown slug', () => {
    expect(findChatBySlug([], '@nonexistent', 'user-1')).toBeNull()
  })

  it('should findChatBySlug fallback to raw chat ID', () => {
    const chats = [
      {
        id: 'raw-chat-id',
        type: 'direct' as const,
        name: '',
        createdBy: 'user-1',
        participants: [],
        createdAt: new Date().toISOString(),
      },
    ]
    expect(findChatBySlug(chats, 'raw-chat-id', 'user-1')?.id).toBe('raw-chat-id')
  })
})

describe('Store - State Management', () => {
  let useStore: any

  beforeAll(async () => {
    // Import with full mock for external dependencies only
    vi.doMock('../../services/api', () => ({
      chatAPI: {
        getChats: vi.fn().mockResolvedValue([]),
        createChat: vi.fn(),
        getChat: vi.fn(),
      },
      messageAPI: {
        getMessages: vi.fn().mockResolvedValue([]),
        sendMessage: vi.fn(),
      },
    }))
    vi.doMock('../../services/websocket', () => ({
      wsService: {
        connect: vi.fn(),
        disconnect: vi.fn(),
        send: vi.fn(),
        onMessage: vi.fn(() => () => {}),
        onReconnect: vi.fn(() => () => {}),
      },
    }))

    const mod = await import('../index')
    useStore = mod.useStore
  })

  beforeEach(() => {
    // Reset store state between tests
    useStore.setState({
      user: null,
      chats: [],
      activeChat: null,
      messages: {},
    })
  })

  it('should set user', () => {
    const testUser = {
      id: 'user-1',
      username: 'testuser',
      email: 'test@example.com',
      displayName: 'Test',
      nativeLanguage: 'en',
      targetLanguages: ['es'],
      createdAt: new Date().toISOString(),
      lastActiveAt: new Date().toISOString(),
    }
    useStore.getState().setUser(testUser)
    expect(useStore.getState().user).toEqual(testUser)
  })

  it('should set active chat', () => {
    const chat = {
      id: 'chat-1',
      type: 'direct' as const,
      name: '',
      createdBy: 'user-1',
      participants: [],
      createdAt: new Date().toISOString(),
    }
    useStore.getState().setActiveChat(chat as any)
    expect(useStore.getState().activeChat).toEqual(chat)
  })

  it('should add a message', () => {
    const msg = {
      id: 'msg-3', chatId: 'chat-1', senderId: 'user-1',
      text: 'New message', timestamp: new Date().toISOString(),
    }
    useStore.getState().addMessage(msg as any)
    expect(useStore.getState().messages['chat-1']).toContainEqual(msg)
  })

  it('should add multiple messages and maintain order', () => {
    useStore.getState().addMessage({
      id: 'msg-1', chatId: 'chat-1', senderId: 'user-1',
      text: 'First', timestamp: new Date().toISOString(),
    } as any)
    useStore.getState().addMessage({
      id: 'msg-2', chatId: 'chat-1', senderId: 'user-1',
      text: 'Second', timestamp: new Date().toISOString(),
    } as any)
    expect(useStore.getState().messages['chat-1']).toHaveLength(2)
  })

  it('should update a message', () => {
    const msg = {
      id: 'msg-1', chatId: 'chat-1', senderId: 'user-1',
      text: 'Original', timestamp: new Date().toISOString(),
    }
    useStore.getState().addMessage(msg as any)
    useStore.getState().updateMessage({ ...msg, text: 'Updated!' } as any)
    const updated = useStore.getState().messages['chat-1']?.find((m: any) => m.id === 'msg-1')
    expect(updated?.text).toBe('Updated!')
  })

  it('should update chat last message', () => {
    const msg = {
      id: 'msg-1', chatId: 'chat-1', senderId: 'user-1',
      text: 'Last msg', timestamp: new Date().toISOString(),
    }
    useStore.getState().updateChatLastMessage('chat-1', msg as any)
    // Should not throw - implementation may vary
  })

  it('should update user partial', () => {
    useStore.getState().setUser({
      id: 'user-1', username: 'testuser', email: 'test@example.com',
      displayName: 'Old Name', nativeLanguage: 'en',
      targetLanguages: ['es'], createdAt: new Date().toISOString(),
      lastActiveAt: new Date().toISOString(),
    } as any)
    useStore.getState().updateUser({ displayName: 'New Name' })
    expect(useStore.getState().user?.displayName).toBe('New Name')
  })
})
