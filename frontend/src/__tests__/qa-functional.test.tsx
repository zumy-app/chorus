import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import fs from 'fs'
import path from 'path'

let mockNavigate: ReturnType<typeof vi.fn> = vi.fn()
let mockParams: Record<string, any> = {}
let mockSearch = ''

vi.mock('react-router-dom', async () => {
  const actual: any = await vi.importActual('react-router-dom')
  return {
    ...actual,
    useNavigate: () => mockNavigate,
    useParams: () => mockParams,
    useLocation: () => ({ search: mockSearch, pathname: '/learn/session', hash: '', state: null, key: 'test' }),
  }
})

vi.mock('../store', () => ({
  useStore: (selector: any) => selector({ user: { id: 'u1', nativeLanguage: 'en', targetLanguages: ['es'] } }),
}))

vi.mock('../components/AppHeader', () => ({ default: () => <div data-testid="app-header" /> }))
vi.mock('../components/BottomNav', () => ({ default: () => <div data-testid="bottom-nav" /> }))
vi.mock('react-i18next', () => ({ useTranslation: () => ({ t: (k: string, opts?: any) => (opts?.defaultValue ?? k) }) }))

vi.mock('../services/api', () => ({
  learningAPI: {
    getDashboard: vi.fn(),
    getPath: vi.fn(),
    getScenarios: vi.fn(),
    getScenario: vi.fn(),
    startScenario: vi.fn(),
    sendScenarioMessage: vi.fn(),
    requestScenarioHint: vi.fn(),
    completeScenario: vi.fn(),
    getMinedItems: vi.fn(),
    acceptMinedItem: vi.fn(),
    ignoreMinedItem: vi.fn(),
    startSession: vi.fn(),
    answerSessionItem: vi.fn(),
    completeSession: vi.fn(),
    recoverStreak: vi.fn(),
    skipPlacement: vi.fn(),
    getRealTalkPrompts: vi.fn(),
  },
  teacherAPI: {
    browse: vi.fn(),
    getProfile: vi.fn(),
    getReviews: vi.fn(),
    getAvailability: vi.fn(),
    book: vi.fn(),
    getTrialCredits: vi.fn(),
  },
  payoutsAPI: {
    overview: vi.fn(),
    methods: vi.fn(),
    history: vi.fn(),
    addMethod: vi.fn(),
    removeMethod: vi.fn(),
    withdraw: vi.fn(),
  },
  authAPI: { getMe: vi.fn() },
}))

import { learningAPI } from '../services/api'
import { teacherAPI } from '../services/api'
import { payoutsAPI } from '../services/api'

import Scenarios from '../pages/Scenarios'
import ScenarioRoleplay from '../pages/ScenarioRoleplay'
import Learn from '../pages/Learn'
import LessonSession from '../pages/LessonSession'
import VocabularyReview from '../pages/VocabularyReview'
import BrowseTutors from '../pages/BrowseTutors'
import TutorProfile from '../pages/TutorProfile'
import TrialCredits from '../pages/TrialCredits'
import TeacherDashboard from '../pages/TeacherDashboard'
import Payouts from '../pages/Payouts'
import LearningRoadmap from '../pages/LearningRoadmap'
import RealTalkHub from '../pages/RealTalkHub'
import StreakRecovery from '../pages/StreakRecovery'

const spanishScenarios = [
  { id: 'es-cafe', title: 'Pedir café en una cafetería', slug: 'pedir-cafe', domain: 'food_drink', cefrLevel: 'A1', canDoStatement: 'Pedir una bebida', estimatedMinutes: 5, metadata: { completed: false } },
  { id: 'es-mercado', title: 'Comprar en el mercado', slug: 'mercado', domain: 'shopping', cefrLevel: 'A2', canDoStatement: 'Comprar frutas', estimatedMinutes: 6, metadata: { completed: false } },
]

