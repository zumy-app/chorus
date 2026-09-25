import { describe, it, expect } from 'vitest'
import { en } from '../locales/en'
import { es } from '../locales/es'
import { ar } from '../locales/ar'
import { bn } from '../locales/bn'
import { de } from '../locales/de'
import { fr } from '../locales/fr'
import { hi } from '../locales/hi'
import { it as itLocale } from '../locales/it'
import { pt } from '../locales/pt'
import { ru } from '../locales/ru'
import { ur } from '../locales/ur'
import { zh } from '../locales/zh'

// Every en key must exist in es (and vice versa) with identical {{placeholders}}.
// i18next falls back per-key to English, so a missing key renders English
// instead of crashing — this test keeps those silent fallbacks intentional.
function flatten(obj: unknown, prefix = '', out: Record<string, string> = {}): Record<string, string> {
  if (typeof obj === 'string') {
    out[prefix] = obj
    return out
  }
  if (obj && typeof obj === 'object') {
    for (const [k, v] of Object.entries(obj)) {
      flatten(v, prefix ? `${prefix}.${k}` : k, out)
    }
  }
  return out
}

function placeholders(s: string): string[] {
  const m = s.match(/\{\{(\w+)\}\}/g) ?? []
  return [...new Set(m)].sort()
}

describe('locale parity (en/es)', () => {
  const enFlat = flatten(en)
  const esFlat = flatten(es)

  it('has identical key sets', () => {
    const enKeys = Object.keys(enFlat).sort()
    const esKeys = Object.keys(esFlat).sort()
    expect(esKeys).toEqual(enKeys)
  })

  it('has identical interpolation placeholders per key', () => {
    const mismatches: string[] = []
    for (const key of Object.keys(enFlat)) {
      const a = placeholders(enFlat[key]).join(',')
      const b = placeholders(esFlat[key] ?? '').join(',')
      if (a !== b) mismatches.push(`${key}: en=[${a}] es=[${b}]`)
    }
    expect(mismatches).toEqual([])
  })

  // The remaining locales are Partial<AppTranslation>: they may lag behind
  // en (runtime falls back per-key), but every key they DO carry must exist
  // in en with matching placeholders — no orphans, no broken interpolation.
  it.each([
    ['ar', ar], ['bn', bn], ['de', de], ['fr', fr], ['hi', hi],
    ['it', itLocale], ['pt', pt], ['ru', ru], ['ur', ur], ['zh', zh],
  ] as const)('%s is a placeholder-consistent subset of en', (_name, table) => {
    const flat = flatten(table)
    const problems: string[] = []
    for (const key of Object.keys(flat)) {
      if (!(key in enFlat)) {
        problems.push(`${key}: orphan key (not in en)`)
        continue
      }
      const a = placeholders(enFlat[key]).join(',')
      const b = placeholders(flat[key]).join(',')
      if (a !== b) problems.push(`${key}: en=[${a}] ${_name}=[${b}]`)
    }
    expect(problems).toEqual([])
  })
})
