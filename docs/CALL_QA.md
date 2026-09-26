# Call QA — Phase 7 (Audio Calling with Smart Captions)

## Scope
7.1 WebRTC audio signaling, 7.2 live transcription + translated captions,
7.3 scrollable transcript panel, 7.4 vocab capture from captions, 7.TE teacher review.

## Backend Contract
| Endpoint | Method | Auth | Validates | Test |
|---|---|---|---|---|
| `/calls/initiate` | POST | JWT | chat participant, type audio|video | Initiate_NotParticipant 403, ChatNotFound 404, Success 201 + offer |
| `/calls/:callId` | GET | JWT | participant | GetSession_NotFound 404, NotParticipant 403 |
| `/calls/:callId/end` | POST | JWT | participant, not ended | AlreadyEnded 400, NotParticipant 403 |
| `/calls/:callId/signal` | POST | JWT | valid type, sdp/candidate for offer/answer/ice | InvalidType 400, MissingSDP 400 |
| `/calls/:callId/captions` | POST | JWT | non-empty text, participant, active | Empty 400, NotParticipant 403, Ended 400 |
| `/calls/:callId/captions` | GET | JWT | participant | NotParticipant 403, pagination limit/offset clamped |
| `/calls/:callId/captions/:index/bookmark` | POST | JWT | index in range, participant | OutOfRange 400 |
| `/calls/:callId/transcribe` | POST | JWT | base64 audio | Missing audio 400, invalid base64 400 |
| `/calls/transcripts/search?q=` | GET | JWT | q required | Missing q 400 |
| `GET /calls/:callId/transcript` | GET | JWT | participant | tested via GetCaptions |
| STT transcribe | internal | — | no STT → speech service not configured | TranscribeAndPublish_NoSTT |

## Web Contract (CallScreen.tsx)
- Header: avatar + chatName + duration + audio/video badge.
- Transcript panel (`#transcript-panel`, `data-testid=transcript-panel`): toggle via header button; close button.
- Live captions list (`data-testid=transcript-scroll`): empty state "Captions will appear here".
- Caption row: originalText + translated toggle (`translated-toggle`), captions toggle (`captions-toggle`), immersive toggle, per-word Save chip (`Save "word"`), Save phrase button (`save-phrase-{idx}`), load more.
- Input: `Type a caption...` + Send caption button; form submit POST `/calls/:id/captions`.
- Controls: Mute (mic/mic_off), Camera (videocam), Screen share (screen_share), Speaker (volume), Captions, End call → POST `/calls/:id/end` + overlay "Call ended".
- WebRTC: RTCPeerConnection with STUN `stun:stun.l.google.com:19302` default; TURN via env; offer/answer/ice-candidate via `/signal`.
- Duration: `formatDuration` mm:ss increments every 1s.

## Mobile Contract (CallScreen.tsx)
- Same backend endpoints via `apiService` (mobile/src/services/api.ts).
- Transcript panel bottom sheet with drag handle, `transcript-panel` testID.
- Controls: mute, camera, screen share, speaker, End; transcript toggles; word chip save via `phrase`.
- Immersive overlay `immersive-captions` shows latest 2 segments.

## Test Files
- `backend/internal/services/call_qa_test.go` — 22 tests: offer defaults, signal validation, initiate/end, publish caption (empty/notParticipant/ended/success), pagination, bookmark OOR, search filter, delete transcript, STT guard.
- `backend/internal/handlers/call_qa_test.go` — 22 tests: initiate/missing chat, notParticipant, end, getSession, postCaption, getCaptions, bookmark OOR, signal invalid/missing SDP/success, transcribe validation, search query.
- `frontend/src/__tests__/qa-call.test.tsx` — 23 tests: header, transcript panel, empty, load, translated/captions toggles, send (button+form), bookmark phrase/word, load more, mute, screen share, speaker, camera, end call, video placeholders, duration, same-translation hidden.
- `mobile/src/__tests__/qa-call.test.tsx` — 18 tests: header, panel, empty, caption+translation, translated toggle, send, bookmark, word chip, toggle collapsed, mute, camera, screen share, end, immersive, load more, etc.
- `e2e/tests/14-call-captions.spec.ts` — 6 scenarios: transcript panel + caption send, invalid signal, empty+paginate, bookmark OOR, unauth 401, controls parity.

## Execution
```
backend:  go test ./...                 — PASS (services 25s + handlers)
frontend: npm test (vitest)             — 23/23 pass in qa-call; full suite 178 tests
mobile:   npm test (jest)               — 18/18 pass in qa-call
e2e:      npx playwright test 14-call-captions -- TBD against running dev stack
```
