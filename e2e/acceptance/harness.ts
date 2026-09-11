/**
 * Acceptance harness for the Chorus rescue plan (plan phase B2).
 *
 * These are QA-owned functional tests that hit the REAL running stack:
 *   - real HTTP backend (default http://localhost:8080)
 *   - real Postgres data seeded by `go run ./cmd/server --seed-dev`
 * No mocks. Unit tests exist elsewhere; passing this suite is the Definition
 * of Done evidence required by the plan (docs/TEST_SPEC.md binding policy).
 *
 * Dev fixture credentials (backend/internal/services/dev_seed.go):
 *   learner (EN→ES): alice.en-es@chorus.test   password: ChorusDev123!
 *   learner (ES→EN): bob.es-en@chorus.test     password: ChorusDev123!
 *   tutor:   sofia.tutor@chorus.test password: ChorusDev123!
 *   invite:  invite.dev@chorus.test  token:    chorus-dev-invite-2026
 */

export const API_BASE = process.env.CHORUS_API || 'http://localhost:8080'

export const DEV_PASSWORD = 'ChorusDev123!'
export const LEARNER_EMAIL = 'alice.en-es@chorus.test'
export const LEARNER2_EMAIL = 'bob.es-en@chorus.test'
export const TUTOR_EMAIL = 'sofia.tutor@chorus.test'
export const INVITE_EMAIL = 'invite.dev@chorus.test'
export const INVITE_TOKEN = 'chorus-dev-invite-2026'

export interface HttpResponse {
  status: number
  body: any
  headers: Record<string, string>
}

/** Minimal JSON HTTP client (no external deps — keeps the suite hermetic). */
export async function http(
  method: string,
  path: string,
  opts: { token?: string; json?: unknown } = {},
): Promise<HttpResponse> {
  const headers: Record<string, string> = {}
  if (opts.json !== undefined) headers['Content-Type'] = 'application/json'
  if (opts.token) headers['Authorization'] = `Bearer ${opts.token}`
  const res = await fetch(`${API_BASE}${path}`, {
    method,
    headers,
    body: opts.json !== undefined ? JSON.stringify(opts.json) : undefined,
  })
  const text = await res.text()
  let body: any = null
  try {
    body = text ? JSON.parse(text) : null
  } catch {
    body = text
  }
  const hdrs: Record<string, string> = {}
  res.headers.forEach((v, k) => (hdrs[k] = v))
  return { status: res.status, body, headers: hdrs }
}

export interface Ctx {
  learnerToken: string
  learner2Token: string
  tutorToken: string
  learnerId: string
  tutorId: string
}

/** Logs in a fixture account; throws a readable error when the stack is broken. */
const loginCache = new Map<string, { token: string; user: any }>()
export async function login(email: string): Promise<{ token: string; user: any }> {
  // Memoized per run: /auth/login is rate-limited (10/15min/IP) and fixtures
  // are seeded once up front, so one login per account is enough. (Throwaway
  // accounts are registered fresh each time and never go through here.)
  const cached = loginCache.get(email)
  if (cached) return cached
  const res = await http('POST', '/api/v1/auth/login', {
    json: { username: email, password: DEV_PASSWORD },
  })
  if (res.status !== 200 || !res.body?.tokens?.accessToken) {
    throw new Error(
      `Login failed for fixture ${email}: HTTP ${res.status} — ` +
        `${JSON.stringify(res.body)} — did you run "go run ./cmd/server --seed-dev"?`,
    )
  }
  const out = { token: res.body.tokens.accessToken, user: res.body.user }
  loginCache.set(email, out)
  return out
}

export async function buildCtx(): Promise<Ctx> {
  const [a, b, s] = await Promise.all([
    login(LEARNER_EMAIL),
    login(LEARNER2_EMAIL),
    login(TUTOR_EMAIL),
  ])
  return {
    learnerToken: a.token,
    learner2Token: b.token,
    tutorToken: s.token,
    learnerId: a.user.id,
    tutorId: s.user.id,
  }
}

/** Test-case definition keyed by TC id from docs/TEST_SPEC.md. */
export interface TestCase {
  id: string
  reqs: string[] // REQ ids this test evidences
  name: string
  fn: (ctx: Ctx) => Promise<void>
}

/**
 * Registers a throwaway account through the real invite gate.
 * Production default is invite-gated (ALLOW_OPEN_REGISTRATION=false), so
 * tests mint an open SMS invite as a fixture user first, then register with
 * its single-use token. Returns the live token + email + id.
 */
export async function registerTemp(
  inviterToken: string,
  prefix: string,
): Promise<{ token: string; email: string; id: string }> {
  const email = `${prefix}-${Date.now()}@chorus.test`
  const inviteRes = await http('POST', '/api/v1/contacts/invites', {
    token: inviterToken,
    json: {
      channel: 'sms',
      contact: { name: prefix, phone: `+1555${String(Date.now()).slice(-7)}` },
    },
  })
  if (inviteRes.status !== 201 || !inviteRes.body?.data?.token) {
    throw new Error(
      `mint invite for ${email}: expected HTTP 201 with token, got ${inviteRes.status} — body: ${JSON.stringify(inviteRes.body)}`,
    )
  }
  const res = await http('POST', '/api/v1/auth/register', {
    json: {
      username: email,
      email,
      password: 'ProbePass123!',
      displayName: prefix,
      nativeLanguage: 'en',
      targetLanguages: ['es'],
      inviteToken: inviteRes.body.data.token,
    },
  })
  if (res.status !== 201 || !res.body?.tokens?.accessToken) {
    throw new Error(
      `temp registration (${email}): expected HTTP 201 with tokens, got ${res.status} — body: ${JSON.stringify(res.body)}`,
    )
  }
  return { token: res.body.tokens.accessToken, email, id: res.body.user.id }
}

export function assert(cond: unknown, msg: string): asserts cond {
  if (!cond) throw new Error(`Assertion failed: ${msg}`)
}

export function assertEq(actual: unknown, expected: unknown, msg: string): void {
  if (actual !== expected) {
    throw new Error(`${msg}: expected ${JSON.stringify(expected)}, got ${JSON.stringify(actual)}`)
  }
}

export function assertStatus(res: HttpResponse, expected: number, msg: string): void {
  if (res.status !== expected) {
    throw new Error(`${msg}: expected HTTP ${expected}, got ${res.status} — body: ${JSON.stringify(res.body)}`)
  }
}