const dashboardEs: any = {
  capability: { supportTier: 'full_course', placementEnabled: true, scenariosEnabled: true, roadmapEnabled: true },
  profile: { placementStatus: 'completed', currentCefrLevel: 'A1', targetLanguage: 'es', nativeLanguage: 'en' },
  dailyGoal: { targetItems: 10, completedItems: 6, percent: 60 },
  streak: { days: 7, atRisk: false, canRecover: false },
  fluency: { readinessScore: 350, readinessPercent: 35, label: 'Construyendo A1' },
  currentUnit: { id: 'u1', title: 'Saludos', cefrLevel: 'A1', progressPct: 40 },
  nextLesson: { id: 'l1', title: 'Saludos básicos', type: 'vocabulary', status: 'available' },
  vocabulary: { total: 30, dueToday: 5, mastered: 10, newFromChats: 3 },
  grammar: { weakestPointTitle: 'ser vs estar', confidencePct: 55, dueToday: 2 },
  monthlyActivity: [{ month: '2026-07', wordsLearned: 15, sentencesUnderstood: 40 }, { month: '2026-08', wordsLearned: 22, sentencesUnderstood: 55 }],
  recommendedActivities: [{ id: 'vocabulary', type: 'vocabulary', title: 'Repaso', description: 'Repaso', priority: 'high', estimatedMinutes: 3, action: 'start_session' }],
  weeklyActivity: [{ date: '2026-08-25', xp: 20, itemsCompleted: 2 }],
}

