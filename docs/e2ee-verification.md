# E2EE Verification Checklist

Use this checklist after running the automated tests to verify that Chorus stores and broadcasts ciphertext only for encrypted chat messages.

## Automated Checks

- Backend: `cd backend && go test ./...`
- Frontend crypto: `cd frontend && npm run test -- src/services/crypto/index.test.ts`
- Frontend build: `cd frontend && npm run build`

The focused tests cover:

- Device public key registration.
- Encrypted per-recipient chat-key envelope storage.
- Encrypted message insertion without plaintext.
- Client chat-key wrapping/unwrapping.
- Client message encryption/decryption.
- Missing local chat key producing a cannot-decrypt state rather than plaintext fallback.

## Manual Flow

1. Register or log in as two users in separate browsers.
2. Create a direct chat and send a message.
3. Inspect the `messages` table:
   - `text` is `NULL` for the new message.
   - `ciphertext`, `nonce`, `algorithm`, `encryption_version`, and `sender_device_id` are populated.
   - `translations` is `{}` and no plaintext translation row is written.
4. Inspect WebSocket `new_message` payloads:
   - The payload contains ciphertext fields.
   - The payload does not contain the original plaintext.
5. Inspect Redis/translation queues if enabled:
   - No encrypted chat plaintext is queued for translation.
6. Confirm each client displays decrypted text locally.
7. Clear one browser's site data and reload the chat:
   - Messages show a cannot-decrypt state on that device.
   - The server never returns plaintext fallback.
8. Try message search inside the chat:
   - Loaded decrypted messages are searched locally.
   - Unloaded encrypted history is not searched server-side.
