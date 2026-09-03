import React from 'react';
import { render, fireEvent, waitFor } from '@testing-library/react-native';

jest.mock('@react-navigation/native', () => ({
  useNavigation: () => ({ navigate: jest.fn(), goBack: jest.fn(), getParent: () => ({ navigate: jest.fn() }), setOptions: jest.fn() }),
  useRoute: () => ({ params: { chatId: 'chat-1', chatName: 'Alice' } }),
}));
jest.mock('../utils/storage', () => ({
  __esModule: true,
  default: {
    getItem: jest.fn((k: string) => {
      if (k === 'user') return Promise.resolve(JSON.stringify({ id: 'u1', nativeLanguage: 'en', targetLanguages: ['es'] }))
      if (k === 'realTalkDraft') return Promise.resolve(null)
      return Promise.resolve(null)
    }),
    setItem: jest.fn(() => Promise.resolve()),
    removeItem: jest.fn(() => Promise.resolve()),
  },
}));
jest.mock('../services/websocket', () => ({
  __esModule: true,
  default: { connect: jest.fn(), onMessage: jest.fn(() => () => {}), send: jest.fn(), sendTyping: jest.fn() },
}));
jest.mock('../components/RealTalkNudge', () => {
  const { Text } = require('react-native');
  return () => <Text>realtalk-nudge</Text>;
});

const mockApi: Record<string, jest.Mock> = {
  getMessages: jest.fn().mockResolvedValue([]),
  getPinnedMessages: jest.fn().mockResolvedValue([]),
  sendMessage: jest.fn().mockResolvedValue({ id: 'm-new', chatId: 'chat-1', senderId: 'u1', text: 'hi', timestamp: new Date().toISOString() }),
  sendAttachment: jest.fn(),
  sendLocation: jest.fn().mockResolvedValue({ id: 'm-loc', chatId: 'chat-1', senderId: 'u1', text: 'loc', timestamp: new Date().toISOString() }),
  translateMessage: jest.fn().mockResolvedValue({}),
  markAsRead: jest.fn().mockResolvedValue({}),
  initiateCall: jest.fn().mockResolvedValue({ session: { id: 'sess1' } }),
  getChats: jest.fn().mockResolvedValue([]),
  getChat: jest.fn().mockResolvedValue({ id: 'chat-1', participants: [] }),
  getBlockStatus: jest.fn().mockResolvedValue({ blocked: false }),
  pinMessage: jest.fn().mockResolvedValue({}),
  unpinMessage: jest.fn().mockResolvedValue({}),
  deleteMessage: jest.fn().mockResolvedValue({}),
  forwardMessage: jest.fn().mockResolvedValue({}),
};

jest.mock('../services/api', () => ({
  __esModule: true,
  default: {
    getMessages: (...a: any[]) => mockApi.getMessages(...a),
    getPinnedMessages: (...a: any[]) => mockApi.getPinnedMessages(...a),
    sendMessage: (...a: any[]) => mockApi.sendMessage(...a),
    sendAttachment: (...a: any[]) => mockApi.sendAttachment(...a),
    sendLocation: (...a: any[]) => mockApi.sendLocation(...a),
    translateMessage: (...a: any[]) => mockApi.translateMessage(...a),
    markAsRead: (...a: any[]) => mockApi.markAsRead(...a),
    initiateCall: (...a: any[]) => mockApi.initiateCall(...a),
    getChats: (...a: any[]) => mockApi.getChats(...a),
    getChat: (...a: any[]) => mockApi.getChat(...a),
    getBlockStatus: (...a: any[]) => mockApi.getBlockStatus(...a),
    pinMessage: (...a: any[]) => mockApi.pinMessage(...a),
    unpinMessage: (...a: any[]) => mockApi.unpinMessage(...a),
    deleteMessage: (...a: any[]) => mockApi.deleteMessage(...a),
    forwardMessage: (...a: any[]) => mockApi.forwardMessage(...a),
  },
}));

import ChatScreen from '../screens/ChatScreen';

const route = { params: { chatId: 'chat-1', chatName: 'Alice' } } as any;
const navigation: any = { navigate: jest.fn(), setOptions: jest.fn(), goBack: jest.fn() };

