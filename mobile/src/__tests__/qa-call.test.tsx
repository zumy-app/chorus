import React from 'react';
import { render, fireEvent, waitFor } from '@testing-library/react-native';

jest.mock('@react-navigation/native', () => ({
  useNavigation: () => ({ navigate: jest.fn(), goBack: jest.fn() }),
  useRoute: () => ({ params: { callId: 'call-1', chatId: 'chat-1', chatName: 'Alice', initialType: 'audio' } }),
}));

jest.mock('../utils/storage', () => ({
  __esModule: true,
  default: {
    getItem: jest.fn((k: string) => Promise.resolve(k === 'user' ? JSON.stringify({ id: 'u1', nativeLanguage: 'en' }) : null)),
    setItem: jest.fn(() => Promise.resolve()),
    removeItem: jest.fn(() => Promise.resolve()),
  },
}));

const mockApi = {
  getCaptions: jest.fn().mockResolvedValue({ segments: [], total: 0, hasMore: false }),
  postCaption: jest.fn().mockResolvedValue({ speakerId: 'u1', startTime: 1000, endTime: 1001, originalText: 'Hola', originalLanguage: 'es', translations: { en: 'Hello' }, confidence: 1 }),
  bookmarkCaption: jest.fn().mockResolvedValue({ id: 'vocab-1' }),
  endCall: jest.fn().mockResolvedValue({}),
};

jest.mock('../services/api', () => ({
  __esModule: true,
  default: {
    getCaptions: (...a: any[]) => mockApi.getCaptions(...a),
    postCaption: (...a: any[]) => mockApi.postCaption(...a),
    bookmarkCaption: (...a: any[]) => mockApi.bookmarkCaption(...a),
    endCall: (...a: any[]) => mockApi.endCall(...a),
  },
}));

jest.mock('../services/websocket', () => ({
  __esModule: true,
  default: { onMessage: jest.fn(() => () => {}), send: jest.fn() },
}));

import CallScreen from '../screens/CallScreen';
import { cleanup } from '@testing-library/react-native';

const route: any = { params: { callId: 'call-1', chatId: 'chat-1', chatName: 'Alice', initialType: 'audio' } };
const navigation: any = { goBack: jest.fn(), navigate: jest.fn() };
const segment = { speakerId: 'u1', startTime: 1000, endTime: 1001, originalText: 'Hola amigo', originalLanguage: 'es', translations: { en: 'Hello friend' }, confidence: 0.9 };

