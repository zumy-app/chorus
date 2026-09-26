import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, fireEvent, waitFor, within } from '@testing-library/react'
import '../../i18n'

let mockNavigate: ReturnType<typeof vi.fn>
let mockParams: Record<string, any> = {}

vi.mock('react-router-dom', () => ({
  useNavigate: () => mockNavigate,
  useParams: () => mockParams,
  useLocation: () => ({ search: '' }),
}))

vi.mock('../../store', () => ({
  useStore: (selector: any) => selector({ user: { targetLanguages: ['es'], nativeLanguage: 'en' } }),
}))

vi.mock('../../components/AppHeader', () => ({ default: () => null }))
vi.mock('../../components/BottomNav', () => ({ default: () => null }))

vi.mock('../../services/api', () => ({
  learningAPI: {
    getDashboard: vi.fn(),
    getPath: vi.fn(),
    getScenario: vi.fn(),
    startScenario: vi.fn(),
    sendScenarioMessage: vi.fn(),
    requestScenarioHint: vi.fn(),
    completeScenario: vi.fn(),
  },
}))

import LearningRoadmap from '../LearningRoadmap'
import ScenarioRoleplay from '../ScenarioRoleplay'
import Learn from '../Learn'
import { learningAPI } from '../../services/api'

beforeEach(() => {
  vi.clearAllMocks()
  mockNavigate = vi.fn()
  mockParams = {}
  learningAPI.getDashboard.mockResolvedValue({
    capability: { supportTier: 'full_course' },
    profile: { placementStatus: 'completed', currentCefrLevel: 'A1' },
    dailyGoal: { targetItems: 10, completedItems: 3, percent: 30 },
    streak: { days: 4, atRisk: false, canRecover: false },
    fluency: { readinessScore: 200, readinessPercent: 20, label: 'Building A1' },
    vocabulary: { total: 12, dueToday: 3, mastered: 2, newFromChats: 1 },
    grammar: { weakestPointTitle: 'ser vs estar', confidencePct: 45, dueToday: 1 },
    scenario: { progressPct: 0, hasNewWords: false },
    recommendedActivities: [],
    weeklyActivity: [],
    monthlyActivity: [
      { month: '2026-07', wordsLearned: 8, sentencesUnderstood: 22 },
      { month: '2026-08', wordsLearned: 12, sentencesUnderstood: 31 },
    ],
  })
  learningAPI.getPath.mockResolvedValue({
    capability: { supportTier: 'full_course' },
    profile: { placementStatus: 'completed' },
    units: [
      { id: 'u1', cefrLevel: 'A1', ordinal: 1, title: 'Introductions', canDoStatement: 'Greet', status: 'available', progressPct: 20, checkpointRequired: false, estimatedMinutes: 30, description: '', slug: 'a1-intro', courseId: 'c1', competencyScore: 0, lessonsCompleted: 0 },
      { id: 'u2', cefrLevel: 'A1', ordinal: 2, title: 'Locked', canDoStatement: 'Locked unit', status: 'locked', progressPct: 0, checkpointRequired: false, estimatedMinutes: 30, description: '', slug: 'a1-locked', courseId: 'c1', competencyScore: 0, lessonsCompleted: 0 },
      { id: 'u3', cefrLevel: 'A1', ordinal: 3, title: 'Done', canDoStatement: 'Done unit', status: 'completed', progressPct: 100, checkpointRequired: false, estimatedMinutes: 30, description: '', slug: 'a1-done', courseId: 'c1', competencyScore: 900, lessonsCompleted: 5 },
    ],
  })
  learningAPI.startScenario.mockResolvedValue({
    run: { id: 'run1', currentPhaseOrdinal: 1, scaffoldLevel: 'guided' },
    aiResponse: { aiMessage: 'Hola. ¿Qué te gustaría pedir hoy?', translation: 'Hi.', suggestedChunks: [] },
  })
  learningAPI.getScenario.mockResolvedValue({
    id: 'sc1',
    courseId: 'c1',
    slug: 'ordering-coffee',
    title: 'Ordering Coffee at a Cafe',
    domain: 'cafe',
    cefrLevel: 'A1',
    canDoStatement: 'Order a drink',
    aiRoleName: 'Barista',
    aiRoleDescription: 'A friendly cafe barista.',
    openingLine: 'Hola. ¿Qué te gustaría pedir hoy?',
    maxTurns: 5,
    estimatedMinutes: 5,
    phases: [
      { id: 'p1', scenarioId: 'sc1', ordinal: 1, title: 'Greeting', learnerGoal: 'Greet the barista.', requiredIntents: ['greet'], chunkBank: [] },
      { id: 'p2', scenarioId: 'sc1', ordinal: 2, title: 'Order', learnerGoal: 'Place an order.', requiredIntents: ['order_drink'], chunkBank: [] },
    ],
  })
  learningAPI.sendScenarioMessage.mockResolvedValue({ aiMessage: 'Muy bien.', translation: 'Very good.', suggestedChunks: [], runCompleted: false })
  learningAPI.completeScenario.mockResolvedValue({ runCompleted: true, summary: { score: 700, xpAwarded: 100, vocabularyAdded: 3 } })
})

