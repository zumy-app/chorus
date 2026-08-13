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
        inviteToken: 'valid-invite',
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

    it('should fetch entitlements via GET', async () => {
      const mockData = {
        plan: 'free',
        planGraceUntil: null,
        effectivePlan: 'free',
        selfHost: false,
        limits: {
          dailyLLMTranslations: 50,
          dailyLLMGrammarAnalyses: 20,
          dailyLLMCorrections: 20,
          dailyVoiceMessages: 10,
          vocabularyItems: 500,
        },
      }
      const mockAxios = createMockAxios({
        get: vi.fn().mockResolvedValue({ data: mockData }),
      })

      vi.doMock('axios', () => ({
        default: { create: vi.fn(() => mockAxios) },
        create: vi.fn(() => mockAxios),
      }))

      const { authAPI } = await import('../../services/api')
      const result = await authAPI.getEntitlements()
      expect(result.effectivePlan).toBe('free')
      expect(result.limits.dailyLLMTranslations).toBe(50)
      expect(mockAxios.get).toHaveBeenCalledWith('/users/me/entitlements')
    })
  })

  describe('waitlistAPI', () => {
    it('submits waitlist interest via POST', async () => {
      const mockAxios = createMockAxios({
        post: vi.fn().mockResolvedValue({ data: { entry: { queuePosition: 7 } } }),
      })
      vi.doMock('axios', () => ({ default: { create: vi.fn(() => mockAxios) }, create: vi.fn(() => mockAxios) }))
      const { waitlistAPI } = await import('../../services/api')

      const result = await waitlistAPI.join({
        email: 'learner@example.com',
        spokenLanguages: ['en', 'fr'],
        targetLanguages: ['es'],
        reasons: ['For travel'],
      })

      expect(result.entry.queuePosition).toBe(7)
      expect(mockAxios.post).toHaveBeenCalledWith('/waitlist', expect.objectContaining({ email: 'learner@example.com' }))
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

  describe('moderationAPI', () => {
    it('blocks a user via POST', async () => {
      const mockPost = vi.fn().mockResolvedValue({ data: { message: 'User blocked' } })
      const mockAxios = createMockAxios({ post: mockPost })

      vi.doMock('axios', () => ({
        default: { create: vi.fn(() => mockAxios) },
        create: vi.fn(() => mockAxios),
      }))

      const { moderationAPI } = await import('../../services/api')
      const result = await moderationAPI.block('user-2', 'spam')
      expect(result.message).toBe('User blocked')
      expect(mockPost).toHaveBeenCalledWith('/blocks', { blockedUserId: 'user-2', reason: 'spam' })
    })

    it('unblocks a user via DELETE', async () => {
      const mockDelete = vi.fn().mockResolvedValue({})
      const mockAxios = createMockAxios({ delete: mockDelete })

      vi.doMock('axios', () => ({
        default: { create: vi.fn(() => mockAxios) },
        create: vi.fn(() => mockAxios),
      }))

      const { moderationAPI } = await import('../../services/api')
      await moderationAPI.unblock('user-2')
      expect(mockDelete).toHaveBeenCalledWith('/blocks/user-2')
    })

    it('lists blocked users via GET', async () => {
      const mockAxios = createMockAxios({
        get: vi.fn().mockResolvedValue({ data: { blocks: [{ id: 'b-1', blockedId: 'user-2' }], total: 1 } }),
      })

      vi.doMock('axios', () => ({
        default: { create: vi.fn(() => mockAxios) },
        create: vi.fn(() => mockAxios),
      }))

      const { moderationAPI } = await import('../../services/api')
      const blocks = await moderationAPI.getBlocked()
      expect(blocks).toHaveLength(1)
      expect(blocks[0].blockedId).toBe('user-2')
      expect(mockAxios.get).toHaveBeenCalledWith('/blocks')
    })

    it('reports a user or message via POST', async () => {
      const mockPost = vi.fn().mockResolvedValue({ data: { id: 'r-1', type: 'message', status: 'open' } })
      const mockAxios = createMockAxios({ post: mockPost })

      vi.doMock('axios', () => ({
        default: { create: vi.fn(() => mockAxios) },
        create: vi.fn(() => mockAxios),
      }))

      const { moderationAPI } = await import('../../services/api')
      const report = await moderationAPI.report({ type: 'message', messageId: 'msg-1', chatId: 'chat-1', reason: 'spam' })
      expect(report.id).toBe('r-1')
      expect(mockPost).toHaveBeenCalledWith('/reports', {
        type: 'message',
        messageId: 'msg-1',
        chatId: 'chat-1',
        reason: 'spam',
      })
    })
  })

  describe('adminAPI reports', () => {
    it('lists reports with status and query params', async () => {
      const mockAxios = createMockAxios({
        get: vi.fn().mockResolvedValue({ data: { reports: [{ id: 'r-1', status: 'open' }], total: 1 } }),
      })

      vi.doMock('axios', () => ({
        default: { create: vi.fn(() => mockAxios) },
        create: vi.fn(() => mockAxios),
      }))

      const { adminAPI } = await import('../../services/api')
      const data = await adminAPI.listReports('open', 'alice')
      expect(data.total).toBe(1)
      expect(mockAxios.get).toHaveBeenCalledWith(expect.stringContaining('/admin/reports?'))
      expect(mockAxios.get).toHaveBeenCalledWith(expect.stringContaining('status=open'))
      expect(mockAxios.get).toHaveBeenCalledWith(expect.stringContaining('q=alice'))
    })

    it('fetches report stats', async () => {
      const mockAxios = createMockAxios({
        get: vi.fn().mockResolvedValue({ data: { open: 3, resolved: 1, dismissed: 2 } }),
      })

      vi.doMock('axios', () => ({
        default: { create: vi.fn(() => mockAxios) },
        create: vi.fn(() => mockAxios),
      }))

      const { adminAPI } = await import('../../services/api')
      const stats = await adminAPI.reportStats()
      expect(stats.open).toBe(3)
      expect(mockAxios.get).toHaveBeenCalledWith('/admin/reports/stats')
    })

    it('resolves a report via POST', async () => {
      const mockPost = vi.fn().mockResolvedValue({ data: { message: 'Report resolved' } })
      const mockAxios = createMockAxios({ post: mockPost })

      vi.doMock('axios', () => ({
        default: { create: vi.fn(() => mockAxios) },
        create: vi.fn(() => mockAxios),
      }))

      const { adminAPI } = await import('../../services/api')
      await adminAPI.resolveReport('r-1')
      expect(mockPost).toHaveBeenCalledWith('/admin/reports/r-1/resolve')
    })

    it('dismisses a report with an optional note', async () => {
      const mockPost = vi.fn().mockResolvedValue({ data: { message: 'Report dismissed' } })
      const mockAxios = createMockAxios({ post: mockPost })

      vi.doMock('axios', () => ({
        default: { create: vi.fn(() => mockAxios) },
        create: vi.fn(() => mockAxios),
      }))

      const { adminAPI } = await import('../../services/api')
      await adminAPI.dismissReport('r-1', 'not actionable')
      expect(mockPost).toHaveBeenCalledWith('/admin/reports/r-1/dismiss', { note: 'not actionable' })
    })
  })

  describe('grammarAPI', () => {
    it('submits an async AI analysis job via POST /grammar/analyze-ai', async () => {
      const mockAxios = createMockAxios({
        post: vi.fn().mockResolvedValue({
          data: { jobId: 'job-1', status: 'queued' },
        }),
      })

      vi.doMock('axios', () => ({
        default: { create: vi.fn(() => mockAxios) },
        create: vi.fn(() => mockAxios),
      }))

      const { grammarAPI } = await import('../../services/api')
      const result = await grammarAPI.analyzeAI({
        text: 'Hola, como estas',
        language: 'es',
        nativeLanguage: 'en',
        messageId: 'm-1',
        chatId: 'c-1',
      })

      expect(result.jobId).toBe('job-1')
      expect(result.status).toBe('queued')
      expect(mockAxios.post).toHaveBeenCalledWith('/grammar/analyze-ai', {
        text: 'Hola, como estas',
        language: 'es',
        nativeLanguage: 'en',
        messageId: 'm-1',
        chatId: 'c-1',
      })
    })

    it('fetches a job result via GET /grammar/analyze/:jobId', async () => {
      const mockJob = {
        jobId: 'job-1',
        messageId: 'm-1',
        status: 'done',
        analysis: { summary: 'Well formed' },
        providerUsed: 'ollama',
      }
      const mockAxios = createMockAxios({
        get: vi.fn().mockResolvedValue({ data: mockJob }),
      })

      vi.doMock('axios', () => ({
        default: { create: vi.fn(() => mockAxios) },
        create: vi.fn(() => mockAxios),
      }))

      const { grammarAPI } = await import('../../services/api')
      const result = await grammarAPI.getAnalysis('job-1')
      expect(result.status).toBe('done')
      expect(result.jobId).toBe('job-1')
      expect(result.analysis.summary).toBe('Well formed')
      expect(mockAxios.get).toHaveBeenCalledWith('/grammar/analyze/job-1')
    })
  })
})
