# Chorus Architecture & Design

A comprehensive guide to understanding the Chorus messenger application's architecture, data flow, and implementation patterns.

## Table of Contents

1. [System Overview](#system-overview)
2. [High-Level Architecture](#high-level-architecture)
3. [Backend Architecture](#backend-architecture)
4. [Frontend Architecture](#frontend-architecture)
5. [Data Flow Diagrams](#data-flow-diagrams)
6. [Database Schema](#database-schema)
7. [Authentication & Security](#authentication--security)
8. [Real-Time Messaging](#real-time-messaging)
9. [Caching & Performance](#caching--performance)
10. [Deployment](#deployment)

---

## System Overview

**Chorus** is a **multilingual real-time messenger** that automatically translates messages between users' languages. It's built as a **full-stack application** with:

- **Backend**: Go (Gin) REST API + WebSocket server
- **Database**: PostgreSQL (durable state) + Redis (cache, pub/sub, sessions)
- **Frontend (Web)**: React + TypeScript (Vite)
- **Frontend (Mobile)**: Capacitor-wrapped React (Android/iOS)

### Core Use Cases

1. **User Registration & Login** → JWT-based authentication
2. **Create Direct/Group Chats** → Multi-user conversations
3. **Send & Receive Messages** → Real-time via WebSocket
4. **Automatic Translation** → Messages shown in recipient's language
5. **User Presence** → Online/offline status
6. **Message Search** → Full-text search across chats
7. **Vocabulary Learning** → Save words with spaced repetition
8. **Grammar Analysis** → Identify learning opportunities
9. **Voice Calls** → Audio/video with transcription

---

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                       Frontend Layer (React)                    │
│  ┌─────────────┐  ┌──────────────┐  ┌────────────────────────┐ │
│  │   Pages     │  │  Components  │  │  Store (Zustand)       │ │
│  │ (Login,     │→ │  (ChatArea,  │→ │ (user, chats,         │ │
│  │  Register,  │  │   ChatList,  │  │  messages, activeChat) │ │
│  │  Chat)      │  │   Bubble)    │  │                        │ │
│  └─────────────┘  └──────────────┘  └────────────────────────┘ │
│                              │                                   │
│         ┌────────────────────┼────────────────────┐              │
│         │                    │                    │              │
│  ┌──────────────┐   ┌────────────────┐   ┌───────────────┐     │
│  │ API Client   │   │ WebSocket      │   │ Components    │     │
│  │ (Axios)      │   │ Service        │   │ (MessageBubble)    │
│  │              │   │ (Real-time)    │   │               │     │
│  └──────────────┘   └────────────────┘   └───────────────┘     │
└──────────┬───────────────────┬───────────────────────────────────┘
           │                   │
        HTTP REST          WebSocket
        (Axios)            (ws://)
           │                   │
┌──────────┴───────────────────┴───────────────────────────────────┐
│                      Backend Layer (Go)                          │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │           Gin HTTP Server + WebSocket Hub               │   │
│  │                                                          │   │
│  │  ┌────────────┐  ┌──────────────┐  ┌──────────────┐    │   │
│  │  │  Handlers  │  │  Middleware  │  │  Routes     │    │   │
│  │  │ (HTTP)     │  │ (Auth,       │  │ (REST API)  │    │   │
│  │  │            │  │  CORS)       │  │             │    │   │
│  │  └────────────┘  └──────────────┘  └──────────────┘    │   │
│  │                         │                               │   │
│  │        ┌────────────────┴────────────────┐              │   │
│  │        ▼                                 ▼              │   │
│  │  ┌──────────────────────────────────────────────┐      │   │
│  │  │         Services Layer (Business Logic)      │      │   │
│  │  │ ┌─────────────────────────────────────────┐ │      │   │
│  │  │ │Auth    │Chat     │Message              │ │      │   │
│  │  │ │        │         │Translation          │ │      │   │
│  │  │ │User    │Search   │WebSocket            │ │      │   │
│  │  │ │        │         │Presence             │ │      │   │
│  │  │ │Inbox   │PubSub   │Grammar              │ │      │   │
│  │  │ │Client  │         │Vocabulary           │ │      │   │
│  │  │ │Call    │Speech   │Speech-to-Text       │ │      │   │
│  │  │ │        │         │                     │ │      │   │
│  │  │ └─────────────────────────────────────────┘ │      │   │
│  │  └──────────────────────────────────────────────┘      │   │
│  └──────────────────────────────────────────────────────────┘   │
│                              │                                   │
│        ┌─────────────────────┼──────────────────┐                │
│        ▼                     ▼                  ▼                │
│   ┌─────────────┐  ┌──────────────┐  ┌──────────────┐           │
│   │PostgreSQL   │  │   Redis      │  │  External    │           │
│   │Database     │  │   Cache +    │  │  Services    │           │
│   │             │  │   Pub/Sub    │  │ (Google API) │           │
│   └─────────────┘  └──────────────┘  └──────────────┘           │
└─────────────────────────────────────────────────────────────────┘
```

---

## Backend Architecture

### Directory Structure

```
backend/
├── cmd/
│   └── server/
│       └── main.go              ← Application entry point
├── internal/
│   ├── config/
│   │   └── config.go            ← Environment configuration loading
│   ├── database/
│   │   ├── postgres.go          ← PostgreSQL connection & migrations
│   │   ├── redis.go             ← Redis client setup
│   │   └── appwrite.go          ← Appwrite (optional) integration
│   ├── models/
│   │   └── models.go            ← Data structures (User, Chat, Message, etc.)
│   ├── services/
│   │   ├── auth.go              ← Authentication, JWT, password hashing
│   │   ├── user.go              ← User profile management
│   │   ├── chat.go              ← Chat CRUD operations
│   │   ├── message.go           ← Message storage & retrieval
│   │   ├── translation.go       ← Google Translate integration
│   │   ├── websocket.go         ← WebSocket hub & client management
│   │   ├── pubsub.go            ← Redis Pub/Sub for real-time events
│   │   ├── presence.go          ← Online/offline status tracking
│   │   ├── search.go            ← Full-text search
│   │   ├── inbox.go             ← Offline message delivery
│   │   ├── grammar.go           ← Grammar analysis & CEFR assessment
│   │   ├── vocabulary.go        ← Vocabulary learning & spaced repetition
│   │   ├── speech_to_text.go    ← Google Speech-to-Text
│   │   ├── call.go              ← Voice/video call management
│   │   ├── client.go            ← Multi-device client management
│   │   └── [other services]
│   ├── handlers/
│   │   ├── auth.go              ← HTTP handlers: register, login, refresh
│   │   ├── chat.go              ← Chat endpoints
│   │   ├── message.go           ← Message endpoints
│   │   ├── websocket.go         ← WebSocket upgrade handler
│   │   ├── presence.go          ← Presence endpoints
│   │   ├── search.go            ← Search endpoints
│   │   ├── grammar.go           ← Grammar analysis endpoints
│   │   ├── vocabulary.go        ← Vocabulary endpoints
│   │   ├── call.go              ← Call endpoints
│   │   └── [other handlers]
│   └── middleware/
│       └── auth.go              ← JWT validation middleware
├── go.mod                        ← Go module dependencies
├── go.sum
├── Dockerfile                    ← Container image definition
└── .env.example                  ← Environment variables template
```

### Layered Architecture Pattern

The backend follows a **3-layer architecture**:

```
┌─────────────────────────────────┐
│   Handlers (HTTP/REST)          │  ← Request parsing, response formatting
│   Middleware (Auth, CORS)       │
├─────────────────────────────────┤
│   Services (Business Logic)     │  ← Core application logic, validation
│   PubSub, Caching               │
├─────────────────────────────────┤
│   Database (Models, Queries)    │  ← Data persistence, retrieval
│   Cache (Redis)                 │
└─────────────────────────────────┘
```

**Benefits:**
- Clear separation of concerns
- Easy to test (mock database layer)
- Business logic independent of transport (HTTP/WebSocket)
- Scalable (services can be distributed)

### Service Responsibilities

| Service | Responsibility |
|---------|-----------------|
| **AuthService** | Password hashing (bcrypt), JWT generation/validation, refresh token management |
| **UserService** | User CRUD, profile searches, language preference management |
| **ChatService** | Chat creation (direct/group), participant management, chat settings |
| **MessageService** | Store/retrieve messages, handle delivery status, pagination |
| **TranslationService** | Detect language, translate via Google API or mock, caching |
| **WebSocketHub** | Client connection management, in-memory message broadcasting |
| **PubSubService** | Redis Pub/Sub channels, cross-server event distribution |
| **PresenceService** | Online status tracking, presence events via Redis |
| **InboxService** | Queue messages for offline clients, multi-device management |
| **SearchService** | Full-text search using PostgreSQL GIN indexes |
| **GrammarService** | CEFR level assessment, grammatical pattern identification |
| **VocabularyService** | Save vocabulary, spaced repetition scheduling |
| **SpeechToTextService** | Audio transcription via Google Cloud Speech API |
| **CallService** | Call session lifecycle, participant tracking, transcription |

---

## Frontend Architecture

### Project Structure

```
frontend/
├── src/
│   ├── pages/
│   │   ├── Landing.tsx         ← Marketing landing page
│   │   ├── Login.tsx           ← Login form
│   │   ├── Register.tsx        ← Registration form (pre-filled)
│   │   └── Chat.tsx            ← Main chat interface
│   ├── components/
│   │   ├── ChatArea.tsx        ← Message display & input
│   │   ├── ChatList.tsx        ← Sidebar with chat list
│   │   ├── MessageBubble.tsx   ← Individual message with translation
│   │   ├── NewChatModal.tsx    ← Create new chat dialog
│   │   └── [other components]
│   ├── services/
│   │   ├── api.ts             ← Axios instance with interceptors
│   │   └── websocket.ts       ← WebSocket client with reconnect logic
│   ├── store/
│   │   └── index.ts           ← Zustand global state store
│   ├── types/
│   │   └── index.ts           ← TypeScript interfaces (matches backend models)
│   ├── App.tsx                ← Root component, auth check, routing
│   ├── main.tsx               ← React entry point
│   └── index.css              ← Base styles
├── android/                    ← Capacitor Android shell
├── package.json
├── tsconfig.json
├── vite.config.ts
├── tailwind.config.js
└── capacitor.config.ts        ← Capacitor settings
```

### State Management (Zustand)

```typescript
interface AppState {
  user: User | null
  chats: Chat[]
  activeChat: Chat | null
  messages: Record<string, Message[]>  // chatId → messages
  
  // Actions
  loadChats()
  loadMessages(chatId)
  sendMessage(chatId, text)
  addMessage(message)  // Via WebSocket broadcast
}
```

**Flow:**
```
User types message
  ↓
ChatArea calls store.sendMessage()
  ↓
API POST /messages
  ↓
Backend stores & broadcasts via WebSocket
  ↓
WebSocket message received
  ↓
Store handler calls addMessage()
  ↓
UI updates (React re-renders)
```

### API Client Pattern

**File:** `src/services/api.ts`

```typescript
// Axios interceptor adds JWT token to all requests
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('accessToken')
  config.headers.Authorization = `Bearer ${token}`
  return config
})

// Auto-refresh token on 401
api.interceptors.response.use(
  (response) => response,
  async (error) => {
    if (error.response?.status === 401) {
      const newToken = await refreshToken()
      // Retry original request
      return api(originalRequest)
    }
  }
)
```

**Benefits:**
- No need to manually add headers
- Automatic token refresh
- Centralized error handling

### WebSocket Client

**File:** `src/services/websocket.ts`

```typescript
class WebSocketService {
  connect(token: string)
  send(message: WebSocketMessage)
  onMessage(handler: (msg) => void)
  sendTyping(chatId, isTyping)
  disconnect()
}

// Auto-reconnect with exponential backoff
// Attempt 1: 1s, Attempt 2: 2s, Attempt 3: 3s, etc. (up to 5 attempts)
```

---

## Data Flow Diagrams

### 1. Registration & Login Flow

```
┌─────────────────────────────────────────────────────┐
│                   Frontend (React)                  │
│                                                     │
│  1. User fills registration form                   │
│     ├─ Email: alice@example.com                    │
│     ├─ Password: SecurePass123                     │
│     ├─ Display Name: Alice                         │
│     └─ Languages: en → es                          │
│                                                     │
│  2. Click "Register"                               │
│     └─ Call authAPI.register(formData)             │
│                                                     │
└──────────────┬──────────────────────────────────────┘
               │ POST /api/v1/auth/register
               ▼
┌─────────────────────────────────────────────────────┐
│                 Backend (Go)                        │
│                                                     │
│  3. AuthHandler.Register()                         │
│     ├─ Validate input (email format, password min) │
│     ├─ authService.Register()                      │
│     │  ├─ Hash password (bcrypt)                   │
│     │  ├─ INSERT INTO users                        │
│     │  └─ Return User object                       │
│     ├─ Generate accessToken (JWT, 24h)            │
│     ├─ Generate refreshToken (UUID, store in DB)  │
│     └─ Return 201 with tokens                      │
│                                                     │
└──────────────┬──────────────────────────────────────┘
               │ {user, tokens}
               ▼
┌─────────────────────────────────────────────────────┐
│                   Frontend (React)                  │
│                                                     │
│  4. Receive response                               │
│     ├─ localStorage.setItem('accessToken', token)  │
│     ├─ localStorage.setItem('refreshToken', token) │
│     ├─ setUser(user)                               │
│     ├─ setIsAuthenticated(true)                    │
│     ├─ wsService.connect(token)                    │
│     └─ navigate('/chat')                           │
│                                                     │
│  5. Chat page loads                                │
│     ├─ loadChats()                                 │
│     └─ Set activeChat = null                       │
│                                                     │
└─────────────────────────────────────────────────────┘
```

### 2. Message Send & Receive Flow

```
┌─────────────────────────────────────────────────────┐
│              User A (Alice, Browser)                │
│                                                     │
│  1. Types in ChatArea                              │
│     └─ Input: "Hola, ¿cómo estás?"                │
│        (Spanish, but Alice's native is English)    │
│                                                     │
│  2. Clicks "Send"                                  │
│     └─ Call sendMessage(chatId, text)              │
│                                                     │
└──────────────┬──────────────────────────────────────┘
               │ POST /chats/{chatId}/messages
               │ Authorization: Bearer <accessToken>
               ▼
┌─────────────────────────────────────────────────────┐
│         Backend API (Port 8080)                     │
│                                                     │
│  3. MessageHandler.SendMessage()                   │
│     ├─ Authenticate (JWT middleware)               │
│     ├─ Extract userID from token                   │
│     ├─ Create Message object                       │
│     ├─ Store in PostgreSQL                         │
│     └─ Return Message object                       │
│                                                     │
│  4. MessageService.Create()                        │
│     ├─ INSERT INTO messages                        │
│     ├─ Set original_language = 'es'               │
│     ├─ Set delivery_status = 'sent'               │
│     └─ Return full message with ID                │
│                                                     │
└──────────────┬──────────────────────────────────────┘
               │ {id, text, original_language, ...}
               ▼
┌─────────────────────────────────────────────────────┐
│                  Backend (Go)                       │
│             (Handler response continues)            │
│                                                     │
│  5. After storing message                          │
│     ├─ Get chat participants                       │
│     ├─ Fetch each participant's target languages  │
│     └─ For each language:                          │
│        ├─ translationService.Translate()           │
│        │  ├─ Check Redis cache                     │
│        │  ├─ Call Google Translate API             │
│        │  ├─ Store result in cache (24h TTL)      │
│        │  └─ Return translated text                │
│        └─ Add to message.translations map          │
│                                                     │
│  6. Broadcast via WebSocket                        │
│     ├─ Send to all connected clients in chat       │
│     ├─ Message format:                             │
│     │  {                                            │
│     │    type: "new_message",                      │
│     │    data: {                                    │
│     │      id: "...",                              │
│     │      text: "Hola, ¿cómo estás?",            │
│     │      originalLanguage: "es",                 │
│     │      translations: {                         │
│     │        en: "Hello, how are you?"              │
│     │      },                                       │
│     │      senderId: "...",                        │
│     │      timestamp: "..."                        │
│     │    }                                          │
│     │  }                                            │
│     └─ Send to Redis Pub/Sub (for other servers)   │
│                                                     │
└──────────────┬──────────────────────────────────────┘
      ┌────────┴──────────┐
      │ WebSocket         │ Pub/Sub
      │ Broadcast         │ (if multiple servers)
      ▼                   ▼
┌─────────────────────────────────────────────────────┐
│   User B (Bob, Browser) - Spanish native, en target │
│                                                     │
│  7. WebSocket message received                     │
│     └─ Browser alert: "new_message"                │
│                                                     │
│  8. Store handler processes                        │
│     ├─ Call store.addMessage(msg)                  │
│     └─ Update state.messages[chatId]               │
│                                                     │
│  9. MessageBubble component renders                │
│     ├─ Show Spanish text: "Hola, ¿cómo estás?"    │
│     ├─ Check translations[bob.targetLanguages]    │
│     ├─ Translation exists: "Hello, how are you?"  │
│     ├─ Show in UI:                                 │
│     │  ┌──────────────────────────┐               │
│     │  │ Alice:                   │               │
│     │  │ Hola, ¿cómo estás?      │               │
│     │  │                          │               │
│     │  │ Translation (en):        │               │
│     │  │ Hello, how are you?      │               │
│     │  │ 2 minutes ago            │               │
│     │  └──────────────────────────┘               │
│     └─ MessageBubble component re-renders         │
│                                                     │
└─────────────────────────────────────────────────────┘
```

### 3. Token Refresh Flow

```
Client makes API call
  │
  ├─ Add Authorization: Bearer <accessToken>
  │
  ▼
API returns 401 (token expired)
  │
  ├─ Axios interceptor catches 401
  │
  ├─ Call POST /auth/refresh
  │  └─ Send refreshToken
  │
  ▼
Backend validates refreshToken
  │
  ├─ Query refresh_tokens table
  │ ├─ Check token exists
  │ └─ Check expiry (30 days)
  │
  ├─ Generate new accessToken (24h)
  │
  ├─ Return 200 with new accessToken
  │
  ▼
Client updates localStorage.accessToken
  │
  ├─ Retry original request with new token
  │
  ▼
API returns 200 (success)
  │
  └─ Application continues as normal

If refresh fails:
  ├─ Clear tokens from localStorage
  ├─ Redirect to /login
  └─ User must log in again
```

---

## Database Schema

### Core Tables (Phase 1)

```sql
users
├─ id (UUID, PK)
├─ username (VARCHAR UNIQUE)
├─ email (VARCHAR UNIQUE)
├─ password_hash (VARCHAR)
├─ display_name (VARCHAR)
├─ native_language (VARCHAR)
├─ target_languages (TEXT[])
├─ created_at (TIMESTAMP)
└─ last_active_at (TIMESTAMP)

chats
├─ id (UUID, PK)
├─ type (VARCHAR: 'direct' or 'group')
├─ name (VARCHAR, optional)
├─ created_by (UUID, FK users)
├─ settings (JSONB)
└─ created_at (TIMESTAMP)

chat_participants
├─ id (UUID, PK)
├─ chat_id (UUID, FK chats)
├─ user_id (UUID, FK users)
├─ role (VARCHAR: 'member' or 'admin')
├─ joined_at (TIMESTAMP)
└─ last_read_message_id (UUID, FK messages)

messages
├─ id (UUID, PK)
├─ chat_id (UUID, FK chats)
├─ sender_id (UUID, FK users)
├─ text (TEXT)
├─ original_language (VARCHAR)
├─ translations (JSONB) ← Stores {en: "...", es: "...", ...}
├─ delivery_status (VARCHAR)
├─ reply_to_id (UUID, optional)
└─ created_at (TIMESTAMP)

refresh_tokens
├─ id (UUID, PK)
├─ user_id (UUID, FK users)
├─ token (VARCHAR UNIQUE)
├─ expires_at (TIMESTAMP)
└─ created_at (TIMESTAMP)
```

### Phase 2 Tables (Multi-Device, Offline Support)

```sql
clients
├─ id (UUID, PK)
├─ user_id (UUID, FK users)
├─ device_type (VARCHAR: mobile, web, desktop)
├─ device_info (JSONB)
├─ connection_status (VARCHAR)
├─ last_active (TIMESTAMP)
└─ created_at (TIMESTAMP)

inbox
├─ client_id (UUID, FK clients)
├─ message_id (UUID, FK messages)
├─ chat_id (UUID, FK chats)
├─ delivery_attempts (INTEGER)
├─ created_at (TIMESTAMP)
└─ ttl (TIMESTAMP, 30-day expiry)

user_settings
├─ user_id (UUID, PK, FK users)
├─ grammar_enabled (BOOLEAN)
├─ vocabulary_enabled (BOOLEAN)
├─ difficulty_level (VARCHAR)
├─ transcript_recording (BOOLEAN)
├─ message_retention_days (INTEGER)
└─ updated_at (TIMESTAMP)

presence_log
├─ user_id (UUID, FK users)
├─ status (VARCHAR)
├─ device_type (VARCHAR)
└─ timestamp (TIMESTAMP)
```

### Phase 3 Tables (Learning, Calls)

```sql
vocabulary
├─ id (UUID, PK)
├─ user_id (UUID, FK users)
├─ term (VARCHAR)
├─ language (VARCHAR)
├─ translation (VARCHAR)
├─ definition (TEXT)
├─ context_message_id (UUID, FK messages)
├─ context_sentence (TEXT)
├─ context_chat_id (UUID, FK chats)
├─ review_count (INTEGER)
├─ correct_count (INTEGER)
├─ next_review (TIMESTAMP)
├─ interval_days (INTEGER)
├─ created_at (TIMESTAMP)
└─ UNIQUE(user_id, term, language)

call_sessions
├─ id (UUID, PK)
├─ chat_id (UUID, FK chats)
├─ type (VARCHAR: audio, video)
├─ status (VARCHAR: active, ended)
├─ started_at (TIMESTAMP)
└─ ended_at (TIMESTAMP, optional)

call_participants
├─ call_id (UUID, FK call_sessions)
├─ user_id (UUID, FK users)
├─ joined_at (TIMESTAMP)
└─ left_at (TIMESTAMP, optional)

call_transcripts
├─ id (UUID, PK)
├─ call_id (UUID, FK call_sessions)
├─ segments (JSONB) ← [{speaker, text, translation, ...}, ...]
└─ created_at (TIMESTAMP)
```

### Indexes for Performance

```sql
-- Message queries (most common)
CREATE INDEX idx_messages_chat_id ON messages(chat_id, created_at DESC)
CREATE INDEX idx_messages_text_search ON messages USING gin(to_tsvector('english', text))

-- User lookup
CREATE INDEX idx_users_username ON users(username)
CREATE INDEX idx_users_email ON users(email)

-- Chat participant queries
CREATE INDEX idx_chat_participants_user_id ON chat_participants(user_id)
CREATE INDEX idx_chat_participants_chat_id ON chat_participants(chat_id)

-- Vocabulary spaced repetition
CREATE INDEX idx_vocabulary_user_due ON vocabulary(user_id, next_review)

-- Presence queries
CREATE INDEX idx_clients_user_status ON clients(user_id, connection_status)
```

---

## Authentication & Security

### Password Security

```go
// Registration
plainPassword := "SecurePass123"
hash := bcrypt.GenerateFromPassword(plainPassword, 14)
// Stored in DB: $2a$14$xyz...

// Login validation
bcrypt.CompareHashAndPassword(storedHash, submittedPassword)
// Returns nil if match, error if no match
```

**Why bcrypt?**
- Salted hashing (salt embedded in hash)
- Slow by design (resistant to brute force)
- Cost factor 14 (increases with hardware improvements)

### JWT Token Structure

```
eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.
eyJ1c2VySWQiOiI1NTBlODQwMC1lNjRiLTQ2ZDctOWY2ZC1kYzEyMDZiODk2ZjAiLCJleHAiOjE2OTQ0MjUyMzZ9.
aBcDeFgHiJkLmNoPqRsTuVwXyZ1a2b3c4d5e6f7g8

Header:  {"alg":"HS256","typ":"JWT"}
Payload: {"userId":"...", "exp":1694425236, "iat":1694338836}
Signature: HMAC-SHA256(header.payload, JWT_SECRET)
```

**Validation:**
1. Parse JWT (decode header + payload)
2. Verify signature using JWT_SECRET
3. Check expiry (current time < exp)
4. Extract and use userID

### Token Lifecycle

| Token | Lifespan | Storage | Purpose |
|-------|----------|---------|---------|
| **Access** | 24 hours | localStorage | Every API request header |
| **Refresh** | 30 days | localStorage + DB | Exchange for new access token |

**Why two tokens?**
- Short-lived access token → less damage if stolen
- Long-lived refresh token → user doesn't need to re-login often
- Refresh token in DB → can be revoked by server if needed

---

## Real-Time Messaging

### WebSocket Connection

```typescript
// Frontend
wsService.connect(accessToken)

// Backend
upgrader.Upgrade(writer, request, nil)
// WebSocket connection established on /ws endpoint
```

### Message Broadcasting

**Single Server (In-Memory):**
```
Client A sends message
  ↓
Handler stores in PostgreSQL
  ↓
WebSocketHub.Broadcast() sends to all connected clients
  ↓
All clients in that chat receive via WebSocket
```

**Multiple Servers (Scaled):**
```
Server 1: Client A sends message
  ↓
Store in PostgreSQL (shared)
  ↓
Publish to Redis Pub/Sub: "chat:{chatId}"
  ↓
All servers receive event
  ↓
Each server broadcasts to its connected clients
```

### Message Types

```typescript
// From client
{ type: "message", data: { text: "..." } }
{ type: "typing_start", data: { chatId: "..." } }
{ type: "typing_stop", data: { chatId: "..." } }

// From server
{ type: "new_message", data: { id, text, translations, ... } }
{ type: "user_typing", data: { userId, chatId } }
{ type: "presence_update", data: { userId, status } }
```

---

## Caching & Performance

### Redis Cache Strategy

| Data | Key Pattern | TTL | Purpose |
|------|-------------|-----|---------|
| Translation results | `translation:{lang}:{text}` | 24h | Avoid re-translating same text |
| Grammar analysis | `grammar:{lang}:{hash}` | 24h | Avoid re-analyzing same text |
| Vocabulary lists | `vocab:{userId}:all` | 1h | Pagination cache |
| Presence status | `presence:{userId}` | 5m | Online status with TTL |
| User's clients | `user:{userId}:clients` | Runtime | Multi-device tracking |

### Database Query Optimization

**Pagination (Common):**
```sql
SELECT * FROM messages 
WHERE chat_id = $1 
ORDER BY created_at DESC 
LIMIT 50
-- Returns: 50 newest messages
```

**Full-Text Search:**
```sql
SELECT * FROM messages 
WHERE to_tsvector('english', text) @@ plainto_tsquery('english', 'search term')
-- Uses GIN index for fast retrieval
```

**Joins for Related Data:**
```sql
SELECT m.*, u.display_name, u.avatar_url
FROM messages m
JOIN users u ON m.sender_id = u.id
WHERE m.chat_id = $1
-- One query instead of N+1
```

---

## Deployment

### Docker Compose

```yaml
services:
  postgres:        # Port 5432 (internal)
  redis:           # Port 6379 (internal)
  backend:         # Port 8080 (external)
  frontend:        # Port 3000 (external)
```

**Scaling:**
- Horizontal: Run multiple backend instances (load balanced)
- All connect to shared PostgreSQL + Redis
- Pub/Sub automatically syncs across instances

### Environment Variables

**Backend:**
```
ENVIRONMENT=production
DATABASE_URL=postgres://...
REDIS_URL=...
JWT_SECRET=<strong-random-key>
GOOGLE_TRANSLATE_API_KEY=...
PORT=8080
```

**Frontend:**
```
VITE_API_URL=/api/v1  # Or http://api.example.com/api/v1
```

---

## Performance Considerations

### Bottlenecks & Solutions

| Issue | Solution |
|-------|----------|
| Message retrieval slow (large chats) | Pagination + index on (chat_id, created_at) |
| Translation latency | Redis cache, async processing |
| WebSocket broadcast to many clients | Pub/Sub for multi-server scaling |
| JWT validation on every request | In-memory JWT cache (could add) |
| Database connection pooling | Set MaxOpenConns=25, MaxIdleConns=5 |

### Monitoring

**Metrics to track:**
- API response times (by endpoint)
- Database query times
- WebSocket connection count
- Redis memory usage
- Message delivery latency

---

## Key Design Patterns

1. **Layered Architecture** — Clean separation: handlers → services → database
2. **Dependency Injection** — Services created with dependencies in main.go
3. **Repository Pattern** — Services encapsulate database queries
4. **Pub/Sub for Scalability** — Redis for cross-server event distribution
5. **Graceful Degradation** — Optional APIs (Google) with fallbacks
6. **Token Refresh Pattern** — Automatic token renewal on client
7. **Spaced Repetition** — Vocabulary review scheduling based on algorithm

---

## Further Reading

- [RUN_GUIDE.md](RUN_GUIDE.md) — How to start and test services
- [API_ENDPOINTS.md](API_ENDPOINTS.md) — Complete API reference
- Backend: `backend/internal/` — Source code
- Frontend: `frontend/src/` — React components & services
- Database: `backend/internal/database/postgres.go` — Schema definitions

