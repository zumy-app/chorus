// Single source of truth for the Chorus domain model.
// Used by both the web frontend (frontend/) and the React Native app (mobile/).

export type PrivacyVisibility = 'everyone' | 'contacts' | 'nobody'

export interface UserPrivacySettings {
  lastSeenVisibility: PrivacyVisibility
  profilePhotoVisibility: PrivacyVisibility
  contactsVisibility: PrivacyVisibility
}

export interface UserSettings {
  userId: string
  grammarEnabled: boolean
  vocabularyEnabled: boolean
  difficultyLevel: string
  transcriptRecording: boolean
  messageRetentionDays: number
  translationEnabled: boolean
  grammarAuto: boolean
  highlightsEnabled: boolean
  lastSeenVisibility: PrivacyVisibility
  profilePhotoVisibility: PrivacyVisibility
  contactsVisibility: PrivacyVisibility
  updatedAt: string
}

export interface User {
  id: string
  username: string
  email: string
  isBlocked?: boolean
  blockedBy?: boolean
  // Onboarding (REQ 2.1): structured name whose displayName ("first last") is
  // composed server-side but still overridable. Optional for accounts created
  // before onboarding was introduced.
  firstName?: string
  lastName?: string
  displayName: string
  // REQ 2.2 / FR-20: deterministic initials avatar. avatarColor is derived
  // server-side from the user's stable ID (identical across clients); avatarUrl
  // is the reserved upload path and stays unset until attachment infra exists,
  // in which case clients render initials + avatarColor instead of an image.
  avatarColor?: string
  avatarUrl?: string | null
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
  phone?: string | null
  phoneVerified?: boolean
  phoneVerifiedAt?: string | null
  twoFactorEnabled?: boolean
}

export interface PhoneStatus {
  phone?: string | null
  phoneMasked?: string
  phoneVerified: boolean
  twoFactorEnabled: boolean
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
export interface BlockStatus {
  blocked: boolean
  blockedBy: boolean
  mutual: boolean
}

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
  // Task 6.4 (archive & mute): per-user conversation state, derived server-side
  // from the chat_preferences table. isArchived hides the chat from the main
  // list; isMuted silences its notifications (optionally until mutedUntil).
  isArchived?: boolean
  isMuted?: boolean
  mutedUntil?: string
}

// Task 6.4: one per-user, per-chat conversation preference row. Strictly
// user-scoped: one user archiving/muting a chat does not affect co-participants.
export interface ChatPreference {
  userId: string
  chatId: string
  archivedAt?: string
  isMuted: boolean
  mutedUntil?: string
  updatedAt: string
}

export interface ChatParticipant {
  chatId: string
  userId: string
  role: 'member' | 'admin'
  joinedAt: string
  lastReadMessageId?: string
  user?: User
}

export interface MessageReceipt {
  messageId: string
  userId: string
  chatId: string
  deliveredAt?: string
  readAt?: string
  status: 'sent' | 'delivered' | 'read'
}

export interface ReceiptEvent {
  chatId: string
  messageId: string
  userId: string
  status: 'delivered' | 'read'
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
  forwarded?: boolean
  forwardedFromMessageId?: string
  forwardedFromChatId?: string
  forwardedFromSenderId?: string
  deletedAt?: string
  media?: MediaAttachment[]
  receipts?: MessageReceipt[]
}

// One chat-scoped pin (task 6.2): the pinned message plus who pinned it and when.
export interface PinnedMessage {
  message: Message
  pinnedBy: string
  pinnedAt: string
}

// A media attachment (task 6.3/6.5/6.7): image, video, audio, document, link or
// location attached to a message. Returned in universal search results, by
// /media/search, on the send response, and in chat history + gallery reads.
export interface MediaAttachment {
  id: string
  messageId: string
  chatId: string
  type: 'image' | 'video' | 'audio' | 'document' | 'link' | 'location'
  fileName: string
  fileSize: number
  mimeType: string
  url: string
  thumbnailUrl?: string
  createdAt: string
  // Location sharing (task 6.7): the map pin's coordinates + optional label.
  // Present only when type === 'location'; the URL above holds a map link and
  // the client renders the pin from these.
  latitude?: number
  longitude?: number
  locationName?: string
}