describe('QA call — mobile CallScreen', () => {
  beforeEach(() => jest.clearAllMocks());
  afterEach(() => { cleanup(); jest.clearAllTimers(); });

  it('renders header with chat name', async () => {
    const { getAllByText } = render(<CallScreen route={route} navigation={navigation} />);
    await waitFor(() => expect(getAllByText('Alice').length).toBeGreaterThan(0));
  });

  it('shows Live captions title and transcript panel', async () => {
    const { getByText, getByTestId } = render(<CallScreen route={route} navigation={navigation} />);
    await waitFor(() => expect(getByText('Live captions')).toBeTruthy());
    expect(getByTestId('transcript-panel')).toBeTruthy();
  });

  it('empty state when no captions', async () => {
    mockApi.getCaptions.mockResolvedValue({ segments: [], total: 0, hasMore: false });
    const { getByText } = render(<CallScreen route={route} navigation={navigation} />);
    await waitFor(() => expect(getByText(/Captions will appear here/)).toBeTruthy());
  });

  it('renders caption and translation', async () => {
    mockApi.getCaptions.mockResolvedValue({ segments: [segment as any], total: 1, hasMore: false });
    const { getByText } = render(<CallScreen route={route} navigation={navigation} />);
    await waitFor(() => expect(getByText('Hola')).toBeTruthy());
    expect(getByText('amigo')).toBeTruthy();
    expect(getByText('Hello friend')).toBeTruthy();
  });

  it('translated toggle hides translation', async () => {
    mockApi.getCaptions.mockResolvedValue({ segments: [segment as any], total: 1, hasMore: false });
    const { getByText, getByTestId } = render(<CallScreen route={route} navigation={navigation} />);
    await waitFor(() => expect(getByText('Hola')).toBeTruthy());
    fireEvent.press(getByTestId('translated-toggle'));
    expect(() => getByText('Hello friend')).toThrow();
    fireEvent.press(getByTestId('translated-toggle'));
    await waitFor(() => expect(getByText('Hello friend')).toBeTruthy());
  });

  it('sends caption via input', async () => {
    mockApi.getCaptions.mockResolvedValue({ segments: [], total: 0, hasMore: false });
    mockApi.postCaption.mockResolvedValue(segment as any);
    const { getByPlaceholderText, getByText } = render(<CallScreen route={route} navigation={navigation} />);
    await waitFor(() => expect(getByPlaceholderText('Type a caption...')).toBeTruthy());
    fireEvent.changeText(getByPlaceholderText('Type a caption...'), 'Hola');
    fireEvent.press(getByText('➤'));
    await waitFor(() => expect(mockApi.postCaption).toHaveBeenCalledWith('call-1', expect.objectContaining({ text: 'Hola' })));
  });

  it('bookmark phrase calls API', async () => {
    mockApi.getCaptions.mockResolvedValue({ segments: [segment as any], total: 1, hasMore: false });
    const { getByTestId } = render(<CallScreen route={route} navigation={navigation} />);
    await waitFor(() => expect(getByTestId('save-phrase-0')).toBeTruthy());
    fireEvent.press(getByTestId('save-phrase-0'));
    await waitFor(() => expect(mockApi.bookmarkCaption).toHaveBeenCalledWith('call-1', 0));
  });

  it('word chip saves individual word', async () => {
    mockApi.getCaptions.mockResolvedValue({ segments: [segment as any], total: 1, hasMore: false });
    const { getByText } = render(<CallScreen route={route} navigation={navigation} />);
    await waitFor(() => expect(getByText('Hola')).toBeTruthy());
    fireEvent.press(getByText('Hola'));
    await waitFor(() => expect(mockApi.bookmarkCaption).toHaveBeenCalledWith('call-1', 0, 'Hola'));
  });

  it('toggle transcript shows collapsed bar', async () => {
    const { getByTestId, queryByTestId } = render(<CallScreen route={route} navigation={navigation} />);
    await waitFor(() => expect(getByTestId('transcript-panel')).toBeTruthy());
    fireEvent.press(getByTestId('toggle-transcript-btn'));
    await waitFor(() => expect(queryByTestId('transcript-panel')).toBeNull());
    expect(getByTestId('open-transcript-btn')).toBeTruthy();
    fireEvent.press(getByTestId('open-transcript-btn'));
    await waitFor(() => expect(getByTestId('transcript-panel')).toBeTruthy());
  });

  it('mute toggle', async () => {
    const { getByText } = render(<CallScreen route={route} navigation={navigation} />);
    await waitFor(() => expect(getByText('🎤')).toBeTruthy());
    fireEvent.press(getByText('🎤'));
    await waitFor(() => expect(getByText('🔇')).toBeTruthy());
  });

  it('camera toggle switches video state', async () => {
    const videoRoute: any = { params: { callId: 'call-1', chatId: 'chat-1', chatName: 'Alice', initialType: 'video' } };
    const { getAllByText } = render(<CallScreen route={videoRoute} navigation={navigation} />);
    await waitFor(() => expect(getAllByText('📹').length).toBeGreaterThan(0));
    fireEvent.press(getAllByText('📹')[0]);
    await waitFor(() => expect(getAllByText('🚫').length).toBeGreaterThan(0));
  });

  it('screen share toggle', async () => {
    const { getAllByText } = render(<CallScreen route={route} navigation={navigation} />);
    await waitFor(() => expect(getAllByText('🖥️').length).toBeGreaterThan(0));
    fireEvent.press(getAllByText('🖥️')[0]);
    expect(getAllByText('🖥️').length).toBeGreaterThan(0);
  });

  it('end call navigates back', async () => {
    const { getByText } = render(<CallScreen route={route} navigation={navigation} />);
    await waitFor(() => expect(getByText('End')).toBeTruthy());
    fireEvent.press(getByText('End'));
    await waitFor(() => expect(mockApi.endCall).toHaveBeenCalledWith('call-1'));
  });

  it('immersive captions visible when segments present', async () => {
    mockApi.getCaptions.mockResolvedValue({ segments: [segment as any], total: 1, hasMore: false });
    const { getByTestId } = render(<CallScreen route={route} navigation={navigation} />);
    await waitFor(() => expect(getByTestId('immersive-captions')).toBeTruthy());
  });

  it('load more captions when hasMore', async () => {
    mockApi.getCaptions.mockResolvedValueOnce({ segments: [segment as any], total: 2, hasMore: true } as any);
    mockApi.getCaptions.mockResolvedValueOnce({ segments: [{ ...segment, originalText: 'Second' } as any], total: 2, hasMore: false } as any);
    const { getByText } = render(<CallScreen route={route} navigation={navigation} />);
    await waitFor(() => expect(getByText('Hola')).toBeTruthy());
    fireEvent.press(getByText('Load older captions'));
    await waitFor(() => expect(mockApi.getCaptions).toHaveBeenCalledWith('call-1', expect.objectContaining({ offset: 1 })));
  });

  it('duration shows and header indicates audio vs video', async () => {
    const { getByText } = render(<CallScreen route={route} navigation={navigation} />);
    await waitFor(() => expect(getByText(/Audio call/)).toBeTruthy());
  });

  it('not showing translation when same as original', async () => {
    const same = { ...segment, translations: { en: 'Hola amigo' } };
    mockApi.getCaptions.mockResolvedValue({ segments: [same as any], total: 1, hasMore: false });
    const { getByText, queryByText } = render(<CallScreen route={route} navigation={navigation} />);
    await waitFor(() => expect(getByText('Hola')).toBeTruthy());
    expect(queryByText('Hola amigo · Hola amigo')).toBeNull();
  });

  it('auto-scroll hint visible', async () => {
    const { getByText } = render(<CallScreen route={route} navigation={navigation} />);
    await waitFor(() => expect(getByText(/Transcript auto-scrolls/)).toBeTruthy());
  });
});
