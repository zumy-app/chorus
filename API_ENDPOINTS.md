# Chorus API Reference

Complete documentation of all REST API endpoints and WebSocket messages.

## Base URL

```
http://localhost:8080/api/v1
```

## Authentication

All protected endpoints require an `Authorization` header:

```
Authorization: Bearer <accessToken>
```

## Content-Type

All requests with body must include:

```
Content-Type: application/json
```

---

## Public Endpoints (No Auth Required)

### Authentication

#### Register User

```http
POST /auth/register
Content-Type: application/json

{
  "username": "alice",           // Optional, defaults to email
  "email": "alice@example.com",  // Required, must be unique
  "password": "Password123!",    // Required, min 8 chars
  "displayName": "Alice",        // Required
  "nativeLanguage": "en",        // Required (language code)
  "targetLanguages": ["es", "fr"] // Optional array of languages
}
```

**Response (201 Created):**
```json
{
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "username": "alice",
    "email": "alice@example.com",
    "displayName": "Alice",
    "nativeLanguage": "en",
    "targetLanguages": ["es", "fr"],
    "createdAt": "2026-06-28T10:00:00Z",
    "lastActiveAt": "2026-06-28T10:00:00Z"
  },
  "tokens": {
    "accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refreshToken": "a5f8c123-e456-7890-abcd-1234567890",
    "expiresIn": 86400
  }
}
```

**Errors:**
- `400 Bad Request` — Invalid email format, password too short, missing required fields
- `409 Conflict` — Email already registered

---

#### Login

```http
POST /auth/login
Content-Type: application/json

{
  "username": "alice",           // Can be username or email
  "password": "Password123!"
}
```

**Response (200 OK):**
```json
{
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "username": "alice",
    "email": "alice@example.com",
    "displayName": "Alice",
    "nativeLanguage": "en",
    "targetLanguages": ["es", "fr"],
    "createdAt": "2026-06-28T10:00:00Z",
    "lastActiveAt": "2026-06-28T10:00:00Z"
  },
  "tokens": {
    "accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refreshToken": "a5f8c123-e456-7890-abcd-1234567890",
    "expiresIn": 86400
  }
}
```

**Errors:**
- `400 Bad Request` — Invalid request format
- `401 Unauthorized` — Invalid username or password

---

#### Refresh Token

```http
POST /auth/refresh
Content-Type: application/json

{
  "refreshToken": "a5f8c123-e456-7890-abcd-1234567890"
}
```

**Response (200 OK):**
```json
{
  "accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expiresIn": 86400
}
```

**Errors:**
- `401 Unauthorized` — Invalid or expired refresh token

---

#### Health Check

```http
GET /health
```

**Response (200 OK):**
```json
{
  "status": "healthy",
  "version": "2.0.0"
}
```

---

## Protected Endpoints (Auth Required)

All requests must include: `Authorization: Bearer <accessToken>`

---

### User Management

#### Get Current User

```http
GET /users/me
```

**Response (200 OK):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "username": "alice",
  "email": "alice@example.com",
  "displayName": "Alice",
  "nativeLanguage": "en",
  "targetLanguages": ["es", "fr"],
  "createdAt": "2026-06-28T10:00:00Z",
  "lastActiveAt": "2026-06-28T10:00:00Z"
}
```

---

#### Update Current User

```http
PUT /users/me
Content-Type: application/json

{
  "displayName": "Alice Smith",           // Optional
  "targetLanguages": ["es", "fr", "de"]  // Optional
}
```

**Response (200 OK):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "username": "alice",
  "email": "alice@example.com",
  "displayName": "Alice Smith",
  "nativeLanguage": "en",
  "targetLanguages": ["es", "fr", "de"],
  "createdAt": "2026-06-28T10:00:00Z",
  "lastActiveAt": "2026-06-28T10:00:00Z"
}
```

---

#### Search Users

```http
GET /users/search?q=alice
```

**Query Parameters:**
- `q` (required) — Search term (username or display name)

