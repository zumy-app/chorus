export interface User {
  id: string
  username: string
  email: string
  displayName: string
  nativeLanguage: string
  targetLanguages: string[]
  role: 'member' | 'moderator' | 'admin'
  createdAt: string
  lastActiveAt: string
  suspendedAt?: string | null
  deletedAt?: string | null
  // Billing. plan mirrors the stored value; entitlements (effective plan,
  // grace deadline, quotas) are resolved server-side via /users/me/entitlements.
  plan?: 'free' | 'premium'
  planGraceUntil?: string | null
}

// Billing plans and resolved entitlements returned by /users/me/entitlements.
export type Plan = 'free' | 'premium'
export type EffectivePlan = 'free' | 'premium' | 'unlimited'

export interface Entitlements {
  plan: Plan
  planGraceUntil?: string | null
  effectivePlan: EffectivePlan
  selfHost: boolean
  showAds: boolean
  features: FeatureFlags
  limits: PlanLimits
}

// Premium feature tiers resolved server-side.
export interface FeatureFlags {
  autoGrammar: boolean
  fasterResponses: boolean
  translationCharLimit?: number | null
}

// Usage quotas for a resolved entitlement set. A null value means unlimited.
export interface PlanLimits {
  dailyLLMTranslations?: number | null
  dailyLLMGrammarAnalyses?: number | null
  dailyLLMCorrections?: number | null
  dailyVoiceMessages?: number | null
  vocabularyItems?: number | null
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

// Translation was intentionally not performed (e.g. free plan char limit).
export interface TranslationBlocked {
  messageId: string
  chatId: string
  charLimit?: number
  reason?: string
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
  inviteToken: string
}

export interface WaitlistRequest {
  email: string
  spokenLanguages: string[]
  targetLanguages: string[]
  reasons: string[]
  comments?: string
}

export interface WaitlistEntry {
  id: string
  email: string
  spokenLanguages: string[]
  targetLanguages: string[]
  reasons: string[]
  comments?: string
  status: 'pending' | 'approved' | 'declined'
  queuePosition: number
  createdAt: string
  approvedAt?: string
}

export interface EmailOutboxEntry {
  id: string
  recipient: string
  subject: string
  status: 'pending' | 'sent' | 'failed'
  attempts: number
  lastError?: string
  createdAt: string
  nextAttemptAt: string
  sentAt?: string
}

export interface AdminStats {
  totalUsers: number
  moderators: number
  admins: number
  suspendedUsers: number
  waitlistPending: number
  waitlistApproved: number
  waitlistDeclined: number
  emailsPending: number
  emailsSent: number
  emailsFailed: number
  translationsPending: number
  translationsCompleted: number
  translationsFailed: number
}

export interface TranslationJob {
  id: string
  messageId: string
  chatId: string
  text: string
  sourceLang: string
  targetLang: string
  status: 'pending' | 'processing' | 'done' | 'failed'
  result: string
  attempts: number
  lastError?: string
  createdAt: string
  nextAttemptAt?: string
  completedAt?: string
}

export interface AdminStatus {
  role: 'member' | 'moderator' | 'admin'
  isAdmin: boolean
  isModerator: boolean
}

export interface ProviderHealth {
  name: string
  ready: boolean
  reason?: string
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
