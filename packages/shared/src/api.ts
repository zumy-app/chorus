// Shared HTTP API client for Chorus. Platform-agnostic: the consuming app
// injects a storage adapter (AsyncStorage / localStorage) and the API base URL.
// Exposes the same endpoint groups the web frontend used to define inline, plus
// the auth helpers the React Native app needs.

import axios, { AxiosInstance } from 'axios'
import type {
  AdminStatus,
  AdminStats,
  AnswerSessionItemResponse,
  AuthTokens,
  Block,
  Chat,
  ChatPreference,
  CheckoutResponse,
  ContactInvite,
  ContactInviteRequest,
  ContactMatch,
  ContactScanRequest,
  CreateChatRequest,
  CurriculumStep,
  EmailOutboxEntry,
  Entitlements,
  GrammarJob,
  GrantPlanRequest,
  LearningDashboard,
  LearningPairCapability,
  LearningPath,
  LearningProfileUpdateRequest,
  LearningSession,
  LessonAttempt,
  LessonStartResponse,
  LessonStepResult,
  LoginRequest,
  Message,
  MinedItem,
  OnboardRequest,
  PinnedMessage,
  PlanChange,
  PlacementResult,
  PremiumAnalytics,
  PremiumUserRow,
  PresenceStatus,
  ProviderHealth,
  RealTalkPrompt,
  RegisterRequest,
  Report,
  ReportRequest,
  ReportStats,
  ScenarioAIReply,
  ScenarioChunk,
  ScenarioRun,
  ScenarioScript,
  ScenarioStartResponse,
  SendLocationRequest,
  SendMessageRequest,
  SessionAnswerRequest,
  StartPlacementResponse,
  StartSessionRequest,
  StartSessionResponse,
  StreakRecoverResult,
  SubscriptionInfo,
  TranslationJob,
  UnitProgressSummary,
  User,
  UserLanguageProfile,
  VocabularyCard,
  WaitlistEntry,
  WaitlistRequest,
} from './types'

export interface StorageAdapter {
  getItem(key: string): Promise<string | null>
  setItem(key: string, value: string): Promise<void>
  removeItem(key: string): Promise<void>
}

export interface ApiClientOptions {
  baseURL: string
  storage: StorageAdapter
  /** Called when a 401 token-refresh attempt fails (e.g. redirect to /login). */
  onAuthFailure?: () => void
}

/** Tiny query-string builder. Avoids URLSearchParams, which Hermes (RN) lacks. */
function qs(params: Record<string, string | number | boolean | null | undefined>): string {
  const parts: string[] = []
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === null || value === '') continue
    parts.push(`${encodeURIComponent(key)}=${encodeURIComponent(String(value))}`)
  }
  return parts.length ? `?${parts.join('&')}` : ''
}

declare module 'axios' {
  interface InternalAxiosRequestConfig {
    _retry?: boolean
  }
}

