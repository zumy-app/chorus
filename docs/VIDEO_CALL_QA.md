# Video Call QA — Phase 8 (8.1-8.4)

## Scope
Video calling, screen sharing, immersive captions, desktop dashboard control center — wireframes `chorus_video_call_experience`, `audio_call_with_live_captions` + `wireframes/1.md` curriculum.

## Backend Contract
| Endpoint | Method | Auth | Validates | Tests |
|---|---|---|---|---|
| `/calls/initiate` (video) | POST | JWT | chat participant, type video | `VideoQA_Initiate_Video_Success` 201 + offer, `InitiateCall_Video_SetsVideoType` |
| `/calls/:callId/signal` video-toggle | POST | JWT | participant, active | `VideoQA_Signal_VideoToggle_Success` 200, `HandleSignal_VideoToggle_Success` |
| `/calls/:callId/signal` screen-share-start | POST | JWT | participant, active | `ScreenShare_StartStop` 200 |
| `/calls/:callId/signal` screen-share-stop | POST | JWT | participant, active | `ScreenShare_StartStop` 200, underscore alias |
| `/calls/:callId/signal` ice-candidate | POST | JWT | candidate required | `IceCandidate_Success`, `JSONCandidate` map handling |
| `/calls/:callId/captions` (video context) | POST/GET | JWT | non-empty, pagination | `ImmersiveCaptions_VideoCaptionFlow` |
| `GET /calls/:callId` video | GET | JWT | participant | `GetSession_Video_ReturnsType` type=video |
| TURN/STUN offer | internal | — | env-driven ICE | `GenerateOffer_TurnServers` 2 servers |

## Web Contract (CallScreen.tsx)
- Header: `formatDuration` mm:ss, type badge `Video`/`Audio`, duration pulsing dot.
- Video dual-view: `remote-video` + `local-video` (or `screen-video` when sharing). `Toggle layout` button switches PiP vs grid (`dualView`).
- Controls: Mute (`Mute microphone`), Camera (`Toggle camera` → video-toggle signal), Screen Share (`Toggle screen share` → screen-share-start/stop), Speaker, Captions, End. All `aria-pressed`.
- Immersive captions: `data-testid=immersive-captions` overlays last 2-3 segments, hidden when `immersive=false` or `captionsEnabled=false`, toggled via header checkbox.
- Transcript panel `#transcript-panel`, `transcript-scroll`, `Translated`/`Captions` toggles, `Save phrase` per segment, word chip `Save "word"`, `Load older captions`, `Type a caption...` input, live transcribe button.
- WebRTC: `RTCPeerConnection` STUN default, TURN via `WEBRTC_*`; `getUserMedia({video:true})` for video, `getDisplayMedia` for screen share, `replaceTrack` for toggling.

## Mobile Contract (CallScreen.tsx)
- Header: avatar + chatName + `Video call` subtitle + `Sharing`/`Remote sharing` badge.
- Video: PiP vs dual view (`▣`/`◫`), remote `Remote video`, local `Cam off` vs `You`/`Your screen`.
- Immersive overlay `immersive-captions` bottom positioned, last 2 segs.
- Transcript bottom sheet `transcript-panel` with drag handle, `Live captions`, `Translated on/off`, `Immersive on/off`, `Load older captions`, `Transcript auto-scrolls`.
- Controls: mute 🎤, camera 📹, screen share 🖥️, speaker, transcript 💬, End.

## Dashboard (Dashboard.tsx)
- Route: `/dashboard`, `data-testid=dashboard-page`. Responsive `lg:grid-cols-12`, 4 stat cards (`Active chats`, `Recent calls`, `Day streak`, `Vocabulary`), 3 panels: `dashboard-chats-panel`, `dashboard-calls-panel` (calls with video/audio icon, status badge), `dashboard-learning-panel` (progress ring, fluency, mastered, `Up next`).

## Test Files
- `backend/internal/services/video_qa_test.go` — 11 tests: video-toggle, screen-share start/stop + alias, video initiate, audio fallback, TURN offer, video getSession, ice-candidate JSON, answer SDP, publish caption in video, valid types.
- `backend/internal/handlers/video_qa_test.go` — 10 tests: initiate video 201, signal video-toggle/screen-share-start/stop/ice-candidate 200, offer missing SDP 400, notParticipant 403, alreadyEnded 400, getSession video type, video caption flow.
- `frontend/src/__tests__/qa-video.test.tsx` — 11 tests: remote/local video, dual-view toggle, immersive overlay + toggle, screen share signals, camera video-toggle, audio→video, captions hide/show, sharing badge, duration.
- `frontend/src/pages/__tests__/Dashboard.test.tsx` — 2 tests: all panels render, responsive grid classes.
- `mobile/src/__tests__/qa-video.test.tsx` — 9 tests: header video badge, dual-view, immersive + toggle, screen share signal, camera, remote video, Cam off, controls, duration.
- `e2e/tests/15-video-call.spec.ts` — 7 scenarios: initiate video 201 type video+offer, screen placeholders, screen-share start/stop 200, video-toggle+ice 200, immersive caption publish+paginate, dashboard panels, unauth 401.

## Execution
```
backend:  go test ./...  — PASS (services 36+ handler 32 video+call tests)
frontend: npm test       — PASS (17 files 191 tests incl qa-video 11 + Dashboard 2)
mobile:   npm test       — PASS (6 suites 84 tests incl qa-video 9)
e2e:      npx playwright test 15-video-call — requires running dev stack (docker-compose.dev.yml)
build:    frontend npm run build — PASS (no type errors)
```
