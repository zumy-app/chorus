import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import { detectBrowserLanguage } from '../services/language'
import { en } from './locales/en'
import { es } from './locales/es'
import { fr } from './locales/fr'
import { de } from './locales/de'
import { pt } from './locales/pt'
import { it } from './locales/it'
import { zh } from './locales/zh'
import { hi } from './locales/hi'
import { ar } from './locales/ar'
import { bn } from './locales/bn'
import { ru } from './locales/ru'
import { ur } from './locales/ur'

export const RTL_LANGS = ['ar', 'ur', 'fa', 'he']

// Pick the initial UI language:
// 1. Saved preference (localStorage `preferredLanguage`) — set by the app's
//    language selectors.
// 2. Browser/OS language detection (respects where the user is coming from).
// 3. English fallback.
function resolveInitialLanguage(): string {
  if (typeof window !== 'undefined') {
    const saved = window.localStorage.getItem('preferredLanguage')
    if (saved && Object.keys(en).some(k => saved === k || saved.startsWith(k + '.'))) {
      // keep any saved code; i18next falls back to 'en' for missing resources
      return saved
    }
  }
  return detectBrowserLanguage()
}

const lng = resolveInitialLanguage()

i18n.use(initReactI18next).init({
  resources: {
    en: { translation: en },
    es: { translation: es },
    fr: { translation: fr },
    de: { translation: de },
    pt: { translation: pt },
    it: { translation: it },
    zh: { translation: zh },
    hi: { translation: hi },
    ar: { translation: ar },
    bn: { translation: bn },
    ru: { translation: ru },
    ur: { translation: ur },
  },
  lng,
  fallbackLng: 'en',
  interpolation: {
    escapeValue: false,
  },
  returnEmptyString: false,
})

// Keep document <html lang> and text direction in sync with the active locale.
const applyDocumentLocale = (lang: string) => {
  if (typeof document === 'undefined') return
  const primary = lang.split('-')[0]
  document.documentElement.lang = primary
  document.documentElement.dir = RTL_LANGS.includes(primary) ? 'rtl' : 'ltr'
}

applyDocumentLocale(lng)

i18n.on('languageChanged', (lang) => {
  applyDocumentLocale(lang)
})

export default i18n