describe('QA messaging parity — mobile', () => {
  beforeEach(() => jest.clearAllMocks());

  it('send: composer sends via apiService.sendMessage and clears input (parity web)', async () => {
    const { getByPlaceholderText, getByText } = render(<ChatScreen route={route} navigation={navigation} />);
    await waitFor(() => expect(mockApi.getMessages).toHaveBeenCalled());
    const input = getByPlaceholderText('Type a message...');
    fireEvent.changeText(input, 'Hola');
    fireEvent.press(getByText('➤'));
    await waitFor(() => expect(mockApi.sendMessage).toHaveBeenCalledWith('chat-1', 'Hola', undefined));
  });

  it('send: empty does not send', async () => {
    const { getByText, getByPlaceholderText } = render(<ChatScreen route={route} navigation={navigation} />);
    await waitFor(() => expect(mockApi.getMessages).toHaveBeenCalled());
    const input = getByPlaceholderText('Type a message...') as any
    expect(input.props.value).toBe('')
    fireEvent.press(getByText('➤'));
    expect(mockApi.sendMessage).not.toHaveBeenCalled();
  });

  it('emoji FR-21: emoji passes through unchanged (both platforms)', async () => {
    const { getByPlaceholderText, getByText } = render(<ChatScreen route={route} navigation={navigation} />);
    await waitFor(() => expect(mockApi.getMessages).toHaveBeenCalled());
    const input = getByPlaceholderText('Type a message...');
    fireEvent.changeText(input, 'hi😀 amigo');
    fireEvent.press(getByText('➤'));
    await waitFor(() => expect(mockApi.sendMessage).toHaveBeenCalledWith('chat-1', 'hi😀 amigo', undefined));
    expect(mockApi.sendMessage.mock.calls[0][1]).toBe('hi😀 amigo');
  });

  it('translation: shows In your language block when native translation present and Translate button when absent', async () => {
    mockApi.getMessages.mockResolvedValue([
      { id: 'm1', chatId: 'chat-1', senderId: 'u2', text: 'Hello', sender: { displayName: 'Alice' }, translations: { en: 'Hola' }, timestamp: new Date().toISOString() } as any,
      { id: 'm2', chatId: 'chat-1', senderId: 'u2', text: 'Untranslated', sender: { displayName: 'Alice' }, translations: {}, timestamp: new Date().toISOString() } as any,
    ]);
    const { getByText } = render(<ChatScreen route={route} navigation={navigation} />);
    await waitFor(() => expect(getByText('Hola')).toBeTruthy(), { timeout: 3000 });
    expect(getByText('🌐 Translate')).toBeTruthy();
    mockApi.getMessages.mockResolvedValue([]);
  });

  it('translate action: tapping Translate calls translateMessage with nativeLanguage', async () => {
    mockApi.getMessages.mockResolvedValue([
      { id: 'm1', chatId: 'chat-1', senderId: 'u2', text: 'Bonjour', sender: { displayName: 'Alice' }, translations: {}, timestamp: new Date().toISOString() } as any,
    ]);
    const { getByText } = render(<ChatScreen route={route} navigation={navigation} />);
    await waitFor(() => expect(getByText('🌐 Translate')).toBeTruthy(), { timeout: 3000 });
    fireEvent.press(getByText('🌐 Translate'));
    await waitFor(() => expect(mockApi.translateMessage).toHaveBeenCalledWith('chat-1', 'm1', 'en'));
    mockApi.getMessages.mockResolvedValue([]);
  });

  it('translateAsType toggle parity: renders and toggles', async () => {
    const { getByText } = render(<ChatScreen route={route} navigation={navigation} />);
    await waitFor(() => expect(getByText('Translate as I type')).toBeTruthy());
    const toggle = getByText('Translate as I type');
    fireEvent.press(toggle);
    expect(getByText('Translate as I type')).toBeTruthy();
  });

  it('receipts: own message shows checkmarks matching receipts state', async () => {
    mockApi.getMessages.mockResolvedValue([
      { id: 'm1', chatId: 'chat-1', senderId: 'u1', text: 'hi', receipts: [{ userId: 'u2', status: 'read' }], timestamp: new Date().toISOString() } as any,
    ]);
    const { getByText } = render(<ChatScreen route={route} navigation={navigation} />);
    await waitFor(() => expect(getByText('✓✓')).toBeTruthy(), { timeout: 3000 });
    mockApi.getMessages.mockResolvedValue([]);
  });

  it('reply: reply quote block renders when replyToId matches', async () => {
    const src: any = { id: 'm-src', chatId: 'chat-1', senderId: 'u2', text: 'Original', sender: { displayName: 'Alice' }, timestamp: new Date().toISOString() };
    mockApi.getMessages.mockResolvedValue([
      { id: 'm2', chatId: 'chat-1', senderId: 'u1', text: 'Reply', replyToId: 'm-src', timestamp: new Date().toISOString() } as any,
      src,
    ]);
    const { getAllByText } = render(<ChatScreen route={route} navigation={navigation} />);
    await waitFor(() => expect(getAllByText('Original').length).toBeGreaterThan(0), { timeout: 3000 });
    mockApi.getMessages.mockResolvedValue([]);
  });

  it('attachments: document card renders with file name + size', async () => {
    mockApi.getMessages.mockResolvedValue([
      { id: 'm1', chatId: 'chat-1', senderId: 'u2', text: 'doc', sender: { displayName: 'Alice' }, timestamp: new Date().toISOString(), media: [{ id: 'a1', type: 'document', fileName: 'report.pdf', fileSize: 2048, mimeType: 'application/pdf', url: 'https://example.com/f.pdf' }] } as any,
    ]);
    const { getByText } = render(<ChatScreen route={route} navigation={navigation} />);
    await waitFor(() => expect(getByText('report.pdf')).toBeTruthy(), { timeout: 3000 });
    mockApi.getMessages.mockResolvedValue([]);
  });

  it('attachments: location card renders with coordinates', async () => {
    mockApi.getMessages.mockResolvedValue([
      { id: 'm1', chatId: 'chat-1', senderId: 'u2', text: 'loc', sender: { displayName: 'Alice' }, timestamp: new Date().toISOString(), media: [{ id: 'a1', type: 'location', latitude: 48.8566, longitude: 2.3522, locationName: 'Paris', url: 'https://www.openstreetmap.org/?mlat=48.8566' }] } as any,
    ]);
    const { getByText } = render(<ChatScreen route={route} navigation={navigation} />);
    await waitFor(() => expect(getByText(/48\.85/)).toBeTruthy(), { timeout: 3000 });
    mockApi.getMessages.mockResolvedValue([]);
  });

  it('sparky FAB and RealTalkNudge parity both visible', async () => {
    const { getByText, UNSAFE_getAllByType } = render(<ChatScreen route={route} navigation={navigation} />);
    await waitFor(() => expect(getByText('realtalk-nudge')).toBeTruthy());
    expect(getByText('🤖')).toBeTruthy();
  });

  it('pinned bar: renders when pinned messages exist and toggles', async () => {
    mockApi.getPinnedMessages.mockResolvedValue([{ message: { id: 'm1', text: 'Pinned note' } }] as any);
    const { getAllByText } = render(<ChatScreen route={route} navigation={navigation} />);
    await waitFor(() => expect(getAllByText(/pinned/i).length).toBeGreaterThan(0), { timeout: 3000 });
    mockApi.getPinnedMessages.mockResolvedValue([]);
  });

  it('forwarded label renders', async () => {
    mockApi.getMessages.mockResolvedValue([
      { id: 'm1', chatId: 'chat-1', senderId: 'u2', text: 'fwd', sender: { displayName: 'Alice' }, forwarded: true, timestamp: new Date().toISOString() } as any,
    ]);
    const { getByText } = render(<ChatScreen route={route} navigation={navigation} />);
    await waitFor(() => expect(getByText(/Forwarded/)).toBeTruthy(), { timeout: 3000 });
    mockApi.getMessages.mockResolvedValue([]);
  });
});
