import axios from 'axios';

jest.mock('axios', () => {
  const instance = {
    get: jest.fn().mockResolvedValue({ data: {} }),
    post: jest.fn().mockResolvedValue({ data: {} }),
    put: jest.fn().mockResolvedValue({ data: {} }),
    delete: jest.fn().mockResolvedValue({ data: {} }),
    interceptors: {
      request: { use: jest.fn() },
      response: { use: jest.fn() },
    },
  };
  return {
    __esModule: true,
    default: {
      create: jest.fn(() => instance),
      get: instance.get,
      post: instance.post,
      put: instance.put,
      delete: instance.delete,
    },
    create: jest.fn(() => instance),
    get: instance.get,
    post: instance.post,
    put: instance.put,
    delete: instance.delete,
  };
});

import apiService from '../services/api';

describe('Mobile Functional API Contract Suite', () => {
  let mockClient: any;

  beforeEach(() => {
    jest.clearAllMocks();
    mockClient = (axios as any).create();
  });

  it('verifies user registration API contract', async () => {
    const registrationResponse = {
      data: {
        user: { id: 'u123', username: 'testuser', email: 'test@example.com' },
        tokens: { accessToken: 'token-abc', refreshToken: 'refresh-abc' },
      },
    };
    mockClient.post.mockResolvedValueOnce(registrationResponse);

    const res = await apiService.register({
      username: 'testuser',
      email: 'test@example.com',
      password: 'TestPassword1!',
      displayName: 'Test User',
      nativeLanguage: 'en',
      targetLanguages: ['es'],
    });

    expect(res.user.id).toBe('u123');
    expect(res.tokens.accessToken).toBe('token-abc');
  });

  it('verifies login API contract', async () => {
    const loginResponse = {
      data: {
        user: { id: 'u123', email: 'test@example.com' },
        tokens: { accessToken: 'token-abc' },
      },
    };
    mockClient.post.mockResolvedValueOnce(loginResponse);

    const res = await apiService.login('testuser', 'TestPassword1!');
    expect(res.tokens.accessToken).toBe('token-abc');
  });

  it('verifies get profile API contract', async () => {
    mockClient.get.mockResolvedValueOnce({
      data: { id: 'u123', displayName: 'Test User', nativeLanguage: 'en' },
    });

    const user = await apiService.getMe();
    expect(user.id).toBe('u123');
    expect(user.displayName).toBe('Test User');
  });

  it('verifies direct chat creation contract', async () => {
    mockClient.post.mockResolvedValueOnce({
      data: { id: 'chat-999', type: 'direct', participants: [{ userId: 'u123' }, { userId: 'u456' }] },
    });

    const chat = await apiService.createChat({ type: 'direct', participants: ['u456'] });
    expect(chat.id).toBe('chat-999');
  });

  it('verifies message sending and fetching contract', async () => {
    mockClient.post.mockResolvedValueOnce({
      data: { id: 'msg-100', text: 'Hello Mobile', chatId: 'chat-999' },
    });
    mockClient.get.mockResolvedValueOnce({
      data: { messages: [{ id: 'msg-100', text: 'Hello Mobile', chatId: 'chat-999' }] },
    });

    const sent = await apiService.sendMessage('chat-999', 'Hello Mobile');
    expect(sent.id).toBe('msg-100');

    const messages = await apiService.getMessages('chat-999');
    expect(messages).toHaveLength(1);
    expect(messages[0].text).toBe('Hello Mobile');
  });

  it('verifies fetching chat list contract', async () => {
    mockClient.get.mockResolvedValueOnce({
      data: { chats: [{ id: 'chat-999', type: 'direct' }] },
    });

    const chats = await apiService.getChats();
    expect(chats).toHaveLength(1);
    expect(chats[0].id).toBe('chat-999');
  });
});