beforeEach(() => {
  vi.clearAllMocks()
  mockNavigate = vi.fn()
  mockParams = {}
  mockSearch = ''
  vi.mocked(learningAPI.getDashboard).mockResolvedValue(dashboardEs)
  vi.mocked(learningAPI.getPath).mockResolvedValue({ capability: { supportTier: 'full_course' }, profile: { placementStatus: 'completed' }, units: [{ id: 'u1', title: 'Saludos', cefrLevel: 'A1', ordinal: 1, slug: 'a1-saludos', courseId: 'c1', status: 'available', progressPct: 30, checkpointRequired: false, estimatedMinutes: 30, description: '', canDoStatement: 'Saludar', competencyScore: 0, lessonsCompleted: 1, lessons: [] }] } as any)
  vi.mocked(learningAPI.getScenarios).mockResolvedValue(spanishScenarios as any)
  vi.mocked(learningAPI.getScenario).mockResolvedValue({ id: 'es-cafe', title: 'Pedir café en una cafetería', cefrLevel: 'A1', canDoStatement: 'Pedir una bebida', aiRoleName: 'Sparky', estimatedMinutes: 5, phases: [{ ordinal: 1, title: 'Greeting', learnerGoal: 'Greet the barista', requiredIntents: ['greet'], chunkBank: [] }] } as any)
  vi.mocked(learningAPI.startScenario).mockResolvedValue({ run: { id: 'run1', currentPhaseOrdinal: 1, scaffoldLevel: 'guided', currentPhase: { ordinal: 1, title: 'Greeting', learnerGoal: 'Greet the barista' } }, aiResponse: { aiMessage: 'Hola, ¿qué te gustaría pedir?', translation: 'Hi, what would you like to order?', suggestedChunks: [{ text: 'Quisiera un café', translation: 'I would like a coffee' }] } } as any)
  vi.mocked(learningAPI.sendScenarioMessage).mockResolvedValue({ aiMessage: 'Perfecto, ¿algo más?', translation: 'Perfect, anything else?', suggestedChunks: [], runCompleted: false, phaseComplete: false, errors: [] } as any)
  vi.mocked(learningAPI.requestScenarioHint).mockResolvedValue([{ text: '¿Cuánto cuesta?', translation: 'How much does it cost?' }] as any)
  vi.mocked(learningAPI.completeScenario).mockResolvedValue({ summary: { score: 700, xpAwarded: 50, vocabularyAdded: 1 } } as any)
  vi.mocked(learningAPI.getMinedItems).mockResolvedValue([{ id: 'm1', surfaceText: 'desayuno', translation: 'breakfast', contextSentence: 'Quiero desayuno.', routeStatus: 'bonus', status: 'candidate' }] as any)
  vi.mocked(learningAPI.acceptMinedItem).mockResolvedValue({ id: 'card1' } as any)
  vi.mocked(learningAPI.ignoreMinedItem).mockResolvedValue({ ok: true } as any)
  vi.mocked((learningAPI as any).getRealTalkPrompts).mockResolvedValue([{ id: 'p1', text: 'Describe your weekend', category: 'Icebreakers' }] as any)
  vi.mocked(learningAPI.startSession).mockResolvedValue({
    session: { id: 'sess1', plannedItemCount: 2, mode: 'daily', status: 'in_progress' },
    items: [
      { id: 'i1', itemType: 'vocabulary', activityType: 'cued_recall', promptType: 'cloze', prompt: { text: 'Yo ____ cansado.', choices: ['estoy', 'soy'], source: '' } },
      { id: 'i2', itemType: 'vocabulary', activityType: 'free_recall', promptType: 'translate', prompt: { text: 'Translate: good morning', source: 'good morning' } },
    ],
  } as any)
  vi.mocked(learningAPI.answerSessionItem).mockResolvedValue({ correct: true, quality: 5, feedback: { message: '¡Excelente!', correctAnswer: 'estoy' }, nextItem: null } as any)
  vi.mocked(learningAPI.completeSession).mockResolvedValue({ id: 'sess1' } as any)
  vi.mocked(learningAPI.recoverStreak).mockResolvedValue({ recovered: true } as any)
  vi.mocked(teacherAPI.browse).mockResolvedValue({ tutors: [{ userId: 't1', displayName: 'María García', languages: ['es','en'], ratingAvg: 4.9, rateCents: 2000, verified: true }], total: 1, hasMore: false } as any)
  vi.mocked(teacherAPI.getProfile).mockResolvedValue({ userId: 't1', displayName: 'María García', bio: 'Native Spanish tutor', languages: ['es'], rateCents: 2000, videoUrl: 'https://example.com/v.mp4', status: 'approved', verified: true, ratingAvg: 4.9, ratingCount: 12 } as any)
  vi.mocked(teacherAPI.getReviews).mockResolvedValue({ reviews: [], total: 0, hasMore: false } as any)
  vi.mocked(teacherAPI.getAvailability).mockResolvedValue([] as any)
  vi.mocked(teacherAPI.book).mockResolvedValue({ id: 'b1', status: 'pending' } as any)
  vi.mocked(payoutsAPI.overview).mockResolvedValue({ availableCents: 5000, pendingCents: 1000, lifetimeGross: 6000, lifetimeNet: 5400, platformFeePct: 10, activeStudents: 3 } as any)
  vi.mocked(payoutsAPI.methods).mockResolvedValue([] as any)
  vi.mocked(payoutsAPI.history).mockResolvedValue({ payouts: [], total: 0, hasMore: false } as any)
  global.fetch = vi.fn(async (url: any) => {
    const s = String(url)
    if (s.includes('/teachers/dashboard')) return { ok: true, json: async () => ({ dashboard: { checklist: { completionPct: 80, hasBio: true, hasVideo: true, hasCertificate: false, percent: 80 }, earnings: { totalGrossCents: 6000, pendingCents: 1000, totalNetCents: 5400, platformFeePct: 10 }, students: [{ displayName: 'Student' }], availability: [] } }) } as any
    if (s.includes('/trial-credits/dashboard')) return { ok: true, json: async () => ({ dashboard: { credits: 1, nextGrantAt: new Date().toISOString(), history: [] }, credits: 1 }) } as any
    if (s.includes('/teachers/trial-credits')) return { ok: true, json: async () => ({ dashboard: { credits: 1 } }) } as any
    return { ok: true, json: async () => ({}) } as any
  }) as any
})

