import React from 'react';
import { render, fireEvent, waitFor } from '@testing-library/react-native';

jest.mock('@react-navigation/native', () => ({
  useNavigation: () => ({ navigate: jest.fn(), goBack: jest.fn(), getParent: () => ({ navigate: jest.fn() }), setOptions: jest.fn() }),
  useRoute: () => ({ params: { chatId: 'chat-1', chatName: 'Alice' } }),
}));

const mockStorageState: Record<string, string | null> = {
  user: JSON.stringify({ id: 'u1', nativeLanguage: 'en', targetLanguages: ['es'] }),
  realTalkDraft: null,
  sparky_pending_job: null,
};
jest.mock('../utils/storage', () => ({
  __esModule: true,
  default: {
    getItem: jest.fn((k: string) => Promise.resolve(mockStorageState[k] ?? null)),
    setItem: jest.fn((k: string, v: string) => { mockStorageState[k] = v; return Promise.resolve(); }),
    removeItem: jest.fn((k: string) => { mockStorageState[k] = null; return Promise.resolve(); }),
  },
}));

let wsHandler: ((m: any) => void) | null = null;
jest.mock('../services/websocket', () => ({
  __esModule: true,
  default: {
    connect: jest.fn(),
    onMessage: jest.fn((h: any) => { wsHandler = h; return () => {}; }),
    send: jest.fn(),
    sendTyping: jest.fn(),
  },
}));
jest.mock('../components/RealTalkNudge', () => {
  const { Text } = require('react-native');
  return () => <Text>realtalk-nudge</Text>;
});
jest.mock('../utils/featureFlags', () => ({
  __esModule: true,
  default: { isEnabled: jest.fn(() => false), init: jest.fn(() => Promise.resolve()) },
}));

const mockApi: Record<string, jest.Mock> = {
  getMessages: jest.fn().mockResolvedValue([]),
  getPinnedMessages: jest.fn().mockResolvedValue([]),
  sendMessage: jest.fn(),
  markAsRead: jest.fn().mockResolvedValue({}),
  getChats: jest.fn().mockResolvedValue([]),
  getChat: jest.fn().mockResolvedValue({ id: 'chat-1', participants: [] }),
  getBlockStatus: jest.fn().mockResolvedValue({ blocked: false }),
  pinMessage: jest.fn().mockResolvedValue({}),
  unpinMessage: jest.fn().mockResolvedValue({}),
  sparkyAsk: jest.fn().mockResolvedValue({ jobId: 'j1', status: 'queued' }),
  sparkyJob: jest.fn(),
  grammarLearn: jest.fn(),
  grammarAnalyzeAI: jest.fn(),
};

jest.mock('../services/api', () => ({
  __esModule: true,
  default: {
    getMessages: (...a: any[]) => mockApi.getMessages(...a),
    getPinnedMessages: (...a: any[]) => mockApi.getPinnedMessages(...a),
    sendMessage: (...a: any[]) => mockApi.sendMessage(...a),
    markAsRead: (...a: any[]) => mockApi.markAsRead(...a),
    getChats: (...a: any[]) => mockApi.getChats(...a),
    getChat: (...a: any[]) => mockApi.getChat(...a),
    getBlockStatus: (...a: any[]) => mockApi.getBlockStatus(...a),
    pinMessage: (...a: any[]) => mockApi.pinMessage(...a),
    unpinMessage: (...a: any[]) => mockApi.unpinMessage(...a),
    sparkyAsk: (...a: any[]) => mockApi.sparkyAsk(...a),
    sparkyJob: (...a: any[]) => mockApi.sparkyJob(...a),
    grammarLearn: (...a: any[]) => mockApi.grammarLearn(...a),
    grammarAnalyzeAI: (...a: any[]) => mockApi.grammarAnalyzeAI(...a),
  },
}));

import ChatScreen from '../screens/ChatScreen';
import storage from '../utils/storage';

const route = { params: { chatId: 'chat-1', chatName: 'Alice' } } as any;
const navigation: any = { navigate: jest.fn(), setOptions: jest.fn(), goBack: jest.fn() };

describe('Sparky sheet — mobile', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    wsHandler = null;
    mockStorageState.sparky_pending_job = null;
  });

  it('async send resolves via WS and the input stays visible', async () => {
    const ui = render(<ChatScreen route={route} navigation={navigation} />);
    await waitFor(() => expect(mockApi.getMessages).toHaveBeenCalled());

    fireEvent.press(ui.getByLabelText('Ask Sparky'));
    const input = await ui.findByTestId('sparky-input');
    fireEvent.changeText(input, 'analyze the last message');
    fireEvent.press(ui.getByTestId('sparky-send'));

    await waitFor(() => expect(mockApi.sparkyAsk).toHaveBeenCalledWith(
      expect.objectContaining({ query: 'analyze the last message', chatId: 'chat-1' })
    ));
    // Job persisted for kill-recovery.
    expect(storage.setItem).toHaveBeenCalledWith(
      'sparky_pending_job', expect.stringContaining('j1')
    );

    // Backend answer arrives over WS.
    wsHandler!({ type: 'sparky_result', data: { jobId: 'j1', chatId: 'chat-1', status: 'done', content: 'Hola means hello' } });
    await waitFor(() => expect(ui.getByText('Hola means hello')).toBeTruthy());
    // Regression: the input must survive the answer (scroll layout, not clip).
    expect(ui.getByTestId('sparky-input')).toBeTruthy();
    expect(ui.getByTestId('sparky-scroll')).toBeTruthy();
  });

  it('resyncs a pending job left by a previous session on open', async () => {
    mockStorageState.sparky_pending_job = JSON.stringify({ jobId: 'j0', chatId: 'chat-1' });
    mockApi.sparkyJob.mockResolvedValueOnce({
      jobId: 'j0', chatId: 'chat-1', status: 'done', content: 'Recovered answer',
    });
    const ui = render(<ChatScreen route={route} navigation={navigation} />);
    await waitFor(() => expect(mockApi.getMessages).toHaveBeenCalled());

    fireEvent.press(ui.getByLabelText('Ask Sparky'));
    await waitFor(() => expect(mockApi.sparkyJob).toHaveBeenCalledWith('j0'));
    await waitFor(() => expect(ui.getByText('Recovered answer')).toBeTruthy());
  });
});
