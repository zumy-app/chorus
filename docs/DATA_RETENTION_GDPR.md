# Data Retention & GDPR — NFR-23 (Issue #26)

## Policy (minimal enforcement)

| Data category | Default window | Configurable | Notes |
|---|---|---|---|
| Messages + media | 365 days | Yes — `user_settings.message_retention_days` (7/30/90/180/365) | Hard-deleted by `RetentionService.PurgeExpiredMessages` after sender's window expires. `deleted_at` soft-delete still respects retention. |
| Inbox (offline queue) | 30 days (TTL) | No | `inbox.ttl` is set on enqueue; sweeper purges expired rows. |
| Translation jobs | 90 days | No | `translation_jobs.completed_at` >90d purged. |
| Call transcripts | 90 days or `transcript_recording=false` disables capture | Per-user `transcript_recording` | Auto-purged after 90d. |
| Email outbox (sent) | 90 days | No | Pending/failed retained until sent or manual retry limit. |
| User account (PII) | Until deletion request | — | `users.deleted_at` soft-delete; GDPR erasure anonymizes. |

Phoenix traces/evals: retained per Phoenix config (separate infra, PII redacted before export).

## GDPR rights (minimal)

- **Right to access (Art. 15):** `GET /api/v1/users/me/export` returns user row, settings, chats, messages (1000), vocabulary, and current retention policy.
- **Right to erasure (Art. 17):** `DELETE /api/v1/users/me` anonymizes PII (`email→deleted_<id>@deleted.local`, `username→deleted_<id>`, clears phone/avatar), soft-deletes account, wipes tokens/clients/settings/vocabulary, and redacts message text to `[deleted]`. Chat membership is removed; other participants' history is preserved minus PII.
- **Rectification / restriction:** `PUT /api/v1/users/me` and `PUT /api/v1/users/me/settings` (including `messageRetentionDays` and `transcriptRecording`).
- **Retention enforcement:** `RetentionService.StartScheduler(24h)` runs daily; `POST /api/v1/admin/retention/purge` (moderator) triggers on demand. Retention windows are documented here and surfaced via `GET /api/v1/privacy/retention-policy`.

All endpoints are authenticated; admin purge requires moderator role. No raw contact data is stored (hashed scan only — FR-22/23).
