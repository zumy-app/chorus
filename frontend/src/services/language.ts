// Language detection utility — detects user's preferred language from browser settings
// Uses the Navigator.language API which respects the user's OS/browser language preferences.
//
// The supported-language catalog now lives in the shared package
// (@chorus/shared); this file retains the browser-specific detection logic.

import { SUPPORTED_LANGUAGES, getNativeLanguageName, getLanguageName } from '@chorus/shared'

export { SUPPORTED_LANGUAGES, getNativeLanguageName, getLanguageName }

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
  'af': 'af', 'af-ZA': 'af',
  'bg': 'bg', 'bg-BG': 'bg',
  'bn': 'bn', 'bn-BD': 'bn', 'bn-IN': 'bn',
  'bs': 'bs', 'bs-BA': 'bs',
  'ca': 'ca', 'ca-ES': 'ca',
  'cs': 'cs', 'cs-CZ': 'cs',
  'cy': 'cy', 'cy-GB': 'cy',
  'da': 'da', 'da-DK': 'da',
  'el': 'el', 'el-GR': 'el',
  'et': 'et', 'et-EE': 'et',
  'fa': 'fa', 'fa-IR': 'fa',
  'fi': 'fi', 'fi-FI': 'fi',
  'ga': 'ga', 'ga-IE': 'ga',
  'gl': 'gl', 'gl-ES': 'gl',
  'gu': 'gu', 'gu-IN': 'gu',
  'ha': 'ha', 'ha-NG': 'ha',
  'he': 'he', 'he-IL': 'he',
  'hi': 'hi', 'hi-IN': 'hi',
  'hr': 'hr', 'hr-HR': 'hr',
  'hu': 'hu', 'hu-HU': 'hu',
  'id': 'id', 'id-ID': 'id',
  'ig': 'ig', 'ig-NG': 'ig',
  'is': 'is', 'is-IS': 'is',
  'kk': 'kk', 'kk-KZ': 'kk',
  'km': 'km', 'km-KH': 'km',
  'kn': 'kn', 'kn-IN': 'kn',
  'ky': 'ky', 'ky-KG': 'ky',
  'lo': 'lo', 'lo-LA': 'lo',
  'lt': 'lt', 'lt-LT': 'lt',
  'lv': 'lv', 'lv-LV': 'lv',
  'mg': 'mg', 'mg-MG': 'mg',
  'mk': 'mk', 'mk-MK': 'mk',
  'ml': 'ml', 'ml-IN': 'ml',
  'mn': 'mn', 'mn-MN': 'mn',
  'mr': 'mr', 'mr-IN': 'mr',
  'ms': 'ms', 'ms-MY': 'ms',
  'mt': 'mt', 'mt-MT': 'mt',
  'my': 'my', 'my-MM': 'my',
  'ne': 'ne', 'ne-NP': 'ne',
  'nb': 'nb', 'nb-NO': 'nb', 'no': 'nb', 'no-NO': 'nb',
  'nn': 'nn', 'nn-NO': 'nn',
  'pa': 'pa', 'pa-IN': 'pa',
  'ps': 'ps', 'ps-AF': 'ps',
  'ro': 'ro', 'ro-RO': 'ro',
  'rw': 'rw', 'rw-RW': 'rw',
  'si': 'si', 'si-LK': 'si',
  'sk': 'sk', 'sk-SK': 'sk',
  'sl': 'sl', 'sl-SI': 'sl',
  'so': 'so', 'so-SO': 'so',
  'sq': 'sq', 'sq-AL': 'sq',
  'sr': 'sr', 'sr-RS': 'sr',
  'sw': 'sw', 'sw-TZ': 'sw', 'sw-KE': 'sw',
  'ta': 'ta', 'ta-IN': 'ta', 'ta-LK': 'ta',
  'te': 'te', 'te-IN': 'te',
  'tg': 'tg', 'tg-TJ': 'tg',
  'th': 'th', 'th-TH': 'th',
  'tk': 'tk', 'tk-TM': 'tk',
  'tr': 'tr', 'tr-TR': 'tr',
  'uk': 'uk', 'uk-UA': 'uk',
  'ur': 'ur', 'ur-PK': 'ur', 'ur-IN': 'ur',
  'uz': 'uz', 'uz-UZ': 'uz',
  'vi': 'vi', 'vi-VN': 'vi',
  'xh': 'xh', 'xh-ZA': 'xh',
  'yo': 'yo', 'yo-NG': 'yo',
  'zu': 'zu', 'zu-ZA': 'zu',
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