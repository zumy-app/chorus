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

  login: async (data: LoginRequest) => {
    const response = await api.post<{ user: User; tokens: AuthTokens }>('/auth/login', data)
    return response.data
  },

  getMe: async () => {
    const response = await api.get<User>('/users/me')
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

export default api