**Response (200 OK):**
```json
{
  "users": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "username": "alice",
      "displayName": "Alice",
      "nativeLanguage": "en",
      "targetLanguages": ["es"]
    },
    {
      "id": "550e8400-e29b-41d4-a716-446655440001",
      "username": "alice.smith",
      "displayName": "Alice Smith",
      "nativeLanguage": "en",
      "targetLanguages": ["fr"]
    }
  ]
}
```

---

### Chat Management

#### Get User's Chats

```http
GET /chats
```

**Response (200 OK):**
```json
{
  "chats": [
    {
      "id": "660f9400-f29b-41d4-a716-446655440001",
      "type": "direct",
      "participants": [
        {
          "chatId": "660f9400-f29b-41d4-a716-446655440001",
          "userId": "550e8400-e29b-41d4-a716-446655440000",
          "role": "member",
          "joinedAt": "2026-06-28T10:00:00Z",
          "user": { "id": "...", "displayName": "Alice", ... }
        },
        {
          "chatId": "660f9400-f29b-41d4-a716-446655440001",
          "userId": "550e8400-e29b-41d4-a716-446655440001",
          "role": "member",
          "joinedAt": "2026-06-28T10:00:00Z",
          "user": { "id": "...", "displayName": "Bob", ... }
        }
      ],
      "createdBy": "550e8400-e29b-41d4-a716-446655440000",
      "settings": { "translationEnabled": true },
      "createdAt": "2026-06-28T10:00:00Z",
      "lastMessage": {
        "id": "770f9500-g29b-41d4-a716-446655440002",
        "text": "How are you?",
        "senderId": "550e8400-e29b-41d4-a716-446655440001",
        "timestamp": "2026-06-28T10:15:00Z"
      },
      "unreadCount": 0
    }
  ]
}
```

---

#### Create Chat

```http
POST /chats
Content-Type: application/json

{
  "type": "direct",                              // 'direct' or 'group'
  "participants": [
    "550e8400-e29b-41d4-a716-446655440001"      // User IDs or emails
  ],
  "name": "Travel Plans"                         // Required for group, optional for direct
}
```

**Response (201 Created):**
```json
{
  "id": "660f9400-f29b-41d4-a716-446655440001",
  "type": "direct",
  "participants": [...],
  "createdBy": "550e8400-e29b-41d4-a716-446655440000",
  "settings": { "translationEnabled": true },
  "createdAt": "2026-06-28T10:00:00Z"
}
```

**Errors:**
- `400 Bad Request` — Invalid type, missing participants
- `409 Conflict` — Direct chat already exists (returns existing chat)

---

#### Get Chat Details

```http
GET /chats/{chatId}
```

**Response (200 OK):**
```json
{
  "id": "660f9400-f29b-41d4-a716-446655440001",
  "type": "direct",
  "participants": [...],
  "createdBy": "550e8400-e29b-41d4-a716-446655440000",
  "settings": { "translationEnabled": true },
  "createdAt": "2026-06-28T10:00:00Z"
}
```

---

#### Update Chat

```http
PUT /chats/{chatId}
Content-Type: application/json

{
  "name": "Updated Group Name",
  "settings": {
    "translationEnabled": false
  }
}
```

**Response (200 OK):**
```json
{
  "id": "660f9400-f29b-41d4-a716-446655440001",
  "type": "group",
  "name": "Updated Group Name",
  "settings": { "translationEnabled": false },
  ...
}
```

---

#### Add Participant to Chat

```http
POST /chats/{chatId}/participants
Content-Type: application/json

{
  "userId": "550e8400-e29b-41d4-a716-446655440002"
}
```

**Response (200 OK):**
```json
{
  "status": "success"
}
```

---

#### Remove Participant from Chat

```http
DELETE /chats/{chatId}/participants/{userId}
```

**Response (200 OK):**
```json
{
  "status": "success"
}
```

---

#### Leave Chat

```http
DELETE /chats/{chatId}/leave
```

**Response (200 OK):**
```json
{
  "status": "success"
}
```

---

### Messaging

#### Get Messages

```http
GET /chats/{chatId}/messages?limit=50&before=<messageId>
```

