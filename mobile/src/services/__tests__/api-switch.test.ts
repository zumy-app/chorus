/**
 * Contract tests for the typed auth boundary in services/api.
 * The quick-switch outage came from screens reaching past the client into a
 * raw response envelope (`(apiService as any).api?.post` → undefined `.data`
 * crash). These tests pin the contract screens rely on: `login` forwards
 * credentials and returns the unwrapped body; `switchUser` resolves
 * {tokens, user} on success and throws human-readable Errors otherwise —
 * never undefined, never a raw envelope.
 */
import { mockLogin } from '../shared-mock';

jest.mock('@chorus/shared', () => {
  const actual = jest.requireActual('@chorus/shared');
  const anything = (): any => {
    const f = jest.fn();
    return new Proxy(f, { get: () => anything() });
  };
  return {
    ...actual,
    createApiClient: jest.fn(() => ({
      auth: {
        login: mockLogin,
        register: anything(),
        refreshToken: anything(),
        logout: anything(),
        getMe: anything(),
        updateMe: anything(),
        searchUsers: anything(),
        forgotPassword: anything(),
        resetPassword: anything(),
      },
      chat: anything(),
      message: anything(),
      translation: anything(),
      health: anything(),
      settings: anything(),
      learning: anything(),
      call: anything(),
      teacher: anything(),
      payouts: anything(),
      search: anything(),
      grammar: anything(),
      moderation: anything(),
      otp: anything(),
      api: anything(),
    })),
    resolveApiConfig: jest.fn(() => ({ baseURL: 'http://localhost:8080' })),
  };
});

import apiService from '../api';

beforeEach(() => {
  jest.clearAllMocks();
});

describe('apiService auth boundary', () => {
  it('login forwards {username, password} and returns the unwrapped body', async () => {
    const body = {
      tokens: { accessToken: 'a', refreshToken: 'r' },
      user: { id: 'u1', email: 'alice.en-es@chorus.test' },
    };
    mockLogin.mockResolvedValue(body);

    await expect(apiService.login('alice.en-es@chorus.test', 'ChorusDev123!')).resolves.toBe(
      body
    );
    expect(mockLogin).toHaveBeenCalledWith({
      username: 'alice.en-es@chorus.test',
      password: 'ChorusDev123!',
    });
  });

  it('switchUser resolves {tokens, user} on success', async () => {
    const session = {
      tokens: { accessToken: 'a', refreshToken: 'r' },
      user: { id: 'u2', email: 'bob.es-en@chorus.test' },
    };
    mockLogin.mockResolvedValue(session);

    await expect(apiService.switchUser('bob.es-en@chorus.test', 'ChorusDev123!')).resolves.toEqual(
      session
    );
  });

  it('switchUser throws a readable error when no session is returned', async () => {
    mockLogin.mockResolvedValue(undefined);
    await expect(apiService.switchUser('x', 'y')).rejects.toThrow(
      'Login did not return session tokens'
    );

    mockLogin.mockResolvedValue({ tokens: {} });
    await expect(apiService.switchUser('x', 'y')).rejects.toThrow(
      'Login did not return session tokens'
    );
  });

  it('switchUser explains 2FA-gated targets instead of crashing', async () => {
    mockLogin.mockResolvedValue({ requires2FA: true, tempToken: 't' });
    await expect(apiService.switchUser('x', 'y')).rejects.toThrow(
      'Target account requires two-factor verification'
    );
  });
});
