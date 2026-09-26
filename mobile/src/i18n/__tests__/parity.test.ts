import { en } from '../en';
import { es } from '../es';

// Every en key must exist in es (and vice versa) with identical {{placeholders}}.
// The runtime falls back per-key to English, so a missing key renders English
// instead of crashing — this test keeps those silent fallbacks intentional.
function flatten(obj: unknown, prefix = '', out: Record<string, string> = {}): Record<string, string> {
  if (typeof obj === 'string') {
    out[prefix] = obj;
    return out;
  }
  if (obj && typeof obj === 'object') {
    for (const [k, v] of Object.entries(obj)) {
      flatten(v, prefix ? `${prefix}.${k}` : k, out);
    }
  }
  return out;
}

function placeholders(s: string): string[] {
  const m = s.match(/\{\{(\w+)\}\}/g) ?? [];
  return [...new Set(m)].sort();
}

describe('locale parity (en/es)', () => {
  const enFlat = flatten(en);
  const esFlat = flatten(es);

  it('has identical key sets', () => {
    expect(Object.keys(esFlat).sort()).toEqual(Object.keys(enFlat).sort());
  });

  it('has identical interpolation placeholders per key', () => {
    const mismatches: string[] = [];
    for (const key of Object.keys(enFlat)) {
      const a = placeholders(enFlat[key]).join(',');
      const b = placeholders(esFlat[key] ?? '').join(',');
      if (a !== b) mismatches.push(`${key}: en=[${a}] es=[${b}]`);
    }
    expect(mismatches).toEqual([]);
  });
});
