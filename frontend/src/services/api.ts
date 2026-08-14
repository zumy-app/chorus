import axios from 'axios'
import { Capacitor } from '@capacitor/core'
import type {
  User,
  Chat,
  Message,
  AuthTokens,
  LoginRequest,
  RegisterRequest,
  CreateChatRequest,
  SendMessageRequest,
  WaitlistRequest,
  WaitlistEntry,
  EmailOutboxEntry,
  AdminStats,
  AdminStatus,
  TranslationJob,
  ProviderHealth,
  Entitlements,
  SubscriptionInfo,
  CheckoutResponse,
  PremiumUserRow,
  PremiumAnalytics,
  PlanChange,
  GrantPlanRequest,
  Block,
  Report,
  ReportRequest,
  ReportStats,
  GrammarJob,
} from '../types'

// Get API URL based on environment
const getAPIUrl = () => {
  const platform = Capacitor.getPlatform()
  const isNative = Capacitor.isNativePlatform()

  if (isNative && platform === 'android') {
    return 'http://10.0.2.2:8080/api/v1'
  }

  if (isNative && platform === 'ios') {
    return 'http://localhost:8080/api/v1'
  }

  return '/api/v1'
}

const API_URL = getAPIUrl()

const api = axios.create({
  baseURL: API_URL,
})

// Add auth token to requests
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('accessToken')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// Handle token refresh on 401
api.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config
    if (error.response?.status === 401 && !originalRequest._retry) {
      originalRequest._retry = true
      try {
        const refreshToken = localStorage.getItem('refreshToken')
        if (refreshToken) {
          const { data } = await axios.post(`${API_URL}/auth/refresh`, {
            refreshToken,
          })
          localStorage.setItem('accessToken', data.accessToken)
          originalRequest.headers.Authorization = `Bearer ${data.accessToken}`
          return api(originalRequest)
        }
      } catch (refreshError) {
        localStorage.removeItem('accessToken')
        localStorage.removeItem('refreshToken')
        window.location.href = '/login'
      }
    }
    return Promise.reject(error)
  }
)

export const authAPI = {
  register: async (data: RegisterRequest) => {
    const response = await api.post<{ user: User; tokens: AuthTokens }>('/auth/register', data)
    return response.data
  },

  inviteEmail: async (token: string) => {
    const response = await api.get<{ email: string }>('/auth/invite', { params: { token } })
    return response.data.email
  },

  login: async (data: LoginRequest) => {
    const response = await api.post<{ user: User; tokens: AuthTokens }>('/auth/login', data)
    return response.data
  },

  forgotPassword: async (email: string) => {
    const response = await api.post<{ message: string }>('/auth/forgot-password', { email })
    return response.data
  },

  resetPassword: async (token: string, password: string) => {
    const response = await api.post<{ message: string }>('/auth/reset-password', { token, password })
    return response.data
  },

  getMe: async () => {
    const response = await api.get<User>('/users/me')
    return response.data
  },

  getEntitlements: async () => {
    const response = await api.get<Entitlements>('/users/me/entitlements')
    return response.data
  },

  updateMe: async (data: { displayName?: string; nativeLanguage?: string; targetLanguages?: string[] }) => {
    const response = await api.put<User>('/users/me', data)
    return response.data
  },

  searchUsers: async (query: string) => {
    const response = await api.get<{ users: User[] }>(`/users/search?q=${query}`)
    return response.data.users
  },
}

export const waitlistAPI = {
  join: async (data: WaitlistRequest) => {
    const response = await api.post<{
      entry: WaitlistEntry
      message: string
      alreadyJoined?: boolean
      emailSent?: boolean
    }>('/waitlist', data)
    return response.data
  },
}

