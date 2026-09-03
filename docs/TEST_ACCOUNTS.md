# Test Accounts — Chorus (dev/test only)

> **Seeded via** `backend/internal/services/dev_seed.go:16` with `go run ./cmd/server --seed-dev` (idempotent — deletes and recreates, invalidates old JWTs).
> **All use password:** `ChorusDev123!` (`DevPassword`). Never used in production.

| # | Email | Username | Display name | Native → Target | Role / Data | Password |
|---|---|---|---|---|---|---|
| 1 | `alice.dev@chorus.test` | `alice.dev` | `Alice Dev` | `en → es` | **Learner** — has 1 trial credit, 2 reviews for Sofia, used in `seedTutorMarketplace` | `ChorusDev123!` |
| 2 | `bob.dev@chorus.test` | `bob.dev` | `Bob Dev` | `es → en` | **Learner** — second learner for booking/review flows | `ChorusDev123!` |
| 3 | `sofia.tutor@chorus.test` | `sofia.tutor` | `Sofia Tutor` | `en → es` | **Approved tutor** — `teacher_applications status=approved`, `rate 2500¢ ($25)`, bio `Hola! I am Sofia...`, `language_certificate` verified `Instituto Cervantes 2018`, 4 future `tutor_availability` slots (1h/day next 4d), ratings 5★+4★ avg 4.5 | `ChorusDev123!` |

### Invite-gated registration helper

| Purpose | Email | Token | Expires | Notes |
|---|---|---|---|---|
| `TC-REG` — test real invite consumption (`RegisterWithInvitation`) | `invite.dev@chorus.test` | `chorus-dev-invite-2026` (`DevInviteToken`) | 30 days (`dev_seed.go:152`) | `sha256(token)` stored in `invitations.status=sent`; `POST /auth/register {inviteToken}` consumes it. `ALLOW_OPEN_REGISTRATION=false` still accepts it (`handlers/auth.go:68`). |

### How to use after a reset

```bash
# re-seed (wipes and recreates the 3 users + invitation)
cd backend && go run ./cmd/server --seed-dev
# Storage on emulator still holds old JWT (old user IDs) → clear or logout
adb -s emulator-5554 shell pm clear com.chorusmobile
# then in app:
# Login as alice.dev@chorus.test / ChorusDev123!  → Learn hub shows dueToday>0 + 5 es scenarios
# Login as sofia.tutor@chorus.test / ChorusDev123! → Profile → Become Teacher shows `approved` + Trial Credits
# Register new user with invite: POST /auth/register {email:"new@...", password:"...", inviteToken:"chorus-dev-invite-2026"}
```

### Notes

- `WAITLIST_ADMIN_EMAILS` admin is `info@chorus.talk` (not seeded) — create via registration then `UPDATE users SET role='admin'` if needed.
- `.env:VITE_DEFAULT_LOGIN_EMAIL=uhsarp@gmail.com / Demor@cer1` is unrelated Appwrite legacy, not Chorus DB.
- If you see `401 Invalid or expired token` after seeding, do `Profile → Logout` or `pm clear` — JWT `userId` changed.

> Run `curl http://localhost:8080/health` — `commit` != `2e55624` proves new binary with `teacher.go:60` fix is live.
