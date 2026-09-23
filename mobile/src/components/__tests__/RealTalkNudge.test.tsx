import React from 'react';
import { render, fireEvent, waitFor } from '@testing-library/react-native';

jest.mock('../../utils/storage', () => ({
  __esModule: true,
  default: {
    getItem: jest.fn(() => Promise.resolve(JSON.stringify({ id: 'u1', nativeLanguage: 'en', targetLanguages: ['es'] }))),
    setItem: jest.fn(() => Promise.resolve()),
    removeItem: jest.fn(() => Promise.resolve()),
  },
}));

jest.mock('../../services/api', () => ({
  __esModule: true,
  default: {
    getRealTalkPrompts: jest.fn().mockResolvedValue([
      {
        id: 'p1',
        category: 'Task-Based',
        text: 'Politely state your perspective when offering a different opinion.',
        targetPhrase: 'Desde mi punto de vista, depende bastante.',
        whyUseful: 'Great for debates.',
      },
    ]),
    markRealTalkUsed: jest.fn().mockResolvedValue({}),
  },
}));

import RealTalkNudge from '../RealTalkNudge';
import apiService from '../../services/api';

const mockApi = apiService as unknown as { getRealTalkPrompts: jest.Mock; markRealTalkUsed: jest.Mock };

describe('RealTalkNudge', () => {
  beforeEach(() => jest.clearAllMocks());

  it('headlines the sendable target phrase with the instruction as context', async () => {
    const { getByText, queryByText } = render(<RealTalkNudge chatId="chat-1" onSendToInput={jest.fn()} />);
    await waitFor(() => expect(getByText(/Desde mi punto de vista/)).toBeTruthy());
    // Instruction visible as helper caption…
    expect(getByText(/Politely state your perspective/)).toBeTruthy();
    expect(getByText(/Great for debates/)).toBeTruthy();
    // …but never as the quoted sendable message.
    expect(queryByText(/^“Politely state your perspective.*”$/)).toBeNull();
  });

  it('Send to Input inserts the target phrase, never the instruction', async () => {
    const onSendToInput = jest.fn();
    const { getByText } = render(<RealTalkNudge chatId="chat-1" onSendToInput={onSendToInput} />);
    await waitFor(() => expect(getByText(/Desde mi punto de vista/)).toBeTruthy());
    fireEvent.press(getByText('Send to Input'));
    await waitFor(() => expect(mockApi.markRealTalkUsed).toHaveBeenCalledWith('p1'));
    await waitFor(() => expect(onSendToInput).toHaveBeenCalledWith('Desde mi punto de vista, depende bastante.'));
  });

  it('falls back to text when a prompt has no target phrase (static trio)', async () => {
    mockApi.getRealTalkPrompts.mockResolvedValueOnce([
      { id: 'rt-1', category: 'Icebreakers', text: 'What do you usually do when you wake up?' },
    ]);
    const onSendToInput = jest.fn();
    const { getByText } = render(<RealTalkNudge chatId="chat-1" onSendToInput={onSendToInput} />);
    await waitFor(() => expect(getByText(/What do you usually do/)).toBeTruthy());
    fireEvent.press(getByText('Send to Input'));
    await waitFor(() => expect(onSendToInput).toHaveBeenCalledWith('What do you usually do when you wake up?'));
  });
});
