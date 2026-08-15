// Single source of truth for the Chorus domain model.
// Used by both the web frontend (frontend/) and the React Native app (mobile/).

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
  translationWordLimit?: number | null
}

// Resolved subscription view from GET /users/me/subscription. EffectivePlan is
// the plan after any grace window is applied.
export interface SubscriptionInfo {
  plan: Plan
  effectivePlan: EffectivePlan
  inGrace: boolean
  premiumSince?: string | null
  status?: string
  provider?: string
  subscriptionId?: string
  nextBillingDate?: string | null
  graceUntil?: string | null
  manageUrl?: string
  wordLimit?: number | null
  autoGrammar: boolean
}

export interface CheckoutRequest {
  plan: 'monthly' | 'annual'
}

export interface CheckoutResponse {
  approvalUrl: string
  plan: 'monthly' | 'annual'
}

// A premium user row from the admin console (GET /admin/premium/users).
export interface PremiumUserRow {
  id: string
  username: string
  displayName: string
  email: string
  plan: 'free' | 'premium'
  effectivePlan: EffectivePlan
  inGrace: boolean
  subscriptionId?: string
  subscriptionStatus?: string
  premiumSince?: string | null
  nextBillingDate?: string | null
  graceUntil?: string | null
  messagesSent: number
  createdAt: string
}

// Aggregated premium metrics (GET /admin/premium/analytics).
export interface PremiumAnalytics {
  totalPremiumUsers: number
  storedPremium: number
  inGrace: number
  monthlySubscriptions: number
  yearlySubscriptions: number
  newThisMonth: number
  churnedThisMonth: number
  projectedMRR: number
  revenueLastYear: number
  topUsersByUsage: PremiumUserRow[]
}

// One row of the plan audit trail (GET /admin/users/:id/plan-history).
export interface PlanChange {
  id: string
  userId: string
  actorId?: string
  fromPlan: string
  toPlan: string
  graceUntil?: string | null
  source: string
  reason: string
  createdAt: string
}

// Admin plan grant/revoke payload.
export interface GrantPlanRequest {
  plan: 'free' | 'premium'
  mode?: 'indefinite' | 'days' | 'until'
  days?: number
  until?: string
  reason?: string
  clearGraceInDays?: number
}

// Report & Block (REQ §8.2)
export interface Block {
  id: string
  blockerId: string
  blockedId: string
  reason?: string
  createdAt: string
  blocked?: User
}

export interface Report {
  id: string
  reporterId: string
  type: 'user' | 'message'
  reportedUserId: string
  messageId?: string
  chatId?: string
  reason: string
  status: 'open' | 'resolved' | 'dismissed'
  resolverId?: string
  resolutionNote?: string
  createdAt: string
  resolvedAt?: string
  reporter?: User
  reportedUser?: User
}

export interface ReportRequest {
  type: 'user' | 'message'
  reportedUserId?: string
  messageId?: string
  chatId?: string
  reason: string
}

export interface ReportStats {
  openReports: number
  userReports: number
  messageReports: number
  resolvedToday: number
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
  wordLimit?: number
  wordCount?: number
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
  inviteToken?: string
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

export type GrammarJobStatus = 'queued' | 'processing' | 'done' | 'failed'

// Payload of the client-initiated AI grammar analysis job. Submitted via
// POST /grammar/analyze-ai (returns the jobId immediately), delivered over the
// WebSocket "grammar_analysis" event, and pollable via GET /grammar/analyze/:jobId.
export interface GrammarJob {
  jobId: string
  messageId?: string
  chatId?: string
  status: GrammarJobStatus
  analysis?: any
  providerUsed?: string
  error?: string
}

export interface Language {
  code: string
  name: string
  nativeName: string
  flag?: string
}
