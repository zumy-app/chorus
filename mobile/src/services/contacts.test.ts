import { describe, expect, it } from 'vitest'

import { canonicalizeContactValue, createContactHashes } from './contacts'

describe('contact privacy helpers', () => {
  it('normalizes contact values before hashing', () => {
    expect(canonicalizeContactValue(' Person@Example.COM ', 'email')).toBe('person@example.com')
    expect(canonicalizeContactValue(' +1 (415) 555-0123 ', 'phone')).toBe('+14155550123')
  })

  it('deduplicates canonicalized contacts and returns only their hashes', async () => {
    const hashes = await createContactHashes(
      [
        { emails: ['PERSON@example.com'], phones: ['+1 415 555 0123'] },
        { emails: [' person@example.com '], phones: ['14155550123'] },
      ],
      async (value) => `hash:${value}`,
    )

    expect(hashes).toEqual(['hash:person@example.com', 'hash:+14155550123', 'hash:14155550123'])
    expect(hashes.join(' ')).not.toContain('PERSON@example.com')
  })
})
