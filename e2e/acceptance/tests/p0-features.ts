/**
 * P0 feature test cases (REQ-APPLY, REQ-MKT, REQ-LEARN).
 * Freeze policy: QA-owned; changes require human approval.
 */
import {
  TestCase, http, assert, assertEq, assertStatus,
} from '../harness.js'

/** Registers a throwaway applicant account (dev open registration). */
async function registerTempApplicant(prefix: string): Promise<{ token: string }> {
  const email = `${prefix}-${Date.now()}@chorus.test`
  const res = await http('POST', '/api/v1/auth/register', {
    json: {
      username: email,
      email,
      password: 'ProbePass123!',
      displayName: prefix,
      nativeLanguage: 'en',
      targetLanguages: ['es'],
    },
  })
  assertStatus(res, 201, `temp applicant registration (${email})`)
  assert(res.body?.tokens?.accessToken, 'temp applicant must receive tokens')
  return { token: res.body.tokens.accessToken }
}

export const featureTests: TestCase[] = [
  // ------------------------------------------------------------------
  // Tutor application (the bug the user reported)
  // ------------------------------------------------------------------
  {
    id: 'TC-APPLY-01',
    reqs: ['REQ-APPLY-01'],
    name: 'a complete application WITHOUT intro video is accepted (video optional per locked decision)',
    fn: async (ctx) => {
      // bob.dev never applies during seeding, so he is the clean applicant.
      const res = await http('POST', '/api/v1/teachers/apply', {
        token: ctx.learner2Token,
        json: {
          bio: 'Native English speaker, two years tutoring Spanish conversationally.',
          languages: ['en'],
          expertise: 'Conversation practice',
          rateCents: 2000,
          videoUrl: '',
          certificates: [],
        },
      })
      assertStatus(res, 200, 'POST /teachers/apply without videoUrl')
      assertEq(res.body.application?.status, 'pending', 'application status')
    },
  },
  {
    id: 'TC-APPLY-02',
    reqs: ['REQ-APPLY-01'],
    name: 'an application WITH a valid intro video URL is accepted and retrievable',
    fn: async (ctx) => {
      const res = await http('POST', '/api/v1/teachers/apply', {
        token: ctx.learnerToken,
        json: {
          bio: 'Experienced language exchange partner offering structured lessons.',
          languages: ['en', 'es'],
          expertise: 'Grammar and pronunciation',
          rateCents: 2500,
          videoUrl: 'https://cdn.example.com/intro-alice.mp4',
          certificates: [
            { type: 'language_certificate', issuer: 'Test Institute', year: 2021, fileUrl: 'https://cdn.example.com/cert.pdf' },
          ],
        },
      })
      assertStatus(res, 200, 'POST /teachers/apply with videoUrl')
      assert(res.body.application, 'response must include application')
      // Round-trip: GET /teachers/me returns the same application.
      const me = await http('GET', '/api/v1/teachers/me', { token: ctx.learnerToken })
      assertStatus(me, 200, 'GET /teachers/me')
      assertEq(me.body.application?.videoUrl, 'https://cdn.example.com/intro-alice.mp4', 'stored videoUrl')
    },
  },
  {
    id: 'TC-APPLY-03',
    reqs: ['REQ-APPLY-02'],
    name: 'an application with a malformed video URL is rejected with a field-level message',
    fn: async () => {
      // Fresh applicant so duplicate-application handling cannot mask the
      // validation path under test.
      const applicant = await registerTempApplicant('tc-apply03')
      const res = await http('POST', '/api/v1/teachers/apply', {
        token: applicant.token,
        json: {
          bio: 'A perfectly fine bio longer than ten characters.',
          languages: ['en'],
          rateCents: 2000,
          videoUrl: 'not-a-url',
        },
      })
      assertStatus(res, 400, 'POST /teachers/apply with invalid videoUrl')
      assertEq(res.body.code, 'VALIDATION', 'error envelope code')
      const msg = String(res.body.message ?? '').toLowerCase()
      assert(msg.includes('video') || msg.includes('application'), `message must hint at the failing field, got: ${res.body.message}`)
    },
  },
  {
    id: 'TC-APPLY-04',
    reqs: ['REQ-APPLY-02'],
    name: 'an application with bio < 10 chars or missing languages is rejected with a readable message',
    fn: async () => {
      const applicant = await registerTempApplicant('tc-apply04')
      const shortBio = await http('POST', '/api/v1/teachers/apply', {
        token: applicant.token,
        json: { bio: 'short', languages: ['en'], rateCents: 2000 },
      })
      assertStatus(shortBio, 400, 'apply with short bio')
      const noLangs = await http('POST', '/api/v1/teachers/apply', {
        token: applicant.token,
        json: { bio: 'This bio is definitely long enough to pass.', languages: [], rateCents: 2000 },
      })
      assertStatus(noLangs, 400, 'apply with no languages')
    },
  },
  // ------------------------------------------------------------------
  // Tutor marketplace (browse / profile / booking)
  // ------------------------------------------------------------------
  {
    id: 'TC-MKT-01',
    reqs: ['REQ-MKT-01'],
    name: 'browse tutors returns the seeded approved tutor with rating + review data',
    fn: async (ctx) => {
      const res = await http('GET', '/api/v1/teachers/browse?language=es', { token: ctx.learnerToken })
      assertStatus(res, 200, 'GET /teachers/browse?language=es')
      const tutors = res.body.tutors
      assert(Array.isArray(tutors) && tutors.length >= 1, `expected at least 1 tutor, got ${JSON.stringify(res.body).slice(0, 200)}`)
      const sofia = tutors.find((t: any) => t.email === 'sofia.tutor@chorus.test' || String(t.displayName ?? '').includes('Sofia'))
      assert(sofia, 'seeded tutor Sofia must appear in browse results')
      assert(sofia.avgRating >= 4, `Sofia avgRating should reflect seeded reviews (>=4), got ${sofia.avgRating}`)
      assert(sofia.reviewCount >= 2, `Sofia reviewCount should be >=2, got ${sofia.reviewCount}`)
    },
  },
  {
    id: 'TC-MKT-02',
    reqs: ['REQ-MKT-02'],
    name: 'tutor profile exposes bio, certificate, availability slots and reviews',
    fn: async (ctx) => {
      const browse = await http('GET', '/api/v1/teachers/browse?language=es', { token: ctx.learnerToken })
      assertStatus(browse, 200, 'browse before profile')
      const sofia = (browse.body.tutors ?? []).find((t: any) => String(t.displayName ?? '').includes('Sofia'))
      assert(sofia, 'Sofia present in browse')
      const profile = await http('GET', `/api/v1/teachers/${sofia.id ?? sofia.userId}`, { token: ctx.learnerToken })
      assertStatus(profile, 200, 'GET /teachers/:id')
      const p = profile.body.tutor
      assert(p && String(p.bio ?? '').length >= 20, 'profile must include a real bio')
      const avail = await http('GET', `/api/v1/teachers/${sofia.id ?? sofia.userId}/availability`, { token: ctx.learnerToken })
      assertStatus(avail, 200, 'GET /teachers/:id/availability')
      const slots = avail.body.availability ?? avail.body.data ?? avail.body
      assert(Array.isArray(slots) && slots.length >= 4, `expected >=4 seeded availability slots, got ${JSON.stringify(avail.body).slice(0, 200)}`)
      const reviews = await http('GET', `/api/v1/teachers/${sofia.id ?? sofia.userId}/reviews`, { token: ctx.learnerToken })
      assertStatus(reviews, 200, 'GET /teachers/:id/reviews')
      const rs = reviews.body.reviews ?? reviews.body.data ?? reviews.body
      assert(Array.isArray(rs) && rs.length >= 2, `expected >=2 seeded reviews, got ${JSON.stringify(reviews.body).slice(0, 200)}`)
    },
  },
  {
    id: 'TC-MKT-03',
    reqs: ['REQ-MKT-03'],
    name: 'learner with a trial credit can book an availability slot',
    fn: async (ctx) => {
      const browse = await http('GET', '/api/v1/teachers/browse?language=es', { token: ctx.learnerToken })
      const sofia = (browse.body.tutors ?? []).find((t: any) => String(t.displayName ?? '').includes('Sofia'))
      assert(sofia, 'Sofia present in browse')
      const tutorId = sofia.id ?? sofia.userId
      const avail = await http('GET', `/api/v1/teachers/${tutorId}/availability`, { token: ctx.learnerToken })
      const slots = avail.body.availability ?? avail.body.data ?? avail.body
      assert(Array.isArray(slots) && slots.length > 0, 'at least one availability slot')
      const slot = slots[0]
      const res = await http('POST', `/api/v1/teachers/${tutorId}/book`, {
        token: ctx.learnerToken,
        json: {
          startTime: slot.startTime,
          endTime: slot.endTime,
          isTrial: true,
          note: 'TC-MKT-03 booking',
        },
      })
      assertStatus(res, 201, 'POST /teachers/:id/book with trial credit')
      assert(res.body.booking, 'booking object returned')
      // Trial credit must be consumed.
      const credits = await http('GET', '/api/v1/teachers/trial-credits', { token: ctx.learnerToken })
      assertStatus(credits, 200, 'GET trial credits')
      const tc = credits.body.trialCredits
      assertEq(typeof tc === 'number' ? tc : tc?.credits, 0, 'trial credits after booking')
    },
  },
  // ------------------------------------------------------------------
  // Learn tab (dashboard, scenarios, drills)
  // ------------------------------------------------------------------
  {
    id: 'TC-LEARN-01',
    reqs: ['REQ-LEARN-01'],
    name: 'learning dashboard returns the EN→ES course (Learn tab is not content-empty)',
    fn: async (ctx) => {
      const res = await http('GET', '/api/v1/learning/dashboard?targetLanguage=es&nativeLanguage=en', {
        token: ctx.learnerToken,
      })
      assertStatus(res, 200, 'GET /learning/dashboard')
      const dash = res.body.data
      assert(dash, 'dashboard payload present')
      const s = JSON.stringify(dash)
      assert(s.includes('es') && s.length > 50, `dashboard must carry course content, got: ${s.slice(0, 300)}`)
    },
  },
  {
    id: 'TC-LEARN-02',
    reqs: ['REQ-LEARN-02'],
    name: 'scenario library has the full seeded set (>=6 scenarios with real metadata)',
    fn: async (ctx) => {
      const res = await http('GET', '/api/v1/learning/scenarios?targetLanguage=es&nativeLanguage=en', {
        token: ctx.learnerToken,
      })
      assertStatus(res, 200, 'GET /learning/scenarios')
      const scenarios = res.body.data
      assert(Array.isArray(scenarios), `scenarios must be an array, got ${JSON.stringify(res.body).slice(0, 200)}`)
      assert(scenarios.length >= 6, `expected >=6 seeded scenarios (ordering coffee + 5), got ${scenarios.length}`)
      const titles = scenarios.map((x: any) => x.title)
      for (const expected of ['Ordering Coffee', 'Dinner at a Restaurant', 'Asking for Directions', 'Shopping at the Market', 'Hotel Check-in', 'Job Interview']) {
        assert(titles.some((t: string) => t.includes(expected)), `scenario library missing "${expected}"`)
      }
      for (const s of scenarios) {
        assert(s.id && s.title && s.domain && s.cefrLevel, `scenario ${JSON.stringify(s).slice(0, 150)} lacks id/title/domain/cefrLevel`)
      }
    },
  },
  {
    id: 'TC-LEARN-03',
    reqs: ['REQ-LEARN-02'],
    name: 'scenario detail includes phases with chunk banks (scaffolding for the roleplay)',
    fn: async (ctx) => {
      const list = await http('GET', '/api/v1/learning/scenarios?targetLanguage=es&nativeLanguage=en', {
        token: ctx.learnerToken,
      })
      const coffee = (list.body.data ?? []).find((s: any) => String(s.title).includes('Ordering Coffee'))
      assert(coffee, 'ordering coffee scenario present')
      const detail = await http('GET', `/api/v1/learning/scenarios/${coffee.id}`, { token: ctx.learnerToken })
      assertStatus(detail, 200, 'GET /learning/scenarios/:id')
      const sc = detail.body.data
      const phases = sc.phases ?? sc.script?.phases
      assert(Array.isArray(phases) && phases.length >= 4, `expected >=4 phases, got ${JSON.stringify(sc).slice(0, 300)}`)
      assert(phases.every((p: any) => Array.isArray(p.chunkBank) && p.chunkBank.length > 0), 'every phase needs a chunk bank')
    },
  },
  {
    id: 'TC-LEARN-04',
    reqs: ['REQ-LEARN-03'],
    name: 'a quick drill session starts and returns practice items (Vocabulary/Drills tab content)',
    fn: async (ctx) => {
      const res = await http('POST', '/api/v1/learning/sessions/start', {
        token: ctx.learnerToken,
        json: { targetLanguage: 'es', nativeLanguage: 'en', mode: 'drill', source: 'acceptance' },
      })
      assertStatus(res, 200, 'POST /learning/sessions/start')
      const data = res.body.data
      assert(data?.session?.id, 'session id returned')
      assert(Array.isArray(data.items) && data.items.length > 0, `drill must return practice items, got ${JSON.stringify(data).slice(0, 300)}`)
    },
  },
]
