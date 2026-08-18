export type ContactValueType = 'email' | 'phone'

export interface ContactRecord {
  emails?: string[]
  phones?: string[]
}

export function canonicalizeContactValue(value: string, type: ContactValueType): string {
  const trimmed = value.trim()
  if (type === 'email') return trimmed.toLowerCase()
  return trimmed.replace(/[^\d+]/g, '')
}

// The native permission layer supplies contacts and a SHA-256 implementation.
// This module deliberately returns only derived hashes so raw address-book data
// never reaches the matching API or application logs.
export async function createContactHashes(
  contacts: ContactRecord[],
  hash: (value: string) => Promise<string>,
): Promise<string[]> {
  const values = new Set<string>()
  for (const contact of contacts) {
    for (const email of contact.emails ?? []) {
      const value = canonicalizeContactValue(email, 'email')
      if (value) values.add(value)
    }
    for (const phone of contact.phones ?? []) {
      const value = canonicalizeContactValue(phone, 'phone')
      if (value) values.add(value)
    }
  }
  return Promise.all([...values].map(hash))
}
