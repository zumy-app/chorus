import React from 'react';
import { render, fireEvent, waitFor } from '@testing-library/react-native';

let mockNavigate: jest.Mock;
let mockGoBack: jest.Mock;
let mockRouteParams: Record<string, any> = {};

jest.mock('@react-navigation/native', () => ({
  useNavigation: () => ({ navigate: mockNavigate, goBack: mockGoBack, getParent: () => ({ navigate: mockNavigate }) }),
  useRoute: () => ({ params: mockRouteParams }),
}));

jest.mock('../../utils/storage', () => ({
  __esModule: true,
  default: {
    getItem: jest.fn((k: string) => Promise.resolve(k === 'user' ? JSON.stringify({ id: 'u1', nativeLanguage: 'en', targetLanguages: ['es'] }) : null)),
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
    getScenario: jest.fn(),
    startScenario: jest.fn(),
    sendScenarioMessage: jest.fn(),
    requestScenarioHint: jest.fn(),
    completeScenario: jest.fn(),
    getScenarioRun: jest.fn(),
    startSession: jest.fn(),
    answerSessionItem: jest.fn(),
    completeSession: jest.fn(),
    getSession: jest.fn(),
    getMinedItems: jest.fn(),
    acceptMinedItem: jest.fn(),
    ignoreMinedItem: jest.fn(),
    startPlacement: jest.fn(),
    answerPlacement: jest.fn(),
    skipPlacement: jest.fn(),
    browseTutors: jest.fn(),
    getTutorProfile: jest.fn(),
    getTutorReviews: jest.fn(),
    getTutorAvailability: jest.fn(),
    bookTutor: jest.fn(),
    getTrialCredits: jest.fn(),
    getTrialCreditsDashboard: jest.fn(),
    getTeacherDashboard: jest.fn(),
    getPayoutOverview: jest.fn(),
    getPayoutMethods: jest.fn(),
    getPayoutHistory: jest.fn(),
    getLearningProfile: jest.fn(),
    getLearningCapabilities: jest.fn(),
  },
}));

import LearnScreen from '../LearnScreen';
import ScenariosScreen from '../ScenariosScreen';
import ScenarioRoleplayScreen from '../ScenarioRoleplayScreen';
import LessonSessionScreen from '../LessonSessionScreen';
import VocabularyReviewScreen from '../VocabularyReviewScreen';
import BrowseTutorsScreen from '../BrowseTutorsScreen';
import TrialCreditsScreen from '../TrialCreditsScreen';
import TeacherDashboardScreen from '../TeacherDashboardScreen';
import PayoutsScreen from '../PayoutsScreen';
import TutorProfileScreen from '../TutorProfileScreen';
import ConfirmBookingScreen from '../ConfirmBookingScreen';
import LearningRoadmapScreen from '../LearningRoadmapScreen';
import apiService from '../../services/api';

const api = apiService as any;

const spanishScenarios = [
  { id: 'es-cafe', title: 'Pedir café en una cafetería', cefrLevel: 'A1', canDoStatement: 'Pedir una bebida', estimatedMinutes: 5, domain: 'food_drink', slug: 'pedir-cafe' },
  { id: 'es-mercado', title: 'Comprar en el mercado', cefrLevel: 'A2', canDoStatement: 'Comprar frutas', estimatedMinutes: 6, domain: 'shopping', slug: 'mercado' },
];

