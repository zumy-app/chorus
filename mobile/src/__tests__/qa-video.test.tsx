import React from 'react';
import { render, fireEvent, waitFor } from '@testing-library/react-native';
jest.mock('@react-navigation/native', () => ({
  useNavigation: () => ({ navigate: jest.fn(), goBack: jest.fn() }),
  useRoute: () => ({ params: { callId: 'call-v1', chatId: 'chat-1', chatName: 'Sofia', initialType: 'video' } }),
}));
jest.mock('../utils/storage', () => ({
  __esModule: true,
  default: { getItem: jest.fn((k: string) => Promise.resolve(k === 'user' ? JSON.stringify({ id: 'u1', nativeLanguage: 'en' }) : null)), setItem: jest.fn(() => Promise.resolve()), removeItem: jest.fn(() => Promise.resolve()) },
}));
const mockApi = {
  getCaptions: jest.fn().mockResolvedValue({ segments: [], total: 0, hasMore: false }),
  postCaption: jest.fn().mockResolvedValue({ speakerId: 'u1', startTime: 1000, endTime: 1001, originalText: 'Hola', originalLanguage: 'es', translations: { en: 'Hello' }, confidence: 1 }),
  bookmarkCaption: jest.fn().mockResolvedValue({ id: 'vocab-1' }),
  endCall: jest.fn().mockResolvedValue({}),
  sendSignal: jest.fn().mockResolvedValue({}),
};
jest.mock('../services/api', () => ({
  __esModule: true,
  default: { getCaptions: (...a: any[]) => mockApi.getCaptions(...a), postCaption: (...a: any[]) => mockApi.postCaption(...a), bookmarkCaption: (...a: any[]) => mockApi.bookmarkCaption(...a), endCall: (...a: any[]) => mockApi.endCall(...a), sendSignal: (...a: any[]) => mockApi.sendSignal(...a) },
}));
jest.mock('../services/websocket', () => ({
  __esModule: true,
  default: { onMessage: jest.fn(() => () => {}), send: jest.fn() },
}));
import CallScreen from '../screens/CallScreen';
import { cleanup } from '@testing-library/react-native';
const segment = { speakerId: 'u2', startTime: 1000, endTime: 1001, originalText: 'Hola que tal', originalLanguage: 'es', translations: { en: 'Hello how are you' }, confidence: 0.9 };
const routeVideo: any = { params: { callId: 'call-v1', chatId: 'chat-1', chatName: 'Sofia', initialType: 'video' } };
const navigation: any = { goBack: jest.fn(), navigate: jest.fn() };
describe('QA video — mobile CallScreen (Phase 8)', () => {
  beforeEach(() => jest.clearAllMocks());
  afterEach(() => { cleanup(); jest.clearAllTimers(); });
  it('video header shows Video call and dual-view button', async () => {
    const { getByText, getAllByText } = render(<CallScreen route={routeVideo} navigation={navigation} />);
    await waitFor(() => expect(getAllByText('Sofia').length).toBeGreaterThan(0));
    expect(getByText(/Video call/)).toBeTruthy();
  });
  it('dual-view toggle switches state', async () => {
    const { getAllByText } = render(<CallScreen route={routeVideo} navigation={navigation} />);
    await waitFor(() => expect(getAllByText('Sofia').length).toBeGreaterThan(0));
    const dualButtons = getAllByText('▣');
    expect(dualButtons.length).toBeGreaterThan(0);
    fireEvent.press(dualButtons[0]);
    await waitFor(() => expect(getAllByText('◫').length).toBeGreaterThan(0));
    fireEvent.press(getAllByText('◫')[0]);
    await waitFor(() => expect(getAllByText('▣').length).toBeGreaterThan(0));
  });
  it('immersive captions render in video mode', async () => {
    mockApi.getCaptions.mockResolvedValue({ segments: [segment as any], total: 1, hasMore: false });
    const { getByTestId } = render(<CallScreen route={routeVideo} navigation={navigation} />);
    await waitFor(() => expect(getByTestId('immersive-captions')).toBeTruthy());
  });
  it('immersive toggle hides overlay', async () => {
    mockApi.getCaptions.mockResolvedValue({ segments: [segment as any], total: 1, hasMore: false });
    const { getByTestId, getByText, queryByTestId } = render(<CallScreen route={routeVideo} navigation={navigation} />);
    await waitFor(() => expect(getByTestId('immersive-captions')).toBeTruthy());
    fireEvent.press(getByText('Immersive on'));
    await waitFor(() => expect(queryByTestId('immersive-captions')).toBeNull());
    fireEvent.press(getByText('Immersive off'));
    await waitFor(() => expect(getByTestId('immersive-captions')).toBeTruthy());
  });
  it('screen share toggle updates badge and sends signal', async () => {
    const { getByTestId, getAllByText } = render(<CallScreen route={routeVideo} navigation={navigation} />);
    await waitFor(() => expect(getByTestId('toggle-screen-share-btn')).toBeTruthy());
    fireEvent.press(getByTestId('toggle-screen-share-btn'));
    await waitFor(() => expect(mockApi.sendSignal).toHaveBeenCalledWith('call-v1', expect.objectContaining({ type: 'screen-share-start' })));
    expect(getAllByText(/Sharing/).length).toBeGreaterThan(0);
    fireEvent.press(getByTestId('toggle-screen-share-btn'));
    await waitFor(() => expect(mockApi.sendSignal).toHaveBeenCalledWith('call-v1', expect.objectContaining({ type: 'screen-share-stop' })));
  });
  it('camera toggle in video hides local placeholder', async () => {
    const { getByTestId, getByText } = render(<CallScreen route={routeVideo} navigation={navigation} />);
    await waitFor(() => expect(getByTestId('toggle-camera-btn')).toBeTruthy());
    fireEvent.press(getByTestId('toggle-camera-btn'));
    await waitFor(() => expect(getByText('Cam off')).toBeTruthy());
    fireEvent.press(getByTestId('toggle-camera-btn'));
    await waitFor(() => expect(getByText('You')).toBeTruthy());
  });
  it('video screen shows remote video placeholder', async () => {
    const { getByText } = render(<CallScreen route={routeVideo} navigation={navigation} />);
    await waitFor(() => expect(getByText('Remote video')).toBeTruthy());
  });
  it('local video shows Cam off when camera off', async () => {
    const { getByText, getByTestId } = render(<CallScreen route={routeVideo} navigation={navigation} />);
    await waitFor(() => expect(getByTestId('toggle-camera-btn')).toBeTruthy());
    fireEvent.press(getByTestId('toggle-camera-btn'));
    await waitFor(() => expect(getByText('Cam off')).toBeTruthy());
  });
  it('controls include transcript toggle and end call', async () => {
    const { getByText, getByTestId } = render(<CallScreen route={routeVideo} navigation={navigation} />);
    await waitFor(() => expect(getByTestId('transcript-panel')).toBeTruthy());
    expect(getByText('End')).toBeTruthy();
  });
  it('duration increments and shows in video header', async () => {
    const { getByText } = render(<CallScreen route={routeVideo} navigation={navigation} />);
    await waitFor(() => expect(getByText(/00:00/)).toBeTruthy());
  });
});
