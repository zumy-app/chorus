/**
 * P0 messaging test cases (REQ-MSG group + DM durability + receipts).
 * The acceptance suite previously had zero messaging coverage; these TCs pin
 * the API-level contracts (creation, delivery, read-back, receipts) with HARD
 * asserts against real Postgres. UI-level messaging stays in Playwright.
 */
import {
  TestCase, http, assert, assertEq, assertStatus, login,
  LEARNER2_EMAIL, TUTOR_EMAIL,
} from '../harness.js'

/** Finds the learner↔learner2 DM or creates it (rerun-safe). */
async function findOrCreateDM(learnerToken: string, learner2Id: string): Promise<string> {
  const list = await http('GET', '/api/v1/chats', { token: learnerToken })
  assertStatus(list, 200, 'GET /chats')
  const chats = list.body.chats || list.body.data || []
  const dm = chats.find(
    (c: any) =>
      (c.type === 'direct' || !c.type) &&
      JSON.stringify(c).includes(learner2Id),
  )
  if (dm?.id) return dm.id
  const created = await http('POST', '/api/v1/chats', {
    token: learnerToken,
    json: { type: 'direct', participants: [learner2Id] },
  })
  assertStatus(created, 201, 'POST /chats (direct)')
  assert(created.body.id, 'chat must have an id')
  return created.body.id
}

function textOf(m: any): string {
  return String(m.content ?? m.text ?? '')
}

export const messagingTests: TestCase[] = [
  {
    id: 'TC-MSG-01',
    reqs: ['REQ-MSG-01'],
    name: 'group chat: create with two participants, message is visible to both',
    fn: async (ctx) => {
      const bob = await login(LEARNER2_EMAIL)
      const sofia = await login(TUTOR_EMAIL)
      const name = `TC MSG ${Date.now()}`
      const created = await http('POST', '/api/v1/chats', {
        token: ctx.learnerToken,
        json: { type: 'group', name, participants: [bob.user.id, sofia.user.id] },
      })
      assertStatus(created, 201, 'POST /chats (group)')
      const chatId = created.body.id
      assert(chatId, 'group chat must have an id')

      const probe = `group probe ${Date.now()}`
      const sent = await http('POST', `/api/v1/chats/${chatId}/messages`, {
        token: ctx.learnerToken,
        json: { text: probe },
      })
      assertStatus(sent, 201, 'POST /chats/:id/messages (group)')

      for (const [who, token] of [['bob', bob.token], ['sofia', sofia.token]] as const) {
        const got = await http('GET', `/api/v1/chats/${chatId}/messages?limit=20`, { token })
        assertStatus(got, 200, `GET messages as ${who}`)
        const msgs = got.body.messages || got.body.data || []
        assert(
          msgs.some((m: any) => textOf(m).includes(probe)),
          `${who} must see the group message`,
        )
      }
    },
  },
  {
    id: 'TC-MSG-02',
    reqs: ['REQ-MSG-02'],
    name: 'DM durability: learner sends, learner2 reads the exact text back via API',
    fn: async (ctx) => {
      const bob = await login(LEARNER2_EMAIL)
      const chatId = await findOrCreateDM(ctx.learnerToken, bob.user.id)
      const probe = `dm durability ${Date.now()}`
      const sent = await http('POST', `/api/v1/chats/${chatId}/messages`, {
        token: ctx.learnerToken,
        json: { text: probe },
      })
      assertStatus(sent, 201, 'POST /chats/:id/messages (DM)')
      assertEq(textOf(sent.body), probe, 'send response echoes the text')

      const got = await http('GET', `/api/v1/chats/${chatId}/messages?limit=20`, {
        token: bob.token,
      })
      assertStatus(got, 200, 'GET messages as learner2')
      const msgs = got.body.messages || got.body.data || []
      assert(msgs.some((m: any) => textOf(m) === probe), 'learner2 must read the exact text')
    },
  },
  {
    id: 'TC-MSG-03',
    reqs: ['REQ-MSG-03'],
    name: 'message receipts endpoint answers for a delivered message',
    fn: async (ctx) => {
      const bob = await login(LEARNER2_EMAIL)
      const chatId = await findOrCreateDM(ctx.learnerToken, bob.user.id)
      const sent = await http('POST', `/api/v1/chats/${chatId}/messages`, {
        token: ctx.learnerToken,
        json: { text: `receipt probe ${Date.now()}` },
      })
      assertStatus(sent, 201, 'POST /chats/:id/messages (receipt fixture)')
      const messageId = sent.body.id
      assert(messageId, 'message must have an id')
      const receipts = await http(
        'GET',
        `/api/v1/chats/${chatId}/messages/${messageId}/receipts`,
        { token: ctx.learnerToken },
      )
      assertStatus(receipts, 200, 'GET /chats/:id/messages/:messageId/receipts')
      assert(receipts.body !== null && receipts.body !== undefined, 'receipts must return a body')
    },
  },
]