const dashboardEs = {
  capability: { supportTier: 'full_course', placementEnabled: true, scenariosEnabled: true },
  profile: { placementStatus: 'completed', currentCefrLevel: 'A1', targetLanguage: 'es', nativeLanguage: 'en' },
  dailyGoal: { targetItems: 10, completedItems: 6, percent: 60 },
  streak: { days: 7, atRisk: false, canRecover: false },
  fluency: { readinessScore: 350, readinessPercent: 35, label: 'Construyendo A1' },
  currentUnit: { id: 'u1', title: 'Saludos', cefrLevel: 'A1', progressPct: 40 },
  nextLesson: { id: 'l1', title: 'Saludos básicos', type: 'vocabulary', status: 'available' },
  vocabulary: { total: 30, dueToday: 5, mastered: 10, newFromChats: 3 },
  grammar: { weakestPointTitle: 'ser vs estar', confidencePct: 55, dueToday: 2 },
  scenario: { nextScenarioId: 'es-cafe', title: 'Pedir café en una cafetería', progressPct: 25, hasNewWords: true },
  recommendedActivities: [{ id: 'vocabulary', type: 'vocabulary', title: 'Repaso', description: 'Repaso', priority: 'high', estimatedMinutes: 3, action: 'start_session' }],
  weeklyActivity: [{ date: '2026-08-25', xp: 20, itemsCompleted: 2 }, { date: '2026-08-26', xp: 30, itemsCompleted: 3 }],
  monthlyActivity: [{ month: '2026-07', wordsLearned: 15, sentencesUnderstood: 40 }, { month: '2026-08', wordsLearned: 22, sentencesUnderstood: 55 }],
};

beforeEach(() => {
  jest.clearAllMocks();
  mockNavigate = jest.fn();
  mockGoBack = jest.fn();
  mockRouteParams = {};
  api.getLearningDashboard.mockResolvedValue(dashboardEs);
  api.getLearningPath.mockResolvedValue({
    capability: { supportTier: 'full_course' },
    profile: { placementStatus: 'completed' },
    units: [{ id: 'u1', cefrLevel: 'A1', ordinal: 1, title: 'Saludos', canDoStatement: 'Saludar', status: 'available', progressPct: 30, checkpointRequired: false, estimatedMinutes: 30, description: '', slug: 'a1-saludos', courseId: 'c1', competencyScore: 0, lessonsCompleted: 1, lessons: [] }],
  });
  api.getScenarios.mockResolvedValue(spanishScenarios);
  api.startScenario.mockResolvedValue({ run: { id: 'run1', currentPhaseOrdinal: 1 }, aiResponse: { aiMessage: 'Hola, ¿qué te gustaría pedir?', translation: 'Hi, what would you like to order?', suggestedChunks: [{ text: 'Quisiera un café', translation: 'I would like a coffee' }] } });
  api.sendScenarioMessage.mockResolvedValue({ aiMessage: 'Perfecto, ¿algo más?', translation: 'Perfect, anything else?', suggestedChunks: [], runCompleted: false });
  api.requestScenarioHint.mockResolvedValue([{ text: '¿Cuánto cuesta?', translation: 'How much does it cost?' }]);
  api.startSession.mockResolvedValue({
    session: { id: 'sess1', plannedItemCount: 2, mode: 'daily', status: 'in_progress' },
    items: [
      { id: 'i1', itemType: 'vocabulary', activityType: 'cued_recall', promptType: 'cloze', prompt: { text: 'Yo ____ cansado.', choices: ['estoy', 'soy'] } },
      { id: 'i2', itemType: 'vocabulary', activityType: 'free_recall', promptType: 'translate', prompt: { text: 'Translate: good morning', source: 'good morning' } },
    ],
  });
  api.answerSessionItem.mockResolvedValue({ correct: true, quality: 5, feedback: { message: '¡Excelente!' }, nextItem: null });
  api.completeSession.mockResolvedValue({ id: 'sess1', completedItemCount: 2 });
  api.getMinedItems.mockResolvedValue([{ id: 'm1', surfaceText: 'desayuno', translation: 'breakfast', contextSentence: 'Quiero desayuno.', routeStatus: 'bonus', status: 'candidate' }]);
  api.acceptMinedItem.mockResolvedValue({ id: 'card1' });
  api.ignoreMinedItem.mockResolvedValue({ ok: true });
  api.browseTutors.mockResolvedValue({ tutors: [{ userId: 't1', displayName: 'María García', languages: ['es', 'en'], ratingAvg: 4.9, rateCents: 2000, verified: true }], total: 1, hasMore: false });
  api.getTutorProfile.mockResolvedValue({ id: 't1', userId: 't1', displayName: 'María García', bio: 'Native Spanish tutor', languages: ['es'], rateCents: 2000, videoUrl: 'https://example.com/v.mp4', status: 'approved', verified: true, ratingAvg: 4.9, ratingCount: 12 });
  api.getTutorReviews.mockResolvedValue([{ id: 'r1', teacherUserId: 't1', studentUserId: 'u1', rating: 5, comment: 'Great!' }]);
  api.getTutorAvailability.mockResolvedValue([{ id: 'a1', teacherUserId: 't1', startTime: new Date(Date.now()+86400000).toISOString(), endTime: new Date(Date.now()+90000000).toISOString() }]);
  api.bookTutor.mockResolvedValue({ id: 'b1', status: 'pending' });
  api.getTrialCreditsDashboard.mockResolvedValue({ credits: 1, nextGrantAt: new Date(Date.now()+86400000).toISOString(), history: [{ id: 'b1', startTime: new Date().toISOString(), status: 'pending', isTrial: true }], dashboard: { credits: 1, nextGrantAt: new Date(Date.now()+86400000).toISOString(), history: [] } });
  api.getTrialCredits.mockResolvedValue({ credits: 1, history: [] });
  api.getTeacherDashboard.mockResolvedValue({ checklist: { items: [], pct: 80 }, earnings: { availableCents: 5000, pendingCents: 1000 }, students: [{ id: 'u1', displayName: 'Student' }] });
  api.getPayoutOverview.mockResolvedValue({ availableCents: 5000, pendingCents: 1000, totalGrossCents: 6000, platformFeePct: 10, recentTransactions: [] });
  api.getPayoutMethods.mockResolvedValue([]);
  api.getPayoutHistory.mockResolvedValue([]);
  api.getScenario.mockResolvedValue(spanishScenarios[0]);
});

