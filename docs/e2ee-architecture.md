# E2EE Chat Messages Architecture

Chorus chat messages use a Signal-inspired phase-1 design: clients create and keep private key material locally, the server stores public device keys and encrypted envelopes, and message content crosses the API and WebSocket boundary only as ciphertext.

## Goals

- Keep message plaintext on user devices.
- Let the backend route, persist, and broadcast encrypted messages without decrypting them.
- Preserve delivery status, reply links, timestamps, sender metadata, and chat participation checks.
- Make translation, search, grammar, and vocabulary features explicit client-side or opt-in flows because they need plaintext.

## Protocol Choice

Signal Protocol and Double Ratchet are strong fits for direct messages and asynchronous delivery, but a complete implementation needs identity verification, one-time pre-key replenishment, skipped-message key handling, and multi-device sessions. MLS is a better long-term group protocol, especially for large dynamic groups, but it is a larger infrastructure investment than this codebase needs for a first encrypted chat phase.

The phase-1 architecture uses per-chat symmetric keys:

1. Each browser device creates an extractable ECDH P-256 key pair for wrapping chat keys and a non-extractable identity signing key placeholder for future verification.
2. The server stores only public device key material in `user_devices`.
3. The creator generates a random AES-GCM chat key and sends one encrypted key envelope per participant device to `chat_recipient_keys`.
4. Each message is encrypted locally with the chat key before `messageAPI.sendMessage`.
5. Clients unwrap their own chat key envelope, decrypt fetched or WebSocket-delivered ciphertext locally, and store decrypted text only in frontend state.

This is intentionally smaller than full Signal or MLS. It gives strict server-side ciphertext storage now while leaving room to add identity verification, one-time pre-keys, and MLS later.

## Server Data

The backend may store:

- User, chat, participant, sender, timestamp, delivery, and reply metadata.
- Device IDs, device labels, public identity keys, public pre-keys, signatures, and key versions.
- Encrypted per-chat key envelopes per user/device.
- Message `ciphertext`, `nonce`, `algorithm`, `encryption_version`, and `sender_device_id`.

The backend must not store plaintext message text or plaintext translations for encrypted messages. Legacy plaintext columns remain nullable for migration and older rows, but encrypted sends should populate ciphertext fields only.

## Threat Model

Protected against:

- Database snapshots exposing message content.
- Backend logs, queues, caches, or WebSocket payloads reading message plaintext.
- Server-side translation/search workers seeing encrypted chat content by default.

Not protected against:

- Malicious or compromised JavaScript served to the browser.
- XSS in the Chorus origin.
- Compromised end-user devices.
- Metadata analysis. The server still sees users, chat IDs, participant graph, timestamps, message sizes, sender IDs, and delivery events.

Before marketing this as strong E2EE, Chorus should harden CSP, remove raw HTML sinks, add key verification UX, and define account recovery behavior.

## Translation And Learning Features

Strict E2EE means the backend cannot automatically translate, index, grammar-check, or extract vocabulary from encrypted message text. The app handles those features as follows:

- Decrypt messages in the client before display.
- Run local/client-side search over loaded decrypted messages.
- Use decrypted text for grammar and vocabulary buttons.
- If a server-backed translation or AI learning request sends plaintext to Chorus, it must be an explicit user action and should be presented as breaking strict E2EE for that request.
- Server-side message search rejects encrypted chat search rather than searching ciphertext.

## Key Loss And Rekeying

Private chat keys are device-local. If IndexedDB/local browser storage is wiped, the UI must show that the message cannot be decrypted on this device. It must not silently ask the server for plaintext fallback.

When participants are added or removed, clients need a re-key workflow before sending new encrypted messages. The current phase stores the envelope format and surfaces rekey-required states; robust multi-device recovery and MLS membership commits are follow-up work.
