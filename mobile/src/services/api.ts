import axios, { type AxiosError, type InternalAxiosRequestConfig } from 'axios'

import { API_URL } from './config'
import { clearSession, getAccessToken, getRefreshToken, saveSession } from './session'
import type { AuthTokens, Chat, Message, User } from '@/types'

type RetriableRequest = InternalAxiosRequestConfig & { _chorusRetried?: boolean }

const api = axios.create({ baseURL: API_URL, headers: { 'Content-Type': 'application/json' } })
let refreshing: Promise<string | null> | null = null

api.interceptors.request.use(async (request) => {
  const token = await getAccessToken()
  if (token) request.headers.Authorization = `Bearer ${token}`
  return request
})

api.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const request = error.config as RetriableRequest | undefined
    if (error.response?.status !== 401 || !request || request._chorusRetried || request.url === '/auth/refresh') {
      return Promise.reject(error)
    }

    request._chorusRetried = true
    refreshing ??= refreshAccessToken()
    const token = await refreshing.finally(() => {
      refreshing = null
    })

    if (!token) {
      await clearSession()
      return Promise.reject(error)
    }

    request.headers.Authorization = `Bearer ${token}`
    return api(request)
  },
)

async function refreshAccessToken(): Promise<string | null> {
  const refreshToken = await getRefreshToken()
  if (!refreshToken) return null

  try {
    const response = await axios.post<Pick<AuthTokens, 'accessToken' | 'expiresIn'>>(`${API_URL}/auth/refresh`, {
      refreshToken,
    })
    await saveSession({ ...response.data, refreshToken })
    return response.data.accessToken
  } catch {
    return null
  }
}

export function getApiErrorMessage(error: unknown): string {
  const status = (error as AxiosError)?.response?.status
  if (status === 401) return 'Your session has expired. Please sign in again.'
  if (status === 403) return 'You do not have permission to do that.'
  if (status === 404) return 'The requested item is no longer available.'
  if (status === 429) return 'Too many requests. Please wait a moment and try again.'
  if (status && status < 500) return 'Please check your information and try again.'
  return 'Something went wrong. Please try again.'
}

export const authApi = {
  async login(email: string, password: string) {
    const response = await api.post<{ user: User; tokens: AuthTokens }>('/auth/login', {
      username: email.trim().toLowerCase(),
      password,
    })
    await saveSession(response.data.tokens)
    return response.data.user
  },
  async register(input: {
    email: string
    password: string
    displayName: string
    nativeLanguage: string
    targetLanguages: string[]
    inviteToken?: string
  }) {
    const email = input.email.trim().toLowerCase()
    const response = await api.post<{ user: User; tokens: AuthTokens }>('/auth/register', {
      ...input,
      email,
      username: email,
    })
    await saveSession(response.data.tokens)
    return response.data.user
  },
  async getMe() {
    return (await api.get<User>('/users/me')).data
  },
  async updateMe(input: Pick<User, 'displayName' | 'nativeLanguage' | 'targetLanguages'>) {
    return (await api.put<User>('/users/me', input)).data
  },
  async forgotPassword(email: string) {
    await api.post('/auth/forgot-password', { email: email.trim().toLowerCase() })
  },
  async resetPassword(token: string, password: string) {
    await api.post('/auth/reset-password', { token, password })
  },
}

export const chatApi = {
  async list() {
    return (await api.get<{ chats: Chat[] }>('/chats')).data.chats
  },
  async get(chatId: string) {
    return (await api.get<Chat>(`/chats/${chatId}`)).data
  },
  async create(input: { type: 'direct' | 'group'; participants: string[]; name?: string }) {
    return (await api.post<Chat>('/chats', input)).data
  },
}

export const messageApi = {
  async list(chatId: string, before?: string) {
    return (await api.get<{ messages: Message[]; hasMore: boolean }>(`/chats/${chatId}/messages`, { params: { before } })).data
  },
  async send(chatId: string, input: { text: string; replyToId?: string }) {
    return (await api.post<Message>(`/chats/${chatId}/messages`, input)).data
  },
  async markRead(chatId: string, messageId: string) {
    await api.put(`/chats/${chatId}/read`, { messageId })
  },
  async translate(chatId: string, messageId: string, targetLang: string) {
    await api.post(`/chats/${chatId}/messages/${messageId}/translate`, { targetLang })
  },
}

export const contactApi = {
  async search(query: string) {
    return (await api.get<{ data: User[] }>('/contacts/search', { params: { q: query } })).data.data
  },
}

export const moderationApi = {
  async block(userId: string, reason = '') {
    await api.post('/blocks', { userId, reason })
  },
  async report(input: { type: 'user' | 'message'; reportedUserId?: string; messageId?: string; chatId?: string; reason: string }) {
    await api.post('/reports', input)
  },
}

export const accountApi = {
  async delete() {
    await api.delete('/users/me')
    await clearSession()
  },
}

export { api }
