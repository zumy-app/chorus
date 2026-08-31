/**
 * Component tests for the Chorus mobile learning screens.
 *
 * Renders each learning screen with a mocked apiService / storage layer and
 * asserts the UI renders real data and key interactions work — including the
 * optimistic-send feedback in the roleplay and the roadmap navigation. These
 * run in Jest (no backend/emulator needed).
 */
import React from 'react';
import { render, fireEvent, waitFor } from '@testing-library/react-native';

let mockNavigate: jest.Mock;
let mockGoBack: jest.Mock;
let mockRouteParams: Record<string, any> = {};

jest.mock('@react-navigation/native', () => ({
  useNavigation: () => ({ navigate: mockNavigate, goBack: mockGoBack }),
  useRoute: () => ({ params: mockRouteParams }),
}));

jest.mock('../../utils/storage', () => ({
  __esModule: true,
  default: {
    getItem: jest.fn((key: string) =>
      Promise.resolve(key === 'user' ? JSON.stringify({ id: 'u1', nativeLanguage: 'en', targetLanguages: ['es'] }) : null)
    ),
    setItem: jest.fn(() => Promise.resolve()),
    removeItem: jest.fn(() => Promise.resolve()),
  },
}));

jest.mock('../../services/api', () => ({
  __esModule: true,
  default: {
    getLearningDashboard: jest.fn(),
    getLearningPath: jest.fn(),
    getScenarios: jest.fn(),
    startScenario: jest.fn(),
    sendScenarioMessage: jest.fn(),
    requestScenarioHint: jest.fn(),
    skipPlacement: jest.fn(),
    startSession: jest.fn(),
    answerSessionItem: jest.fn(),
    completeSession: jest.fn(),
    getMinedItems: jest.fn(),
    acceptMinedItem: jest.fn(),
    ignoreMinedItem: jest.fn(),
    startPlacement: jest.fn(),
    answerPlacement: jest.fn(),
  },
}));

import LearnScreen from '../../screens/LearnScreen';
import LearningRoadmapScreen from '../../screens/LearningRoadmapScreen';
import VocabularyReviewScreen from '../../screens/VocabularyReviewScreen';
import ScenarioRoleplayScreen from '../../screens/ScenarioRoleplayScreen';
import LessonSessionScreen from '../../screens/LessonSessionScreen';
import ScenariosScreen from '../../screens/ScenariosScreen';
import PlacementScreen from '../../screens/PlacementScreen';
import apiService from '../../services/api';

const api = apiService as any;

