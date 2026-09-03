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
 *   learner: alice.dev@chorus.test   password: ChorusDev123!
 *   learner: bob.dev@chorus.test     password: ChorusDev123!
 *   tutor:   sofia.tutor@chorus.test password: ChorusDev123!
 *   invite:  invite.dev@chorus.test  token:    chorus-dev-invite-2026
 */

export const API_BASE = process.env.CHORUS_API || 'http://localhost:8080'

export const DEV_PASSWORD = 'ChorusDev123!'
export const LEARNER_EMAIL = 'alice.dev@chorus.test'
export const LEARNER2_EMAIL = 'bob.dev@chorus.test'
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
export async function login(email: string): Promise<{ token: string; user: any }> {
  const res = await http('POST', '/api/v1/auth/login', {
    json: { username: email, password: DEV_PASSWORD },
  })
  if (res.status !== 200 || !res.body?.tokens?.accessToken) {
    throw new Error(
      `Login failed for fixture ${email}: HTTP ${res.status} — ` +
        `${JSON.stringify(res.body)} — did you run "go run ./cmd/server --seed-dev"?`,
    )
  }
  return { token: res.body.tokens.accessToken, user: res.body.user }
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
