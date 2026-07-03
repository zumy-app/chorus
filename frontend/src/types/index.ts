export interface User {
  id: string
  username: string
  email: string
  displayName: string
  nativeLanguage: string
  targetLanguages: string[]
  createdAt: string
  lastActiveAt: string
}

export interface Chat {
  id: string
  type: 'direct' | 'group'
  name?: string
  participants: ChatParticipant[]
  createdBy: string
  settings?: {
    translationEnabled?: boolean
  }
  createdAt: string
  lastMessage?: Message
  unreadCount?: number
}

export interface ChatParticipant {
  chatId: string
  userId: string
  role: 'member' | 'admin'
  joinedAt: string
  lastReadMessageId?: string
  user?: User
}

export interface Message {
  id: string
  chatId: string
  senderId: string
  text: string
  originalLanguage?: string
  translations?: Record<string, string>
  translationEnhanced?: boolean
  deliveryStatus: 'sent' | 'delivered' | 'failed'
  replyToId?: string
  timestamp: string
  sender?: User
}

export interface AuthTokens {
  accessToken: string
  refreshToken: string
  expiresIn: number
}

export interface LoginRequest {
  username: string
  password: string
}

export interface RegisterRequest {
  username?: string
  email: string
  password: string
  displayName: string
  nativeLanguage: string
  targetLanguages: string[]
}

export interface CreateChatRequest {
  type: 'direct' | 'group'
  participants: string[]
  name?: string
}

export interface SendMessageRequest {
  text: string
  replyToId?: string
}

export interface WebSocketMessage {
  type: string
  data: any
}
