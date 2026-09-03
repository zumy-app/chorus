/**
 * P0 foundation test cases (REQ-ENV, REQ-AUTH, REQ-REG, REQ-ERR).
 * Freeze policy: this file is QA-owned; changing/weakening tests requires
 * human approval (rescue plan §4b binding policy).
 */
import {
  TestCase, http, assert, assertEq, assertStatus,
  INVITE_EMAIL, INVITE_TOKEN,
} from '../harness.js'

export const foundationTests: TestCase[] = [
  {
    id: 'TC-ENV-01',
    reqs: ['REQ-ENV-01'],
    name: 'health endpoint reports version + build commit so start scripts can verify the served binary',
    fn: async () => {
      const res = await http('GET', '/health')
      assertStatus(res, 200, 'GET /health')
      assertEq(res.body.status, 'healthy', 'health status')
      assert(typeof res.body.commit === 'string' && res.body.commit.length > 0, 'health payload must include "commit"')
      assert(typeof res.body.checks === 'object', 'health payload must include per-dependency checks')
      // When CHORUS_BUILD_COMMIT is set (start scripts do this), the served
      // commit must match git HEAD so stale binaries are detected.
      const expected = process.env.CHORUS_EXPECTED_COMMIT
      if (expected) {
        assertEq(res.body.commit, expected, 'served commit must match git HEAD')
      }
    },
  },
  {
    id: 'TC-AUTH-01',
    reqs: ['REQ-AUTH-01'],
    name: 'fixture learner can log in and the token authenticates /users/me',
    fn: async (ctx) => {
      const res = await http('GET', '/api/v1/users/me', { token: ctx.learnerToken })
      assertStatus(res, 200, 'GET /users/me')
      const user = res.body.data ?? res.body.user ?? res.body
      assertEq(user.email, 'alice.dev@chorus.test', 'authenticated user email')
    },
  },
  {
    id: 'TC-AUTH-02',
    reqs: ['REQ-AUTH-02'],
    name: 'a valid token for an existing user never fails with "User not found" (split-brain regression)',
    fn: async (ctx) => {
      // Repeat several times: the production bug was intermittent (wrong DB
      // listener), so a single pass is not sufficient evidence.
      for (let i = 0; i < 5; i++) {
        const res = await http('GET', '/api/v1/users/me', { token: ctx.tutorToken })
        assertStatus(res, 200, `GET /users/me (attempt ${i + 1})`)
        assert(
          !JSON.stringify(res.body).includes('User not found'),
          `attempt ${i + 1} must not return "User not found"`,
        )
      }
    },
  },
  {
    id: 'TC-REG-01',
    reqs: ['REQ-REG-01'],
    name: 'open registration (dev flag) creates an account without an invite token',
    fn: async () => {
      const suffix = Date.now()
      const email = `tc-reg01-${suffix}@chorus.test`
      const res = await http('POST', '/api/v1/auth/register', {
        json: {
          username: email,
          email,
          password: 'ProbePass123!',
          displayName: 'TC Reg 01',
          nativeLanguage: 'en',
          targetLanguages: ['es'],
        },
      })
      assertStatus(res, 201, 'POST /auth/register without invite token (ALLOW_OPEN_REGISTRATION=true)')
      assert(res.body?.tokens?.accessToken, 'registration response must include tokens')
      // The new account must be immediately usable.
      const me = await http('GET', '/api/v1/users/me', { token: res.body.tokens.accessToken })
      assertStatus(me, 200, 'GET /users/me with fresh token')
    },
  },
  {
    id: 'TC-REG-02',
    reqs: ['REQ-REG-02'],
    name: 'registration with the seeded dev invite token succeeds and consumes the invitation',
    fn: async () => {
      const res = await http('POST', '/api/v1/auth/register', {
        json: {
          username: INVITE_EMAIL,
          email: INVITE_EMAIL,
          password: 'ProbePass123!',
          displayName: 'TC Reg 02',
          nativeLanguage: 'en',
          targetLanguages: ['es'],
          inviteToken: INVITE_TOKEN,
        },
      })
      assertStatus(res, 201, 'POST /auth/register with dev invite token')
      assert(res.body?.tokens?.accessToken, 'invited registration must return tokens')
      // Invitation is consumed: a second registration with the same token
      // (different email) must be rejected.
      const email2 = `tc-reg02-dupe-${Date.now()}@chorus.test`
      const dupe = await http('POST', '/api/v1/auth/register', {
        json: {
          username: email2,
          email: email2,
          password: 'ProbePass123!',
          displayName: 'TC Reg 02 dupe',
          nativeLanguage: 'en',
          targetLanguages: [],
          inviteToken: INVITE_TOKEN,
        },
      })
      assert(dupe.status >= 400 && dupe.status < 500, `consumed invite must be rejected, got HTTP ${dupe.status}`)
    },
  },
  {
    id: 'TC-REG-03',
    reqs: ['REQ-REG-03'],
    name: 'registration with an invalid invite token is rejected with a readable error',
    fn: async () => {
      const email = `tc-reg03-${Date.now()}@chorus.test`
      const res = await http('POST', '/api/v1/auth/register', {
        json: {
          username: email,
          email,
          password: 'ProbePass123!',
          displayName: 'TC Reg 03',
          nativeLanguage: 'en',
          targetLanguages: [],
          inviteToken: 'definitely-not-a-real-token',
        },
      })
      assert(res.status >= 400 && res.status < 500, `invalid invite must be rejected, got HTTP ${res.status}`)
      const msg = JSON.stringify(res.body).toLowerCase()
      assert(
        msg.includes('invitation') || msg.includes('invite'),
        'rejection message must mention the invitation',
      )
    },
  },
  {
    id: 'TC-ERR-01',
    reqs: ['REQ-ERR-01'],
    name: 'unauthenticated requests get the standard error envelope {code, message}',
    fn: async () => {
      const res = await http('GET', '/api/v1/users/me')
      assertStatus(res, 401, 'GET /users/me without token')
      assertEq(res.body.code, 'UNAUTHORIZED', 'error envelope code')
      assert(typeof res.body.message === 'string' && res.body.message.length > 0, 'error envelope message')
    },
  },
  {
    id: 'TC-ERR-02',
    reqs: ['REQ-ERR-01'],
    name: 'unknown tutor id returns a structured 404, not a crash',
    fn: async (ctx) => {
      const res = await http('GET', '/api/v1/teachers/00000000-0000-0000-0000-000000000000', {
        token: ctx.learnerToken,
      })
      assertStatus(res, 404, 'GET unknown tutor')
      assertEq(res.body.code, 'NOT_FOUND', 'error envelope code')
    },
  },
]