// Payload for sharing a location pin into a chat (task 6.7). Latitude/Longitude
// are required and bounded to world coordinates; label is an optional place name.
export interface SendLocationRequest {
  latitude: number
  longitude: number
  label?: string
  replyToId?: string
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
  firstName?: string
  lastName?: string
  displayName: string
  nativeLanguage: string
  targetLanguages: string[]
  inviteToken?: string
}

export interface OnboardRequest {
  firstName: string
  lastName: string
  // Optional override; when omitted the backend composes "first last".
  displayName?: string
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

// Contacts & Invites epic (REQ 2.4 / FR-22-23). Privacy-preserving: the scan
// uploads only SHA-256 hashes of normalized identifiers; raw contacts never
// leave the device.

export type ContactInviteChannel = 'email' | 'sms' | 'whatsapp'

export interface ContactMatch {
  userId: string
  username: string
  displayName: string
  email: string
  emailHash: string
  nativeLanguage: string
  targetLanguages: string[]
  isBlocked?: boolean
  blockedBy?: boolean
}

export interface ContactScanRequest {
  hashes: string[]
}

export interface ContactInviteRequest {
  channel: ContactInviteChannel
  contact: {
    name?: string
    email?: string
    phone?: string
  }
}

export type ContactInviteStatus = 'pending' | 'sent' | 'redeemed' | 'expired'

export interface ContactInvite {
  id: string
  inviterId: string
  channel: ContactInviteChannel
  recipient: string
  name?: string
  token?: string
  link?: string
  status: ContactInviteStatus
  expiresAt: string
  createdAt: string
  redeemedAt?: string
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

export interface SearchRequest {
  query: string
  chatIds?: string[]
  language?: string
  mediaType?: string
  limit?: number
  offset?: number
}

export interface SearchResult {
  messages: Message[]
  media: MediaAttachment[]
  total: number
  mediaTotal: number
  hasMore: boolean
}

export interface MediaSearchResult {
  media: MediaAttachment[]
  total: number
  hasMore: boolean
}

export interface ChatSearchResult {
  data: Chat[]
}

export interface ContactSearchResult {
  data: User[]
}

export interface WebSocketMessage {
  type: string
  data: any
}

// Presence status returned by GET /presence/:userId and POST /presence/batch,
// and pushed over the WebSocket as a "user_presence" / "presence_update" event.
export interface PresenceStatus {
  userId: string
  status: 'online' | 'offline' | 'away'
  lastSeen: string
  deviceType?: string
}

// Payload of the client-initiated typing indicator (WebSocket "user_typing").
export interface TypingEvent {
  chatId: string
  userId: string
  isTyping: boolean
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
  placementStatus: 'not_started' | 'in_progress' | 'completed' | 'skipped' | 'self_selected'
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

export interface MonthlyActivityPoint {
  month: string
  wordsLearned: number
  sentencesUnderstood: number
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
  monthlyActivity: MonthlyActivityPoint[]
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
export interface CallSession {
  id: string
  chatId: string
  participants: string[]
  type: 'audio' | 'video'
  status: 'active' | 'ended'
  startedAt: string
  endedAt?: string
}

export interface TranscriptSegment {
  speakerId: string
  startTime: number
  endTime: number
  originalText: string
  originalLanguage: string
  translations: Record<string, string>
  confidence: number
}

export interface CallTranscript {
  id: string
  callId: string
  segments: TranscriptSegment[]
  createdAt: string
}

export interface WebRTCOffer {
  callId: string
  sdp: string
  type: string
  iceServers: { urls: string[]; username?: string; credential?: string }[]
}

export interface TeacherCertificate {
  id?: string
  type: 'teaching_degree' | 'language_certificate' | 'other'
  issuer: string
  year: number
  fileUrl: string
  verified?: boolean
}

export interface TeacherApplication {
  id: string
  userId: string
  bio: string
  languages: string[]
  expertise?: string
  rateCents: number
  videoUrl: string
  status: string
  createdAt: string
  updatedAt: string
  certificates?: TeacherCertificate[]
}

export interface TeacherApplyRequest {
  bio: string
  languages: string[]
  expertise?: string
  rateCents: number
  videoUrl: string
  certificates?: TeacherCertificate[]
}

export interface TutorProfile {
  id: string
  userId: string
  displayName: string
  bio: string
  languages: string[]
  expertise?: string
  rateCents: number
  videoUrl: string
  status: string
  verified: boolean
  ratingAvg: number
  ratingCount: number
  avatarColor?: string
  avatarUrl?: string | null
  createdAt: string
  updatedAt: string
  certificates?: TeacherCertificate[]
}

export interface TutorBrowseResult {
  tutors: TutorProfile[]
  total: number
  hasMore: boolean
}

export interface TutorReview {
  id: string
  teacherUserId: string
  studentUserId: string
  rating: number
  comment: string
  createdAt: string
  studentName?: string
}

export interface TrialCredit {
  userId: string
  credits: number
  updatedAt: string
  grantedAt: string
}

export interface TutorAvailability {
  id: string
  teacherUserId: string
  startTime: string
  endTime: string
  createdAt: string
}

export interface TutorBooking {
  id: string
  teacherUserId: string
  studentUserId: string
  startTime: string
  endTime: string
  status: 'pending' | 'confirmed' | 'cancelled' | 'completed'
  isTrial: boolean
  note?: string
  reviewNotes?: string
  confirmedAt?: string
  completedAt?: string
  createdAt: string
  updatedAt: string
  teacherName?: string
  studentName?: string
}

export interface SrsPushCard {
  term: string
  translation?: string
  definition?: string
  contextSentence?: string
  cefrLevel?: 'A1' | 'A2' | 'B1' | 'B2'
}

export interface TeacherSrsPushRequest {
  studentId: string
  bookingId?: string
  language: string
  note?: string
  cards: SrsPushCard[]
}

export interface TeacherSrsPushItem {
  id: string
  pushId: string
  vocabularyId: string
  term: string
  translation?: string
  definition?: string
  createdAt: string
}

export interface TeacherSrsPush {
  id: string
  teacherUserId: string
  studentUserId: string
  bookingId?: string
  language: string
  note?: string
  itemCount: number
  createdAt: string
  teacherName?: string
  studentName?: string
  items?: TeacherSrsPushItem[]
}

export interface PayoutMethod {
  id: string
  teacherUserId: string
  type: 'paypal' | 'bank'
  label: string
  details: string
  isDefault: boolean
  createdAt: string
}

export interface PayoutRecord {
  id: string
  teacherUserId: string
  amountCents: number
  feeCents: number
  grossCents: number
  methodId?: string
  destination: string
  status: 'pending' | 'processing' | 'completed' | 'failed'
  reference: string
  paypalBatchId?: string
  createdAt: string
  completedAt?: string
}

export interface PayoutOverview {
  availableCents: number
  pendingCents: number
  pendingGrossCents: number
  totalGrossCents: number
  totalNetCents: number
  lifetimeGross: number
  lifetimeNet: number
  totalPaidCents: number
  platformFeePct: number
  completedCount: number
  pendingCount: number
  cancelledCount: number
  totalBookings: number
  nextPayoutDate?: string
  hoursTaught: number
  activeStudents: number
  recentTransactions: Array<{
    studentName: string
    initials: string
    minutes: number
    amountCents: number
    grossCents: number
    feeCents: number
    date: string
    status: string
  }>
}

export interface CaptionReview {
  id: string
  callId: string
  segmentIndex: number
  originalText: string
  originalLanguage: string
  translatedText: string
  targetLanguage: string
  reviewerId: string
  rating: number
  correctedText?: string
  feedback?: string
  reviewerName?: string
  createdAt: string
}
export interface CaptionReviewQueueItem {
  callId: string
  segmentIndex: number
  originalText: string
  originalLanguage: string
  translations: Record<string,string>
  speakerId: string
  confidence: number
  reviewCount: number
  avgRating?: number
}
export interface CaptionQualityStats {
  totalCaptions: number
  reviewedCount: number
  avgRating: number
  ratingCounts: Record<number,number>
  pendingCount: number
}
export type { PlacementMainStartResponse as StartPlacementResponse }
