import { useCallback, useEffect, useState } from 'react'
import { vocabularyAPI } from '../services/api'
import { normalizeWord } from '../utils/words'

// Known-words are cached per language so a chat full of bubbles only fetches
// the user's vocabulary once per language instead of once per message.
const cache = new Map<string, Set<string>>()
const inFlight = new Map<string, Promise<Set<string>>>()

function cacheKey(language: string): string {
  return language.toLowerCase()
}

async function fetchKnownWords(language: string): Promise<Set<string>> {
  const key = cacheKey(language)
  const hit = inFlight.get(key)
  if (hit) return hit

  const promise = vocabularyAPI
    .getAll(language)
    .then((entries: any[]) => {
      const set = new Set<string>()
      for (const entry of entries) {
        if (entry?.term) set.add(normalizeWord(entry.term))
      }
      cache.set(key, set)
      return set
    })
    .catch(() => {
      // No known-words set (offline / empty / API error) → treat everything as
      // unknown so highlighting still works; the set just stays empty.
      const empty = new Set<string>()
      cache.set(key, empty)
      return empty
    })
    .finally(() => {
      inFlight.delete(key)
    })

  inFlight.set(key, promise)
  return promise
}

interface UseKnownWordsResult {
  knownWords: Set<string>
  addKnownWord: (word: string) => void
  refresh: () => Promise<void>
}

/**
 * Loads the user's known vocabulary for a language and exposes it as a set of
 * normalized terms, so MessageBubble can highlight new/unlearned words (FR-28)
 * and dim words already in the word bank (FR-26).
 */
export function useKnownWords(language?: string): UseKnownWordsResult {
  const [knownWords, setKnownWords] = useState<Set<string>>(() =>
    language && cache.has(cacheKey(language)) ? cache.get(cacheKey(language))! : new Set()
  )

  useEffect(() => {
    if (!language) return
    const key = cacheKey(language)
    if (cache.has(key)) {
      setKnownWords(cache.get(key)!)
      return
    }
    let cancelled = false
    fetchKnownWords(language).then((set) => {
      if (!cancelled) setKnownWords(set)
    })
    return () => {
      cancelled = true
    }
  }, [language])

  const addKnownWord = useCallback(
    (word: string) => {
      const normalized = normalizeWord(word)
      setKnownWords((prev) => {
        const next = new Set(prev)
        next.add(normalized)
        return next
      })
      if (language) {
        const key = cacheKey(language)
        const next = new Set(cache.get(key) || [])
        next.add(normalized)
        cache.set(key, next)
      }
    },
    [language]
  )

  const refresh = useCallback(async () => {
    if (!language) return
    const set = await fetchKnownWords(language)
    setKnownWords(set)
  }, [language])

  return { knownWords, addKnownWord, refresh }
}
