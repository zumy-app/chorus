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

// ---------------------------------------------------------------------------
// Learning engine types. Shared by web and mobile Learn surfaces. The backend
// is pair-aware: structured courses exist only for curated native->target pairs
// (launch: en -> es). Unseeded pairs get vocab_only behavior.
// ---------------------------------------------------------------------------

export type LearningSupportTier =
  | 'full_course'
  | 'beta_ai_assisted'
  | 'vocab_only'
  | 'disabled'

export interface LearningPairCapability {
  nativeLanguage: string
  targetLanguage: string
  supportTier: LearningSupportTier
  activeCourseId?: string
  placementEnabled: boolean
  roadmapEnabled: boolean
  scenariosEnabled: boolean
  srsEnabled: boolean
  miningEnabled: boolean
  grammarFeedbackEnabled: boolean
  qualityNotes: string
  createdAt: string
  updatedAt: string
}

export type CefrLevel = 'A1' | 'A2' | 'B1' | 'B2'

export interface UserLanguageProfile {
  userId: string
  nativeLanguage: string
  targetLanguage: string
  currentCefrLevel: CefrLevel
  readinessScore: number
  activeCourseId?: string
  activeUnitId?: string
  placementStatus: 'not_started' | 'in_progress' | 'completed' | 'skipped'
  primaryGoal: string
  dailyGoalItems: number
  miningEnabled: boolean
  nudgesEnabled: boolean
  scenarioHintsEnabled: boolean
  createdAt: string
  updatedAt: string
}

export interface DailyGoalSummary {
  targetItems: number
  completedItems: number
  percent: number
}

export interface StreakSummary {
  days: number
  atRisk: boolean
  canRecover: boolean
}

export interface FluencySummary {
  readinessScore: number
  readinessPercent: number
  label: string
  componentScores: Record<string, number>
}

export interface VocabularySummary {
  total: number
  dueToday: number
  mastered: number
  newFromChats: number
}

export interface GrammarSummary {
  weakestPointTitle: string
  confidencePct: number
  dueToday: number
}

export interface ScenarioSummary {
  nextScenarioId?: string
  title?: string
  progressPct: number
  hasNewWords: boolean
}

export interface RecommendedActivity {
  id: string
  type: string
  title: string
  description: string
  priority: string
  estimatedMinutes: number
  action: string
}

export interface DailyActivityPoint {
  date: string
  xp: number
  itemsCompleted: number
}

export interface LessonSummary {
  id: string
  unitId: string
  ordinal: number
  slug: string
  type: string
  title: string
  objective: string
  estimatedMinutes: number
  status: string
}

export interface UnitProgressSummary {
  id: string
  courseId: string
  cefrLevel: CefrLevel
  ordinal: number
  slug: string
  title: string
  canDoStatement: string
  description: string
  estimatedMinutes: number
  checkpointRequired: boolean
  status: string
  progressPct: number
  competencyScore: number
  lessonsCompleted: number
  checkpointScore?: number
  startedAt?: string
  completedAt?: string
  lessons?: LessonSummary[]
}

export interface LearningDashboard {
  capability: LearningPairCapability
  profile: UserLanguageProfile
  dailyGoal: DailyGoalSummary
  streak: StreakSummary
  fluency: FluencySummary
  currentUnit?: UnitProgressSummary
  nextLesson?: LessonSummary
  vocabulary: VocabularySummary
  grammar: GrammarSummary
  scenario: ScenarioSummary
  recommendedActivities: RecommendedActivity[]
  weeklyActivity: DailyActivityPoint[]
}

export interface LearningPath {
  capability: LearningPairCapability
  profile: UserLanguageProfile
  units: UnitProgressSummary[]
}

export interface LearningProfileUpdateRequest {
  nativeLanguage?: string
  targetLanguage: string
  primaryGoal?: string
  dailyGoalItems?: number
  miningEnabled?: boolean
  nudgesEnabled?: boolean
  scenarioHintsEnabled?: boolean
}

// ---------------------------------------------------------------------------
// Practice sessions / SRS
// ---------------------------------------------------------------------------

export interface LearningSession {
  id: string
  userId: string
  targetLanguage: string
  mode: string
  status: string
  sourceUnitId?: string
  sourceLessonId?: string
  plannedItemCount: number
  completedItemCount: number
  score: number
  xpAwarded: number
  startedAt: string
  completedAt?: string
  progressPct: number
}

export interface StartSessionRequest {
  targetLanguage: string
  nativeLanguage?: string
  mode?: string
  source?: string
}

export interface SessionPrompt {
  text?: string
  source?: string
  translation?: string
  choices?: string[]
  term?: string
  tone?: string
  grammarHint?: string
}

export interface SessionQuestion {
  id: string
  itemType: string
  activityType: string
  promptType: string
  prompt: SessionPrompt
}

export interface SessionAnswerRequest {
  text?: string
  choice?: string
}

export interface SessionFeedback {
  message: string
  correctAnswer?: string
  grammarPointId?: string
  masteryState?: string
}

export interface AnswerSessionItemResponse {
  correct: boolean
  quality: number
  feedback: SessionFeedback
  nextItem?: SessionQuestion
}

export interface StartSessionResponse {
  session: LearningSession
  items: SessionQuestion[]
}