beforeEach(() => {
  jest.clearAllMocks();
  mockNavigate = jest.fn();
  mockGoBack = jest.fn();

  api.getLearningDashboard.mockResolvedValue({
    capability: { supportTier: 'full_course', placementEnabled: true, scenariosEnabled: true },
    profile: { placementStatus: 'completed', currentCefrLevel: 'A1' },
    dailyGoal: { targetItems: 10, completedItems: 3, percent: 30 },
    streak: { days: 4, atRisk: false, canRecover: false },
    fluency: { readinessScore: 200, readinessPercent: 20, label: 'Building A1' },
    currentUnit: { id: 'u1', title: 'Introductions', cefrLevel: 'A1', progressPct: 20 },
    nextLesson: { id: 'l1', title: 'Greetings', type: 'vocabulary', status: 'available' },
    vocabulary: { total: 12, dueToday: 3, mastered: 2, newFromChats: 1 },
    grammar: { weakestPointTitle: 'ser vs estar', confidencePct: 45, dueToday: 1 },
    scenario: { nextScenarioId: 'sc1', title: 'Ordering Coffee', progressPct: 0, hasNewWords: true },
    recommendedActivities: [
      { id: 'vocabulary', type: 'vocabulary', title: 'Vocabulary Review', description: 'Review', priority: 'high', estimatedMinutes: 3, action: 'start_session' },
      { id: 'scene', type: 'scenario', title: 'Scenario', description: 'Practice', priority: 'medium', estimatedMinutes: 5, action: 'open_scenarios' },
    ],
    weeklyActivity: [],
    monthlyActivity: [
      { month: '2026-07', wordsLearned: 8, sentencesUnderstood: 22 },
      { month: '2026-08', wordsLearned: 12, sentencesUnderstood: 31 },
    ],
  });

  api.getLearningPath.mockResolvedValue({
    capability: { supportTier: 'full_course' },
    profile: { placementStatus: 'completed' },
    units: [
      { id: 'u1', cefrLevel: 'A1', ordinal: 1, title: 'Introductions', canDoStatement: 'Greet', status: 'available', progressPct: 20, checkpointRequired: false, estimatedMinutes: 30, description: '', slug: 'a1-intro', courseId: 'c1', competencyScore: 0, lessonsCompleted: 0, lessons: [{ id: 'l1', unitId: 'u1', ordinal: 1, slug: 'vocab', type: 'vocabulary', title: 'Vocab', objective: '', estimatedMinutes: 5, status: 'available' }] },
      { id: 'u2', cefrLevel: 'A1', ordinal: 2, title: 'Locked', canDoStatement: 'Locked unit', status: 'locked', progressPct: 0, checkpointRequired: false, estimatedMinutes: 30, description: '', slug: 'a1-locked', courseId: 'c1', competencyScore: 0, lessonsCompleted: 0 },
      { id: 'u3', cefrLevel: 'A1', ordinal: 3, title: 'Done', canDoStatement: 'Done unit', status: 'completed', progressPct: 100, checkpointRequired: false, estimatedMinutes: 30, description: '', slug: 'a1-done', courseId: 'c1', competencyScore: 900, lessonsCompleted: 5 },
    ],
  });

  api.getScenarios.mockResolvedValue([
    { id: 'sc1', title: 'Ordering Coffee at a Cafe', cefrLevel: 'A1', canDoStatement: 'Order a drink', estimatedMinutes: 5, domain: 'food_drink', slug: 'ordering-coffee' },
  ]);

  api.startScenario.mockResolvedValue({
    run: { id: 'run1', currentPhaseOrdinal: 1 },
    aiResponse: {
      aiMessage: 'Hola. ¿Qué te gustaría pedir hoy?',
      translation: 'Hi. What would you like?',
      suggestedChunks: [{ text: 'Hola', translation: 'Hello' }],
    },
  });

  api.sendScenarioMessage.mockResolvedValue({
    aiMessage: 'Muy bien.',
    translation: 'Very good.',
    suggestedChunks: [],
    runCompleted: false,
  });
  api.requestScenarioHint.mockResolvedValue([]);

  api.startSession.mockResolvedValue({
    session: { id: 'sess1', plannedItemCount: 1, mode: 'daily', status: 'in_progress' },
    items: [{ id: 'i1', itemType: 'vocabulary', activityType: 'cued_recall', promptType: 'cloze', prompt: { text: 'Yo ____ cansado.', choices: ['estoy', 'soy', 'es', 'eres'] } }],
  });
  api.answerSessionItem.mockResolvedValue({ correct: true, quality: 4, feedback: { message: '¡Bien!' }, nextItem: null });
  api.completeSession.mockResolvedValue({ id: 'sess1', completedItemCount: 1 });

  api.getMinedItems.mockResolvedValue([
    { id: 'm1', surfaceText: 'café con leche', translation: 'coffee with milk', contextSentence: 'Quiero café con leche.', routeStatus: 'bonus', status: 'candidate' },
  ]);
  api.acceptMinedItem.mockResolvedValue({ id: 'card1' });
  api.ignoreMinedItem.mockResolvedValue({ ok: true });

  api.startPlacement.mockResolvedValue({
    attemptId: 'p1',
    totalQuestions: 12,
    question: { id: 'q1', ref: 'r1', itemType: 'vocabulary', cefrLevel: 'A1', prompt: { text: 'Which word?' }, choices: ['hola', 'adiós'] },
  });
  api.answerPlacement.mockResolvedValue({
    attemptId: 'p1',
    totalQuestions: 12,
    question: { id: 'q2', ref: 'r2', itemType: 'vocabulary', cefrLevel: 'A2', prompt: { text: 'Next?' }, choices: ['ayer'] },
  });
});

describe('LearnScreen', () => {
  it('renders dashboard metrics and streak', async () => {
    const { getByText } = render(<LearnScreen />);
    await waitFor(() => getByText('Your Learning Path'));
    expect(getByText('4 Days')).toBeTruthy();
    expect(getByText('Drills')).toBeTruthy();
    expect(getByText('Quick Drills')).toBeTruthy();
  });

  it('navigates to scenarios from the AI scenario card', async () => {
    const { getAllByText } = render(<LearnScreen />);
    await waitFor(() => getAllByText('Scenarios').length > 0);
    fireEvent.press(getAllByText('Scenarios')[0]);
    expect(mockNavigate).toHaveBeenCalledWith('Scenarios');
  });

  it('shows monthly activity metrics for the most recent month and lets you page back (FR-31)', async () => {
    const { getByTestId, getByText } = render(<LearnScreen />);
    await waitFor(() => getByTestId('learn-monthly'));
    expect(getByText('Monthly Activity')).toBeTruthy();
    expect(getByText('August 2026')).toBeTruthy();
    expect(getByText('12')).toBeTruthy();
    expect(getByText('31')).toBeTruthy();
    fireEvent.press(getByText('‹'));
    expect(getByText('July 2026')).toBeTruthy();
    expect(getByText('8')).toBeTruthy();
    expect(getByText('22')).toBeTruthy();
  });
});

