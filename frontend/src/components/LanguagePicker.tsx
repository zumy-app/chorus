import { useState } from 'react'
import { SUPPORTED_LANGUAGES } from '../services/language'

interface LanguagePickerProps {
  label: string
  hint?: string
  multiple?: boolean
  selected: string | string[]
  topCodes: string[]
  exclude?: string[]
  onChange: (code: string, add: boolean) => void
}

export default function LanguagePicker({
  label,
  hint,
  multiple = false,
  selected,
  topCodes,
  exclude = [],
  onChange,
}: LanguagePickerProps) {
  const [query, setQuery] = useState('')
  const selectedArr = multiple
    ? (selected as string[])
    : selected
      ? [selected as string]
      : []
  const isSelected = (code: string) => selectedArr.includes(code)

  const topLangs = SUPPORTED_LANGUAGES.filter(
    (l) => topCodes.includes(l.code) && !exclude.includes(l.code)
  )
  const q = query.trim().toLowerCase()
  const results = q
    ? SUPPORTED_LANGUAGES.filter(
        (l) =>
          !exclude.includes(l.code) &&
          (l.name.toLowerCase().includes(q) || l.nativeName.toLowerCase().includes(q))
      ).slice(0, 8)
    : []

  const handlePick = (code: string) => {
    if (multiple) {
      onChange(code, !isSelected(code))
    } else {
      onChange(code, true)
      setQuery('')
    }
  }

  const text = query.trim()

  return (
    <div>
      <span className="block font-semibold text-gray-700">{label}</span>
      {hint && <span className="mt-0.5 block text-sm text-gray-400">{hint}</span>}

      {selectedArr.length > 0 && (
        <div className="mt-2 flex flex-wrap gap-2">
          {selectedArr.map((code) => {
            const lang = SUPPORTED_LANGUAGES.find((l) => l.code === code)
            if (!lang) return null
            return (
              <span
                key={code}
                className="inline-flex items-center gap-1 rounded-full bg-primary/10 py-1 pl-3 pr-1 text-sm font-medium text-primary"
              >
                {lang.flag} {lang.name}
                {multiple && (
                  <button
                    type="button"
                    onClick={() => onChange(code, false)}
                    className="text-primary hover:text-primary/70"
                    aria-label={`Remove ${lang.name}`}
                  >
                    ×
                  </button>
                )}
              </span>
            )
          })}
        </div>
      )}

      <div className="mt-2 flex flex-wrap gap-2">
        {topLangs.map((lang) => (
          <button
            key={lang.code}
            type="button"
            onClick={() => handlePick(lang.code)}
            className={`rounded-full border px-3 py-1.5 text-sm font-medium transition ${
              isSelected(lang.code)
                ? 'border-primary bg-primary text-white'
                : 'border-gray-300 text-gray-700 hover:border-primary hover:text-primary'
            }`}
          >
            {lang.flag} {lang.name}
          </button>
        ))}
      </div>

      <div className="relative mt-2">
        <input
          type="text"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Type another language…"
          className="w-full rounded-lg border border-gray-300 px-4 py-2.5 text-sm focus:border-primary focus:outline-none"
        />
        {q && (
          <div className="absolute z-10 mt-1 max-h-56 w-full overflow-y-auto rounded-lg border border-gray-200 bg-white shadow-lg">
            {results.length === 0 && !text ? (
              <div />
            ) : results.length === 0 ? (
              <div className="px-4 py-3 text-sm text-gray-400">No matching languages.</div>
            ) : (
              results.map((lang) => (
                <button
                  key={lang.code}
                  type="button"
                  onClick={() => handlePick(lang.code)}
                  className={`flex w-full items-center justify-between px-4 py-2.5 text-left text-sm transition hover:bg-indigo-50 ${
                    isSelected(lang.code) ? 'bg-indigo-50 text-indigo-700' : 'text-gray-800'
                  }`}
                >
                  <span>
                    {lang.flag} {lang.name}
                  </span>
                  {isSelected(lang.code) && <span className="text-indigo-600">✓</span>}
                </button>
              ))
            )}
          </div>
        )}
      </div>
    </div>
  )
}