// ---------------------------------------------------------------------------
// Mined vocabulary
// ---------------------------------------------------------------------------

export interface MinedItem {
  id: string
  userId: string
  jobId?: string
  chatId?: string
  messageId?: string
  sourceType: string
  surfaceText: string
  lemma: string
  normalizedText: string
  language: string
  partOfSpeech: string
  translation: string
  definition: string
  contextSentence: string
  cefrLevel?: string
  confidence: number
  teachabilityScore: number
  isChunk: boolean
  isProperNoun: boolean
  grammarTags?: string[]
  curriculumUnitId?: string
  routeStatus: string
  status: string
  routeReason?: string
  createdAt: string
}

// ---------------------------------------------------------------------------
// Lessons
// ---------------------------------------------------------------------------

export interface CurriculumStep {
  id: string
  lessonId: string
  ordinal: number
  type: string
  prompt: any
  answerKey?: any
  contentRefs?: any
}

export interface LessonAttempt {
  id: string
  userId: string
  lessonId: string
  targetLanguage: string
  status: string
  score: number
  correctCount: number
  totalCount: number
  startedAt: string
}

export interface LessonStartResponse {
  attempt: LessonAttempt
  steps: CurriculumStep[]
}

export interface LessonStepResult {
  id: string
  stepId: string
  userAnswer: any
  correct: boolean
  score: number
  feedback: any
}

// ---------------------------------------------------------------------------
// Placement
// ---------------------------------------------------------------------------

export interface PlacementQuestion {
  id: string
  ref: string
  itemType: string
  cefrLevel: CefrLevel
  prompt: any
  choices?: string[]
}

export interface PlacementMainStartResponse {
  attemptId: string
  status: string
  question: PlacementQuestion
  totalQuestions: number
}

export interface PlacementResult {
  attemptId: string
  estimatedCefr: CefrLevel
  readinessScore: number
  activeUnitId: string
  skippedUnits?: string[]
}

// ---------------------------------------------------------------------------
// Scenarios
// ---------------------------------------------------------------------------

export interface ScenarioChunk {
  text: string
  translation: string
}

export interface ScenarioPhase {
  id: string
  scenarioId: string
  ordinal: number
  title: string
  learnerGoal: string
  requiredIntents: string[]
  chunkBank: ScenarioChunk[]
}

export interface ScenarioScript {
  id: string
  courseId: string
  unitId?: string
  slug: string
  title: string
  domain: string
  cefrLevel: CefrLevel
  canDoStatement: string
  aiRoleName: string
  aiRoleDescription: string
  openingLine: string
  maxTurns: number
  estimatedMinutes: number
  completionCriteria?: any
  phases?: ScenarioPhase[]
  metadata?: any
}

export interface ScenarioTurn {
  id: string
  runId: string
  ordinal: number
  speaker: 'user' | 'ai' | 'system'
  text: string
  translation: string
  phaseOrdinal: number
  evaluation?: any
  createdAt: string
}

export interface ScenarioRun {
  id: string
  userId: string
  scenarioId: string
  targetLanguage: string
  nativeLanguage: string
  status: string
  scaffoldLevel: string
  currentPhaseOrdinal: number
  coveredIntents?: string[]
  score: number
  xpAwarded: number
  startedAt: string
  currentPhase?: ScenarioPhase
  suggestedChunks?: ScenarioChunk[]
  turns?: ScenarioTurn[]
}

export interface ScenarioError {
  span?: string
  correction?: string
  grammarTag?: string
  explanation?: string
}

export interface ScenarioSummaryResult {
  score: number
  xpAwarded: number
  minutes: number
  vocabularyAdded: number
}

export interface ScenarioAIReply {
  aiMessage: string
  translation: string
  phaseComplete: boolean
  nextPhaseOrdinal?: number
  coveredIntents?: string[]
  errors?: ScenarioError[]
  score: number
  runCompleted: boolean
  summary?: ScenarioSummaryResult
  suggestedChunks?: ScenarioChunk[]
}

export interface ScenarioStartResponse {
  run: ScenarioRun
  aiResponse: ScenarioAIReply
}

export interface RealTalkPrompt {
  id: string
  category: string
  text: string
  sourcePrompt?: boolean
}

export interface StreakRecoverResult {
  newStreak?: number
  recovered: boolean
}

// A fully-elevated vocabulary card (the SRS card model) returned by accept/ignore
// and practice endpoints.
export interface VocabularyCard {
  id: string
  userId: string
  term: string
  language: string
  translation: string
  definition: string
  lemma: string
  normalizedTerm: string
  partOfSpeech: string
  isChunk: boolean
  sourceType: string
  sourceMessageId?: string
  cefrLevel?: string
  curriculumUnitId?: string
  routeStatus: string
  masteryStage: number
  masteryState: string
  easeFactor: number
  lapses: number
  stageSuccessCount: number
  productionSuccessCount: number
  spontaneousUseCount: number
  teachabilityScore: number
  confidence: number
  reviewCount: number
  correctCount: number
  intervalDays: number
  nextReview: string
  contextSentence?: string
  contextMessageId?: string
  contextChatId?: string
  createdAt: string
}

// Export alias used by API methods returning placement.
export type { PlacementMainStartResponse as StartPlacementResponse }