**Query Parameters:**
- `limit` (optional) — Number of messages to return (default: 50, max: 100)
- `before` (optional) — Return messages before this message ID (for pagination)

**Response (200 OK):**
```json
{
  "messages": [
    {
      "id": "770f9500-g29b-41d4-a716-446655440002",
      "chatId": "660f9400-f29b-41d4-a716-446655440001",
      "senderId": "550e8400-e29b-41d4-a716-446655440001",
      "text": "How are you?",
      "originalLanguage": "en",
      "translations": {
        "es": "¿Cómo estás?",
        "fr": "Comment allez-vous?"
      },
      "deliveryStatus": "delivered",
      "timestamp": "2026-06-28T10:15:00Z",
      "sender": {
        "id": "550e8400-e29b-41d4-a716-446655440001",
        "displayName": "Bob",
        "nativeLanguage": "es"
      }
    }
  ]
}
```

---

#### Send Message

```http
POST /chats/{chatId}/messages
Content-Type: application/json

{
  "text": "Hello, how are you?",
  "replyToId": "770f9500-g29b-41d4-a716-446655440001"  // Optional
}
```

**Response (201 Created):**
```json
{
  "id": "770f9500-g29b-41d4-a716-446655440003",
  "chatId": "660f9400-f29b-41d4-a716-446655440001",
  "senderId": "550e8400-e29b-41d4-a716-446655440000",
  "text": "Hello, how are you?",
  "originalLanguage": "en",
  "translations": {
    "es": "Hola, ¿cómo estás?",
    "fr": "Bonjour, comment allez-vous?"
  },
  "deliveryStatus": "sent",
  "timestamp": "2026-06-28T10:20:00Z"
}
```

**Errors:**
- `400 Bad Request` — Text is empty or too long (max 10000 chars)
- `404 Not Found` — Chat not found

---

#### Mark Messages as Read

```http
PUT /chats/{chatId}/read
Content-Type: application/json

{
  "messageId": "770f9500-g29b-41d4-a716-446655440003"
}
```

**Response (200 OK):**
```json
{
  "status": "success"
}
```

---

### Search

#### Search Messages

```http
GET /messages/search?q=hello&chatId={chatId}
```

**Query Parameters:**
- `q` (required) — Search term
- `chatId` (optional) — Filter to specific chat

**Response (200 OK):**
```json
{
  "messages": [
    {
      "id": "770f9500-g29b-41d4-a716-446655440003",
      "text": "Hello, how are you?",
      "chatId": "660f9400-f29b-41d4-a716-446655440001",
      "senderId": "550e8400-e29b-41d4-a716-446655440000",
      "timestamp": "2026-06-28T10:20:00Z"
    }
  ]
}
```

---

#### Search Chats

```http
GET /chats/search?q=alice
```

**Query Parameters:**
- `q` (required) — Search term (chat name or participant name)

**Response (200 OK):**
```json
{
  "chats": [
    {
      "id": "660f9400-f29b-41d4-a716-446655440001",
      "type": "direct",
      "participants": [...],
      "createdAt": "2026-06-28T10:00:00Z"
    }
  ]
}
```

---

#### Search Contacts

```http
GET /contacts/search?q=alice
```

**Query Parameters:**
- `q` (required) — Search term (name or username)

**Response (200 OK):**
```json
{
  "users": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "username": "alice",
      "displayName": "Alice",
      "nativeLanguage": "en"
    }
  ]
}
```

---

### Presence

#### Get Presence Status

```http
GET /presence/{userId}
```

**Response (200 OK):**
```json
{
  "userId": "550e8400-e29b-41d4-a716-446655440001",
  "status": "online",
  "lastSeen": "2026-06-28T10:20:00Z",
  "deviceType": "web"
}
```

---

#### Update Presence

```http
PUT /presence
Content-Type: application/json

{
  "status": "online",        // 'online' or 'away'
  "deviceType": "web"        // 'mobile', 'web', 'desktop'
}
```

**Response (200 OK):**
```json
{
  "status": "success"
}
```

---

#### Update Activity

