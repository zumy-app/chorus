export interface AuthTokens {
  accessToken: string
  refreshToken: string
  expiresIn: number
}

export interface User {
  id: string
  email: string
  username: string
  displayName: string
  nativeLanguage: string
  targetLanguages: string[]
}

export interface Chat {
  id: string
  type: 'direct' | 'group'
  name?: string
  unreadCount?: number
  participants?: ChatParticipant[]
  lastMessage?: Message
  createdAt?: string
}

export interface ChatParticipant {
  userId: string
  role: 'member' | 'admin'
  user?: User
}

export interface Message {
  id: string
  chatId: string
  senderId: string
  text: string
  originalLanguage: string
  translations?: Record<string, string>
  translationEnhanced?: boolean
  deliveryStatus: 'pending' | 'sent' | 'delivered' | 'failed'
  replyToId?: string
  timestamp: string
  sender?: User
}

export interface WebSocketEvent<T = unknown> {
  type: string
  data: T
}

export interface TypingEvent {
  chatId: string
  userId: string
  isTyping: boolean
}

export interface MessageStatusEvent {
  chatId: string
  messageId: string
  status: 'delivered' | 'read'
}
