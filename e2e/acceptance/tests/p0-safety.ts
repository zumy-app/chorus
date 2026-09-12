/**
 * P0 safety test cases (REQ-SAFETY blocks/reports, REQ-GDPR export/erasure).
 * All asserts are HARD (throw on failure) — no mocks, real backend + Postgres.
 * Throwaway accounts keep fixtures pristine; block state is always cleaned up
 * in a finally so later suites never inherit a block edge.
 */
import {
  TestCase, http, assert, assertEq, assertStatus, login, registerTemp,
  LEARNER2_EMAIL,
} from '../harness.js'

export const safetyTests: TestCase[] = [
  {
    id: 'TC-SAFE-01',
    reqs: ['REQ-SAFETY-01'],
    name: 'filing a report on a real message returns 201 with a report id',
    fn: async (ctx) => {
      // Build a real message to report: DM learner -> learner2.
      const bob = await login(LEARNER2_EMAIL)
      const chatRes = await http('POST', '/api/v1/chats', {
        token: ctx.learnerToken,
        json: { type: 'direct', participants: [bob.user.id] },
      })
      assertStatus(chatRes, 201, 'POST /chats (direct for report fixture)')
      const chatId = chatRes.body.id
      assert(chatId, 'chat must have an id')
      const msgRes = await http('POST', `/api/v1/chats/${chatId}/messages`, {
        token: ctx.learnerToken,
        json: { text: `report fixture ${Date.now()}` },
      })
      assertStatus(msgRes, 201, 'POST /chats/:id/messages')
      const messageId = msgRes.body.id
      assert(messageId, 'message must have an id')

      const rep = await http('POST', '/api/v1/reports', {
        token: ctx.learner2Token,
        json: { type: 'message', messageId, reason: 'acceptance test spam probe' },
      })
      assertStatus(rep, 201, 'POST /reports')
      assert(rep.body.id, 'report must have an id')
    },
  },
  {
    id: 'TC-SAFE-02',
    reqs: ['REQ-SAFETY-02'],
    name: 'block lifecycle: block bob → direct chat creation is 403 → unblock restores (finally-guarded)',
    fn: async (ctx) => {
      const bob = await login(LEARNER2_EMAIL)
      const bobId = bob.user.id
      assert(bobId, 'bob must have an id')
      try {
        const block = await http('POST', '/api/v1/blocks', {
          token: ctx.learnerToken,
          json: { blockedUserId: bobId },
        })
        assertStatus(block, 201, 'POST /blocks')

        const listed = await http('GET', '/api/v1/blocks', { token: ctx.learnerToken })
        assertStatus(listed, 200, 'GET /blocks')
        const blocks = listed.body.blocks || listed.body.data || []
        assert(
          blocks.some((b: any) => b.blockedId === bobId || b.blocked?.id === bobId),
          'GET /blocks must list bob after block',
        )

        // Enforcement proof: starting a NEW direct chat with a blocked user is forbidden.
        const forbidden = await http('POST', '/api/v1/chats', {
          token: ctx.learnerToken,
          json: { type: 'direct', participants: [bobId] },
        })
        assertStatus(forbidden, 403, 'POST /chats (direct, blocked user)')
      } finally {
        // Unblock answers 204 No Content.
        const unblock = await http('DELETE', `/api/v1/blocks/${bobId}`, { token: ctx.learnerToken })
        assertStatus(unblock, 204, 'DELETE /blocks/:userId')
        const listed = await http('GET', '/api/v1/blocks', { token: ctx.learnerToken })
        assertStatus(listed, 200, 'GET /blocks after unblock')
        const blocks = listed.body.blocks || listed.body.data || []
        assert(
          !blocks.some((b: any) => b.blockedId === bobId || b.blocked?.id === bobId),
          'block edge must be gone after unblock',
        )
      }
    },
  },
  {
    id: 'TC-GDPR-01',
    reqs: ['REQ-GDPR-01'],
    name: 'GDPR export returns the account data as JSON for the owner',
    fn: async (ctx) => {
      const temp = await registerTemp(ctx.learnerToken, 'tc-gdpr-export')
      const exp = await http('GET', '/api/v1/users/me/export', { token: temp.token })
      assertStatus(exp, 200, 'GET /users/me/export')
      assertEq(JSON.stringify(exp.body).includes(temp.email), true, 'export must contain the account email')
      // Cleanup: erase the throwaway so reruns stay deterministic.
      const del = await http('DELETE', '/api/v1/users/me', { token: temp.token })
      assertStatus(del, 200, 'DELETE /users/me (throwaway cleanup)')
    },
  },
  {
    id: 'TC-GDPR-02',
    reqs: ['REQ-GDPR-02'],
    name: 'GDPR erasure: DELETE /users/me removes the account and its tokens die',
    fn: async (ctx) => {
      const temp = await registerTemp(ctx.learnerToken, 'tc-gdpr-erase')
      const del = await http('DELETE', '/api/v1/users/me', { token: temp.token })
      assertStatus(del, 200, 'DELETE /users/me')
      const relogin = await http('POST', '/api/v1/auth/login', {
        json: { username: temp.email, password: 'ProbePass123!' },
      })
      assertStatus(relogin, 401, 'login after erasure must fail')
      // Soft-deleted accounts authenticate as disabled (403), not unknown
      // (401): the token still decodes, the account gate rejects it.
      const stale = await http('GET', '/api/v1/users/me', { token: temp.token })
      assertStatus(stale, 403, 'old access token must die with the account')
      assert(
        String(stale.body?.error?.message ?? '').toLowerCase().includes('disabled'),
        `stale token must report a disabled account, got: ${JSON.stringify(stale.body)}`,
      )
    },
  },
]
