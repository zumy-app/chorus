# BACKLOG — Contextual Reply Suggestions (SHELVED, not in prod release)

> **Status: SHELVED.** Do not implement until after the minimal prod release.
> Tracked in GitHub issues (see footer). This doc is the full spec snapshot so
> the work restarts from design, not from zero.

## Problem
Sparky's nudge recommendations (`RealTalkNudge`) are generic daily practice
prompts, unrelated to the conversation. E.g. after receiving
"can you meet me at 8pm", the nudge may suggest an opinion phrase instead of a
usable reply like "yes, that's fine".

## Agreed design (reviewed, not yet built)

**Separate feature from RealTalk practice prompts.** Reply suggestions are a
conversational aid; RealTalk prompts are pedagogical material. Different data,
latency, and UX needs — do not merge them.

**Precompute at send, don't generate on view.** When a message is sent, enqueue
a suggestion job for each recipient alongside the existing translation/grammar
fan-out. The recipient sees instant results; the 3–25s LLM latency happens
before they open the chat. (Generating on view reintroduces the Sparky-timeout
saga.)

**Thread context, not single message.** Job input = last ~10 messages of the
chat with the newest marked as reply target. Trigger timing and input breadth
are independent: send-time triggering does not limit context (history is in DB).

**Keying + display rules.**
- Cache key `(message_id, lang)`, TTL ~30 days.
- Client renders only the set for the **latest incoming message** (sender != me).
- Suppress once the user has sent anything after that message (dead turn).
- Skip when message language == recipient native language (zero learning
  value; mirrors the grammar native-skip).
- 1:1 chats only in v1; groups explicitly skipped in the send hook.
- Tap inserts into the composer for editing — **never auto-sends**.

**Fallback endpoint.** `GET /messages/:id/reply-suggestions`: fast-fail sync,
~8s timeout, silent 204 on failure (chips simply don't appear). No WS plumbing
for the fallback. Client fires on chat-open/new-message, retries once after
~5s, then drops it.

**Result shape.** 3 short replies in the message's language + optional
native-language gloss; CEFR-aware wording from the recipient profile;
mechanical brevity cap (≤12 words A1, ≤20 otherwise).

**Cost control.** Precompute only when the recipient was recently active
(presence service); otherwise generate lazily on first open. One short
completion (~100 tokens out) per message per language, cached indefinitely.

**Safety.** Suggestions are LLM output derived from user messages shown to
another user — blast radius is contained solely because nothing auto-sends.
Name that as load-bearing. Short/neutral/beginner-worded; skip own + system +
stale messages.

**Quality loop (required, not optional).** Tap-through logging (shown vs tapped)
+ a ~20-thread golden set reviewed by a speaker before enabling for all.
Contract tests cover shape/cache/skip-rules only.

## Open questions (unresolved at shelve time)
1. Reply language = message language always? (Recommendation: yes.)
2. Show the native-language gloss under each chip? (Recommendation: yes, subtle.)
3. 1:1 chats only to start? (Recommendation: yes.)

## Tracking
- GitHub issue: _reply-suggestions umbrella_ (created via `GITHUB_CHORUS_ISSUES_PAT`; link added when created).
- Revisit only after: minimal prod release is stable + tap-through metrics infra exists.