export const adminAPI = {
  status: async () => {
    const response = await api.get<AdminStatus>('/admin/status')
    return response.data
  },

  isAdmin: async () => {
    const data = await adminAPI.status()
    return data.isAdmin
  },

  isModerator: async () => {
    const data = await adminAPI.status()
    return data.isModerator
  },

  stats: async () => {
    const response = await api.get<AdminStats>('/admin/stats')
    return response.data
  },

  listWaitlist: async (status?: string, q?: string) => {
    const params = new URLSearchParams()
    if (status) params.append('status', status)
    if (q) params.append('q', q)
    const response = await api.get<{ entries: WaitlistEntry[] }>(`/admin/waitlist?${params}`)
    return response.data.entries
  },

  approve: async (id: string) => {
    const response = await api.post<{ message: string }>(`/admin/waitlist/${id}/approve`)
    return response.data
  },

  decline: async (id: string) => {
    const response = await api.post<{ message: string }>(`/admin/waitlist/${id}/decline`)
    return response.data
  },

  resendInvite: async (id: string) => {
    const response = await api.post<{ message: string }>(`/admin/waitlist/${id}/resend-invite`)
    return response.data
  },

  emails: async (status?: string) => {
    const params = new URLSearchParams()
    if (status) params.append('status', status)
    const response = await api.get<{ emails: EmailOutboxEntry[] }>(`/admin/emails?${params}`)
    return response.data.emails
  },

  retryEmail: async (id: string) => {
    const response = await api.post<{ message: string }>(`/admin/emails/${id}/retry`)
    return response.data
  },

  listUsers: async (params: { q?: string; role?: string; status?: string; limit?: number } = {}) => {
    const query = new URLSearchParams()
    if (params.q) query.append('q', params.q)
    if (params.role) query.append('role', params.role)
    if (params.status) query.append('status', params.status)
    if (params.limit) query.append('limit', String(params.limit))
    const response = await api.get<{ users: User[]; total: number }>(`/admin/users?${query}`)
    return response.data
  },

  setUserRole: async (id: string, role: string) => {
    const response = await api.put<{ user: User }>(`/admin/users/${id}/role`, { role })
    return response.data
  },

  suspendUser: async (id: string) => {
    const response = await api.post<{ message: string }>(`/admin/users/${id}/suspend`)
    return response.data
  },

  unsuspendUser: async (id: string) => {
    const response = await api.post<{ message: string }>(`/admin/users/${id}/unsuspend`)
    return response.data
  },

  deleteUser: async (id: string) => {
    const response = await api.delete<{ message: string }>(`/admin/users/${id}`)
    return response.data
  },

  listTranslations: async (status?: string, q?: string) => {
    const query = new URLSearchParams()
    if (status && status !== 'all') query.append('status', status)
    if (q) query.append('q', q)
    const response = await api.get<{ jobs: TranslationJob[]; total: number }>(`/admin/translations?${query}`)
    return response.data
  },

  retryTranslation: async (id: string) => {
    const response = await api.post<{ message: string }>(`/admin/translations/${id}/retry`)
    return response.data
  },

  translationHealth: async () => {
    const response = await api.get<{ providers: ProviderHealth[] }>('/admin/translations/health')
    return response.data.providers
  },

  premiumUsers: async (q?: string, limit = 50, offset = 0) => {
    const params = new URLSearchParams({ limit: String(limit), offset: String(offset) })
    if (q) params.append('q', q)
    const response = await api.get<{ users: PremiumUserRow[]; total: number }>('/admin/premium/users?' + params)
    return response.data
  },

  premiumAnalytics: async () => {
    const response = await api.get<PremiumAnalytics>('/admin/premium/analytics')
    return response.data
  },

  grantPlan: async (id: string, data: GrantPlanRequest) => {
    const response = await api.put<{ message: string }>(`/admin/users/${id}/plan`, data)
    return response.data
  },

  revokePlan: async (id: string, clearGraceInDays = 0, reason?: string) => {
    const response = await api.post<{ message: string }>(`/admin/users/${id}/plan/revoke`, {
      plan: 'free',
      clearGraceInDays,
      reason,
    })
    return response.data
  },

  planHistory: async (id: string) => {
    const response = await api.get<{ history: PlanChange[] }>(`/admin/users/${id}/plan-history`)
    return response.data.history
  },

  listReports: async (status?: string, q?: string, limit = 50, offset = 0) => {
    const params = new URLSearchParams({ limit: String(limit), offset: String(offset) })
    if (status && status !== 'all') params.append('status', status)
    if (q) params.append('q', q)
    const response = await api.get<{ reports: Report[]; total: number }>(`/admin/reports?${params}`)
    return response.data
  },

  reportStats: async () => {
    const response = await api.get<ReportStats>('/admin/reports/stats')
    return response.data
  },

  resolveReport: async (id: string) => {
    const response = await api.post<{ message: string }>(`/admin/reports/${id}/resolve`)
    return response.data
  },

  dismissReport: async (id: string, note?: string) => {
    const response = await api.post<{ message: string }>(`/admin/reports/${id}/dismiss`, { note })
    return response.data
  },
}

export const moderationAPI = {
  block: async (blockedUserId: string, reason?: string) => {
    const response = await api.post<{ message: string }>('/blocks', { blockedUserId, reason })
    return response.data
  },

  unblock: async (userId: string) => {
    await api.delete(`/blocks/${userId}`)
  },

  getBlocked: async () => {
    const response = await api.get<{ blocks: Block[]; total: number }>('/blocks')
    return response.data.blocks
  },

  report: async (data: ReportRequest) => {
    const response = await api.post<Report>('/reports', data)
    return response.data
  },
}

