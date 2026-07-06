import { useState, useRef, useEffect } from 'react'
import { detectBrowserLanguage, SUPPORTED_LANGUAGES } from '../services/language'

interface LanguageSelectorProps {
  /** Currently selected language code */
  currentLang?: string
  /** Called when user selects a language */
  onLanguageChange: (langCode: string) => void
  /** Visual variant */
  variant?: 'navbar' | 'compact' | 'full'
}

export default function LanguageSelector({ currentLang, onLanguageChange, variant = 'compact' }: LanguageSelectorProps) {
  const [isOpen, setIsOpen] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)

  const detectedLang = detectBrowserLanguage()
  const activeLang = currentLang || detectedLang
  const activeLanguage = SUPPORTED_LANGUAGES.find(l => l.code === activeLang) || SUPPORTED_LANGUAGES[0]

  // Show language code text for English (avoids UK flag confusion), flag emoji for others
  const displayLabel = activeLang === 'en' ? 'EN' : activeLanguage.flag

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setIsOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  const handleSelect = (code: string) => {
    onLanguageChange(code)
    setIsOpen(false)
  }

  if (variant === 'navbar') {
    return (
      <div className="relative" ref={menuRef}>
        <button
          onClick={() => setIsOpen(!isOpen)}
          className="flex items-center gap-2 px-3 py-2 border border-gray-300 rounded-lg hover:bg-gray-50 transition text-sm"
          title="Select language"
        >
          <span className="text-base">{activeLanguage.flag}</span>
          <span className="hidden sm:inline text-gray-700 font-medium">{activeLanguage.nativeName}</span>
          <svg className={`w-3 h-3 text-gray-500 transition-transform ${isOpen ? 'rotate-180' : ''}`} fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
          </svg>
        </button>

        {isOpen && (
          <div className="absolute right-0 mt-2 w-56 bg-white rounded-lg shadow-xl border border-gray-200 z-[100]">
            <div className="p-2">
              <p className="text-xs text-gray-500 px-3 py-1.5 font-medium uppercase tracking-wider">Select Language</p>
              {SUPPORTED_LANGUAGES.map((lang) => (
                <button
                  key={lang.code}
                  onClick={() => handleSelect(lang.code)}
                  className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm transition ${
                    activeLang === lang.code
                      ? 'bg-indigo-50 text-indigo-700 font-semibold'
                      : 'text-gray-700 hover:bg-gray-50'
                  }`}
                >
                  <span className="text-lg">{lang.flag}</span>
                  <div className="text-left">
                    <div>{lang.nativeName}</div>
                    <div className="text-xs text-gray-400">{lang.name}</div>
                  </div>
                  {activeLang === lang.code && (
                    <span className="ml-auto text-indigo-600">✓</span>
                  )}
                </button>
              ))}
            </div>
          </div>
        )}
      </div>
    )
  }

  // Compact variant (icon-only for header)
  return (
    <div className="relative" ref={menuRef}>
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="w-9 h-9 flex items-center justify-center rounded-full hover:bg-gray-100 transition text-sm font-bold"
        title={`Language: ${activeLanguage.nativeName}`}
      >
        {displayLabel}
      </button>

      {isOpen && (
        <div className="absolute right-0 mt-2 w-56 bg-white rounded-lg shadow-xl border border-gray-200 z-[100]">
          <div className="p-2">
            <p className="text-xs text-gray-500 px-3 py-1.5 font-medium uppercase tracking-wider">Select Language</p>
            {SUPPORTED_LANGUAGES.map((lang) => (
              <button
                key={lang.code}
                onClick={() => handleSelect(lang.code)}
                className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm transition ${
                  activeLang === lang.code
                    ? 'bg-indigo-50 text-indigo-700 font-semibold'
                    : 'text-gray-700 hover:bg-gray-50'
                }`}
              >
                <span className="text-lg">{lang.flag}</span>
                <div className="text-left">
                  <div>{lang.nativeName}</div>
                  <div className="text-xs text-gray-400">{lang.name}</div>
                </div>
                {activeLang === lang.code && (
                  <span className="ml-auto text-indigo-600">✓</span>
                )}
              </button>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
