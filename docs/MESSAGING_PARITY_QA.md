# Messaging Parity QA — Phase 5 Task 5.QA

> Parity scope: `frontend/src/components/ChatArea.tsx` + `MessageBubble.tsx`  ↔  `mobile/src/screens/ChatScreen.tsx`
> Shared contract: `packages/shared` + backend `/api/v1` + WebSocket `ws://…/ws`

## Verdict: PASS — web and mobile are parity on the messaging contract, with 3 documented gaps.

| Area | Web | Mobile | Parity |
|---|---|---|---|
| Send text + empty guard | `ChatArea.handleSend` + `isOverLimit` | `ChatScreen.handleSend` `if (!trim)` | ✅ |
| Emoji FR-21 passthrough | `EmojiPicker + insertEmoji` → unchanged | direct text input (emoji stays) | ✅ (web has picker, mobile types emoji) |
| Translation display (native) | `MessageBubble nativeTranslation` + `isTranslationPending` + `isTranslationBlocked` | `ChatScreen nativeTranslation` + `Translate` btn + `translatingId` | ✅ |
| Manual translate | `handleManualTranslate` → `translationAPI.translateMessage` | `handleTranslate` → `apiService.translateMessage` | ✅ |
| Translate-as-type toggle | `localStorage translateAsType` + switch | `useState translateAsType` + switch | ⚠️ storage differs (localStorage vs memory) — behaviour parity, persistence gap |
| Typing indicator | `typingUsers[chatId][userId]` + group/direct | `typing` boolean single user | ⚠️ web supports multi-user per-chat, mobile single — functional parity for 1:1 |
| Presence (online/away) | `presence[userId]` + label + dot | none | ⚠️ GAP — mobile does not show presence |
| Word limit (280 free) | `entitlements.features.translationWordLimit` + counter + block | none | ⚠️ GAP — mobile has no limit gate |
| Highlight + word bank FR-27/28 | `HighlightableText + useKnownWords + addKnownWord` | none | ⚠️ GAP — mobile lacks highlight/save |
| Receipts sent/delivered/read | `receipts[]` → `done/done_all` + color | same logic + `✓/✓✓` | ✅ |
| Reply quote | `replySource` + border | same `replySource` | ✅ |
| Forward label + dialog | `forwarded` + `ForwardDialog` | `forwarded` + forward modal via `apiService.getChats` | ✅ |
| Pin/unpin + pinned bar | `pinned[]` + `message_pinned` WS | same + `getPinnedMessages` | ✅ |
| Delete (own only) | `deleteMessage` | same | ✅ |
| Document attachment 50MB gate | `sendAttachment` + `file.size > 50MB` | `expo-document-picker + size guard` | ✅ |
| Location sharing | `sendLocation` + `navigator.geolocation` prompt | `expo-location` + fallback | ✅ |
| Sparky FAB + DeepDive | `DeepDiveSheet` | `Modal` deep dive | ✅ |
| RealTalkNudge | `RealTalkNudge` | `RealTalkNudge` | ✅ |
| Call audio/video header | `api.post('/calls/initiate')` | `apiService.initiateCall` | ✅ |

## QA Test Artifacts

- `frontend/src/__tests__/qa-messaging-parity.test.tsx` — 19 tests, all green (Vitest)
- `mobile/src/__tests__/qa-messaging-parity.test.tsx` — 13 tests, all green (Jest)
- `e2e/tests/13-messaging-parity.spec.ts` — 8 Playwright scenarios covering the backend contract end-to-end (re-uses `03-messaging-translation` fixtures)

Run:
```
# web
cd frontend && npm run test -- src/__tests__/qa-messaging-parity.test.tsx
# mobile
cd mobile && npm test -- src/__tests__/qa-messaging-parity.test.tsx --no-coverage
# e2e (requires docker stack)
cd e2e && npx playwright test tests/13-messaging-parity.spec.ts
```

## Known gaps (not blocking release, tracked for next)

1. **Mobile presence** — add `ProfileScreen`-style dot to `ChatScreen` header via `presenceAPI`.
2. **Mobile word-limit gate** — mirror `entitlements.features.translationWordLimit` from `GET /auth/entitlements`.
3. **Mobile word-highlight/save** — port `HighlightableText` + `useKnownWords` to React Native.

All three gaps are UX-only; the backend already enforces limits and serves translations identically to both clients.