describe('LearningRoadmapScreen', () => {
  it('renders units and disables locked ones', async () => {
    const { getByText, queryByText } = render(<LearningRoadmapScreen navigation={{ navigate: mockNavigate }} />);
    await waitFor(() => getByText('Introductions'));
    expect(getByText('Done')).toBeTruthy();
  });

  it('tapping an available unit starts a lesson session (no-op fix)', async () => {
    const { getByText } = render(<LearningRoadmapScreen navigation={{ navigate: mockNavigate }} />);
    await waitFor(() => getByText('Introductions'));
    fireEvent.press(getByText('Introductions'));
    expect(mockNavigate).toHaveBeenCalledWith('LessonSession', { mode: 'lesson', sessionId: undefined });
  });

  it('tapping a completed unit opens vocabulary practice', async () => {
    const { getByText } = render(<LearningRoadmapScreen navigation={{ navigate: mockNavigate }} />);
    await waitFor(() => getByText('Done'));
    fireEvent.press(getByText('Done'));
    expect(mockNavigate).toHaveBeenCalledWith('LessonSession', { mode: 'vocabulary', sessionId: undefined });
  });
});

describe('ScenarioRoleplayScreen', () => {
  it('opens with the AI greeting turn', async () => {
    mockRouteParams = { scenarioId: 'sc1' };
    const { getByText } = render(<ScenarioRoleplayScreen navigation={{ navigate: mockNavigate, goBack: mockGoBack }} />);
    await waitFor(() => getByText('Hola. ¿Qué te gustaría pedir hoy?'));
    expect(getByText('Hola. ¿Qué te gustaría pedir hoy?')).toBeTruthy();
  });

  it('optimistically shows the user bubble, then the AI reply + typing indicator', async () => {
    mockRouteParams = { scenarioId: 'sc1' };
    const { getByText, getByPlaceholderText, findByText } = render(<ScenarioRoleplayScreen navigation={{ navigate: mockNavigate, goBack: mockGoBack }} />);
    await waitFor(() => getByText('Hola. ¿Qué te gustaría pedir hoy?'));

    let resolveReply!: (v: any) => void;
    api.sendScenarioMessage.mockReturnValue(new Promise((res) => (resolveReply = res)));

    fireEvent.changeText(getByPlaceholderText('Escribe en español...'), 'Hola, buenos días.');
    fireEvent.press(getByText('➤'));

    // User bubble + typing indicator appear immediately, before the reply.
    expect(getByText('Hola, buenos días.')).toBeTruthy();
    expect(getByText('Sparky is typing…')).toBeTruthy();

    resolveReply({ aiMessage: 'Muy bien.', translation: 'Very good.', suggestedChunks: [], runCompleted: false });

    await waitFor(() => expect(getByText('Muy bien.')).toBeTruthy());
    expect(() => getByText('Sparky is typing…')).toThrow();
  });
});

describe('LessonSessionScreen', () => {
  it('renders a question and shows feedback after answering', async () => {
    mockRouteParams = { mode: 'daily' };
    const { getByText } = render(<LessonSessionScreen navigation={{ navigate: mockNavigate, goBack: mockGoBack }} />);
    await waitFor(() => getByText('Yo ____ cansado.'));
    fireEvent.press(getByText('estoy'));
    await waitFor(() => expect(getByText('¡Bien!')).toBeTruthy());
  });
});

describe('VocabularyReviewScreen', () => {
  it('lists mined items and can accept one', async () => {
    const { getByText, queryByText } = render(<VocabularyReviewScreen navigation={{ navigate: mockNavigate, goBack: mockGoBack }} />);
    await waitFor(() => getByText('café con leche'));
    fireEvent.press(getByText('Save'));
    await waitFor(() => expect(api.acceptMinedItem).toHaveBeenCalledWith('m1'));
    expect(queryByText('café con leche')).toBeNull();
  });
});

describe('ScenariosScreen', () => {
  it('lists scenarios and navigates to roleplay on tap', async () => {
    const { getByText } = render(<ScenariosScreen navigation={{ navigate: mockNavigate, goBack: mockGoBack }} />);
    await waitFor(() => getByText('Ordering Coffee at a Cafe'));
    fireEvent.press(getByText('Ordering Coffee at a Cafe'));
    expect(mockNavigate).toHaveBeenCalledWith('ScenarioRoleplay', { scenarioId: 'sc1' });
  });
});

describe('PlacementScreen', () => {
  it('renders the first placement question and advances', async () => {
    const { getByText } = render(<PlacementScreen navigation={{ navigate: mockNavigate, goBack: mockGoBack }} />);
    await waitFor(() => getByText('Which word?'));
    fireEvent.press(getByText('hola'));
    fireEvent.press(getByText('Check'));
    await waitFor(() => getByText('Next?'));
  });
});
