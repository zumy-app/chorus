
## Phase 1 Requirements: Stateless Chat with Redis Pub/Sub

- Backend must be stateless; all state externalized
- Use Redis Pub/Sub for chat and presence event delivery
- On message send: publish to Redis channel
- On message receive: subscribe and deliver via WebSocket
- Store messages in PostgreSQL for history
- Redis is not a source of truth
- Presence/typing via Redis Pub/Sub
