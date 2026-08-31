import { describe, it, expect } from 'vitest'
import { tokenize, normalizeWord, isHighlightableWord } from '../words'

describe('tokenize', () => {
  it('splits a sentence into words and delimiters preserving order', () => {
    const tokens = tokenize('Hola mi amigo')
    expect(tokens).toEqual([
      { type: 'word', value: 'Hola' },
      { type: 'delim', value: ' ' },
      { type: 'word', value: 'mi' },
      { type: 'delim', value: ' ' },
      { type: 'word', value: 'amigo' },
    ])
  })

  it('treats punctuation and emoji as delimiters', () => {
    const tokens = tokenize('¡Qué bien! 😀')
    const words = tokens.filter((t) => t.type === 'word').map((t) => t.value)
    expect(words).toEqual(['Qué', 'bien'])
  })

  it('keeps apostrophes and hyphens inside a single word', () => {
    const tokens = tokenize("l'école well-being")
    const words = tokens.filter((t) => t.type === 'word').map((t) => t.value)
    expect(words).toEqual(["l'école", 'well-being'])
  })

  it('handles empty and accented input', () => {
    expect(tokenize('')).toEqual([])
    const words = tokenize('café amore').filter((t) => t.type === 'word').map((t) => t.value)
    expect(words).toEqual(['café', 'amore'])
  })
})

describe('normalizeWord', () => {
  it('lowercases and strips diacritics', () => {
    expect(normalizeWord('Café')).toBe('cafe')
    expect(normalizeWord('CAFE')).toBe('cafe')
    expect(normalizeWord('amigO')).toBe('amigo')
  })
})

describe('isHighlightableWord', () => {
  it('ignores short words and numbers', () => {
    expect(isHighlightableWord('mi')).toBe(false)
    expect(isHighlightableWord('la')).toBe(false)
    expect(isHighlightableWord('42')).toBe(false)
    expect(isHighlightableWord('hola')).toBe(true)
    expect(isHighlightableWord('amigo')).toBe(true)
  })
})
