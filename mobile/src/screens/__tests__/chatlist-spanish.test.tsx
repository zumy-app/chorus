import React from 'react';
import { render, waitFor } from '@testing-library/react-native';

jest.mock('../../utils/storage', () => ({
  __esModule: true,
  default: {
    getItem: jest.fn((k: string) => {
      if (k === 'user')
        return Promise.resolve(
          JSON.stringify({ id: 'bob-id', nativeLanguage: 'es', targetLanguages: ['en'] })
        );
      return Promise.resolve(null);
    }),
    setItem: jest.fn(() => Promise.resolve()),
    removeItem: jest.fn(() => Promise.resolve()),
  },
}));

jest.mock('../../services/websocket', () => ({
  __esModule: true,
  default: {
    connect: jest.fn(),
    onMessage: jest.fn(() => () => {}),
    onReconnect: jest.fn(() => () => {}),
    send: jest.fn(),
    sendTyping: jest.fn(),
  },
}));

jest.mock('../../services/api', () => ({
  __esModule: true,
  default: {
    getChats: jest.fn().mockResolvedValue([]),
  },
}));

import ChatListScreen from '../ChatListScreen';
import { setExplicitLanguage } from '../../i18n';
import { LanguageProvider } from '../../i18n';

const navigation: any = { navigate: jest.fn(), setOptions: jest.fn() };

describe('ChatListScreen in Spanish', () => {
  beforeEach(async () => {
    jest.clearAllMocks();
    await setExplicitLanguage('es');
  });

  afterEach(async () => {
    await setExplicitLanguage('en');
  });

  it('renders chrome in Spanish for an es user', async () => {
    const ui = render(
      <LanguageProvider>
        <ChatListScreen navigation={navigation} />
      </LanguageProvider>
    );
    await waitFor(() => expect(ui.getByText('CONVERSACIONES ACTIVAS')).toBeTruthy());
    expect(ui.getByText('Buscar mensajes, archivos o personas...')).toBeTruthy();
    expect(ui.getByText('Repaso diario')).toBeTruthy();
    expect(ui.getByText('Sin chats aún')).toBeTruthy();
    expect(ui.getByText('¡Inicia una conversación!')).toBeTruthy();
  });
});
