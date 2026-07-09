import { describe, it, expect, vi, beforeEach } from 'vitest'

// Set up localStorage mock
const localStorageMock = (() => {
  let store: Record<string, string> = {}
  return {
    getItem: (key: string) => store[key] || null,
    setItem: (key: string, value: string) => { store[key] = value },
    removeItem: (key: string) => { delete store[key] },
    clear: () => { store = {} },
  }
})()

Object.defineProperty(window, 'localStorage', { value: localStorageMock })

// Helper to create a mock axios instance
function createMockAxios(methods: Record<string, any> = {}) {
  return {
    get: vi.fn().mockResolvedValue({ data: {} }),
    post: vi.fn().mockResolvedValue({ data: {} }),
    put: vi.fn().mockResolvedValue({ data: {} }),
    delete: vi.fn().mockResolvedValue({ data: {} }),
    interceptors: {
      request: { use: vi.fn() },
      response: { use: vi.fn() },
    },
    ...methods,
  }
}

// We'll test by importing the module with a mocked axios.create
describe('API Service', () => {
  beforeEach(() => {
    localStorageMock.clear()
    vi.clearAllMocks()
    // Reset modules so each test gets a fresh import with fresh mocks
    vi.resetModules()
  })

  describe('authAPI', () => {
    it('should register a user via POST', async () => {
      const mockData = {
        user: { id: '1', email: 'test@example.com', displayName: 'Test' },
        tokens: { accessToken: 'access-123', refreshToken: 'refresh-123', expiresIn: 86400 },
      }
      const mockAxios = createMockAxios({
        post: vi.fn().mockResolvedValue({ data: mockData }),
      })

      vi.doMock('axios', () => ({
        default: {
          create: vi.fn(() => mockAxios),
        },
        create: vi.fn(() => mockAxios),
      }))

      const { authAPI } = await import('../../services/api')
      const result = await authAPI.register({
        username: 'testuser',
        email: 'test@example.com',
        password: 'Password123!',
        displayName: 'Test',
        nativeLanguage: 'en',
        targetLanguages: ['es'],
      })

      expect(result.user).toBeDefined()
      expect(result.tokens.accessToken).toBe('access-123')
      expect(mockAxios.post).toHaveBeenCalled()
    })

    it('should login a user via POST', async () => {
      const mockData = {
        user: { id: '1', email: 'test@example.com' },
        tokens: { accessToken: 'access-123', refreshToken: 'refresh-123', expiresIn: 86400 },
      }
      const mockAxios = createMockAxios({
        post: vi.fn().mockResolvedValue({ data: mockData }),
      })

      vi.doMock('axios', () => ({
        default: { create: vi.fn(() => mockAxios) },
        create: vi.fn(() => mockAxios),
      }))

      const { authAPI } = await import('../../services/api')
      const result = await authAPI.login({ username: 'testuser', password: 'Password123!' })
      expect(result.tokens.accessToken).toBe('access-123')
    })

    it('should fetch current user via GET', async () => {
      const mockAxios = createMockAxios({
        get: vi.fn().mockResolvedValue({ data: { id: '1', email: 'test@example.com' } }),
      })

      vi.doMock('axios', () => ({
        default: { create: vi.fn(() => mockAxios) },
        create: vi.fn(() => mockAxios),
      }))

      const { authAPI } = await import('../../services/api')
      const result = await authAPI.getMe()
      expect(result.id).toBe('1')
    })

    it('should search users via GET', async () => {
      const mockAxios = createMockAxios({
        get: vi.fn().mockResolvedValue({ data: { users: [{ id: '2', username: 'founduser' }] } }),
      })

      vi.doMock('axios', () => ({
        default: { create: vi.fn(() => mockAxios) },
        create: vi.fn(() => mockAxios),
      }))

      const { authAPI } = await import('../../services/api')
      const users = await authAPI.searchUsers('found')
      expect(users).toHaveLength(1)
      expect(users[0].username).toBe('founduser')
    })
  })

  describe('chatAPI', () => {
    it('should get chats via GET', async () => {
      const mockAxios = createMockAxios({
        get: vi.fn().mockResolvedValue({ data: { chats: [{ id: 'chat-1', type: 'direct' }] } }),
      })

      vi.doMock('axios', () => ({
        default: { create: vi.fn(() => mockAxios) },
        create: vi.fn(() => mockAxios),
      }))

      const { chatAPI } = await import('../../services/api')
      const chats = await chatAPI.getChats()
      expect(chats).toHaveLength(1)
      expect(chats[0].id).toBe('chat-1')
    })

    it('should create a chat via POST', async () => {
      const mockAxios = createMockAxios({
        post: vi.fn().mockResolvedValue({ data: { id: 'new-chat', type: 'direct' } }),
      })

      vi.doMock('axios', () => ({
        default: { create: vi.fn(() => mockAxios) },
        create: vi.fn(() => mockAxios),
      }))

      const { chatAPI } = await import('../../services/api')
      const chat = await chatAPI.createChat({ type: 'direct', participants: ['user-2'] })
      expect(chat.id).toBe('new-chat')
    })

    it('should add and remove participants', async () => {
      const mockPost = vi.fn().mockResolvedValue({})
      const mockDelete = vi.fn().mockResolvedValue({})
      const mockAxios = createMockAxios({ post: mockPost, delete: mockDelete })

      vi.doMock('axios', () => ({
        default: { create: vi.fn(() => mockAxios) },
        create: vi.fn(() => mockAxios),
      }))

      const { chatAPI } = await import('../../services/api')
      await chatAPI.addParticipant('chat-1', 'user-3')
      expect(mockPost).toHaveBeenCalledWith('/chats/chat-1/participants', { userId: 'user-3' })

      await chatAPI.removeParticipant('chat-1', 'user-3')
      expect(mockDelete).toHaveBeenCalledWith('/chats/chat-1/participants/user-3')
    })

    it('should let user leave a chat', async () => {
      const mockDelete = vi.fn().mockResolvedValue({})
      const mockAxios = createMockAxios({ delete: mockDelete })

      vi.doMock('axios', () => ({
        default: { create: vi.fn(() => mockAxios) },
        create: vi.fn(() => mockAxios),
      }))

      const { chatAPI } = await import('../../services/api')
      await chatAPI.leaveChat('chat-1')
      expect(mockDelete).toHaveBeenCalledWith('/chats/chat-1/leave')
    })
  })

  describe('messageAPI', () => {
    it('should get messages via GET', async () => {
      const mockAxios = createMockAxios({
        get: vi.fn().mockResolvedValue({ data: { messages: [{ id: 'msg-1', text: 'Hello' }] } }),
      })

      vi.doMock('axios', () => ({
        default: { create: vi.fn(() => mockAxios) },
        create: vi.fn(() => mockAxios),
      }))

      const { messageAPI } = await import('../../services/api')
      const messages = await messageAPI.getMessages('chat-1')
      expect(messages).toHaveLength(1)
      expect(messages[0].text).toBe('Hello')
    })

    it('should send a message via POST', async () => {
      const mockAxios = createMockAxios({
        post: vi.fn().mockResolvedValue({ data: { id: 'msg-new', text: 'Hi!', chatId: 'chat-1' } }),
      })

      vi.doMock('axios', () => ({
        default: { create: vi.fn(() => mockAxios) },
        create: vi.fn(() => mockAxios),
      }))

      const { messageAPI } = await import('../../services/api')
      const msg = await messageAPI.sendMessage('chat-1', { text: 'Hi!' })
      expect(msg.id).toBe('msg-new')
    })

    it('should mark messages as read', async () => {
      const mockPut = vi.fn().mockResolvedValue({})
      const mockAxios = createMockAxios({ put: mockPut })

      vi.doMock('axios', () => ({
        default: { create: vi.fn(() => mockAxios) },
        create: vi.fn(() => mockAxios),
      }))

      const { messageAPI } = await import('../../services/api')
      await messageAPI.markAsRead('chat-1', 'msg-5')
      expect(mockPut).toHaveBeenCalledWith('/chats/chat-1/read', { messageId: 'msg-5' })
    })
  })

  describe('vocabularyAPI', () => {
    it('should get vocabulary entries via GET', async () => {
      const mockAxios = createMockAxios({
        get: vi.fn().mockResolvedValue({ data: { data: { entries: [{ id: 'v-1', term: 'hola' }] } } }),
      })

      vi.doMock('axios', () => ({
        default: { create: vi.fn(() => mockAxios) },
        create: vi.fn(() => mockAxios),
      }))

      const { vocabularyAPI } = await import('../../services/api')
      const entries = await vocabularyAPI.getAll('es')
      expect(entries).toHaveLength(1)
      expect(entries[0].term).toBe('hola')
    })

    it('should save a vocabulary entry via POST', async () => {
      const mockAxios = createMockAxios({
        post: vi.fn().mockResolvedValue({ data: { data: { id: 'v-new', term: 'gracias' } } }),
      })

      vi.doMock('axios', () => ({
        default: { create: vi.fn(() => mockAxios) },
        create: vi.fn(() => mockAxios),
      }))

      const { vocabularyAPI } = await import('../../services/api')
      const entry = await vocabularyAPI.save('gracias', 'es', 'msg-1')
      expect(entry.id).toBe('v-new')
    })
  })
})