```http
POST /presence/activity
Content-Type: application/json

{
  "type": "typing",          // Activity type
  "chatId": "660f9400-f29b-41d4-a716-446655440001"
}
```

**Response (200 OK):**
```json
{
  "status": "success"
}
```

---

### Grammar Analysis

#### Analyze Message Grammar

```http
POST /grammar/analyze
Content-Type: application/json

{
  "messageId": "770f9500-g29b-41d4-a716-446655440003"
}
```

**Response (200 OK):**
```json
{
  "difficulty": "A2",
  "patterns": [
    "simple_present_question",
    "verb_to_be"
  ],
  "explanations": [
    "Uses simple present tense for questions",
    "Includes question word 'how'"
  ]
}
```

---

#### Analyze Text Grammar

```http
POST /grammar/analyze-text
Content-Type: application/json

{
  "text": "How are you?",
  "language": "en"
}
```

**Response (200 OK):**
```json
{
  "difficulty": "A1",
  "patterns": ["simple_present_question"],
  "explanations": ["Basic greeting question"]
}
```

---

#### Get Grammar Suggestions

```http
GET /grammar/suggestions?messageId={messageId}
```

**Response (200 OK):**
```json
{
  "suggestions": [
    {
      "original": "How are you?",
      "suggestion": "How are you doing?",
      "explanation": "Alternative form"
    }
  ]
}
```

---

#### Get Grammar Report

```http
GET /grammar/report?period=week
```

**Query Parameters:**
- `period` (optional) — 'day', 'week', 'month' (default: week)

**Response (200 OK):**
```json
{
  "totalAnalyzed": 45,
  "averageDifficulty": "B1",
  "patternFrequency": {
    "simple_present_question": 15,
    "verb_to_be": 12,
    "conditionals": 5
  }
}
```

---

### Vocabulary

#### Save Vocabulary Word

```http
POST /vocabulary
Content-Type: application/json

{
  "term": "serendipity",
  "language": "en",
  "messageId": "770f9500-g29b-41d4-a716-446655440003"
}
```

**Response (201 Created):**
```json
{
  "id": "880g9600-h29b-41d4-a716-446655440004",
  "userId": "550e8400-e29b-41d4-a716-446655440000",
  "term": "serendipity",
  "language": "en",
  "translation": "serendipia (es)",
  "definition": "Finding something good by luck",
  "context": {
    "messageId": "770f9500-g29b-41d4-a716-446655440003",
    "sentence": "It was pure serendipity that we met",
    "chatId": "660f9400-f29b-41d4-a716-446655440001"
  },
  "learningData": {
    "reviewCount": 0,
    "correctCount": 0,
    "nextReview": "2026-06-29T10:20:00Z",
    "interval": 1
  },
  "createdAt": "2026-06-28T10:20:00Z"
}
```

---

#### Get Vocabulary List

```http
GET /vocabulary?language=en&limit=20&offset=0
```

**Query Parameters:**
- `language` (optional) — Filter by language
- `limit` (optional) — Items per page (default: 50)
- `offset` (optional) — Pagination offset (default: 0)

**Response (200 OK):**
```json
[
  {
    "id": "880g9600-h29b-41d4-a716-446655440004",
    "term": "serendipity",
    "language": "en",
    "translation": "serendipia",
    "learningData": {
      "reviewCount": 2,
      "correctCount": 2,
      "nextReview": "2026-06-30T10:20:00Z",
      "interval": 3
    }
  }
]
```

---

#### Get Due Vocabulary (Spaced Repetition)

```http
GET /vocabulary/due
```

**Response (200 OK):**
```json
[
  {
    "id": "880g9600-h29b-41d4-a716-446655440004",
    "term": "serendipity",
    "language": "en",
    "translation": "serendipia",
    "nextReview": "2026-06-28T10:20:00Z"
  }
]
```

---

#### Get Vocabulary by ID

```http
GET /vocabulary/{vocabularyId}
```

**Response (200 OK):**
```json
{
  "id": "880g9600-h29b-41d4-a716-446655440004",
  "term": "serendipity",
  "language": "en",
  "translation": "serendipia",
  "definition": "Finding something good by luck",
  "learningData": { ... }
}
```