describe('QA Spanish scenarios', () => {
  it('lists Spanish scenarios (es domain)', async () => {
    const { getByText } = render(<ScenariosScreen navigation={{ navigate: mockNavigate, goBack: mockGoBack }} />);
    await waitFor(() => getByText('Pedir café en una cafetería'));
    expect(getByText('Comprar en el mercado')).toBeTruthy();
    expect(api.getScenarios).toHaveBeenCalled();
  });
  it('starts Spanish scenario and shows opening line + translation', async () => {
    mockRouteParams = { scenarioId: 'es-cafe' };
    const { getByText } = render(<ScenarioRoleplayScreen navigation={{ navigate: mockNavigate, goBack: mockGoBack }} />);
    await waitFor(() => getByText('Hola, ¿qué te gustaría pedir?'));
    expect(getByText('Hi, what would you like to order?')).toBeTruthy();
  });
  it('shows suggested Spanish chunk for scaffolding', async () => {
    mockRouteParams = { scenarioId: 'es-cafe' };
    const { getByText } = render(<ScenarioRoleplayScreen navigation={{ navigate: mockNavigate, goBack: mockGoBack }} />);
    await waitFor(() => getByText('Quisiera un café'));
  });
  it('sends Spanish user message and receives AI reply with translation', async () => {
    mockRouteParams = { scenarioId: 'es-cafe' };
    const { getByText, getByPlaceholderText } = render(<ScenarioRoleplayScreen navigation={{ navigate: mockNavigate, goBack: mockGoBack }} />);
    await waitFor(() => getByText('Hola, ¿qué te gustaría pedir?'));
    fireEvent.changeText(getByPlaceholderText('Escribe en español...'), 'Quisiera un café por favor');
    fireEvent.press(getByText('➤'));
    await waitFor(() => getByText('Perfecto, ¿algo más?'));
    expect(getByText('Perfect, anything else?')).toBeTruthy();
    expect(api.sendScenarioMessage).toHaveBeenCalled();
  });
  it('requests Spanish hint', async () => {
    mockRouteParams = { scenarioId: 'es-cafe' };
    const { getByText } = render(<ScenarioRoleplayScreen navigation={{ navigate: mockNavigate, goBack: mockGoBack }} />);
    await waitFor(() => getByText('Hola, ¿qué te gustaría pedir?'));
    const hintBtn = getByText('💡');
    fireEvent.press(hintBtn);
    await waitFor(() => getByText('¿Cuánto cuesta?'));
  });
  it('navigates from Scenarios list to Spanish roleplay', async () => {
    const { getByText } = render(<ScenariosScreen navigation={{ navigate: mockNavigate, goBack: mockGoBack }} />);
    await waitFor(() => getByText('Pedir café en una cafetería'));
    fireEvent.press(getByText('Pedir café en una cafetería'));
    expect(mockNavigate).toHaveBeenCalledWith('ScenarioRoleplay', { scenarioId: 'es-cafe' });
  });
});

