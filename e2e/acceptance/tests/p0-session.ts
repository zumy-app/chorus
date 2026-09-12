/**
 * P0 session test cases (REQ-AUTH session lifecycle).
 * Token refresh, password-reset flow contracts, 2FA negatives, auth gating.
 * All asserts are HARD (throw on failure) — no mocks, real backend + Postgres.
 */
import {
  TestCase, http, assert, assertEq, assertStatus,
  DEV_PASSWORD, LEARNER_EMAIL,
} from '../harness.js'

export const sessionTests: TestCase[] = [
  {
    id: 'TC-SESS-01',
    reqs: ['REQ-AUTH-03'],
    name: 'refresh token rotation works: POST /auth/refresh yields a live access token, garbage is 401',
    fn: async () => {
      const loginRes = await http('POST', '/api/v1/auth/login', {
        json: { username: LEARNER_EMAIL, password: DEV_PASSWORD },
      })
      assertStatus(loginRes, 200, 'POST /auth/login')
      const refreshToken = loginRes.body?.tokens?.refreshToken
      assert(refreshToken, 'login must issue a refresh token')

      const ref = await http('POST', '/api/v1/auth/refresh', { json: { refreshToken } })
      assertStatus(ref, 200, 'POST /auth/refresh')
      assert(ref.body?.accessToken, 'refresh must return a new accessToken')

      const me = await http('GET', '/api/v1/users/me', { token: ref.body.accessToken })
      assertStatus(me, 200, 'GET /users/me with refreshed token')

      const bad = await http('POST', '/api/v1/auth/refresh', { json: { refreshToken: 'not-a-token' } })
      assertStatus(bad, 401, 'POST /auth/refresh with garbage')
    },
  },
  {
    id: 'TC-SESS-02',
    reqs: ['REQ-AUTH-04'],
    name: 'forgot-password is anti-enumeration: known and unknown emails get the identical 200 message',
    fn: async () => {
      const known = await http('POST', '/api/v1/auth/forgot-password', {
        json: { email: LEARNER_EMAIL },
      })
      assertStatus(known, 200, 'POST /auth/forgot-password (known email)')
      const unknown = await http('POST', '/api/v1/auth/forgot-password', {
        json: { email: `nobody-${Date.now()}@chorus.test` },
      })
      assertStatus(unknown, 200, 'POST /auth/forgot-password (unknown email)')
      assertEq(known.body?.message, unknown.body?.message, 'responses must be identical')
    },
  },
  {
    id: 'TC-SESS-03',
    reqs: ['REQ-AUTH-04'],
    name: 'reset-password rejects bogus tokens and short passwords with 400',
    fn: async () => {
      const bogus = await http('POST', '/api/v1/auth/reset-password', {
        json: { token: 'bogus-token', password: 'NewPass123!' },
      })
      assertStatus(bogus, 400, 'POST /auth/reset-password (bogus token)')
      const short = await http('POST', '/api/v1/auth/reset-password', {
        json: { token: 'bogus-token', password: 'short' },
      })
      assertStatus(short, 400, 'POST /auth/reset-password (short password)')
    },
  },
  {
    id: 'TC-SESS-04',
    reqs: ['REQ-AUTH-05'],
    name: '2FA negatives: enabling without a verified phone is 400, verifying with a bogus temp token is 401',
    fn: async (ctx) => {
      const enable = await http('PUT', '/api/v1/users/me/2fa', {
        token: ctx.learnerToken,
        json: { enabled: true },
      })
      assertStatus(enable, 400, 'PUT /users/me/2fa without verified phone')
      const errMsg = String(enable.body?.error?.message ?? enable.body?.message ?? '')
      assert(
        errMsg.toLowerCase().includes('phone'),
        `message must mention phone verification, got: ${JSON.stringify(enable.body)}`,
      )
      const verify = await http('POST', '/api/v1/auth/2fa/verify', {
        json: { tempToken: 'bogus-temp-token', code: '123456' },
      })
      assertStatus(verify, 401, 'POST /auth/2fa/verify with bogus temp token')
    },
  },
  {
    id: 'TC-SESS-05',
    reqs: ['REQ-AUTH-01'],
    name: 'protected endpoints reject missing/garbage tokens with 401',
    fn: async () => {
      const noAuth = await http('GET', '/api/v1/users/me')
      assertStatus(noAuth, 401, 'GET /users/me without token')
      const garbage = await http('GET', '/api/v1/users/me', { token: 'garbage.token.here' })
      assertStatus(garbage, 401, 'GET /users/me with garbage token')
    },
  },
]