---

#### Update Practice Result

```http
POST /vocabulary/practice
Content-Type: application/json

{
  "vocabularyId": "880g9600-h29b-41d4-a716-446655440004",
  "correct": true     // true or false
}
```

**Response (200 OK):**
```json
{
  "id": "880g9600-h29b-41d4-a716-446655440004",
  "term": "serendipity",
  "learningData": {
    "reviewCount": 3,
    "correctCount": 3,
    "nextReview": "2026-07-01T10:20:00Z",
    "interval": 5
  }
}
```

---

#### Get Vocabulary Progress

```http
GET /vocabulary/progress
```

**Response (200 OK):**
```json
{
  "totalVocabulary": 45,
  "masteredCount": 15,
  "learningCount": 20,
  "newCount": 10,
  "accuracy": 0.89,
  "dueToday": 5,
  "averageInterval": 3.5
}
```

---

#### Delete Vocabulary

```http
DELETE /vocabulary/{vocabularyId}
```

**Response (200 OK):**
```json
{
  "status": "success"
}
```

---

#### Search Vocabulary

```http
GET /vocabulary/search?q=ser
```

**Query Parameters:**
- `q` (required) — Search term

**Response (200 OK):**
```json
[
  {
    "id": "880g9600-h29b-41d4-a716-446655440004",
    "term": "serendipity",
    "language": "en",
    "translation": "serendipia"
  }
]
```

---

### Calls

#### Initiate Call

```http
POST /calls/initiate
Content-Type: application/json

{
  "chatId": "660f9400-f29b-41d4-a716-446655440001",
  "type": "audio"  // 'audio' or 'video'
}
```

**Response (201 Created):**
```json
{
  "id": "990h9700-i29b-41d4-a716-446655440005",
  "chatId": "660f9400-f29b-41d4-a716-446655440001",
  "participants": ["550e8400-e29b-41d4-a716-446655440000", "550e8400-e29b-41d4-a716-446655440001"],
  "type": "audio",
  "status": "active",
  "startedAt": "2026-06-28T10:25:00Z"
}
```

---

#### End Call

```http
POST /calls/{callId}/end
```

**Response (200 OK):**
```json
{
  "id": "990h9700-i29b-41d4-a716-446655440005",
  "status": "ended",
  "endedAt": "2026-06-28T10:35:00Z"
}
```

---

#### Get Call Session

```http
GET /calls/{callId}
```

**Response (200 OK):**
```json
{
  "id": "990h9700-i29b-41d4-a716-446655440005",
  "chatId": "660f9400-f29b-41d4-a716-446655440001",
  "participants": [...],
  "type": "audio",
  "status": "ended",
  "startedAt": "2026-06-28T10:25:00Z",
  "endedAt": "2026-06-28T10:35:00Z"
}
```

---

#### Get Call Transcript

```http
GET /calls/{callId}/transcript
```

**Response (200 OK):**
```json
{
  "id": "aa0i9800-j29b-41d4-a716-446655440006",
  "callId": "990h9700-i29b-41d4-a716-446655440005",
  "segments": [
    {
      "speakerId": "550e8400-e29b-41d4-a716-446655440000",
      "startTime": 0.0,
      "endTime": 2.5,
      "originalText": "Hello, how are you?",
      "originalLanguage": "en",
      "translations": {
        "es": "Hola, ¿cómo estás?"
      },
      "confidence": 0.95
    }
  ],
  "createdAt": "2026-06-28T10:35:00Z"
}
```

---

#### Get Call History

```http
GET /calls/history?limit=20&offset=0
```

**Query Parameters:**
- `limit` (optional) — Items per page
- `offset` (optional) — Pagination offset

**Response (200 OK):**
```json
[
  {
    "id": "990h9700-i29b-41d4-a716-446655440005",
    "type": "audio",
    "status": "ended",
    "startedAt": "2026-06-28T10:25:00Z",
    "endedAt": "2026-06-28T10:35:00Z",
    "duration": 600
  }
]
```

