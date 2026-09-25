import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from 'react';
import storage from '../utils/storage';
import { en, type Strings } from './en';
import { es } from './es';

// Bundled UI locales. The in-app language selector lists exactly these —
// never more (showing a language we fall back to English for is deceptive).
export const BUNDLED_LOCALES = ['en', 'es'] as const;
export type Locale = (typeof BUNDLED_LOCALES)[number];

const TABLES: Record<Locale, Strings> = { en, es };
const UI_LANGUAGE_KEY = 'ui_language';

let current: Locale = 'en';
// True once the user has explicitly picked a language (this session or a
// persisted one). Explicit choice wins over profile/device in the chain.
let hasExplicitChoice = false;
const listeners = new Set<() => void>();
const notify = () => listeners.forEach((fn) => { try { fn(); } catch {} });

export function normalizeLocale(input?: string | null): Locale | null {
  if (!input) return null;
  const base = input.trim().toLowerCase().split(/[-_]/)[0];
  return (BUNDLED_LOCALES as readonly string[]).includes(base) ? (base as Locale) : null;
}

// Device locale with zero native dependencies (Hermes ships full Intl).
// Used pre-auth and as a fallback; profile native wins after login.
export function deviceLocale(): Locale | null {
  try {
    const tag = new Intl.DateTimeFormat().resolvedOptions().locale;
    return normalizeLocale(tag);
  } catch {
    return null;
  }
}

function lookup(table: unknown, path: string): string | undefined {
  const v = path.split('.').reduce<any>((o, k) => (o == null ? o : o[k]), table);
  return typeof v === 'string' ? v : undefined;
}

// Imperative lookup for non-render paths (Alert.alert, error toasts).
// Falls back per-key to English, then to the key itself (never crashes).
export function t(path: string, vars?: Record<string, string | number | null | undefined>): string {
  const raw = lookup(TABLES[current], path) ?? lookup(TABLES.en, path) ?? path;
  if (!vars) return raw;
  return raw.replace(/\{\{(\w+)\}\}/g, (m, k: string) =>
    vars[k] != null ? String(vars[k]) : m
  );
}

export function getLocale(): Locale {
  return current;
}

async function readStoredChoice(): Promise<Locale | null> {
  try {
    return normalizeLocale(await storage.getItem(UI_LANGUAGE_KEY));
  } catch {
    return null;
  }
}

// Explicit choice (language selector): persists and wins over everything.
export async function setExplicitLanguage(lang: Locale): Promise<void> {
  current = lang;
  hasExplicitChoice = true;
  notify();
  try {
    await storage.setItem(UI_LANGUAGE_KEY, lang);
  } catch {}
}

// Implicit choice (login / register / session restore / profile save):
// profile native → device → English. Never overrides an explicit choice and
// never persists (a future explicit pick still wins).
export async function applyImplicitLanguage(nativeLang?: string | null): Promise<void> {
  if (hasExplicitChoice) return;
  const stored = await readStoredChoice();
  if (stored) {
    hasExplicitChoice = true;
    if (stored !== current) {
      current = stored;
      notify();
    }
    return;
  }
  const next = normalizeLocale(nativeLang) ?? deviceLocale() ?? 'en';
  if (next !== current) {
    current = next;
    notify();
  }
}

interface LangCtx {
  lang: Locale;
  setLanguage: (l: Locale) => void;
}

const LanguageContext = createContext<LangCtx>({ lang: 'en', setLanguage: () => {} });

export function LanguageProvider({ children }: { children: React.ReactNode }) {
  const [lang, setLangState] = useState<Locale>(current);

  useEffect(() => {
    const sync = () => setLangState(current);
    listeners.add(sync);
    // Boot: explicit stored choice wins; otherwise device locale (pre-auth).
    // Post-auth callers refine via applyImplicitLanguage().
    (async () => {
      const stored = await readStoredChoice();
      if (stored) {
        hasExplicitChoice = true;
        if (stored !== current) {
          current = stored;
          notify();
        }
        return;
      }
      const dev = deviceLocale();
      if (dev && dev !== current) {
        current = dev;
        notify();
      }
    })();
    return () => {
      listeners.delete(sync);
    };
  }, []);

  const setLanguage = useCallback((l: Locale) => {
    void setExplicitLanguage(l);
  }, []);

  const value = useMemo(() => ({ lang, setLanguage }), [lang, setLanguage]);
  return <LanguageContext.Provider value={value}>{children}</LanguageContext.Provider>;
}

// Render path: typed table for the active locale. Components re-render on
// language change through context (no remount, no nav-state loss).
export function useStrings(): Strings {
  return TABLES[useContext(LanguageContext).lang];
}

// Locale code for non-string locale-sensitive APIs (toLocaleDateString, …).
export function useAppLocale(): Locale {
  return useContext(LanguageContext).lang;
}