describe('QA Spanish scenarios — web', () => {
  it('lists Spanish scenarios (es domain) with titles', async () => {
    render(<MemoryRouter><Scenarios /></MemoryRouter>)
    await waitFor(() => expect(screen.getByText('Pedir café en una cafetería')).toBeTruthy())
    expect(screen.getByText('Comprar en el mercado')).toBeTruthy()
    expect(learningAPI.getScenarios).toHaveBeenCalled()
    const args = (learningAPI.getScenarios as any).mock.calls[0]
    expect(args[0]).toBe('es')
  })
  it('shows loading then scenarios', async () => {
    render(<MemoryRouter><Scenarios /></MemoryRouter>)
    expect(screen.getByText('Loading scenarios...')).toBeTruthy()
    await waitFor(() => screen.getByText('Pedir café en una cafetería'))
  })
  it('starts Spanish scenario and shows opening line + translation', async () => {
    mockParams = { scenarioId: 'es-cafe' }
    render(<MemoryRouter><ScenarioRoleplay /></MemoryRouter>)
    await waitFor(() => expect(screen.getByText('Hola, ¿qué te gustaría pedir?')).toBeTruthy())
    expect(screen.getByText('Hi, what would you like to order?')).toBeTruthy()
  })
  it('shows suggested Spanish chunk for scaffolding', async () => {
    mockParams = { scenarioId: 'es-cafe' }
    render(<MemoryRouter><ScenarioRoleplay /></MemoryRouter>)
    await waitFor(() => screen.getByText('Hola, ¿qué te gustaría pedir?'))
    fireEvent.click(screen.getByText('Show suggestions'))
    expect(screen.getByText('Quisiera un café')).toBeTruthy()
  })
  it('sends Spanish user message and receives AI reply with translation', async () => {
    mockParams = { scenarioId: 'es-cafe' }
    render(<MemoryRouter><ScenarioRoleplay /></MemoryRouter>)
    await waitFor(() => screen.getByText('Hola, ¿qué te gustaría pedir?'))
    const input = screen.getByPlaceholderText('Escribe en español...')
    fireEvent.change(input, { target: { value: 'Quisiera un café por favor' } })
    fireEvent.click(screen.getByLabelText('Send'))
    await waitFor(() => expect(screen.getByText('Perfecto, ¿algo más?')).toBeTruthy())
    expect(screen.getByText('Perfect, anything else?')).toBeTruthy()
    expect(learningAPI.sendScenarioMessage).toHaveBeenCalledWith('run1', 'Quisiera un café por favor')
  })
  it('requests Spanish hint via lightbulb', async () => {
    mockParams = { scenarioId: 'es-cafe' }
    render(<MemoryRouter><ScenarioRoleplay /></MemoryRouter>)
    await waitFor(() => screen.getByText('Hola, ¿qué te gustaría pedir?'))
    fireEvent.click(screen.getByLabelText('Hint'))
    await waitFor(() => expect(learningAPI.requestScenarioHint).toHaveBeenCalledWith('run1'))
  })
  it('shows chunk suggestions after hint', async () => {
    mockParams = { scenarioId: 'es-cafe' }
    vi.mocked(learningAPI.requestScenarioHint).mockResolvedValue([{ text: '¿Cuánto cuesta?', translation: 'How much does it cost?' }, { text: 'Para llevar', translation: 'To go' }] as any)
    render(<MemoryRouter><ScenarioRoleplay /></MemoryRouter>)
    await waitFor(() => screen.getByText('Hola, ¿qué te gustaría pedir?'))
    fireEvent.click(screen.getByLabelText('Hint'))
    await waitFor(() => expect(screen.getByText('¿Cuánto cuesta?')).toBeTruthy())
  })
  it('navigates from Scenarios list to Spanish roleplay', async () => {
    render(<MemoryRouter><Scenarios /></MemoryRouter>)
    await waitFor(() => screen.getByText('Pedir café en una cafetería'))
    fireEvent.click(screen.getByText('Pedir café en una cafetería'))
    expect(mockNavigate).toHaveBeenCalledWith('/learn/scenarios/es-cafe')
  })
  it('shows can-do statement for Spanish scenario', async () => {
    render(<MemoryRouter><Scenarios /></MemoryRouter>)
    await waitFor(() => expect(screen.getByText('Pedir una bebida')).toBeTruthy())
  })
  it('shows CEFR badges for Spanish scenarios', async () => {
    render(<MemoryRouter><Scenarios /></MemoryRouter>)
    await waitFor(() => expect(screen.getByText('A1')).toBeTruthy())
    expect(screen.getByText('A2')).toBeTruthy()
  })
})