export function createApiClient(options: ApiClientOptions) {
  const { baseURL, storage, onAuthFailure } = options
  const client: AxiosInstance = axios.create({
    baseURL,
    timeout: 10000,
    headers: {
      'Content-Type': 'application/json',
    },
  })

  client.interceptors.request.use(
    async config => {
      const token = await storage.getItem('accessToken')
      if (token) {
        config.headers.Authorization = `Bearer ${token}`
      }
      return config
    },
    error => Promise.reject(error)
  )

  client.interceptors.response.use(
    response => response,
    async error => {
      const originalRequest = error.config
      if (error.response?.status === 401 && !originalRequest._retry) {
        originalRequest._retry = true
        try {
          const refreshToken = await storage.getItem('refreshToken')
          if (refreshToken) {
            const { data } = await client.post<{ accessToken: string; refreshToken: string }>(
              '/auth/refresh',
              { refreshToken }
            )
            await storage.setItem('accessToken', data.accessToken)
            if (data.refreshToken) {
              await storage.setItem('refreshToken', data.refreshToken)
            }
            originalRequest.headers.Authorization = `Bearer ${data.accessToken}`
            return client(originalRequest)
          }
        } catch (refreshError) {
          await storage.removeItem('accessToken')
          await storage.removeItem('refreshToken')
          onAuthFailure?.()
          throw refreshError
        }
      }
      return Promise.reject(error)
    }
  )

  const auth = {
    register: async (data: RegisterRequest) => {
      const response = await client.post<{ user: User; tokens: AuthTokens }>('/auth/register', data)
      return response.data
    },

    inviteEmail: async (token: string) => {
      const response = await client.get<{ email: string }>('/auth/invite', { params: { token } })
      return response.data.email
    },

    login: async (data: LoginRequest) => {
      const response = await client.post<{ user: User; tokens: AuthTokens }>('/auth/login', data)
      return response.data
    },

    refreshToken: async (refreshToken: string) => {
      const response = await client.post<{ user: User; tokens: AuthTokens }>('/auth/refresh', {
        refreshToken,
      })
      return response.data
    },

    forgotPassword: async (email: string) => {
      const response = await client.post<{ message: string }>('/auth/forgot-password', { email })
      return response.data
    },

    resetPassword: async (token: string, password: string) => {
      const response = await client.post<{ message: string }>('/auth/reset-password', {
        token,
        password,
      })
      return response.data
    },

    getMe: async () => {
      const response = await client.get<User>('/users/me')
      return response.data
    },

    getEntitlements: async () => {
      const response = await client.get<Entitlements>('/users/me/entitlements')
      return response.data
    },

    updateMe: async (data: {
      firstName?: string
      lastName?: string
      displayName?: string
      nativeLanguage?: string
      targetLanguages?: string[]
    }) => {
      const response = await client.put<User>('/users/me', data)
      return response.data
    },

    // Onboarding name capture (REQ 2.1): composes displayName from first + last
    // unless an explicit displayName override is provided.
    onboard: async (data: OnboardRequest) => {
      const response = await client.put<User>('/users/me/onboard', data)
      return response.data
    },

    searchUsers: async (query: string) => {
      const response = await client.get<{ users: User[] }>(`/users/search${qs({ q: query })}`)
      return response.data.users
    },

    logout: async () => {
      await storage.removeItem('accessToken')
      await storage.removeItem('refreshToken')
      await storage.removeItem('user')
    },
  }

  const waitlist = {
    join: async (data: WaitlistRequest) => {
      const response = await client.post<{
        entry: WaitlistEntry
        message: string
        alreadyJoined?: boolean
        emailSent?: boolean
      }>('/waitlist', data)
      return response.data
    },
  }

  // Contacts & Invites epic (REQ 2.4 / FR-22-23): hashed on-platform scan and
  // single-use off-platform invites (email dispatched, sms/whatsapp link).
  const contacts = {
    scan: async (data: ContactScanRequest) => {
      const response = await client.post<{ data: ContactMatch[] }>('/contacts/scan', data)
      return response.data.data
    },
    createInvite: async (data: ContactInviteRequest) => {
      const response = await client.post<{ data: ContactInvite }>('/contacts/invites', data)
      return response.data.data
    },
    listInvites: async (params?: { limit?: number; offset?: number }) => {
      const response = await client.get<{ data: ContactInvite[] }>(
        `/contacts/invites${qs({ limit: params?.limit, offset: params?.offset })}`
      )
      return response.data.data
    },
  }

  const admin = {
    status: async () => {
      const response = await client.get<AdminStatus>('/admin/status')
      return response.data
    },

    isAdmin: async () => {
      const data = await admin.status()
      return data.isAdmin
    },

    isModerator: async () => {
      const data = await admin.status()
      return data.isModerator
    },

    stats: async () => {
      const response = await client.get<AdminStats>('/admin/stats')
      return response.data
    },

    listWaitlist: async (status?: string, q?: string) => {
      const response = await client.get<{ entries: WaitlistEntry[] }>(
        `/admin/waitlist${qs({ status, q })}`
      )
      return response.data.entries
    },

    approve: async (id: string) => {
      const response = await client.post<{ message: string }>(`/admin/waitlist/${id}/approve`)
      return response.data
    },

    decline: async (id: string) => {
      const response = await client.post<{ message: string }>(`/admin/waitlist/${id}/decline`)
      return response.data
    },

    resendInvite: async (id: string) => {
      const response = await client.post<{ message: string }>(`/admin/waitlist/${id}/resend-invite`)
      return response.data
    },

    emails: async (status?: string) => {
      const response = await client.get<{ emails: EmailOutboxEntry[] }>(
        `/admin/emails${qs({ status })}`
      )
      return response.data.emails
    },

    retryEmail: async (id: string) => {
      const response = await client.post<{ message: string }>(`/admin/emails/${id}/retry`)
      return response.data
    },

    listUsers: async (
      params: { q?: string; role?: string; status?: string; limit?: number } = {}
    ) => {
      const response = await client.get<{ users: User[]; total: number }>(
        `/admin/users${qs(params)}`
      )
      return response.data
    },

    setUserRole: async (id: string, role: string) => {
      const response = await client.put<{ user: User }>(`/admin/users/${id}/role`, { role })
      return response.data
    },

    suspendUser: async (id: string) => {
      const response = await client.post<{ message: string }>(`/admin/users/${id}/suspend`)
      return response.data
    },

    unsuspendUser: async (id: string) => {
      const response = await client.post<{ message: string }>(`/admin/users/${id}/unsuspend`)
      return response.data
    },

    deleteUser: async (id: string) => {
      const response = await client.delete<{ message: string }>(`/admin/users/${id}`)
      return response.data
    },

    listTranslations: async (status?: string, q?: string) => {
      const response = await client.get<{ jobs: TranslationJob[]; total: number }>(
        `/admin/translations${qs({ status: status && status !== 'all' ? status : '', q })}`
      )
      return response.data
    },

    retryTranslation: async (id: string) => {
      const response = await client.post<{ message: string }>(`/admin/translations/${id}/retry`)
      return response.data
    },

    translationHealth: async () => {
      const response = await client.get<{ providers: ProviderHealth[] }>(
        '/admin/translations/health'
      )
      return response.data.providers
    },

    premiumUsers: async (q?: string, limit = 50, offset = 0) => {
      const response = await client.get<{ users: PremiumUserRow[]; total: number }>(
        `/admin/premium/users${qs({ limit, offset, q })}`
      )
      return response.data
    },

    premiumAnalytics: async () => {
      const response = await client.get<PremiumAnalytics>('/admin/premium/analytics')
      return response.data
    },

    grantPlan: async (id: string, data: GrantPlanRequest) => {
      const response = await client.put<{ message: string }>(`/admin/users/${id}/plan`, data)
      return response.data
    },

    revokePlan: async (id: string, clearGraceInDays = 0, reason?: string) => {
      const response = await client.post<{ message: string }>(`/admin/users/${id}/plan/revoke`, {
        plan: 'free',
        clearGraceInDays,
        reason,
      })
      return response.data
    },

    planHistory: async (id: string) => {
      const response = await client.get<{ history: PlanChange[] }>(`/admin/users/${id}/plan-history`)
      return response.data.history
    },

    listReports: async (status?: string, q?: string, limit = 50, offset = 0) => {
      const response = await client.get<{ reports: Report[]; total: number }>(
        `/admin/reports${qs({
          limit,
          offset,
          status: status && status !== 'all' ? status : '',
          q,
        })}`
      )
      return response.data
    },

    reportStats: async () => {
      const response = await client.get<ReportStats>('/admin/reports/stats')
      return response.data
    },

    resolveReport: async (id: string) => {
      const response = await client.post<{ message: string }>(`/admin/reports/${id}/resolve`)
      return response.data
    },

    dismissReport: async (id: string, note?: string) => {
      const response = await client.post<{ message: string }>(`/admin/reports/${id}/dismiss`, {
        note,
      })
      return response.data
    },
  }

  const moderation = {
    block: async (blockedUserId: string, reason?: string) => {
      const response = await client.post<{ message: string }>('/blocks', {
        blockedUserId,
        reason,
      })
      return response.data
    },

    unblock: async (userId: string) => {
      await client.delete(`/blocks/${userId}`)
    },

    getBlocked: async () => {
      const response = await client.get<{ blocks: Block[]; total: number }>('/blocks')
      return response.data.blocks
    },

    getBlockStatus: async (userId: string) => {
      const response = await client.get<import('./types').BlockStatus>(`/blocks/${userId}/status`)
      return response.data
    },

    report: async (data: ReportRequest) => {
      const response = await client.post<Report>('/reports', data)
      return response.data
    },
  }

  const chat = {
    getChats: async () => {
      const response = await client.get<{ chats: Chat[] }>('/chats')
      return response.data.chats
    },

    createChat: async (data: CreateChatRequest) => {
      const response = await client.post<Chat>('/chats', data)
      return response.data
    },

    getChat: async (chatId: string) => {
      const response = await client.get<Chat>(`/chats/${chatId}`)
      return response.data
    },

    updateChat: async (chatId: string, data: { name?: string; settings?: any }) => {
      const response = await client.put<Chat>(`/chats/${chatId}`, data)
      return response.data
    },

    addParticipant: async (chatId: string, userId: string) => {
      await client.post(`/chats/${chatId}/participants`, { userId })
    },

    removeParticipant: async (chatId: string, userId: string) => {
      await client.delete(`/chats/${chatId}/participants/${userId}`)
    },

    leaveChat: async (chatId: string) => {
      await client.delete(`/chats/${chatId}/leave`)
    },

    // Task 6.4 (archive & mute): per-user, per-chat conversation preferences.
    archiveChat: async (chatId: string, archived = true) => {
      const response = await client.post<ChatPreference>(`/chats/${chatId}/archive`, { archived })
      return response.data
    },

    unarchiveChat: async (chatId: string) => {
      const response = await client.delete<ChatPreference>(`/chats/${chatId}/archive`)
      return response.data
    },

    muteChat: async (chatId: string, until?: string) => {
      const response = await client.post<ChatPreference>(`/chats/${chatId}/mute`, {
        muted: true,
        until: until ?? null,
      })
      return response.data
    },

    unmuteChat: async (chatId: string) => {
      const response = await client.delete<ChatPreference>(`/chats/${chatId}/mute`)
      return response.data
    },

    getChatPreference: async (chatId: string) => {
      const response = await client.get<ChatPreference>(`/chats/${chatId}/preferences`)
      return response.data
    },

    getChatPreferences: async () => {
      const response = await client.get<{ preferences: Record<string, ChatPreference> }>(
        '/chats/preferences'
      )
      return response.data.preferences
    },
  }

  const message = {
    getMessages: async (chatId: string, limit = 50, before?: string) => {
      const response = await client.get<{ messages: Message[] }>(
        `/chats/${chatId}/messages${qs({ limit, before })}`
      )
      return response.data.messages
    },

    sendMessage: async (chatId: string, data: SendMessageRequest) => {
      const response = await client.post<Message>(`/chats/${chatId}/messages`, data)
      return response.data
    },

    // Task 6.6: file/document sharing. Multipart upload that creates a media
    // message; the returned message carries media[0] with the public URL.
    sendAttachment: async (chatId: string, file: Blob, fileName: string, opts?: { caption?: string; type?: string }) => {
      const form = new FormData()
      ;(form as any).append('file', file, fileName)
      if (opts?.caption) form.append('caption', opts.caption)
      if (opts?.type) form.append('type', opts.type)
      const response = await client.post<Message>(`/chats/${chatId}/attachments`, form, {
        headers: { 'Content-Type': 'multipart/form-data' },
      })
      return response.data
    },

    // Task 6.7: location sharing. A validated lat/lng (plus optional label) that
    // creates a location message; the returned message carries media[0] with the
    // pin's coordinates + map URL.
    sendLocation: async (chatId: string, data: SendLocationRequest) => {
      const response = await client.post<Message>(`/chats/${chatId}/location`, data)
      return response.data
    },

    markAsRead: async (chatId: string, messageId: string) => {
      await client.put(`/chats/${chatId}/read`, { messageId })
    },

    searchMessages: async (query: string, chatId?: string) => {
      const response = await client.get<{ messages: Message[] }>(
        `/messages/search${qs({ q: query, chatId })}`
      )
      return response.data.messages
    },

    universalSearch: async (query: string, params?: { chatId?: string; type?: string; limit?: number; offset?: number }) => {
      const response = await client.get<import('./types').SearchResult>(
        `/messages/search${qs({ q: query, chatId: params?.chatId, type: params?.type, limit: params?.limit, offset: params?.offset })}`
      )
      return response.data
    },

    searchMedia: async (query: string, params?: { type?: string; chatId?: string; limit?: number; offset?: number }) => {
      const response = await client.get<import('./types').MediaSearchResult>(
        `/media/search${qs({ q: query, ...params })}`
      )
      return response.data
    },

    // Message actions (task 6.2).
    forwardMessage: async (chatId: string, messageId: string, targetChatId: string) => {
      const response = await client.post<Message>(
        `/chats/${chatId}/messages/${messageId}/forward`,
        { targetChatId }
      )
      return response.data
    },

    deleteMessage: async (chatId: string, messageId: string) => {
      await client.delete(`/chats/${chatId}/messages/${messageId}`)
    },

    pinMessage: async (chatId: string, messageId: string) => {
      const response = await client.post(`/chats/${chatId}/pins`, { messageId })
      return response.data
    },

    unpinMessage: async (chatId: string, messageId: string) => {
      await client.delete(`/chats/${chatId}/pins/${messageId}`)
    },

    getPinnedMessages: async (chatId: string) => {
      const response = await client.get<{ pins: PinnedMessage[] }>(`/chats/${chatId}/pins`)
      return response.data.pins
    },
  }

  const vocabulary = {
    getAll: async (language?: string) => {
      const response = await client.get(`/vocabulary${qs({ language })}`)
      return response.data.data?.entries || []
    },

    getDue: async () => {
      const response = await client.get('/vocabulary/due')
      return response.data.data || []
    },

    save: async (term: string, language: string, messageId: string) => {
      const response = await client.post('/vocabulary', { term, language, messageId })
      return response.data.data
    },

    practice: async (vocabularyId: string, correct: boolean) => {
      await client.post('/vocabulary/practice', { vocabularyId, correct })
    },

    getProgress: async () => {
      const response = await client.get('/vocabulary/progress')
      return response.data.data
    },

    delete: async (id: string) => {
      await client.delete(`/vocabulary/${id}`)
    },
  }

  const billing = {
    getMySubscription: async () => {
      const response = await client.get<SubscriptionInfo>('/users/me/subscription')
      return response.data
    },

    checkout: async (plan: 'monthly' | 'annual', returnUrl?: string, cancelUrl?: string) => {
      const response = await client.post<CheckoutResponse>(
        `/users/me/subscription/checkout${qs({ return_url: returnUrl, cancel_url: cancelUrl })}`,
        { plan }
      )
      return response.data
    },
  }

  const grammar = {
    analyze: async (text: string, language: string, nativeLanguage?: string) => {
      const response = await client.post('/grammar/analyze-text', {
        text,
        language,
        nativeLanguage,
      })
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
      const response = await client.post<{
        jobId: string
        status: string
        messageId?: string
        analysis?: any
        providerUsed?: string
      }>('/grammar/analyze-ai', data)
      return response.data
    },

    getAnalysis: async (jobId: string) => {
      const response = await client.get<GrammarJob>(`/grammar/analyze/${jobId}`)
      return response.data
    },

    learn: async (
      text: string,
      language: string,
      nativeLanguage: string,
      action: string,
      customQuery?: string
    ) => {
      const response = await client.post('/grammar/learn', {
        text,
        language,
        nativeLanguage,
        action,
        customQuery,
      })
      return response.data.data
    },

    getSuggestions: async (level: string, language: string) => {
      const response = await client.get(`/grammar/suggestions${qs({ level, language })}`)
      return response.data.suggestions
    },

    getReport: async (language: string) => {
      const response = await client.get(`/grammar/report${qs({ language })}`)
      return response.data
    },
  }

  // Per-message translations requested manually by a participant (the
  // "Translate" button). The result is delivered over the WebSocket
  // "message_updated" event once the async job completes.
  const translation = {
    translateMessage: async (chatId: string, messageId: string, targetLang: string) => {
      const response = await client.post(`/chats/${chatId}/messages/${messageId}/translate`, {
        targetLang,
      })
      return response.data
    },
  }

  // Pair-aware learning engine. Structured courses exist only for curated
  // native->target pairs (launch: en -> es); unseeded pairs return vocab_only.
  const learning = {
    getCapabilities: async (nativeLanguage: string, targetLanguage: string) => {
      const response = await client.get<{ data: LearningPairCapability }>(
        `/learning/capabilities${qs({ nativeLanguage, targetLanguage })}`
      )
      return response.data.data
    },

    getProfile: async (targetLanguage: string, nativeLanguage?: string) => {
      const response = await client.get<{ data: UserLanguageProfile }>(
        `/learning/profile${qs({ targetLanguage, nativeLanguage })}`
      )
      return response.data.data
    },

    updateProfile: async (data: LearningProfileUpdateRequest) => {
      const response = await client.put<{ data: UserLanguageProfile }>('/learning/profile', data)
      return response.data.data
    },

    getDashboard: async (targetLanguage: string, nativeLanguage?: string) => {
      const response = await client.get<{ data: LearningDashboard }>(
        `/learning/dashboard${qs({ targetLanguage, nativeLanguage })}`
      )
      return response.data.data
    },

    getPath: async (targetLanguage: string, nativeLanguage?: string) => {
      const response = await client.get<{ data: LearningPath }>(
        `/learning/path${qs({ targetLanguage, nativeLanguage })}`
      )
      return response.data.data
    },

    // Placement
    startPlacement: async (targetLanguage: string, nativeLanguage?: string) => {
      const response = await client.post<{ data: StartPlacementResponse }>(
        `/learning/placement/start${qs({ targetLanguage, nativeLanguage })}`
      )
      return response.data.data
    },
    answerPlacement: async (attemptId: string, answer: string) => {
      const response = await client.post<{ data: PlacementResult | StartPlacementResponse }>(
        `/learning/placement/${attemptId}/answer`,
        { answer }
      )
      return response.data.data
    },
    skipPlacement: async (targetLanguage: string, nativeLanguage?: string) => {
      const response = await client.post<{ data: PlacementResult }>(
        `/learning/placement/skip${qs({ targetLanguage, nativeLanguage })}`
      )
      return response.data.data
    },
    selectLevel: async (level: 'beginner' | 'intermediate' | 'advanced', targetLanguage: string, nativeLanguage?: string) => {
      const response = await client.post<{ data: PlacementResult }>(
        '/learning/level/select',
        { level, targetLanguage, nativeLanguage }
      )
      return response.data.data
    },
    getPlacement: async (attemptId: string) => {
      const response = await client.get<{ data: StartPlacementResponse }>(
        `/learning/placement/${attemptId}`
      )
      return response.data.data
    },

    // Lessons
    getUnit: async (unitId: string) => {
      const response = await client.get<{ data: UnitProgressSummary }>(`/learning/units/${unitId}`)
      return response.data.data
    },
    startLesson: async (lessonId: string, targetLanguage: string, nativeLanguage?: string) => {
      const response = await client.post<{ data: LessonStartResponse }>(
        `/learning/lessons/${lessonId}/start${qs({ targetLanguage, nativeLanguage })}`
      )
      return response.data.data
    },
    answerLessonStep: async (attemptId: string, stepId: string, answer: string) => {
      const response = await client.post<{ data: LessonStepResult }>(
        `/learning/lesson-attempts/${attemptId}/steps/${stepId}/answer`,
        { answer }
      )
      return response.data.data
    },
    completeLesson: async (attemptId: string) => {
      const response = await client.post<{ data: LessonAttempt }>(
        `/learning/lesson-attempts/${attemptId}/complete`
      )
      return response.data.data
    },
    getLessonAttempt: async (attemptId: string) => {
      const response = await client.get<{ data: { attempt: LessonAttempt; steps: CurriculumStep[] } }>(
        `/learning/lesson-attempts/${attemptId}`
      )
      return response.data.data
    },

    // Sessions
    startSession: async (data: StartSessionRequest) => {
      const response = await client.post<{ data: StartSessionResponse }>('/learning/sessions/start', data)
      return response.data.data
    },
    getSession: async (sessionId: string) => {
      const response = await client.get<{ data: LearningSession }>(`/learning/sessions/${sessionId}`)
      return response.data.data
    },
    answerSessionItem: async (sessionId: string, itemId: string, answer: SessionAnswerRequest, latencyMs?: number) => {
      const response = await client.post<{ data: AnswerSessionItemResponse }>(
        `/learning/sessions/${sessionId}/items/${itemId}/answer`,
        { answer, latencyMs }
      )
      return response.data.data
    },
    completeSession: async (sessionId: string) => {
      const response = await client.post<{ data: LearningSession }>(`/learning/sessions/${sessionId}/complete`)
      return response.data.data
    },

    // Mined vocabulary
    getMinedItems: async (targetLanguage: string, status?: string) => {
      const response = await client.get<{ data: MinedItem[] | null }>(
        `/learning/vocabulary/mined${qs({ targetLanguage, status })}`
      )
      return response.data.data ?? []
    },
    acceptMinedItem: async (id: string) => {
      const response = await client.post<{ data: VocabularyCard }>(`/learning/vocabulary/mined/${id}/accept`)
      return response.data.data
    },
    ignoreMinedItem: async (id: string) => {
      const response = await client.post<{ data: { ok: boolean } }>(`/learning/vocabulary/mined/${id}/ignore`)
      return response.data.data
    },
    reviewVocabulary: async (id: string, answer: string, latencyMs?: number) => {
      const response = await client.post<{ data: { card: VocabularyCard; correct: boolean; quality: number } }>(`/learning/vocabulary/${id}/review`, { answer, latencyMs })
      return response.data.data
    },

    // Scenarios
    getScenarios: async (targetLanguage: string, nativeLanguage?: string) => {
      const response = await client.get<{ data: ScenarioScript[] }>(
        `/learning/scenarios${qs({ targetLanguage, nativeLanguage })}`
      )
      return response.data.data
    },
    getScenario: async (scenarioId: string) => {
      const response = await client.get<{ data: ScenarioScript }>(`/learning/scenarios/${scenarioId}`)
      return response.data.data
    },
    startScenario: async (scenarioId: string, targetLanguage: string, nativeLanguage?: string) => {
      const response = await client.post<{ data: ScenarioStartResponse }>(
        `/learning/scenarios/${scenarioId}/start${qs({ targetLanguage, nativeLanguage })}`
      )
      return response.data.data
    },
    getScenarioRun: async (runId: string) => {
      const response = await client.get<{ data: ScenarioRun }>(`/learning/scenario-runs/${runId}`)
      return response.data.data
    },
    sendScenarioMessage: async (runId: string, message: string) => {
      const response = await client.post<{ data: ScenarioAIReply }>(
        `/learning/scenario-runs/${runId}/message`,
        { message }
      )
      return response.data.data
    },
    requestScenarioHint: async (runId: string) => {
      const response = await client.post<{ data: ScenarioChunk[] }>(`/learning/scenario-runs/${runId}/hint`)
      return response.data.data
    },
    completeScenario: async (runId: string) => {
      const response = await client.post<{ data: ScenarioAIReply }>(`/learning/scenario-runs/${runId}/complete`)
      return response.data.data
    },

    // Real talk + streak
    getRealTalkPrompts: async (targetLanguage: string, nativeLanguage?: string, chatId?: string) => {
      const response = await client.get<{ data: RealTalkPrompt[] }>(
        `/learning/real-talk/prompts${qs({ targetLanguage, nativeLanguage, chatId })}`
      )
      return response.data.data
    },
    markRealTalkUsed: async (promptId: string) => {
      const response = await client.post<{ data: { ok: boolean } }>(`/learning/real-talk/prompts/${promptId}/used`)
      return response.data.data
    },
    recoverStreak: async (targetLanguage: string, nativeLanguage?: string) => {
      const response = await client.post<{ data: StreakRecoverResult }>(
        `/learning/streak/recover${qs({ targetLanguage, nativeLanguage })}`
      )
      return response.data.data
    },
  }

  const payouts = {
    overview: async () => {
      const response = await client.get<{ overview: import('./types').PayoutOverview }>('/teachers/payouts/overview')
      return response.data.overview
    },
    methods: async () => {
      const response = await client.get<{ methods: import('./types').PayoutMethod[] }>('/teachers/payouts/methods')
      return response.data.methods
    },
    addMethod: async (data: { type: 'paypal' | 'bank'; label: string; details: string; isDefault?: boolean }) => {
      const response = await client.post<{ method: import('./types').PayoutMethod }>('/teachers/payouts/methods', data)
      return response.data.method
    },
    removeMethod: async (id: string) => {
      await client.delete(`/teachers/payouts/methods/${id}`)
    },
    setDefaultMethod: async (id: string) => {
      await client.put(`/teachers/payouts/methods/${id}/default`)
    },
    history: async (params?: { limit?: number; offset?: number }) => {
      const response = await client.get<{ payouts: import('./types').PayoutRecord[]; total: number; hasMore: boolean }>(`/teachers/payouts/history${qs({ limit: params?.limit, offset: params?.offset })}`)
      return response.data
    },
    withdraw: async (data: { amountCents: number; methodId?: string }) => {
      const response = await client.post<{ payout: import('./types').PayoutRecord }>('/teachers/payouts/withdraw', data)
      return response.data.payout
    },
  }

  const health = async () => {
    // The health endpoint sits at <origin>/health, outside the /api/v1 prefix.
    const healthUrl = baseURL.replace('/api/v1', '/health')
    const response = await axios.get(healthUrl)
    return response.data
  }

  const otp = {
    getPhoneStatus: async () => {
      const response = await client.get<import('./types').PhoneStatus>('/users/me/phone')
      return response.data
    },
    requestOTP: async (phone: string) => {
      const response = await client.post<{ phoneMasked: string }>('/users/me/phone/request-otp', { phone })
      return response.data
    },
    verifyPhone: async (phone: string, code: string) => {
      const response = await client.post<{ status: import('./types').PhoneStatus }>('/users/me/phone/verify', { phone, code })
      return response.data
    },
    setTwoFactor: async (enabled: boolean) => {
      const response = await client.put<import('./types').PhoneStatus>('/users/me/2fa', { enabled })
      return response.data
    },
    verify2FA: async (tempToken: string, code: string) => {
      const response = await client.post<{ user: User; tokens: AuthTokens }>('/auth/2fa/verify', { tempToken, code })
      return response.data
    },
  }

  const captionReview = {
    getQueue: async (params?: { limit?: number; offset?: number }) => {
      const r = await client.get<{ items: import('./types').CaptionReviewQueueItem[]; total: number; hasMore: boolean }>(`/captions/review-queue${qs({ limit: params?.limit, offset: params?.offset })}`)
      return r.data
    },
    getStats: async () => {
      const r = await client.get<import('./types').CaptionQualityStats>('/captions/quality-stats')
      return r.data
    },
    review: async (callId: string, index: number, data: { rating: number; correctedText?: string; feedback?: string; targetLanguage?: string }) => {
      const r = await client.post<import('./types').CaptionReview>(`/calls/${callId}/captions/${index}/review`, data)
      return r.data
    },
    getReviews: async (callId: string, index: number) => {
      const r = await client.get<{ reviews: import('./types').CaptionReview[] }>(`/calls/${callId}/captions/${index}/reviews`)
      return r.data.reviews
    },
  }
  const call = {
    initiate: async (chatId: string, type: 'audio' | 'video' = 'audio') => {
      const response = await client.post<{ session: import('./types').CallSession; offer: import('./types').WebRTCOffer }>('/calls/initiate', { chatId, type })
      return response.data
    },
    getSession: async (callId: string) => {
      const response = await client.get<import('./types').CallSession>(`/calls/${callId}`)
      return response.data
    },
    end: async (callId: string) => {
      const response = await client.post<{ message: string }>(`/calls/${callId}/end`)
      return response.data
    },
    getCaptions: async (callId: string, params?: { limit?: number; offset?: number }) => {
      const response = await client.get<{ segments: import('./types').TranscriptSegment[]; total: number; hasMore: boolean }>(`/calls/${callId}/captions${qs({ limit: params?.limit, offset: params?.offset })}`)
      return response.data
    },
    postCaption: async (callId: string, data: { text: string; language?: string }) => {
      const response = await client.post<import('./types').TranscriptSegment>(`/calls/${callId}/captions`, data)
      return response.data
    },
    bookmarkCaption: async (callId: string, index: number, phrase?: string) => {
      const response = await client.post(`/calls/${callId}/captions/${index}/bookmark`, phrase ? { phrase } : {})
      return response.data
    },
    transcribe: async (callId: string, data: { audio: string; language?: string }) => {
      const response = await client.post<import('./types').TranscriptSegment>(`/calls/${callId}/transcribe`, data)
      return response.data
    },
    signal: async (callId: string, data: { type: string; sdp?: string; candidate?: string; data?: Record<string, unknown> }) => {
      const response = await client.post(`/calls/${callId}/signal`, data)
      return response.data
    },
    getTranscript: async (callId: string) => {
      const response = await client.get<import('./types').CallTranscript>(`/calls/${callId}/transcript`)
      return response.data
    },
    getHistory: async (params?: { limit?: number; offset?: number }) => {
      const response = await client.get<import('./types').CallSession[]>(`/calls/history${qs({ limit: params?.limit, offset: params?.offset })}`)
      return response.data
    },
    searchTranscripts: async (query: string, language?: string) => {
      const response = await client.get<import('./types').CallTranscript[]>(`/calls/transcripts/search${qs({ q: query, language })}`)
      return response.data
    },
  }

  const search = {
    universal: async (query: string, params?: { chatId?: string; type?: string; limit?: number; offset?: number; language?: string }) => {
      const r = await client.get<import('./types').SearchResult>(`/messages/search${qs({ q: query, chatId: params?.chatId, type: params?.type, limit: params?.limit, offset: params?.offset, language: params?.language })}`)
      return r.data
    },
    media: async (query: string, params?: { chatId?: string; type?: string; limit?: number; offset?: number }) => {
      const r = await client.get<import('./types').MediaSearchResult>(`/media/search${qs({ q: query, chatId: params?.chatId, type: params?.type, limit: params?.limit, offset: params?.offset })}`)
      return r.data
    },
    chats: async (query: string) => {
      const r = await client.get<{ data: import('./types').Chat[] }>(`/chats/search${qs({ q: query })}`)
      return r.data.data
    },
    contacts: async (query: string) => {
      const r = await client.get<{ data: import('./types').User[] }>(`/contacts/search${qs({ q: query })}`)
      return r.data.data
    },
  }

  const presence = {
    get: async (userId: string) => {
      const response = await client.get<{ data: PresenceStatus }>(`/presence/${userId}`)
      return response.data.data
    },

    getMultiple: async (userIds: string[]) => {
      const response = await client.post<{ data: Record<string, PresenceStatus> }>(
        '/presence/batch',
        { userIds }
      )
      return response.data.data
    },

    update: async (data: { status: PresenceStatus['status']; deviceType?: string }) => {
      return client.put('/presence', data)
    },

    heartbeat: async (deviceType?: string) => {
      return client.post(`/presence/heartbeat${qs({ deviceType })}`)
    },

    activity: async () => {
      return client.post('/presence/activity')
    },
  }

  const settings = {
    getSettings: async () => {
      const response = await client.get<import('./types').UserSettings>('/users/me/settings')
      return response.data
    },
    updateSettings: async (data: Partial<Record<'translationEnabled' | 'grammarAuto' | 'highlightsEnabled', boolean>> & Partial<Record<'lastSeenVisibility' | 'profilePhotoVisibility' | 'contactsVisibility', import('./types').PrivacyVisibility>>) => {
      const response = await client.put<import('./types').UserSettings>('/users/me/settings', data)
      return response.data
    },
  }

  const teacher = {
    getMyApplication: async () => {
      const response = await client.get<{ application: import('./types').TeacherApplication | null }>('/teachers/me')
      return response.data.application
    },
    apply: async (data: import('./types').TeacherApplyRequest) => {
      const response = await client.post<{ application: import('./types').TeacherApplication }>('/teachers/apply', data)
      return response.data.application
    },
    browse: async (params?: { language?: string; search?: string; verified?: boolean; minRating?: number; maxRate?: number; minRate?: number; sort?: string; limit?: number; offset?: number }) => {
      const response = await client.get<{ tutors: import('./types').TutorProfile[]; total: number; hasMore: boolean }>(`/teachers/browse${qs({ language: params?.language, search: params?.search, verified: params?.verified, minRating: params?.minRating, maxRate: params?.maxRate, minRate: params?.minRate, sort: params?.sort, limit: params?.limit, offset: params?.offset })}`)
      return response.data
    },
    getProfile: async (userId: string) => {
      const response = await client.get<{ tutor: import('./types').TutorProfile }>(`/teachers/${userId}`)
      return response.data.tutor
    },
    getTrialCredits: async () => {
      const response = await client.get<{ trialCredits: import('./types').TrialCredit }>('/teachers/trial-credits')
      return response.data.trialCredits
    },
    getReviews: async (userId: string, params?: { limit?: number; offset?: number }) => {
      const response = await client.get<{ reviews: import('./types').TutorReview[]; total: number; hasMore: boolean }>(`/teachers/${userId}/reviews${qs({ limit: params?.limit, offset: params?.offset })}`)
      return response.data
    },
    addReview: async (userId: string, data: { rating: number; comment?: string }) => {
      const response = await client.post<{ review: import('./types').TutorReview }>(`/teachers/${userId}/reviews`, data)
      return response.data.review
    },
    getAvailability: async (userId: string) => {
      const response = await client.get<{ availability: import('./types').TutorAvailability[] }>(`/teachers/${userId}/availability`)
      return response.data.availability
    },
    addAvailability: async (data: { startTime: string; endTime: string }) => {
      const response = await client.post<{ availability: import('./types').TutorAvailability }>('/teachers/availability', data)
      return response.data.availability
    },
    removeAvailability: async (id: string) => {
      await client.delete(`/teachers/availability/${id}`)
    },
    book: async (userId: string, data: { startTime: string; endTime: string; isTrial?: boolean; note?: string }) => {
      const response = await client.post<{ booking: import('./types').TutorBooking }>(`/teachers/${userId}/book`, data)
      return response.data.booking
    },
    listBookings: async (params?: { role?: string; limit?: number; offset?: number }) => {
      const response = await client.get<{ bookings: import('./types').TutorBooking[]; total: number; hasMore: boolean }>(`/teachers/bookings${qs({ role: params?.role, limit: params?.limit, offset: params?.offset })}`)
      return response.data
    },
    cancelBooking: async (id: string) => {
      const response = await client.post<{ booking: import('./types').TutorBooking }>(`/teachers/bookings/${id}/cancel`)
      return response.data.booking
    },
    confirmBooking: async (id: string) => {
      const response = await client.post<{ booking: import('./types').TutorBooking }>(`/teachers/bookings/${id}/confirm`)
      return response.data.booking
    },
    completeBooking: async (id: string) => {
      const response = await client.post<{ booking: import('./types').TutorBooking }>(`/teachers/bookings/${id}/complete`)
      return response.data.booking
    },
    updateReviewNotes: async (id: string, notes: string) => {
      const response = await client.put<{ booking: import('./types').TutorBooking }>(`/teachers/bookings/${id}/review-notes`, { notes })
      return response.data.booking
    },
    pushSrs: async (data: import('./types').TeacherSrsPushRequest) => {
      const response = await client.post<{ push: import('./types').TeacherSrsPush }>('/teachers/srs/push', data)
      return response.data.push
    },
    listSrsPushes: async (params?: { role?: string; peerId?: string; limit?: number; offset?: number }) => {
      const response = await client.get<{ pushes: import('./types').TeacherSrsPush[]; total: number; hasMore: boolean }>(`/teachers/srs/pushes${qs({ role: params?.role, peerId: params?.peerId, limit: params?.limit, offset: params?.offset })}`)
      return response.data
    },
    getSrsPush: async (id: string) => {
      const response = await client.get<{ push: import('./types').TeacherSrsPush }>(`/teachers/srs/pushes/${id}`)
      return response.data.push
    },
    getSrsSandbox: async (studentId: string) => {
      const response = await client.get<{ pushes: import('./types').TeacherSrsPush[] }>(`/teachers/srs/sandbox/${studentId}`)
      return response.data.pushes
    },
  }

  return {
    api: client,
    auth,
    waitlist,
    admin,
    moderation,
    chat,
    message,
    vocabulary,
    billing,
    grammar,
    translation,
    learning,
    contacts,
    otp,
    presence,
    settings,
    call,
    captionReview,
    teacher,
    payouts,
    search,
    health,
  }
}