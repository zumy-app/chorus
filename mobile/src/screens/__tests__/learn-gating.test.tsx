import React from 'react';
import { render, waitFor } from '@testing-library/react-native';

jest.mock('@react-navigation/native', () => ({
  useNavigation: () => ({ navigate: jest.fn(), goBack: jest.fn(), getParent: () => ({ navigate: jest.fn() }), setOptions: jest.fn() }),
  useRoute: () => ({ params: {} }),
}));
jest.mock('../../utils/storage', () => ({
  __esModule: true,
  default: {
    getItem: jest.fn(() => Promise.resolve(JSON.stringify({ id: 'u1', nativeLanguage: 'en', targetLanguages: ['es'] }))),
    setItem: jest.fn(() => Promise.resolve()),
    removeItem: jest.fn(() => Promise.resolve()),
  },
}));
// NOTE: no featureFlags mock — the real manager with fail-closed defaults
// must hide every release-gated entry. This is the general-user prod posture.
jest.mock('../../services/api', () => ({
  __esModule: true,
  default: {
    getLearningDashboard: jest.fn().mockResolvedValue({
      profile: { placementStatus: 'completed' },
      capability: { supportTier: 'full_course' },
      dailyGoal: { percent: 0, completedItems: 0, targetItems: 10 },
      streak: { days: 0, atRisk: false, canRecover: false },
      vocabulary: { dueToday: 0 },
      scenario: { title: 'Scenario', progressPct: 0 },
      grammar: { weakestPointTitle: '', confidencePct: 0 },
      weeklyActivity: [],
      monthlyActivity: [],
    }),
    skipPlacement: jest.fn(),
  },
}));

import LearnScreen from '../LearnScreen';

describe('LearnScreen release gating (general user)', () => {
  it('hides placement, scenarios, and tutor entries when flags are off', async () => {
    const { queryByText, queryByTestId } = render(<LearnScreen />);
    await waitFor(() => expect(queryByText('Quick Drills')).toBeTruthy());
    // Core learning stays; gated entries disappear.
    expect(queryByText('Quick Drills')).toBeTruthy();
    expect(queryByText('Find your starting level')).toBeNull();
    expect(queryByText('Scenarios')).toBeNull();
    expect(queryByText('Find a Tutor')).toBeNull();
    expect(queryByTestId('learn-find-tutors')).toBeNull();
  });
});
