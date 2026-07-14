# Phase 1 Implementation Plan: Stateless Chat with Redis Pub/Sub

## Objective
Refactor the backend to use a stateless architecture and Redis Pub/Sub for message ingestion and delivery, following the WhatsApp-inspired "great solutions" approach.

## Steps

1. **Refactor Backend to Stateless**
   - Remove any session or user state from in-memory server variables.
   - Store all state in Redis, PostgreSQL, or external services.

2. **Integrate Redis Pub/Sub**
   - Set up Redis as a central pub/sub broker for chat messages and presence events.
   - On message send: publish to a Redis channel (e.g., `chat:{chatId}`).
   - On message receive: backend subscribes to relevant channels and delivers via WebSocket.

3. **Update Message Flow**
   - Client sends message → REST/WebSocket API → Backend publishes to Redis channel.
   - All backend instances subscribe to channels and push messages to connected clients.

4. **Persistence**
   - Store messages in PostgreSQL for history and retrieval.
   - Use Redis only for real-time delivery, not as a source of truth.

5. **Presence & Typing Indicators**
   - Use Redis Pub/Sub for presence/typing events.

6. **Testing & Validation**
   - Unit and integration tests for pub/sub flow.
   - Simulate multiple backend instances to verify statelessness and scaling.

## Deliverables
- Stateless backend code
- Redis Pub/Sub integration for chat and presence
- Updated documentation
- Tests for new architecture

---

## Redis Pub/Sub vs Kafka Pub/Sub for Chat

- **Redis Pub/Sub** is simple, fast, and easy to integrate for real-time, transient messaging (like chat, presence, typing). It is ideal for low-latency, in-memory pub/sub where message durability is not required.
- **Kafka Pub/Sub** is better for high-throughput, persistent, and replayable messaging. It is more complex and adds operational overhead, but is ideal for analytics, audit logs, or guaranteed delivery.
- **For chat applications:** Redis Pub/Sub is usually better for real-time delivery, while Kafka is better for analytics or persistent event streams. For Phase 1, Redis Pub/Sub is the right choice.