describe('LearningRoadmap (web)', () => {
  it('renders units and starts a lesson session from an available unit', async () => {
    render(<LearningRoadmap />)
    await waitFor(() => expect(screen.getByText('Introductions')).toBeTruthy())
    fireEvent.click(screen.getByText('Introductions'))
    expect(mockNavigate).toHaveBeenCalledWith('/learn/session?mode=lesson')
  })

  it('opens vocabulary practice from a completed unit', async () => {
    render(<LearningRoadmap />)
    await waitFor(() => expect(screen.getByText('Done')).toBeTruthy())
    fireEvent.click(screen.getByText('Done'))
    expect(mockNavigate).toHaveBeenCalledWith('/learn/session?mode=vocabulary')
  })
})

describe('ScenarioRoleplay (web)', () => {
  it('optimistically shows the user bubble, then the AI reply + typing indicator', async () => {
    mockParams = { scenarioId: 'sc1' }
    render(<ScenarioRoleplay />)
    await waitFor(() => expect(screen.getByText('Hola. ¿Qué te gustaría pedir hoy?')).toBeTruthy())

    let resolveReply!: (v: any) => void
    learningAPI.sendScenarioMessage.mockReturnValue(new Promise(res => (resolveReply = res)))

    const input = screen.getByPlaceholderText('Escribe en español...')
    fireEvent.change(input, { target: { value: 'Hola, buenos días.' } })
    fireEvent.click(screen.getByRole('button', { name: /send/i }))

    expect(screen.getByText('Hola, buenos días.')).toBeTruthy()
    expect(screen.getByText('Sparky is typing…')).toBeTruthy()

    resolveReply({ aiMessage: 'Muy bien.', translation: 'Very good.', suggestedChunks: [], runCompleted: false })
    await waitFor(() => expect(screen.getByText('Muy bien.')).toBeTruthy())
    expect(screen.queryByText('Sparky is typing…')).toBeNull()
  })

  it('renders the scenario header with AI role, CEFR, and phase progress (FR-34 polish)', async () => {
    mockParams = { scenarioId: 'sc1' }
    render(<ScenarioRoleplay />)
    expect(await screen.findByText('Ordering Coffee at a Cafe')).toBeTruthy()
    expect(screen.getByText('Barista')).toBeTruthy()
    expect(screen.getByText('A1')).toBeTruthy()
    expect(screen.getByText(/Phase 1 of 2/)).toBeTruthy()
  })

  it('finishes an in-progress run early and shows the summary (FR-34)', async () => {
    mockParams = { scenarioId: 'sc1' }
    render(<ScenarioRoleplay />)
    await screen.findByText('Ordering Coffee at a Cafe')
    fireEvent.click(screen.getByRole('button', { name: /Finish/ }))
    await waitFor(() => expect(learningAPI.completeScenario).toHaveBeenCalledWith('run1'))
    expect(await screen.findByText('Scenario complete!')).toBeTruthy()
    expect(screen.getByText(/Score 700/)).toBeTruthy()
  })

  it('shows gentle grammar corrections and a phase-complete chip from the AI reply (FR-34)', async () => {
    mockParams = { scenarioId: 'sc1' }
    learningAPI.sendScenarioMessage.mockResolvedValue({
      aiMessage: 'Muy bien.',
      translation: 'Very good.',
      suggestedChunks: [],
      runCompleted: false,
      phaseComplete: true,
      errors: [{ span: 'yo quiero café', correction: 'Quiero un café', explanation: 'Use "querer" + article.' }],
    })
    render(<ScenarioRoleplay />)
    await screen.findByText('Ordering Coffee at a Cafe')
    const input = screen.getByPlaceholderText('Escribe en español...')
    fireEvent.change(input, { target: { value: 'yo quiero café' } })
    fireEvent.click(screen.getByRole('button', { name: /send/i }))
    expect(await screen.findByText(/Quiero un café/)).toBeTruthy()
    expect(screen.getByText('Phase complete!')).toBeTruthy()
  })
})

describe('Learn (web)', () => {
  it('renders monthly activity for the most recent month (FR-31)', async () => {
    render(<Learn />)
    await waitFor(() => expect(screen.getByText('August 2026')).toBeTruthy())
    const card = screen.getByTestId('learn-monthly')
    expect(within(card).getByText('Words learned')).toBeTruthy()
    expect(within(card).getByText('Sentences understood')).toBeTruthy()
    expect(within(card).getByText('12')).toBeTruthy()
    expect(within(card).getByText('31')).toBeTruthy()
  })

  it('navigates to the previous month with the month picker', async () => {
    render(<Learn />)
    await waitFor(() => expect(screen.getByText('August 2026')).toBeTruthy())
    fireEvent.click(screen.getByRole('button', { name: 'Previous month' }))
    expect(await screen.findByText('July 2026')).toBeTruthy()
    const card = screen.getByTestId('learn-monthly')
    expect(within(card).getByText('8')).toBeTruthy()
    expect(within(card).getByText('22')).toBeTruthy()
  })
})
