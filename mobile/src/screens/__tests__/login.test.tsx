/**
 * Wiring tests for LoginScreen auth flows. The screen previously reached the
 * backend through a dead `(apiService as any).api?.post` branch (always
 * undefined) plus a fallback import — the same untyped-envelope confusion
 * that broke ProfileScreen's quick-switch. These tests pin the typed
 * `apiService.login` / `apiService.verify2FA` wiring: correct args, token
 * persistence, navigation, 2FA branch, and failure alerts.
 */
import React from 'react';
import { Alert } from 'react-native';
import { render, fireEvent, waitFor } from '@testing-library/react-native';

jest.mock('../../utils/storage', () => ({
  __esModule: true,
  default: {
    getItem: jest.fn(() => Promise.resolve(null)),
    setItem: jest.fn(() => Promise.resolve()),
    removeItem: jest.fn(() => Promise.resolve()),
  },
}));

jest.mock('../../services/api', () => ({
  __esModule: true,
  default: {
    login: jest.fn(),
    verify2FA: jest.fn(),
  },
}));

import LoginScreen from '../LoginScreen';
import storage from '../../utils/storage';
import apiService from '../../services/api';

jest.spyOn(Alert, 'alert').mockImplementation(() => {});
const alertMock = Alert.alert as jest.Mock;

const store = storage as unknown as { setItem: jest.Mock };
const api = apiService as unknown as { login: jest.Mock; verify2FA: jest.Mock };

const aliceUser = {
  id: 'alice-id',
  username: 'alice.en-es',
  email: 'alice.en-es@chorus.test',
  displayName: 'Alice Dev',
  nativeLanguage: 'en',
  targetLanguages: ['es'],
};

function renderLogin() {
  const navigation = { replace: jest.fn(), navigate: jest.fn() };
  const ui = render(<LoginScreen navigation={navigation} />);
  return { navigation, ui };
}

async function fillAndSubmit(ui: ReturnType<typeof render>, email: string, password: string) {
  fireEvent.changeText(ui.getByPlaceholderText('you@example.com'), email);
  fireEvent.changeText(ui.getByPlaceholderText('Enter your password'), password);
  fireEvent.press(ui.getByText('Log In'));
}

beforeEach(() => {
  jest.clearAllMocks();
});

describe('LoginScreen', () => {
  it('logs in with the typed client, persists tokens+user, navigates to MainTabs', async () => {
    api.login.mockResolvedValue({
      tokens: { accessToken: 'a1', refreshToken: 'r1' },
      user: aliceUser,
    });
    const { navigation, ui } = renderLogin();

    await fillAndSubmit(ui, 'alice.en-es@chorus.test', 'ChorusDev123!');

    await waitFor(() => {
      expect(api.login).toHaveBeenCalledWith('alice.en-es@chorus.test', 'ChorusDev123!');
    });
    await waitFor(() => {
      expect(store.setItem).toHaveBeenCalledWith('accessToken', 'a1');
      expect(store.setItem).toHaveBeenCalledWith('refreshToken', 'r1');
      expect(store.setItem).toHaveBeenCalledWith('user', JSON.stringify(aliceUser));
      expect(navigation.replace).toHaveBeenCalledWith('MainTabs');
    });
  });

  it('routes 2FA-challenged accounts to the code step and verifies', async () => {
    api.login.mockResolvedValue({
      requires2FA: true,
      tempToken: 'tmp-123',
      phoneMasked: '•••12',
    });
    api.verify2FA.mockResolvedValue({
      tokens: { accessToken: 'a2', refreshToken: 'r2' },
      user: aliceUser,
    });
    const { navigation, ui } = renderLogin();

    await fillAndSubmit(ui, 'alice.en-es@chorus.test', 'ChorusDev123!');

    const codeInput = await ui.findByPlaceholderText('123456');
    fireEvent.changeText(codeInput, '123456');
    fireEvent.press(ui.getByText('Verify'));

    await waitFor(() => {
      expect(api.verify2FA).toHaveBeenCalledWith('tmp-123', '123456');
    });
    await waitFor(() => {
      expect(store.setItem).toHaveBeenCalledWith('accessToken', 'a2');
      expect(navigation.replace).toHaveBeenCalledWith('MainTabs');
    });
  });

  it('shows Login Failed on rejected credentials without navigating', async () => {
    api.login.mockRejectedValue({ response: { data: { error: 'Invalid credentials' } } });
    const { navigation, ui } = renderLogin();

    await fillAndSubmit(ui, 'nobody@chorus.test', 'wrong');

    await waitFor(() => {
      expect(alertMock).toHaveBeenCalledWith('Login Failed', 'Invalid credentials');
    });
    expect(navigation.replace).not.toHaveBeenCalled();
  });

  it('DevAccountSwitcher Fill populates the form fields', async () => {
    const { ui } = renderLogin();

    const fillButtons = await ui.findAllByText('Fill →');
    fireEvent.press(fillButtons[0]);

    await waitFor(() => {
      expect(ui.getByPlaceholderText('you@example.com').props.value).toBe(
        'alice.en-es@chorus.test'
      );
      expect(ui.getByPlaceholderText('Enter your password').props.value).toBe('ChorusDev123!');
    });
  });
});
