// Token helpers for the word-highlighting feature (FR-27 / FR-28). These keep
// the "which words are new" logic pure and unit-testable, independent of React
// and of the vocabulary API.

export type WordToken =
  | { type: 'word'; value: string }
  | { type: 'delim'; value: string }

// Split text into word tokens and the delimiters between them, preserving the
// original run order. A "word" is a run of letters/numbers that may contain a
// single internal apostrophe or hyphen ("café", "don't", "well-being"). Every
// other character (whitespace, punctuation, emoji) stays a delimiter token.
const WORD_RE = /[\p{L}\p{N}]+(?:['\u2019-][\p{L}\p{N}]+)*/gu

export function tokenize(text: string): WordToken[] {
  const tokens: WordToken[] = []
  let last = 0
  let match: RegExpExecArray | null
  WORD_RE.lastIndex = 0
  while ((match = WORD_RE.exec(text)) !== null) {
    if (match.index > last) {
      tokens.push({ type: 'delim', value: text.slice(last, match.index) })
    }
    tokens.push({ type: 'word', value: match[0] })
    last = match.index + match[0].length
  }
  if (last < text.length) {
    tokens.push({ type: 'delim', value: text.slice(last) })
  }
  return tokens
}

// Normalize a word into a stable matchable key: strip combining diacritics and
// lowercase. "Café" and "CAFE" both collapse to "cafe" so the vocabulary known
// set matches regardless of how the term was saved or accented in the message.
export function normalizeWord(word: string): string {
  return word
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '')
    .toLowerCase()
}

// A word worth surfacing as a "new" word. Short function words (and bare
// numbers/symbols) are ignored so we don't crowd a sentence with highlights.
export function isHighlightableWord(word: string): boolean {
  return normalizeWord(word).length >= 3
}
