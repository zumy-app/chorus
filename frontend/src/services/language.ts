// Language detection utility — detects user's preferred language from browser settings
// Uses the Navigator.language API which respects the user's OS/browser language preferences

export const SUPPORTED_LANGUAGES = [
  { code: 'en', name: 'English', nativeName: 'English', flag: '🇬🇧' },
  { code: 'es', name: 'Spanish', nativeName: 'Español', flag: '🇪🇸' },
  { code: 'fr', name: 'French', nativeName: 'Français', flag: '🇫🇷' },
  { code: 'de', name: 'German', nativeName: 'Deutsch', flag: '🇩🇪' },
  { code: 'it', name: 'Italian', nativeName: 'Italiano', flag: '🇮🇹' },
  { code: 'pt', name: 'Portuguese', nativeName: 'Português', flag: '🇵🇹' },
  { code: 'ja', name: 'Japanese', nativeName: '日本語', flag: '🇯🇵' },
  { code: 'ko', name: 'Korean', nativeName: '한국어', flag: '🇰🇷' },
  { code: 'zh', name: 'Chinese', nativeName: '中文', flag: '🇨🇳' },
  { code: 'ar', name: 'Arabic', nativeName: 'العربية', flag: '🇸🇦' },
  { code: 'nl', name: 'Dutch', nativeName: 'Nederlands', flag: '🇳🇱' },
  { code: 'pl', name: 'Polish', nativeName: 'Polski', flag: '🇵🇱' },
  { code: 'ru', name: 'Russian', nativeName: 'Русский', flag: '🇷🇺' },
  { code: 'sv', name: 'Swedish', nativeName: 'Svenska', flag: '🇸🇪' },
]

// Map browser language codes to our supported codes
const LANGUAGE_MAP: Record<string, string> = {
  'en': 'en', 'en-US': 'en', 'en-GB': 'en', 'en-AU': 'en',
  'es': 'es', 'es-ES': 'es', 'es-MX': 'es', 'es-AR': 'es',
  'fr': 'fr', 'fr-FR': 'fr', 'fr-CA': 'fr',
  'de': 'de', 'de-DE': 'de', 'de-AT': 'de', 'de-CH': 'de',
  'it': 'it', 'it-IT': 'it',
  'pt': 'pt', 'pt-PT': 'pt', 'pt-BR': 'pt',
  'ja': 'ja', 'ja-JP': 'ja',
  'ko': 'ko', 'ko-KR': 'ko',
  'zh': 'zh', 'zh-CN': 'zh', 'zh-TW': 'zh', 'zh-HK': 'zh',
  'ar': 'ar', 'ar-SA': 'ar', 'ar-AE': 'ar', 'ar-EG': 'ar',
  'nl': 'nl', 'nl-NL': 'nl', 'nl-BE': 'nl',
  'pl': 'pl', 'pl-PL': 'pl',
  'ru': 'ru', 'ru-RU': 'ru',
  'sv': 'sv', 'sv-SE': 'sv',
}

/**
 * Detects the user's preferred language from browser settings.
 * Uses navigator.language (primary) or navigator.languages (fallback).
 * Returns a 2-letter language code matching our supported languages.
 * Defaults to 'en' if detection fails or language is not supported.
 */
export function detectBrowserLanguage(): string {
  if (typeof navigator === 'undefined') return 'en'

  // Get the user's preferred languages (in order of preference)
  const browserLangs = navigator.languages || [navigator.language]

  for (const lang of browserLangs) {
    // Try exact match first (e.g., 'es' → 'es')
    if (LANGUAGE_MAP[lang]) {
      return LANGUAGE_MAP[lang]
    }
    // Try primary language match (e.g., 'es-MX' → 'es')
    const primary = lang.split('-')[0]
    if (LANGUAGE_MAP[primary]) {
      return LANGUAGE_MAP[primary]
    }
  }

  return 'en'
}

/**
 * Returns the native name of a language code.
 * e.g., 'es' → 'Español', 'fr' → 'Français'
 */
export function getNativeLanguageName(code: string): string {
  const lang = SUPPORTED_LANGUAGES.find(l => l.code === code)
  return lang?.nativeName || 'English'
}

/**
 * Returns the English name of a language code.
 */
export function getLanguageName(code: string): string {
  const lang = SUPPORTED_LANGUAGES.find(l => l.code === code)
  return lang?.name || 'English'
}