describe('QA daily drills — web', () => {
  it('Learn shows daily goal progress and streak', async () => {
    render(<MemoryRouter><Learn /></MemoryRouter>)
    await waitFor(() => expect(screen.getByText(/6 \/ 10/)).toBeTruthy())
    expect(screen.getByText('learn.dayStreak')).toBeTruthy()
  })
  it('Learn vocabulary due section', async () => {
    render(<MemoryRouter><Learn /></MemoryRouter>)
    await waitFor(() => expect(screen.getByText('30')).toBeTruthy())
  })
  it('Learn Quick Drills start navigates to session', async () => {
    render(<MemoryRouter><Learn /></MemoryRouter>)
    await waitFor(() => screen.getByText('Drills'))
    fireEvent.click(screen.getByText('Drills'))
    expect(mockNavigate).toHaveBeenCalledWith(expect.stringContaining('/learn/session'))
  })
  it('Learn vocabulary card navigates', async () => {
    render(<MemoryRouter><Learn /></MemoryRouter>)
    await waitFor(() => screen.getByText('Vocabulary'))
    fireEvent.click(screen.getByText('Vocabulary'))
    expect(mockNavigate).toHaveBeenCalledWith('/learn/vocabulary')
  })
  it('Learn scenarios card navigates', async () => {
    render(<MemoryRouter><Learn /></MemoryRouter>)
    await waitFor(() => screen.getByText('Scenarios'))
    fireEvent.click(screen.getByText('Scenarios'))
    expect(mockNavigate).toHaveBeenCalledWith('/learn/scenarios')
  })
  it('LessonSession renders Spanish cloze with choices and feedback', async () => {
    mockSearch = '?mode=daily'
    render(<MemoryRouter><LessonSession /></MemoryRouter>)
    await waitFor(() => expect(screen.getByText('Yo ____ cansado.')).toBeTruthy())
    fireEvent.click(screen.getByText('estoy'))
    await waitFor(() => expect(screen.getByText('¡Excelente!')).toBeTruthy())
    fireEvent.click(screen.getByText('Continue'))
    await waitFor(() => expect(screen.getByText(/Translate: good morning/)).toBeTruthy())
  })
  it('LessonSession free-recall has Spanish placeholder', async () => {
    mockSearch = '?mode=daily'
    vi.mocked(learningAPI.startSession).mockResolvedValueOnce({
      session: { id: 'sess1', plannedItemCount: 1, mode: 'daily', status: 'in_progress' },
      items: [{ id: 'i2', itemType: 'vocabulary', activityType: 'free_recall', promptType: 'translate', prompt: { text: 'Translate: good morning', source: '' } }],
    } as any)
    render(<MemoryRouter><LessonSession /></MemoryRouter>)
    await waitFor(() => expect(screen.getByPlaceholderText('Escribe aquí...')).toBeTruthy())
  })
  it('LessonSession completes and shows XP', async () => {
    mockSearch = '?mode=daily'
    vi.mocked(learningAPI.startSession).mockResolvedValueOnce({
      session: { id: 'sess1', plannedItemCount: 1, mode: 'daily', status: 'in_progress' },
      items: [{ id: 'i1', itemType: 'vocabulary', activityType: 'cued_recall', promptType: 'cloze', prompt: { text: 'Yo ____ cansado.', choices: ['estoy'] } }],
    } as any)
    render(<MemoryRouter><LessonSession /></MemoryRouter>)
    await waitFor(() => screen.getByText('Yo ____ cansado.'))
    fireEvent.click(screen.getByText('estoy'))
    await waitFor(() => screen.getByText('Continue'))
    fireEvent.click(screen.getByText('Continue'))
    await waitFor(() => expect(screen.getByText('Session complete!')).toBeTruthy())
    expect(screen.getByText(/You earned/)).toBeTruthy()
  })
  it('VocabularyReview lists mined Spanish item and saves', async () => {
    render(<MemoryRouter><VocabularyReview /></MemoryRouter>)
    await waitFor(() => expect(screen.getByText('desayuno')).toBeTruthy())
    fireEvent.click(screen.getByText('Save'))
    await waitFor(() => expect(learningAPI.acceptMinedItem).toHaveBeenCalledWith('m1'))
  })
  it('VocabularyReview dismiss removes item', async () => {
    render(<MemoryRouter><VocabularyReview /></MemoryRouter>)
    await waitFor(() => screen.getByText('desayuno'))
    fireEvent.click(screen.getByText('Dismiss'))
    await waitFor(() => expect(learningAPI.ignoreMinedItem).toHaveBeenCalledWith('m1'))
  })
  it('Learn monthly activity shows Spanish month data', async () => {
    render(<MemoryRouter><Learn /></MemoryRouter>)
    await waitFor(() => expect(screen.getByTestId('learn-monthly')).toBeTruthy())
    expect(screen.getByText('August 2026')).toBeTruthy()
    expect(screen.getByText('22')).toBeTruthy()
  })
  it('streak recovery banner appears when at risk', async () => {
    vi.mocked(learningAPI.getDashboard).mockResolvedValueOnce({ ...dashboardEs, streak: { days: 7, atRisk: true, canRecover: true } } as any)
    render(<MemoryRouter><Learn /></MemoryRouter>)
    await waitFor(() => expect(screen.getByTestId('streak-at-risk-banner')).toBeTruthy())
    expect(screen.getByText('Recover')).toBeTruthy()
  })
})