describe('QA daily drills', () => {
  it('LearnScreen shows daily goal progress (60%)', async () => {
    const { getByText } = render(<LearnScreen />);
    await waitFor(() => getByText('Your Learning Path'));
    expect(getByText('6/10 completed')).toBeTruthy();
  });
  it('Quick Drills Start navigates to LessonSession quick_drill', async () => {
    const { getByText } = render(<LearnScreen />);
    await waitFor(() => getByText('Your Learning Path'));
    fireEvent.press(getByText('Start'));
    expect(mockNavigate).toHaveBeenCalledWith('LessonSession', { sessionId: undefined, mode: 'quick_drill' });
  });
  it('LessonSession renders cloze with Spanish choices and feedback', async () => {
    mockRouteParams = { mode: 'daily' };
    const { getByText } = render(<LessonSessionScreen navigation={{ navigate: mockNavigate, goBack: mockGoBack }} />);
    await waitFor(() => getByText('Yo ____ cansado.'));
    fireEvent.press(getByText('estoy'));
    await waitFor(() => getByText('¡Excelente!'));
    fireEvent.press(getByText('Continue'));
    await waitFor(() => getByText('Translate: good morning'));
  });
  it('LessonSession free-recall Spanish input placeholder', async () => {
    mockRouteParams = { mode: 'daily' };
    api.startSession.mockResolvedValueOnce({
      session: { id: 'sess1', plannedItemCount: 1, mode: 'daily', status: 'in_progress' },
      items: [{ id: 'i2', itemType: 'vocabulary', activityType: 'free_recall', promptType: 'translate', prompt: { text: 'Translate: good morning' } }],
    });
    const { getByPlaceholderText } = render(<LessonSessionScreen navigation={{ navigate: mockNavigate, goBack: mockGoBack }} />);
    await waitFor(() => getByPlaceholderText('Escribe aquí...'));
    expect(getByPlaceholderText('Escribe aquí...')).toBeTruthy();
  });
  it('LessonSession completes and shows XP', async () => {
    mockRouteParams = { mode: 'daily' };
    api.startSession.mockResolvedValueOnce({
      session: { id: 'sess1', plannedItemCount: 1, mode: 'daily', status: 'in_progress' },
      items: [{ id: 'i1', itemType: 'vocabulary', activityType: 'cued_recall', promptType: 'cloze', prompt: { text: 'Yo ____ cansado.', choices: ['estoy'] } }],
    });
    const { getByText } = render(<LessonSessionScreen navigation={{ navigate: mockNavigate, goBack: mockGoBack }} />);
    await waitFor(() => getByText('Yo ____ cansado.'));
    fireEvent.press(getByText('estoy'));
    await waitFor(() => getByText('Continue'));
    fireEvent.press(getByText('Continue'));
    await waitFor(() => getByText('Session complete!'));
    expect(getByText(/You earned/)).toBeTruthy();
  });
  it('VocabularyReview lists mined Spanish item and saves', async () => {
    const { getByText } = render(<VocabularyReviewScreen navigation={{ navigate: mockNavigate, goBack: mockGoBack }} />);
    await waitFor(() => getByText('desayuno'));
    fireEvent.press(getByText('Save'));
    await waitFor(() => expect(api.acceptMinedItem).toHaveBeenCalledWith('m1'));
  });
});

