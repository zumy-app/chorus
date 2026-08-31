import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'
import { vocabularyAPI } from '../services/api'
import { tokenize, normalizeWord, isHighlightableWord } from '../utils/words'

interface HighlightableTextProps {
  text: string
  language: string
  messageId: string
  knownWords: Set<string>
  onWordSaved?: (word: string) => void
  className?: string
}

/**
 * Renders a chunk of text (a translation or an original message) so unknown
 * words are highlighted and tappable. Tapping a highlighted word opens a quick
 * affordance to add it to the word bank (FR-27) and a practice CTA (FR-28).
 * Words already in the user's vocabulary render dimmed (FR-26).
 */
export default function HighlightableText({
  text,
  language,
  messageId,
  knownWords,
  onWordSaved,
  className,
}: HighlightableTextProps) {
  const { t } = useTranslation()
  const [activeWord, setActiveWord] = useState<string | null>(null)
  const [savingWord, setSavingWord] = useState<string | null>(null)
  const [savedWord, setSavedWord] = useState<string | null>(null)
  const containerRef = useRef<HTMLSpanElement>(null)

  // Close the open affordance when clicking anywhere outside the component.
  useEffect(() => {
    if (!activeWord) return
    const onMouseDown = (e: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setActiveWord(null)
      }
    }
    document.addEventListener('mousedown', onMouseDown)
    return () => document.removeEventListener('mousedown', onMouseDown)
  }, [activeWord])

  useEffect(() => {
    if (!savedWord) return
    const timer = setTimeout(() => setSavedWord(null), 2000)
    return () => clearTimeout(timer)
  }, [savedWord])

  const handleSave = async (word: string) => {
    if (savingWord) return
    setSavingWord(word)
    try {
      await vocabularyAPI.save(word, language, messageId)
      setSavedWord(word)
      setActiveWord(null)
      onWordSaved?.(word)
    } catch (err) {
      console.error('Failed to save highlighted word:', err)
    } finally {
      setSavingWord(null)
    }
  }

  const tokens = tokenize(text)

  return (
    <span className={className} ref={containerRef}>
      {tokens.map((token, i) => {
        if (token.type === 'delim') {
          return <span key={i}>{token.value}</span>
        }
        const normalized = normalizeWord(token.value)
        const isKnown = knownWords.has(normalized)
        const isActive = activeWord === token.value
        const isSaved = savedWord === token.value
        const saving = savingWord === token.value
        const isWord = isHighlightableWord(token.value)

        if (!isWord || isKnown) {
          // Known / too-short words render dimmed so focus stays on the new ones.
          return (
            <span key={i} className={isWord ? 'opacity-60' : undefined}>
              {token.value}
            </span>
          )
        }

        return (
          <span key={i} className="relative inline-block">
            <button
              type="button"
              onClick={(e) => {
                e.stopPropagation()
                setActiveWord((prev) => (prev === token.value ? null : token.value))
              }}
              title={t('grammar.tapWordToSave')}
              aria-label={`${token.value} — ${t('grammar.tapWordToSave')}`}
              className={`word-highlight inline-block rounded px-0.5 mx-[1px] font-semibold transition-colors ${
                isSaved
                  ? 'bg-emerald-200 text-emerald-800'
                  : isActive
                    ? 'bg-amber-300 text-amber-900'
                    : 'bg-amber-100/80 text-amber-900 hover:bg-amber-200'
              }`}
            >
              {token.value}
              {isSaved && <span className="ml-0.5">✓</span>}
            </button>

            {isActive && (
              <span
                className="word-affordance absolute left-0 top-full z-20 mt-1 flex flex-col gap-1 rounded-lg border border-outline-variant bg-surface-container-lowest px-2.5 py-2 shadow-lg min-w-[10rem]"
                role="menu"
              >
                <span className="font-semibold text-sm text-on-surface whitespace-nowrap">
                  {token.value}
                </span>
                <button
                  type="button"
                  disabled={saving}
                  onClick={(e) => {
                    e.stopPropagation()
                    handleSave(token.value)
                  }}
                  className="rounded-md bg-primary px-2.5 py-1 text-xs font-semibold text-on-primary hover:bg-primary/90 disabled:opacity-50"
                >
                  {saving ? t('common.saving') : `+ ${t('grammar.addToWordBank')}`}
                </button>
                <Link
                  to="/learn/vocabulary"
                  onClick={(e) => e.stopPropagation()}
                  className="rounded-md bg-secondary-container px-2.5 py-1 text-xs font-semibold text-on-secondary-container hover:opacity-90"
                >
                  {t('grammar.practice')}
                </Link>
              </span>
            )}
          </span>
        )
      })}
    </span>
  )
}
