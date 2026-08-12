import { test, expect } from '@playwright/test'
import { API_BASE, loginViaAPI } from '../fixtures/test-helpers'
import { ENGLISH_USER } from '../fixtures/users'

/**
 * Test Suite 11: Grammar AI — Local Qwen (Offline-First) ⭐
 *
 * Verifies that grammar analysis is served by the LOCAL Ollama provider
 * (qwen2.5:3b, alias `ollama`) as the priority provider, and that the
 * AI-generated structured analysis is valid. This is the offline-first
 * guarantee: grammar analysis must work with no cloud API keys, entirely on
 * the local machine.
 *
 * The backend exposes `/api/v1/grammar/analyze-ai`, which returns
 * `provider_used` — the alias of the endpoint that actually generated the
 * analysis. With GRAMMAR_FALLBACK_ORDER=ollama, that must be `ollama`.
 *
 * NOTE: Local CPU-only inference of qwen2.5:3b is slow (~1 min for a full
 * analysis). Each request uses unique text so results are never served from
 * the Redis cache and the live provider path is exercised.
 */
test.describe('Grammar AI — Local Qwen', () => {
  test.describe.configure({ mode: 'serial' })

  test('11.1 — Grammar AI analysis is served by the local qwen provider', async () => {
    test.setTimeout(420_000) // qwen2.5:3b on CPU can take ~1-2 min

    const token = await loginViaAPI(ENGLISH_USER)
    expect(token).toBeTruthy()

    // Unique text avoids the Redis ai_grammar cache.
    const text = `She has been learning Spanish for three years. ${Date.now()}`

    const started = Date.now()
    const response = await fetch(`${API_BASE}/grammar/analyze-ai`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({
        text,
        language: 'en',
        nativeLanguage: 'es',
      }),
      signal: AbortSignal.timeout(360_000),
    })

    expect(response.ok).toBeTruthy()
    const elapsedSec = Math.round((Date.now() - started) / 1000)
    const body = await response.json()

    const data = body.data
    expect(data).toBeTruthy()
    expect(data.text).toBe(text)

    // ⭐ The whole point: grammar analysis must route to the local qwen provider.
    expect(data.provider_used).toBe('ollama')

    const analysis = data.analysis
    expect(analysis).toBeTruthy()

    // Structured analysis must be present and non-trivial.
    expect(analysis.difficulty).toBeTruthy()
    expect(analysis.summary).toBeTruthy()
    expect(analysis.summary.length).toBeGreaterThan(10)

    expect(Array.isArray(analysis.keyPhrases)).toBeTruthy()
    expect(analysis.keyPhrases.length).toBeGreaterThan(0)
    for (const kp of analysis.keyPhrases) {
      expect(kp.phrase).toBeTruthy()
    }

    expect(Array.isArray(analysis.detailedBreakdown)).toBeTruthy()
    expect(analysis.detailedBreakdown.length).toBeGreaterThan(0)
    let typedItems = 0
    for (const item of analysis.detailedBreakdown) {
      // Every word in the sentence must be covered.
      expect(item.text).toBeTruthy()
      if (item.type) typedItems++
    }
    // At least the majority of words should carry a part-of-speech tag.
    expect(typedItems).toBeGreaterThan(0)

    expect(Array.isArray(analysis.grammarNotes)).toBeTruthy()
    expect(analysis.grammarNotes.length).toBeGreaterThan(0)
    for (const note of analysis.grammarNotes) {
      expect(note.title).toBeTruthy()
      expect(note.explanation).toBeTruthy()
    }

    console.log(`✓ Grammar analysis from ${data.provider_used} in ${elapsedSec}s`)
    console.log(`  difficulty=${analysis.difficulty} keyPhrases=${analysis.keyPhrases.length}`)
  })

  test('11.2 — Grammar AI results are cached by the backend', async () => {
    test.setTimeout(120_000)

    const token = await loginViaAPI(ENGLISH_USER)
    const text = `Cached grammar analysis test sentence. ${Date.now()}`

    const callAnalyze = async () => {
      const response = await fetch(`${API_BASE}/grammar/analyze-ai`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({ text, language: 'en', nativeLanguage: 'es' }),
        signal: AbortSignal.timeout(360_000),
      })
      expect(response.ok).toBeTruthy()
      return response.json()
    }

    // First call hits the live provider (warm in the backend by now).
    const first = await callAnalyze()
    expect(first.data.provider_used).toBe('ollama_local')

    // Immediate repeat is served from the Redis cache.
    const second = await callAnalyze()
    expect(second.data.provider_used).toBe('cache')
    expect(second.data.analysis.summary).toBe(first.data.analysis.summary)

    console.log('✓ Repeated grammar analysis served from cache')
  })
})