describe('QA marketplace — web', () => {
  it('BrowseTutors loads Spanish tutor and shows rating', async () => {
    render(<MemoryRouter><BrowseTutors /></MemoryRouter>)
    await waitFor(() => expect(screen.getByText('María García')).toBeTruthy())
    expect(screen.getByText(/4\.9/)).toBeTruthy()
  })
  it('BrowseTutors search input exists and filters', async () => {
    render(<MemoryRouter><BrowseTutors /></MemoryRouter>)
    await waitFor(() => screen.getByTestId('tutor-search'))
    fireEvent.change(screen.getByTestId('tutor-search'), { target: { value: 'María' } })
    fireEvent.click(screen.getByText('Search'))
    await waitFor(() => expect(teacherAPI.browse).toHaveBeenCalled())
  })
  it('BrowseTutors empty state', async () => {
    vi.mocked(teacherAPI.browse).mockResolvedValueOnce({ tutors: [], total: 0, hasMore: false } as any)
    render(<MemoryRouter><BrowseTutors /></MemoryRouter>)
    await waitFor(() => expect(screen.getByText('No tutors yet')).toBeTruthy())
  })
  it('BrowseTutors links to TrialCredits/Dashboard/Payouts', async () => {
    render(<MemoryRouter><BrowseTutors /></MemoryRouter>)
    await waitFor(() => screen.getByText('Trial credits'))
    expect(screen.getByText('Teacher dashboard')).toBeTruthy()
    expect(screen.getByText('Payouts')).toBeTruthy()
  })
  it('TutorProfile renders Spanish tutor', async () => {
    mockParams = { id: 't1' }
    render(<MemoryRouter><TutorProfile /></MemoryRouter>)
    await waitFor(() => expect(screen.getByText('María García')).toBeTruthy())
  })
  it('TrialCredits reachable', async () => {
    render(<MemoryRouter><TrialCredits /></MemoryRouter>)
    await waitFor(() => expect(screen.getByText(/Trial Credits/)).toBeTruthy())
  })
  it('TeacherDashboard reachable', async () => {
    render(<MemoryRouter><TeacherDashboard /></MemoryRouter>)
    await waitFor(() => expect(screen.getByText(/Teacher Dashboard/)).toBeTruthy())
  })
  it('Payouts reachable', async () => {
    render(<MemoryRouter><Payouts /></MemoryRouter>)
    await waitFor(() => expect(screen.getByText(/Payout Settings/)).toBeTruthy())
  })
})