describe('QA marketplace', () => {
  it('BrowseTutors loads Spanish tutor and shows rating', async () => {
    const { getByText } = render(<BrowseTutorsScreen />);
    await waitFor(() => getByText('María García'));
    expect(getByText(/4\.9/)).toBeTruthy();
  });
  it('BrowseTutors search filters', async () => {
    const { getByTestId } = render(<BrowseTutorsScreen />);
    await waitFor(() => getByTestId('tutor-search'));
    fireEvent.changeText(getByTestId('tutor-search'), 'María');
    fireEvent(getByTestId('tutor-search'), 'submitEditing');
    await waitFor(() => expect(api.browseTutors).toHaveBeenCalled());
  });
  it('BrowseTutors Book Trial navigates to profile', async () => {
    api.browseTutors.mockResolvedValueOnce({ tutors: [{ userId: 't1', displayName: 'María García', languages: ['es'], ratingAvg: 4.9, rateCents: 2000 }, { userId: 't2', displayName: 'Carlos', languages: ['es'], ratingAvg: 4.8, rateCents: 1800 }, { userId: 't3', displayName: 'Ana', languages: ['es'], ratingAvg: 5, rateCents: 2200 }], total: 3, hasMore: false });
    const { getByText } = render(<BrowseTutorsScreen />);
    await waitFor(() => getByText('Book Trial'));
    fireEvent.press(getByText('Book Trial'));
    expect(mockNavigate).toHaveBeenCalled();
  });
  it('BrowseTutors empty state shows Become a teacher', async () => {
    api.browseTutors.mockResolvedValueOnce({ tutors: [], total: 0, hasMore: false });
    const { getByText } = render(<BrowseTutorsScreen />);
    await waitFor(() => getByText('No tutors yet'));
    expect(getByText('Become a teacher')).toBeTruthy();
  });
  it('BrowseTutors links to TrialCredits/Dashboard/Payouts', async () => {
    const { getByText } = render(<BrowseTutorsScreen />);
    await waitFor(() => getByText('Trial credits'));
    fireEvent.press(getByText('Trial credits'));
    expect(mockNavigate).toHaveBeenCalledWith('TrialCredits');
    fireEvent.press(getByText('Dashboard'));
    expect(mockNavigate).toHaveBeenCalledWith('TeacherDashboard');
    fireEvent.press(getByText('Payouts'));
    expect(mockNavigate).toHaveBeenCalledWith('Payouts');
  });
  it('TutorProfile renders Spanish tutor and Book Trial', async () => {
    mockRouteParams = { userId: 't1' };
    const { getByText } = render(<TutorProfileScreen navigation={{ navigate: mockNavigate, goBack: mockGoBack }} />);
    await waitFor(() => getByText('María García'));
    expect(getByText('Book Trial')).toBeTruthy();
  });
  it('ConfirmBooking shows tutor and confirms booking', async () => {
    mockRouteParams = { userId: 't1' };
    const { getByText } = render(<ConfirmBookingScreen navigation={{ navigate: mockNavigate, goBack: mockGoBack }} />);
    await waitFor(() => getByText(/Confirm/i));
    const btn = getByText(/Confirm Booking|Book Trial|Confirm/i);
    fireEvent.press(btn);
    await waitFor(() => expect(api.bookTutor).toHaveBeenCalled());
  });
  it('TrialCredits shows credits and How Trials', async () => {
    const { getByText, getAllByText } = render(<TrialCreditsScreen />);
    await waitFor(() => getByText('Trial credits available'));
    expect(getAllByText('1').length >= 1).toBeTruthy();
    expect(getByText('How Trials Work')).toBeTruthy();
    expect(getByText(/Find a Tutor/)).toBeTruthy();
  });
  it('TrialCredits shows recommended tutors', async () => {
    const { getByText } = render(<TrialCreditsScreen />);
    await waitFor(() => getByText('Recommended for Trials'));
    expect(getByText('María García')).toBeTruthy();
  });
  it('TeacherDashboard renders earnings and students', async () => {
    const { getByText } = render(<TeacherDashboardScreen />);
    await waitFor(() => getByText('Teacher Dashboard'));
    expect(getByText('Earnings Overview')).toBeTruthy();
  });
  it('Payouts renders overview', async () => {
    const { getByText } = render(<PayoutsScreen />);
    await waitFor(() => getByText('Earnings'));
    expect(getByText(/available/)).toBeTruthy();
  });
});

