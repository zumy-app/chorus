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
  CheckoutResponse,
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
  PlanChange,
  PlacementResult,
  PremiumAnalytics,
  PremiumUserRow,
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
      displayName?: string
      nativeLanguage?: string
      targetLanguages?: string[]
    }) => {
      const response = await client.put<User>('/users/me', data)
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

    markAsRead: async (chatId: string, messageId: string) => {
      await client.put(`/chats/${chatId}/read`, { messageId })
    },

    searchMessages: async (query: string, chatId?: string) => {
      const response = await client.get<{ messages: Message[] }>(
        `/messages/search${qs({ q: query, chatId })}`
      )
      return response.data.messages
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
      const response = await client.get<{ data: MinedItem[] }>(
        `/learning/vocabulary/mined${qs({ targetLanguage, status })}`
      )
      return response.data.data
    },
    acceptMinedItem: async (id: string) => {
      const response = await client.post<{ data: VocabularyCard }>(`/learning/vocabulary/mined/${id}/accept`)
      return response.data.data
    },
    ignoreMinedItem: async (id: string) => {
      const response = await client.post<{ data: { ok: boolean } }>(`/learning/vocabulary/mined/${id}/ignore`)
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

  const health = async () => {
    // The health endpoint sits at <origin>/health, outside the /api/v1 prefix.
    const healthUrl = baseURL.replace('/api/v1', '/health')
    const response = await axios.get(healthUrl)
    return response.data
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
    health,
  }
}