describe('QA learn hub — web', () => {
  it('Learn hub shows sections', async () => {
    render(<MemoryRouter><Learn /></MemoryRouter>)
    await waitFor(() => expect(screen.getByText('Drills')).toBeTruthy())
    expect(screen.getByText('Vocabulary')).toBeTruthy()
    expect(screen.getByText('Scenarios')).toBeTruthy()
  })
  it('Learn Find a Tutor bridge exists', async () => {
    render(<MemoryRouter><Learn /></MemoryRouter>)
    await waitFor(() => expect(screen.getByTestId('learn-find-tutors')).toBeTruthy())
    fireEvent.click(screen.getByTestId('learn-find-tutors'))
    expect(mockNavigate).toHaveBeenCalledWith('/tutors')
  })
  it('LearningRoadmap renders unit', async () => {
    render(<MemoryRouter><LearningRoadmap /></MemoryRouter>)
    await waitFor(() => expect(screen.getByText('Saludos')).toBeTruthy())
  })
  it('RealTalkHub reachable', async () => {
    render(<MemoryRouter><RealTalkHub /></MemoryRouter>)
    await waitFor(() => expect(screen.getAllByText(/Real Talk/).length).toBeGreaterThan(0))
  })
  it('StreakRecovery reachable', async () => {
    render(<MemoryRouter><StreakRecovery /></MemoryRouter>)
    expect(screen.getByText(/missed a day/)).toBeTruthy()
    expect(screen.getByTestId('streak-recovery-scenario')).toBeTruthy()
  })
})

describe('QA navigation parity — web + mobile', () => {
  it('App.tsx exposes marketplace + learn hub routes', () => {
    const appPath = path.resolve(__dirname, '../App.tsx')
    const content = fs.readFileSync(appPath, 'utf-8')
    const requiredRoutes = ['/tutors', '/tutors/:id', '/tutors/:id/confirm', '/trial-credits', '/teacher/dashboard', '/teacher/payouts', '/learn', '/learn/scenarios', '/learn/scenarios/:scenarioId', '/learn/vocabulary', '/learn/session', '/learn/roadmap', '/learn/real-talk', '/learn/streak-recovery', '/learn/placement']
    for (const r of requiredRoutes) {
      expect(content).toContain(r)
    }
  })
  it('MainTabs.tsx exposes marketplace + learn hub tabs on mobile', () => {
    const tabsPath = path.resolve(__dirname, '../../../mobile/src/components/MainTabs.tsx')
    const content = fs.readFileSync(tabsPath, 'utf-8')
    const requiredScreens = ['BrowseTutors', 'TutorProfile', 'ConfirmBooking', 'TrialCredits', 'TeacherDashboard', 'Payouts', 'Learn', 'Scenarios', 'ScenarioRoleplay', 'VocabularyReview', 'LessonSession', 'LearningRoadmap', 'RealTalkHub']
    for (const s of requiredScreens) {
      expect(content).toContain(s)
    }
    expect(content).toContain('MarketplaceTab')
    expect(content).toContain('LearnTab')
  })
})