describe('QA learn hub', () => {
  it('LearnScreen renders hub title and streak', async () => {
    const { getByText, getAllByText } = render(<LearnScreen />);
    await waitFor(() => getByText('Your Learning Path'));
    expect(getByText('7 Days')).toBeTruthy();
    expect(getByText('Quick Drills')).toBeTruthy();
    expect(getAllByText('Vocabulary').length >= 1).toBeTruthy();
    expect(getAllByText('Scenarios').length >= 1).toBeTruthy();
    expect(getByText('Grammar Deep Dive')).toBeTruthy();
  });
  it('LearnScreen scenario card navigates to Scenarios', async () => {
    const { getAllByText } = render(<LearnScreen />);
    await waitFor(() => getAllByText('Scenarios').length > 0);
    fireEvent.press(getAllByText('Scenarios')[0]);
    expect(mockNavigate).toHaveBeenCalledWith('Scenarios');
  });
  it('LearnScreen monthly activity shows Spanish month data and pages', async () => {
    const { getByTestId, getByText } = render(<LearnScreen />);
    await waitFor(() => getByTestId('learn-monthly'));
    expect(getByText('August 2026')).toBeTruthy();
    expect(getByText('22')).toBeTruthy();
    fireEvent.press(getByText('‹'));
    expect(getByText('July 2026')).toBeTruthy();
  });
  it('LearnScreen weekly goal and Find a Tutor bridge', async () => {
    const { getByText, getByTestId } = render(<LearnScreen />);
    await waitFor(() => getByText('Weekly Goal'));
    expect(getByTestId('learn-find-tutors')).toBeTruthy();
    fireEvent.press(getByTestId('learn-find-tutors'));
    expect(mockNavigate).toHaveBeenCalled();
  });
  it('LearnScreen vocabulary card navigates to VocabularyReview', async () => {
    const { getAllByText } = render(<LearnScreen />);
    await waitFor(() => getAllByText('Vocabulary').length > 0);
    fireEvent.press(getAllByText('Vocabulary')[0]);
    expect(mockNavigate).toHaveBeenCalledWith('VocabularyReview');
  });
  it('LearningRoadmap renders Spanish unit Saludos', async () => {
    const { getByText } = render(<LearningRoadmapScreen navigation={{ navigate: mockNavigate }} />);
    await waitFor(() => getByText('Saludos'));
  });
  it('Learn hub Drills quick action starts session', async () => {
    const { getByText } = render(<LearnScreen />);
    await waitFor(() => getByText('Drills'));
    fireEvent.press(getByText('Drills'));
    expect(mockNavigate).toHaveBeenCalledWith('LessonSession', expect.objectContaining({ mode: 'quick_drill' }));
  });
});
