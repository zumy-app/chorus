import {
  normalizeLocale,
  deviceLocale,
  t,
  getLocale,
  setExplicitLanguage,
  applyImplicitLanguage,
} from '../index';
import storage from '../../utils/storage';

jest.mock('../../utils/storage', () => ({
  __esModule: true,
  default: {
    getItem: jest.fn(() => Promise.resolve(null)),
    setItem: jest.fn(() => Promise.resolve()),
    removeItem: jest.fn(() => Promise.resolve()),
  },
}));

const mockStorage = storage as unknown as {
  getItem: jest.Mock;
  setItem: jest.Mock;
  removeItem: jest.Mock;
};

describe('i18n module', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockStorage.getItem.mockResolvedValue(null);
  });

  it('normalizes locale tags to bundled locales', () => {
    expect(normalizeLocale('es')).toBe('es');
    expect(normalizeLocale('es-ES')).toBe('es');
    expect(normalizeLocale('es_MX')).toBe('es');
    expect(normalizeLocale('EN')).toBe('en');
    expect(normalizeLocale('fr')).toBeNull();
    expect(normalizeLocale('')).toBeNull();
    expect(normalizeLocale(null)).toBeNull();
  });

  it('deviceLocale never crashes and returns a Locale or null', () => {
    const loc = deviceLocale();
    expect(loc === null || loc === 'en' || loc === 'es').toBe(true);
  });

  it('t() interpolates vars and falls back to the key for unknown paths', () => {
    expect(t('chatList.bentoReviewSub', { count: 3 })).toBe('3 new vocab words');
    expect(t('does.not.exist')).toBe('does.not.exist');
  });

  it('applyImplicitLanguage adopts the profile native when nothing stored', async () => {
    await applyImplicitLanguage('es');
    expect(getLocale()).toBe('es');
    // Spanish table resolves (spot-check, not exhaustive — parity test owns that).
    expect(t('auth.loginBtn')).toBe('Iniciar sesión');
    expect(t('chatList.sectionActive')).toBe('CONVERSACIONES ACTIVAS');
  });

  it('applyImplicitLanguage respects a stored explicit choice', async () => {
    mockStorage.getItem.mockResolvedValue('en');
    await applyImplicitLanguage('es');
    expect(getLocale()).toBe('en');
  });

  it('setExplicitLanguage persists, wins, and blocks later implicit flips', async () => {
    await setExplicitLanguage('es');
    expect(getLocale()).toBe('es');
    expect(mockStorage.setItem).toHaveBeenCalledWith('ui_language', 'es');
    await applyImplicitLanguage('en');
    expect(getLocale()).toBe('es');
    await setExplicitLanguage('en');
    expect(getLocale()).toBe('en');
  });
});