export const chatAPI = {
  getChats: async () => {
    const response = await api.get<{ chats: Chat[] }>('/chats')
    return response.data.chats
  },

  createChat: async (data: CreateChatRequest) => {
    const response = await api.post<Chat>('/chats', data)
    return response.data
  },

  getChat: async (chatId: string) => {
    const response = await api.get<Chat>(`/chats/${chatId}`)
    return response.data
  },

  updateChat: async (chatId: string, data: { name?: string; settings?: any }) => {
    const response = await api.put<Chat>(`/chats/${chatId}`, data)
    return response.data
  },

  addParticipant: async (chatId: string, userId: string) => {
    await api.post(`/chats/${chatId}/participants`, { userId })
  },

  removeParticipant: async (chatId: string, userId: string) => {
    await api.delete(`/chats/${chatId}/participants/${userId}`)
  },

  leaveChat: async (chatId: string) => {
    await api.delete(`/chats/${chatId}/leave`)
  },
}

export const messageAPI = {
  getMessages: async (chatId: string, limit = 50, before?: string) => {
    const params = new URLSearchParams({ limit: limit.toString() })
    if (before) params.append('before', before)
    const response = await api.get<{ messages: Message[] }>(
      `/chats/${chatId}/messages?${params}`
    )
    return response.data.messages
  },

  sendMessage: async (chatId: string, data: SendMessageRequest) => {
    const response = await api.post<Message>(`/chats/${chatId}/messages`, data)
    return response.data
  },

  markAsRead: async (chatId: string, messageId: string) => {
    await api.put(`/chats/${chatId}/read`, { messageId })
  },

  searchMessages: async (query: string, chatId?: string) => {
    const params = new URLSearchParams({ q: query })
    if (chatId) params.append('chatId', chatId)
    const response = await api.get<{ messages: Message[] }>(`/messages/search?${params}`)
    return response.data.messages
  },
}

export const vocabularyAPI = {
  getAll: async (language?: string) => {
    const params = new URLSearchParams()
    if (language) params.append('language', language)
    const response = await api.get(`/vocabulary?${params}`)
    return response.data.data?.entries || []
  },

  getDue: async () => {
    const response = await api.get('/vocabulary/due')
    return response.data.data || []
  },

  save: async (term: string, language: string, messageId: string) => {
    const response = await api.post('/vocabulary', { term, language, messageId })
    return response.data.data
  },

  practice: async (vocabularyId: string, correct: boolean) => {
    await api.post('/vocabulary/practice', { vocabularyId, correct })
  },

  getProgress: async () => {
    const response = await api.get('/vocabulary/progress')
    return response.data.data
  },

  delete: async (id: string) => {
    await api.delete(`/vocabulary/${id}`)
  },
}

export const billingAPI = {
  getMySubscription: async () => {
    const response = await api.get<SubscriptionInfo>('/users/me/subscription')
    return response.data
  },

  checkout: async (plan: 'monthly' | 'annual', returnUrl?: string, cancelUrl?: string) => {
    const params = new URLSearchParams()
    if (returnUrl) params.append('return_url', returnUrl)
    if (cancelUrl) params.append('cancel_url', cancelUrl)
    const query = params.toString()
    const response = await api.post<CheckoutResponse>(
      `/users/me/subscription/checkout${query ? `?${query}` : ''}`,
      { plan }
    )
    return response.data
  },
}

export const grammarAPI = {
  analyze: async (text: string, language: string, nativeLanguage?: string) => {
    const response = await api.post('/grammar/analyze-text', { text, language, nativeLanguage })
    return response.data.data
  },
  // Submits an AI grammar analysis. Returns immediately with a job id; the
  // result arrives over the WebSocket "grammar_analysis" event (or via getAnalysis).
  analyzeAI: async (data: {
    text: string
    language: string
    nativeLanguage?: string
    messageId?: string
    chatId?: string
  }) => {
    const response = await api.post<{
      jobId: string
      status: string
      messageId?: string
      analysis?: any
      providerUsed?: string
    }>('/grammar/analyze-ai', data)
    return response.data
  },

  getAnalysis: async (jobId: string) => {
    const response = await api.get<GrammarJob>(`/grammar/analyze/${jobId}`)
    return response.data
  },

  learn: async (text: string, language: string, nativeLanguage: string, action: string, customQuery?: string) => {
    const response = await api.post('/grammar/learn', { text, language, nativeLanguage, action, customQuery })
    return response.data.data
  },

  getSuggestions: async (level: string, language: string) => {
    const response = await api.get(`/grammar/suggestions?level=${level}&language=${language}`)
    return response.data.suggestions
  },

  getReport: async (language: string) => {
    const response = await api.get(`/grammar/report?language=${language}`)
    return response.data
  },
}

// Per-message translations requested manually by a participant (the
// "Translate" button). The result is delivered over the WebSocket
// "message_updated" event once the async job completes.
export const translationAPI = {
  translateMessage: async (chatId: string, messageId: string, targetLang: string) => {
    const response = await api.post(`/chats/${chatId}/messages/${messageId}/translate`, {
      targetLang,
    })
    return response.data
  },
}

export default api
