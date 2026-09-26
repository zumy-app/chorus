# Security Test Plan — Phase 6 QA

## Scope
Block/Report (6.1), Two-Factor Auth / OTP (6.2), Privacy Settings (6.3), Auth middleware, Rate limiting.

## Threat Model
| Vector | Mitigation | Test |
|---|---|---|
| Anonymous message send | WebSocket auth, JWT required | AuthMiddleware unauthenticated -> 401 |
| Block bypass | IsBlocked check on chat create/addParticipant | Blocked chat returns 403 |
| Self-block/report | ErrBlockSelf / ErrReportSelf | 400 |
| OTP brute force | otpMaxAttempts=5, 429 after | TooManyAttempts -> 429 |
| OTP flood | otpMaxPerHour=5 + UserRateLimiter 5/15m | RateLimited -> 429 |
| Phone injection | E.164 regex ^\+[1-9]\d{7,14}$ | InvalidPhone -> 400 |
| 2FA bypass | SetTwoFactor requires phone_verified; Verify2FA requires valid temp JWT with purpose=2fa | unverified -> 400, bad token -> 401 |
| 2FA token replay/expire | 5m expiry, purpose claim | expired token -> 401 |
| Privacy leak | FilterUser blanks avatar/lastActive when visibility=nobody/contacts | asserts |
| Suspended/deleted access | AuthMiddleware checks SuspendedAt/DeletedAt | 403 |
| Cost exhaustion | UserRateLimiter on OTP, translation | burst > limit -> 429 |

## Execution
`go test ./...` must pass. See `backend/internal/handlers/security_qa_test.go` and `backend/internal/services/security_qa_test.go`.