---

#### Delete Call Transcript

```http
DELETE /calls/{callId}/transcript
```

**Response (200 OK):**
```json
{
  "status": "success"
}
```

---

#### Search Transcripts

```http
GET /calls/transcripts/search?q=hello
```

**Query Parameters:**
- `q` (required) — Search term

**Response (200 OK):**
```json
[
  {
    "id": "aa0i9800-j29b-41d4-a716-446655440006",
    "callId": "990h9700-i29b-41d4-a716-446655440005",
    "segments": [...]
  }
]
```

---

#### Handle WebRTC Signaling

```http
POST /calls/{callId}/signal
Content-Type: application/json

{
  "type": "offer|answer|ice-candidate",
  "data": { /* WebRTC data */ }
}
```

**Response (200 OK):**
```json
{
  "status": "success"
}
```

---

## WebSocket Events

### Connection

**Client connects to:** `ws://localhost:8080/ws`

With header: `Authorization: Bearer <accessToken>`

```javascript
// Frontend
wsService.connect(accessToken)
```

---

### Incoming Events (Server → Client)

#### New Message

```json
{
  "type": "new_message",
  "data": {
    "id": "770f9500-g29b-41d4-a716-446655440003",
    "chatId": "660f9400-f29b-41d4-a716-446655440001",
    "senderId": "550e8400-e29b-41d4-a716-446655440001",
    "text": "How are you?",
    "originalLanguage": "en",
    "translations": {
      "es": "¿Cómo estás?",
      "fr": "Comment allez-vous?"
    },
    "deliveryStatus": "delivered",
    "timestamp": "2026-06-28T10:15:00Z"
  }
}
```

---

#### Message Updated

```json
{
  "type": "message_updated",
  "data": {
    "id": "770f9500-g29b-41d4-a716-446655440003",
    "deliveryStatus": "delivered"
  }
}
```

---

#### Chat Updated

```json
{
  "type": "chat_updated",
  "data": {
    "chatId": "660f9400-f29b-41d4-a716-446655440001",
    "name": "Updated Group Name",
    "settings": { "translationEnabled": false }
  }
}
```

---

#### User Typing

```json
{
  "type": "user_typing",
  "data": {
    "chatId": "660f9400-f29b-41d4-a716-446655440001",
    "userId": "550e8400-e29b-41d4-a716-446655440001",
    "isTyping": true
  }
}
```

---

#### Presence Update

```json
{
  "type": "presence_update",
  "data": {
    "userId": "550e8400-e29b-41d4-a716-446655440001",
    "status": "online",
    "deviceType": "web"
  }
}
```

---

### Outgoing Events (Client → Server)

#### Send Typing Indicator

```json
{
  "type": "typing_start",
  "data": {
    "chatId": "660f9400-f29b-41d4-a716-446655440001"
  }
}
```

Stop typing:

```json
{
  "type": "typing_stop",
  "data": {
    "chatId": "660f9400-f29b-41d4-a716-446655440001"
  }
}
```

---

## HTTP Status Codes

| Code | Meaning |
|------|---------|
| `200` | OK — Request succeeded |
| `201` | Created — Resource created successfully |
| `400` | Bad Request — Invalid input or format |
| `401` | Unauthorized — Missing or invalid token |
| `403` | Forbidden — Not permitted to access |
| `404` | Not Found — Resource doesn't exist |
| `409` | Conflict — Resource already exists |
| `500` | Server Error — Internal server error |

---

## Error Response Format

All errors follow this format:

```json
{
  "error": "Description of the error"
}
```

Example:
```json
{
  "error": "Invalid credentials"
}
```

---

## Rate Limiting

Currently no rate limiting implemented, but planned for production.

---

## Best Practices

1. **Store tokens** in localStorage or secure storage
2. **Send Authorization header** on all protected requests
3. **Handle 401 responses** by refreshing token
4. **Implement pagination** for large lists (use `limit` and `offset`)
5. **Handle WebSocket reconnection** with exponential backoff
6. **Validate input** on client before sending
7. **Use HTTPS** in production (for secure token transmission)

