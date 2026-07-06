// Language detection utility — detects user's preferred language from browser settings
// Uses the Navigator.language API which respects the user's OS/browser language preferences

export const SUPPORTED_LANGUAGES = [
  // Most common languages (top of list for easy access)
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

  // Extended European
  { code: 'af', name: 'Afrikaans', nativeName: 'Afrikaans', flag: '🇿🇦' },
  { code: 'bg', name: 'Bulgarian', nativeName: 'Български', flag: '🇧🇬' },
  { code: 'bs', name: 'Bosnian', nativeName: 'Bosanski', flag: '🇧🇦' },
  { code: 'ca', name: 'Catalan', nativeName: 'Català', flag: '🇪🇸' },
  { code: 'cs', name: 'Czech', nativeName: 'Čeština', flag: '🇨🇿' },
  { code: 'cy', name: 'Welsh', nativeName: 'Cymraeg', flag: '🏴󠁧󠁢󠁷󠁬󠁳󠁿' },
  { code: 'da', name: 'Danish', nativeName: 'Dansk', flag: '🇩🇰' },
  { code: 'el', name: 'Greek', nativeName: 'Ελληνικά', flag: '🇬🇷' },
  { code: 'et', name: 'Estonian', nativeName: 'Eesti', flag: '🇪🇪' },
  { code: 'fi', name: 'Finnish', nativeName: 'Suomi', flag: '🇫🇮' },
  { code: 'ga', name: 'Irish', nativeName: 'Gaeilge', flag: '🇮🇪' },
  { code: 'gl', name: 'Galician', nativeName: 'Galego', flag: '🇪🇸' },
  { code: 'hr', name: 'Croatian', nativeName: 'Hrvatski', flag: '🇭🇷' },
  { code: 'hu', name: 'Hungarian', nativeName: 'Magyar', flag: '🇭🇺' },
  { code: 'is', name: 'Icelandic', nativeName: 'Íslenska', flag: '🇮🇸' },
  { code: 'lt', name: 'Lithuanian', nativeName: 'Lietuvių', flag: '🇱🇹' },
  { code: 'lv', name: 'Latvian', nativeName: 'Latviešu', flag: '🇱🇻' },
  { code: 'mk', name: 'Macedonian', nativeName: 'Македонски', flag: '🇲🇰' },
  { code: 'mt', name: 'Maltese', nativeName: 'Malti', flag: '🇲🇹' },
  { code: 'nb', name: 'Norwegian Bokmål', nativeName: 'Norsk Bokmål', flag: '🇳🇴' },
  { code: 'nn', name: 'Norwegian Nynorsk', nativeName: 'Norsk Nynorsk', flag: '🇳🇴' },
  { code: 'ro', name: 'Romanian', nativeName: 'Română', flag: '🇷🇴' },
  { code: 'sk', name: 'Slovak', nativeName: 'Slovenčina', flag: '🇸🇰' },
  { code: 'sl', name: 'Slovenian', nativeName: 'Slovenščina', flag: '🇸🇮' },
  { code: 'sq', name: 'Albanian', nativeName: 'Shqip', flag: '🇦🇱' },
  { code: 'sr', name: 'Serbian', nativeName: 'Српски', flag: '🇷🇸' },
  { code: 'tr', name: 'Turkish', nativeName: 'Türkçe', flag: '🇹🇷' },
  { code: 'uk', name: 'Ukrainian', nativeName: 'Українська', flag: '🇺🇦' },

  // Asian
  { code: 'bn', name: 'Bengali', nativeName: 'বাংলা', flag: '🇧🇩' },
  { code: 'gu', name: 'Gujarati', nativeName: 'ગુજરાતી', flag: '🇮🇳' },
  { code: 'he', name: 'Hebrew', nativeName: 'עברית', flag: '🇮🇱' },
  { code: 'hi', name: 'Hindi', nativeName: 'हिन्दी', flag: '🇮🇳' },
  { code: 'id', name: 'Indonesian', nativeName: 'Bahasa Indonesia', flag: '🇮🇩' },
  { code: 'kn', name: 'Kannada', nativeName: 'ಕನ್ನಡ', flag: '🇮🇳' },
  { code: 'lo', name: 'Lao', nativeName: 'ລາວ', flag: '🇱🇦' },
  { code: 'ml', name: 'Malayalam', nativeName: 'മലയാളം', flag: '🇮🇳' },
  { code: 'mr', name: 'Marathi', nativeName: 'मराठी', flag: '🇮🇳' },
  { code: 'ms', name: 'Malay', nativeName: 'Bahasa Melayu', flag: '🇲🇾' },
  { code: 'my', name: 'Burmese', nativeName: 'မြန်မာဘာသာ', flag: '🇲🇲' },
  { code: 'ne', name: 'Nepali', nativeName: 'नेपाली', flag: '🇳🇵' },
  { code: 'pa', name: 'Punjabi', nativeName: 'ਪੰਜਾਬੀ', flag: '🇮🇳' },
  { code: 'si', name: 'Sinhala', nativeName: 'සිංහල', flag: '🇱🇰' },
  { code: 'ta', name: 'Tamil', nativeName: 'தமிழ்', flag: '🇮🇳' },
  { code: 'te', name: 'Telugu', nativeName: 'తెలుగు', flag: '🇮🇳' },
  { code: 'th', name: 'Thai', nativeName: 'ไทย', flag: '🇹🇭' },
  { code: 'ur', name: 'Urdu', nativeName: 'اردو', flag: '🇵🇰' },
  { code: 'vi', name: 'Vietnamese', nativeName: 'Tiếng Việt', flag: '🇻🇳' },

  // African
  { code: 'am', name: 'Amharic', nativeName: 'አማርኛ', flag: '🇪🇹' },
  { code: 'ha', name: 'Hausa', nativeName: 'Hausa', flag: '🇳🇬' },
  { code: 'ig', name: 'Igbo', nativeName: 'Igbo', flag: '🇳🇬' },
  { code: 'km', name: 'Khmer', nativeName: 'ភាសាខ្មែរ', flag: '🇰🇭' },
  { code: 'mg', name: 'Malagasy', nativeName: 'Malagasy', flag: '🇲🇬' },
  { code: 'rw', name: 'Kinyarwanda', nativeName: 'Ikinyarwanda', flag: '🇷🇼' },
  { code: 'so', name: 'Somali', nativeName: 'Soomaali', flag: '🇸🇴' },
  { code: 'sw', name: 'Swahili', nativeName: 'Kiswahili', flag: '🇹🇿' },
  { code: 'xh', name: 'Xhosa', nativeName: 'isiXhosa', flag: '🇿🇦' },
  { code: 'yo', name: 'Yoruba', nativeName: 'Yorùbá', flag: '🇳🇬' },
  { code: 'zu', name: 'Zulu', nativeName: 'isiZulu', flag: '🇿🇦' },

  // Middle Eastern / Central Asian
  { code: 'az', name: 'Azerbaijani', nativeName: 'Azərbaycan', flag: '🇦🇿' },
  { code: 'fa', name: 'Persian', nativeName: 'فارسی', flag: '🇮🇷' },
  { code: 'kk', name: 'Kazakh', nativeName: 'Қазақ', flag: '🇰🇿' },
  { code: 'ky', name: 'Kyrgyz', nativeName: 'Кыргыз', flag: '🇰🇬' },
  { code: 'mn', name: 'Mongolian', nativeName: 'Монгол', flag: '🇲🇳' },
  { code: 'ps', name: 'Pashto', nativeName: 'پښتو', flag: '🇦🇫' },
  { code: 'tg', name: 'Tajik', nativeName: 'Тоҷикӣ', flag: '🇹🇯' },
  { code: 'tk', name: 'Turkmen', nativeName: 'Türkmen', flag: '🇹🇲' },
  { code: 'uz', name: 'Uzbek', nativeName: 'Oʻzbek', flag: '🇺🇿' },
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