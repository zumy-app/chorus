/**
 * Regression test for the dev quick-switch outage:
 * `handleDevSwitch` called `(apiService as any).api?.post(...)`, but the
 * default-exported apiService has no `.api` member, so `raw` was undefined
 * and `raw.data` threw "Cannot read properties of undefined (reading
 * 'data')" → Alert "Switch failed". The mock below intentionally exposes
 * NO `.api` (like production), so this suite fails on the old code and
 * passes with the typed `apiService.switchUser()` fix.
 */
import React from 'react';
import { Alert } from 'react-native';
import { render, fireEvent, waitFor } from '@testing-library/react-native';

jest.mock('../../utils/storage', () => ({
  __esModule: true,
  default: {
    getItem: jest.fn((k: string) =>
      Promise.resolve(
        k === 'user'
          ? JSON.stringify({
              id: 'alice-id',
              username: 'alice.en-es',
              email: 'alice.en-es@chorus.test',
              displayName: 'Alice Dev',
              nativeLanguage: 'en',
              targetLanguages: ['es'],
            })
          : null
      )
    ),
    setItem: jest.fn(() => Promise.resolve()),
    removeItem: jest.fn(() => Promise.resolve()),
  },
}));

jest.mock('../../services/api', () => ({
  __esModule: true,
  default: {
    getMe: jest.fn(),
    updateProfile: jest.fn(),
    logout: jest.fn(),
    getSettings: jest.fn(() =>
      Promise.resolve({
        lastSeenVisibility: 'everyone',
        profilePhotoVisibility: 'everyone',
        contactsVisibility: 'everyone',
      })
    ),
    updateSettings: jest.fn(),
    getBlocked: jest.fn(() => Promise.resolve([])),
    getPhoneStatus: jest.fn(() => Promise.resolve({})),
    // NOTE: no `.api` member — matches the real default export.
    switchUser: jest.fn(),
  },
}));

jest.mock('../../services/websocket', () => ({
  __esModule: true,
  default: { disconnect: jest.fn(), connect: jest.fn() },
}));

import ProfileScreen from '../ProfileScreen';
import storage from '../../utils/storage';
import apiService from '../../services/api';
import webSocketService from '../../services/websocket';

jest.spyOn(Alert, 'alert').mockImplementation(() => {});
const alertMock = Alert.alert as jest.Mock;

const store = storage as unknown as { setItem: jest.Mock };
const api = apiService as unknown as { switchUser: jest.Mock };
const ws = webSocketService as unknown as { disconnect: jest.Mock };

const bobUser = {
  id: 'bob-id',
  username: 'bob.es-en',
  email: 'bob.es-en@chorus.test',
  displayName: 'Bob Dev',
  nativeLanguage: 'es',
  targetLanguages: ['en'],
};

function renderProfile() {
  const navigation = { replace: jest.fn(), navigate: jest.fn() };
  const ui = render(<ProfileScreen navigation={navigation} />);
  return { navigation, ui };
}

beforeEach(() => {
  jest.clearAllMocks();
});

describe('ProfileScreen dev quick-switch', () => {
  it('switches to the tapped account: tokens+user persisted, navigates to MainTabs', async () => {
    api.switchUser.mockResolvedValue({
      tokens: { accessToken: 'bob-access', refreshToken: 'bob-refresh' },
      user: bobUser,
    });
    const { navigation, ui } = renderProfile();

    // DEV_ACCOUNTS order: alice, bob, sofia → bob is index 1.
    const switchButtons = await ui.findAllByText('Switch →');
    fireEvent.press(switchButtons[1]);

    await waitFor(() => {
      expect(api.switchUser).toHaveBeenCalledWith('bob.es-en@chorus.test', 'ChorusDev123!');
    });
    await waitFor(() => {
      expect(store.setItem).toHaveBeenCalledWith('accessToken', 'bob-access');
      expect(store.setItem).toHaveBeenCalledWith('refreshToken', 'bob-refresh');
      expect(store.setItem).toHaveBeenCalledWith('user', JSON.stringify(bobUser));
      expect(navigation.replace).toHaveBeenCalledWith('MainTabs');
    });
    expect(ws.disconnect).toHaveBeenCalled();
  });

  it('surfaces a rejected login as a Switch failed alert and does not navigate', async () => {
    api.switchUser.mockRejectedValue(new Error('Invalid credentials'));
    const { navigation, ui } = renderProfile();

    const switchButtons = await ui.findAllByText('Switch →');
    fireEvent.press(switchButtons[1]);

    await waitFor(() => {
      expect(alertMock).toHaveBeenCalledWith('Switch failed', 'Invalid credentials');
    });
    expect(navigation.replace).not.toHaveBeenCalled();
  });

  it('explains when the target account needs 2FA instead of crashing', async () => {
    api.switchUser.mockRejectedValue(
      new Error('Target account requires two-factor verification — log in manually')
    );
    const { navigation, ui } = renderProfile();

    const switchButtons = await ui.findAllByText('Switch →');
    fireEvent.press(switchButtons[2]);

    await waitFor(() => {
      expect(alertMock).toHaveBeenCalledWith(
        'Switch failed',
        'Target account requires two-factor verification — log in manually'
      );
    });
    expect(navigation.replace).not.toHaveBeenCalled();
  });
});
