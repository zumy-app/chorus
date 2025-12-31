# Design Document: Multilingual Messenger - Iterative Development

## Overview

The multilingual messenger (Chorus) will be developed in iterative phases, starting with a minimal viable product (MVP) and evolving into a full-featured platform. This approach allows for rapid delivery, user feedback incorporation, and risk mitigation while building toward WhatsApp-scale architecture.

Each phase has clear goals, deliverables, and architectural evolution to ensure manageable development cycles and early user validation.

## Development Phases

### Phase 1: Production-Ready Messaging System (4-5 weeks)
**Goal:** Complete messaging platform with group chats and translation
**Scope:** PostgreSQL, Redis, group chats, message persistence, Docker deployment

### Phase 2: Multi-Device & Scaling (4-5 weeks)
**Goal:** Horizontal scaling with multi-device support
**Scope:** Redis Pub/Sub, Kubernetes, offline message delivery, monitoring

### Phase 3: Advanced Features (5-6 weeks)
**Goal:** Language learning and advanced communication features
**Scope:** Grammar analysis, vocabulary management, voice calls, ML services

---

## Phase 1: Production-Ready Messaging System

### Phase 1 Goals
- **Primary:** Complete messaging platform with group chats and translation
- **Secondary:** Production-ready infrastructure with Docker deployment
- **Delivery:** Scalable messaging system supporting 1K-10K concurrent users

### Phase 1 Tech Stack (Simplified for MVP)

**Backend Services (Go + Appwrite - Simple & Fast):**
- **Language:** Go 1.21+ with Gin framework (single binary, easy deployment)
- **Database:** Appwrite Database (PostgreSQL backend, managed by Appwrite)
- **Authentication:** Appwrite Auth service (handles JWT, user management, sessions)
- **Real-time:** Appwrite Realtime for WebSocket connections (no custom WebSocket needed)
- **File Storage:** Appwrite Storage for media files (images, documents, voice messages)
- **Translation Processing:** Custom Go service running on your VPS (no external dependencies)
- **Caching:** Simple in-memory cache in Go (no Redis needed initially)
- **Testing:** Go standard testing package (keep it simple)

**Translation Architecture (All On-VPS):**
```go
// Translation service running on your VPS
type TranslationService struct {
    googleClient *translate.Client  // Google Translate API client
    cache        map[string]string  // Simple in-memory cache
    rateLimiter  *time.Ticker      // Simple rate limiting
}

// Process translation entirely on your VPS
func (t *TranslationService) TranslateMessage(text, targetLang string) (string, error) {
    // 1. Check cache first (avoid API calls)
    cacheKey := fmt.Sprintf("%s:%s", text, targetLang)
    if cached, exists := t.cache[cacheKey]; exists {
        return cached, nil
    }
    
    // 2. Call Google Translate API from your VPS
    result, err := t.googleClient.Translate(context.Background(), []string{text}, targetLang, nil)
    if err != nil {
        return text, err  // Return original on error
    }
    
    // 3. Cache result in memory
    t.cache[cacheKey] = result[0].Text
    
    return result[0].Text, nil
}

// Simple message flow with translation
func (s *MessageService) SendMessage(chatID, userID, text string) error {
    // 1. Store original message in Appwrite
    message := map[string]interface{}{
        "chatId":   chatID,
        "senderId": userID,
        "text":     text,
        "originalLanguage": detectLanguage(text),
        "createdAt": time.Now(),
    }
    
    doc, err := s.appwrite.CreateDocument("messages", message)
    if err != nil {
        return err
    }
    
    // 2. Get chat participants and their target languages
    participants, err := s.getChatParticipants(chatID)
    if err != nil {
        return err
    }
    
    // 3. Translate for each participant (simple loop)
    translations := make(map[string]string)
    for _, participant := range participants {
        if participant.TargetLanguage != message["originalLanguage"] {
            translated, err := s.translator.TranslateMessage(text, participant.TargetLanguage)
            if err == nil {
                translations[participant.TargetLanguage] = translated
            }
        }
    }
    
    // 4. Update message with translations
    s.appwrite.UpdateDocument("messages", doc.ID, map[string]interface{}{
        "translations": translations,
    })
    
    // 5. Broadcast via Appwrite Realtime (automatic to all connected clients)
    return nil
}
```

**Backend Service Architecture:**
```go
// Simple Go service structure for Phase 1
cmd/
├── main.go                 // Single main server binary

internal/
├── handlers/
│   ├── auth.go            // Appwrite auth integration
│   ├── messages.go        // Message CRUD operations  
│   ├── chats.go          // Chat management
│   └── translation.go     // Translation endpoints
├── services/
│   ├── appwrite.go       // Appwrite client wrapper
│   ├── translator.go     // Google Translate integration
│   └── message.go        // Business logic
└── models/
    ├── user.go           // User data structures
    ├── chat.go           // Chat data structures
    └── message.go        // Message data structures

// Single Docker container deployment
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o messenger ./cmd/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/messenger .
CMD ["./messenger"]
```

**Frontend Applications (Simplified):**

**Web Application (React - Keep It Simple):**
- **Framework:** React 18+ with TypeScript
- **State Management:** React's built-in useState and useContext (no Redux!)
- **UI Library:** Tailwind CSS for styling (no component library needed)
- **Appwrite SDK:** Appwrite Web SDK for all backend operations
- **Real-time:** Appwrite Realtime subscriptions (no custom WebSocket code)
- **Forms:** Simple controlled components (no form library needed)
- **Build Tool:** Vite for fast development
- **Testing:** Basic React Testing Library (minimal tests)

```tsx
// Simple React state management for Phase 1
function App() {
  const [user, setUser] = useState(null);
  const [chats, setChats] = useState([]);
  const [messages, setMessages] = useState({});
  
  // Simple Appwrite integration
  useEffect(() => {
    // Subscribe to real-time updates
    appwrite.subscribe('documents', (response) => {
      if (response.events.includes('databases.messages.documents.create')) {
        // New message received
        const message = response.payload;
        setMessages(prev => ({
          ...prev,
          [message.chatId]: [...(prev[message.chatId] || []), message]
        }));
      }
    });
  }, []);
  
  return (
    <div className="flex h-screen">
      <ChatList chats={chats} />
      <MessageArea messages={messages} />
    </div>
  );
}
```

**Mobile Application (React Native - Simplified):**
- **Framework:** React Native 0.72+ with TypeScript
- **Navigation:** React Navigation v6 (basic stack navigation)
- **State Management:** React's built-in state (no external state library)
- **UI Components:** React Native built-in components + Tailwind CSS (NativeWind)
- **Appwrite SDK:** Appwrite React Native SDK
- **Storage:** AsyncStorage for simple offline data
- **Push Notifications:** Basic React Native Push Notification
- **Testing:** Minimal Jest tests

**Desktop Application (Phase 3):**
- **Move to Phase 3:** Desktop app adds complexity, focus on web/mobile first

**Simplified API Design:**
```typescript
// Phase 1 API - Simple REST endpoints
GET /api/chats                    // Get user's chats
POST /api/chats                   // Create new chat
GET /api/chats/{id}/messages      // Get chat messages  
POST /api/chats/{id}/messages     // Send message (auto-translates)
PUT /api/users/me                 // Update user profile/languages

// Appwrite handles:
// - Authentication (login, register, sessions)
// - Real-time subscriptions (message updates)
// - File uploads (media messages)
// - User management (profiles, preferences)
```

**What Moves to Later Phases:**

**Phase 2 (Multi-Device & Scaling):**
- Advanced state management (Zustand/Redux)
- Redis caching and sessions
- Rate limiting and abuse prevention
- Advanced error handling and retry logic
- Horizontal scaling with multiple Go instances
- Database sharding preparation

**Phase 3 (Advanced Features):**
- Complex UI components and libraries
- Desktop application (Tauri)
- Advanced WebSocket management
- Grammar analysis and vocabulary features
- Voice/video calling
- Advanced search and indexing
- Comprehensive monitoring and analytics

**Phase 1 Deployment (Single VPS):**
```yaml
# Simple docker-compose.yml for Phase 1
version: '3.8'
services:
  messenger-api:
    build: .
    ports:
      - "8080:8080"
    environment:
      - APPWRITE_ENDPOINT=http://appwrite/v1
      - APPWRITE_PROJECT_ID=your-project-id
      - APPWRITE_API_KEY=your-api-key
      - GOOGLE_TRANSLATE_API_KEY=your-translate-key
    depends_on:
      - appwrite
      
  appwrite:
    # Your existing Appwrite installation
    # Handles: auth, database, storage, realtime
    
  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
```

**Phase 1 Focus:**
- ✅ Core messaging works reliably
- ✅ Real-time translation on your VPS
- ✅ Simple, maintainable codebase
- ✅ Fast development and deployment
- ✅ Appwrite handles complexity (auth, realtime, storage)
- ❌ No premature optimization
- ❌ No complex state management
- ❌ No advanced scaling features

**Go + Appwrite Architecture:**
- **Go Services:** Custom messaging logic, translation processing, WebSocket management
- **Appwrite Services:** Authentication, user management, database operations, file storage
- **Integration:** Go services use Appwrite Server SDK for database and auth operations
- **API Style:** RESTful APIs with Go Gin framework + Appwrite REST API
- **Documentation:** Go Swagger for custom APIs + Appwrite built-in API docs
- **Rate Limiting:** Go middleware + Appwrite built-in rate limiting
- **CORS:** Configured in both Go services and Appwrite
- **Logging:** Go structured logging (logrus/zap) + Appwrite audit logs

**Cost Analysis:**
- **Socket.io:** Free and open source (but not needed with Appwrite Realtime)
- **Appwrite:** Free and open source, self-hosted on your VPS
- **Go:** Free and open source, excellent performance
- **Translation APIs:** Pay-per-use (Google Translate ~$20/1M characters)
- **Total Backend Cost:** $0 infrastructure + translation API usage only

**External APIs:**
- **Translation:** Google Translate API (primary), DeepL API (fallback)
- **Language Detection:** Google Cloud Translation API
- **Appwrite Services:** Self-hosted Appwrite for auth, database, storage, realtime
- **Monitoring:** Prometheus + Grafana for metrics, built-in Go pprof for profiling

**Infrastructure:**
- **Containerization:** Docker with multi-stage builds (smaller Go binaries)
- **Orchestration:** Docker Compose for development, Kubernetes for production
- **Load Balancing:** NGINX for HTTP traffic (Appwrite handles WebSocket load balancing)
- **SSL/TLS:** Let's Encrypt certificates with automatic renewal
- **Self-hosted:** Appwrite on your VPS for auth, database, storage, and realtime features

### Phase 1 Enhanced Entities
```typescript
interface User {
  id: string;
  username: string;
  displayName: string;
  email: string;
  nativeLanguage: string;
  targetLanguages: string[]; // Multiple target languages
  createdAt: Date;
  lastActiveAt: Date;
}

interface Chat {
  id: string;
  type: 'direct' | 'group';
  participants: string[]; // 2-100 participants
  name?: string; // Group name for group chats
  createdBy: string;
  settings: {
    translationEnabled: boolean;
  };
  createdAt: Date;
}

interface Message {
  id: string;
  chatId: string;
  senderId: string;
  text: string;
  originalLanguage: string;
  translations: Record<string, string>; // Multiple translations
  deliveryStatus: 'sent' | 'delivered' | 'failed';
  timestamp: Date;
}

interface ChatParticipant {
  chatId: string;
  userId: string;
  role: 'member' | 'admin';
  joinedAt: Date;
  lastReadMessageId?: string;
}
```

### Phase 1 Complete API Specification

**Base URL:** `https://api.multilingual-messenger.com/v1`

**Authentication:**
All API endpoints (except auth) require JWT token in Authorization header:
```
Authorization: Bearer <jwt_token>
```

**Error Response Format:**
```typescript
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid request parameters",
    "details": [
      {
        "field": "username",
        "message": "Username must be at least 3 characters"
      }
    ]
  },
  "timestamp": "2024-01-15T10:30:00Z",
  "requestId": "req_123456"
}
```

**Authentication Endpoints:**
```typescript
// User Registration
POST /auth/register
Content-Type: application/json
{
  "username": "john_doe",
  "email": "john@example.com", 
  "password": "SecurePass123!",
  "displayName": "John Doe",
  "nativeLanguage": "en",
  "targetLanguages": ["es", "fr"]
}
→ 201 Created
{
  "user": {
    "id": "user_123",
    "username": "john_doe",
    "displayName": "John Doe",
    "email": "john@example.com",
    "nativeLanguage": "en",
    "targetLanguages": ["es", "fr"]
  },
  "tokens": {
    "accessToken": "eyJhbGciOiJIUzI1NiIs...",
    "refreshToken": "eyJhbGciOiJIUzI1NiIs...",
    "expiresIn": 3600
  }
}

// User Login
POST /auth/login
Content-Type: application/json
{
  "username": "john_doe",
  "password": "SecurePass123!"
}
→ 200 OK
{
  "user": { /* user object */ },
  "tokens": { /* tokens object */ }
}

// Refresh Token
POST /auth/refresh
Content-Type: application/json
{
  "refreshToken": "eyJhbGciOiJIUzI1NiIs..."
}
→ 200 OK
{
  "accessToken": "eyJhbGciOiJIUzI1NiIs...",
  "expiresIn": 3600
}

// Logout
POST /auth/logout
Authorization: Bearer <token>
→ 204 No Content
```

**User Management Endpoints:**
```typescript
// Get Current User Profile
GET /users/me
Authorization: Bearer <token>
→ 200 OK
{
  "id": "user_123",
  "username": "john_doe",
  "displayName": "John Doe",
  "email": "john@example.com",
  "nativeLanguage": "en",
  "targetLanguages": ["es", "fr"],
  "createdAt": "2024-01-01T00:00:00Z",
  "lastActiveAt": "2024-01-15T10:30:00Z"
}

// Update User Profile
PUT /users/me
Authorization: Bearer <token>
Content-Type: application/json
{
  "displayName": "John Smith",
  "targetLanguages": ["es", "fr", "de"]
}
→ 200 OK
{
  /* updated user object */
}

// Search Users
GET /users/search?q=john&limit=10
Authorization: Bearer <token>
→ 200 OK
{
  "users": [
    {
      "id": "user_456",
      "username": "john_smith",
      "displayName": "John Smith"
    }
  ],
  "total": 1,
  "hasMore": false
}
```

**Chat Management Endpoints:**
```typescript
// Get User's Chats
GET /chats?limit=20&offset=0
Authorization: Bearer <token>
→ 200 OK
{
  "chats": [
    {
      "id": "chat_123",
      "type": "group",
      "name": "Study Group",
      "participants": [
        {
          "id": "user_123",
          "displayName": "John Doe",
          "role": "admin"
        },
        {
          "id": "user_456", 
          "displayName": "Jane Smith",
          "role": "member"
        }
      ],
      "lastMessage": {
        "id": "msg_789",
        "text": "Hello everyone!",
        "senderId": "user_456",
        "timestamp": "2024-01-15T10:30:00Z"
      },
      "unreadCount": 2,
      "createdAt": "2024-01-10T00:00:00Z"
    }
  ],
  "total": 5,
  "hasMore": true
}

// Create New Chat
POST /chats
Authorization: Bearer <token>
Content-Type: application/json
{
  "type": "group",
  "name": "Study Group",
  "participants": ["user_456", "user_789"]
}
→ 201 Created
{
  "id": "chat_123",
  "type": "group",
  "name": "Study Group",
  "participants": [/* participant objects */],
  "createdBy": "user_123",
  "settings": {
    "translationEnabled": true
  },
  "createdAt": "2024-01-15T10:30:00Z"
}

// Get Chat Details
GET /chats/{chatId}
Authorization: Bearer <token>
→ 200 OK
{
  /* complete chat object with participants */
}

// Update Chat
PUT /chats/{chatId}
Authorization: Bearer <token>
Content-Type: application/json
{
  "name": "Updated Study Group",
  "settings": {
    "translationEnabled": false
  }
}
→ 200 OK
{
  /* updated chat object */
}

// Add Participant to Chat
POST /chats/{chatId}/participants
Authorization: Bearer <token>
Content-Type: application/json
{
  "userId": "user_999"
}
→ 201 Created
{
  "message": "User added successfully"
}

// Remove Participant from Chat
DELETE /chats/{chatId}/participants/{userId}
Authorization: Bearer <token>
→ 204 No Content

// Leave Chat
DELETE /chats/{chatId}/participants/me
Authorization: Bearer <token>
→ 204 No Content
```

**Messaging Endpoints:**
```typescript
// Get Chat Messages
GET /chats/{chatId}/messages?limit=50&before=msg_456
Authorization: Bearer <token>
→ 200 OK
{
  "messages": [
    {
      "id": "msg_789",
      "chatId": "chat_123",
      "senderId": "user_456",
      "text": "Hello everyone, how are you?",
      "originalLanguage": "en",
      "translations": {
        "es": "Hola a todos, ¿cómo están?",
        "fr": "Salut tout le monde, comment allez-vous?"
      },
      "timestamp": "2024-01-15T10:30:00Z",
      "deliveryStatus": "delivered"
    }
  ],
  "hasMore": true,
  "nextCursor": "msg_456"
}

// Send Message
POST /chats/{chatId}/messages
Authorization: Bearer <token>
Content-Type: application/json
{
  "text": "Hello everyone, how are you?",
  "replyToId": "msg_456" // Optional
}
→ 201 Created
{
  "id": "msg_789",
  "chatId": "chat_123",
  "senderId": "user_123",
  "text": "Hello everyone, how are you?",
  "originalLanguage": "en",
  "translations": {
    "es": "Hola a todos, ¿cómo están?",
    "fr": "Salut tout le monde, comment allez-vous?"
  },
  "timestamp": "2024-01-15T10:30:00Z",
  "deliveryStatus": "sent"
}

// Mark Messages as Read
PUT /chats/{chatId}/read
Authorization: Bearer <token>
Content-Type: application/json
{
  "messageId": "msg_789"
}
→ 204 No Content

// Search Messages
GET /messages/search?q=hello&chatId=chat_123&limit=20
Authorization: Bearer <token>
→ 200 OK
{
  "messages": [/* matching messages with highlighted text */],
  "total": 5,
  "hasMore": false
}
```

**WebSocket Events:**
```typescript
// Connection
WebSocket: wss://api.multilingual-messenger.com/ws
Authorization: Bearer <token>

// Client → Server Events
{
  "type": "join_chat",
  "chatId": "chat_123"
}

{
  "type": "typing_start",
  "chatId": "chat_123"
}

{
  "type": "typing_stop", 
  "chatId": "chat_123"
}

// Server → Client Events
{
  "type": "new_message",
  "data": {
    "chatId": "chat_123",
    "message": {/* message object */}
  }
}

{
  "type": "message_updated",
  "data": {
    "chatId": "chat_123",
    "messageId": "msg_789",
    "deliveryStatus": "delivered"
  }
}

{
  "type": "chat_updated",
  "data": {
    "chatId": "chat_123",
    "participants": [/* updated participants */]
  }
}

{
  "type": "user_typing",
  "data": {
    "chatId": "chat_123",
    "userId": "user_456",
    "isTyping": true
  }
}

{
  "type": "user_online",
  "data": {
    "userId": "user_456",
    "status": "online"
  }
}
```

**Rate Limiting:**
- **Authentication endpoints:** 5 requests per minute per IP
- **Message sending:** 100 messages per minute per user
- **API calls:** 1000 requests per hour per user
- **WebSocket connections:** 5 concurrent connections per user

**Response Headers:**
```
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 999
X-RateLimit-Reset: 1642262400
X-Request-ID: req_123456
```
```typescript
// Authentication
POST /api/auth/register
{
  "username": "john_doe",
  "email": "john@example.com",
  "password": "secure_password",
  "displayName": "John Doe",
  "nativeLanguage": "en",
  "targetLanguages": ["es", "fr"]
}
→ { "token": "jwt_token", "refreshToken": "refresh_token", "userId": "user_123" }

POST /api/auth/login
{
  "username": "john_doe",
  "password": "secure_password"
}
→ { "token": "jwt_token", "refreshToken": "refresh_token", "userId": "user_123" }

// Chat Management
POST /api/chats
{
  "type": "group",
  "participants": ["user_123", "user_456", "user_789"],
  "name": "Study Group"
}
→ { "chatId": "chat_123" }

POST /api/chats/{chatId}/participants
{
  "userId": "user_999",
  "action": "add"
}
→ "SUCCESS"

GET /api/chats
→ [
  {
    "id": "chat_123",
    "type": "group", 
    "name": "Study Group",
    "participants": ["user_123", "user_456"],
    "lastMessage": {
      "text": "Hello everyone",
      "timestamp": "2024-01-15T10:30:00Z"
    }
  }
]

// Messaging
POST /api/messages
{
  "chatId": "chat_123",
  "text": "Hello everyone, how are you?"
}
→ { 
  "messageId": "msg_789",
  "translations": {
    "es": "Hola a todos, ¿cómo están?",
    "fr": "Salut tout le monde, comment allez-vous?"
  }
}

GET /api/messages/{chatId}?limit=50&before=msg_456
→ [
  {
    "id": "msg_789",
    "senderId": "user_123",
    "text": "Hello everyone",
    "originalLanguage": "en",
    "translations": { "es": "Hola a todos", "fr": "Salut tout le monde" },
    "timestamp": "2024-01-15T10:30:00Z"
  }
]

// Real-time WebSocket Events
WebSocket /ws
← newMessage
{
  "type": "newMessage",
  "chatId": "chat_123",
  "message": {
    "id": "msg_789",
    "senderId": "user_456",
    "text": "Hello everyone",
    "translations": { "es": "Hola a todos" },
    "timestamp": "2024-01-15T10:30:00Z"
  }
}

← chatUpdate
{
  "type": "chatUpdate", 
  "chatId": "chat_123",
  "participants": ["user_123", "user_456", "user_999"],
  "name": "Updated Study Group"
}

← typingIndicator
{
  "type": "typing",
  "chatId": "chat_123",
  "userId": "user_456",
  "isTyping": true
}
```

### Phase 1 Architecture with Go + Appwrite

```mermaid
graph TB
    subgraph "Client Applications"
        WebApp[Web App - React + TypeScript]
        MobileApp[Mobile App - React Native]
        DesktopApp[Desktop App - Electron]
    end
    
    subgraph "Load Balancer"
        NGINX[NGINX Load Balancer]
    end
    
    subgraph "Go Backend Services"
        API1[Translation API Server 1 - Go]
        API2[Translation API Server 2 - Go]
        Worker1[Translation Worker 1 - Go]
        Worker2[Translation Worker 2 - Go]
    end
    
    subgraph "Appwrite Services (Self-hosted VPS)"
        AppwriteAuth[Appwrite Auth Service]
        AppwriteDB[Appwrite Database Service]
        AppwriteStorage[Appwrite Storage Service]
        AppwriteRealtime[Appwrite Realtime Service]
    end
    
    subgraph "Additional Services"
        RedisCache[(Redis Cache)]
        RedisQueue[(Redis Queue)]
    end
    
    subgraph "External Services"
        GoogleTranslate[Google Translate API]
        DeepLAPI[DeepL API - Fallback]
    end
    
    subgraph "Monitoring"
        Prometheus[Prometheus Metrics]
        Grafana[Grafana Dashboard]
    end
    
    %% Client connections
    WebApp --> NGINX
    MobileApp --> NGINX
    DesktopApp --> NGINX
    
    %% Load balancing to Go services
    NGINX --> API1
    NGINX --> API2
    
    %% Direct client connections to Appwrite Realtime
    WebApp -.->|WebSocket| AppwriteRealtime
    MobileApp -.->|WebSocket| AppwriteRealtime
    DesktopApp -.->|WebSocket| AppwriteRealtime
    
    %% Go services to Appwrite
    API1 --> AppwriteAuth
    API1 --> AppwriteDB
    API1 --> AppwriteStorage
    API2 --> AppwriteAuth
    API2 --> AppwriteDB
    API2 --> AppwriteStorage
    
    %% Translation processing
    API1 --> RedisQueue
    API2 --> RedisQueue
    RedisQueue --> Worker1
    RedisQueue --> Worker2
    Worker1 --> RedisCache
    Worker2 --> RedisCache
    
    %% External APIs
    Worker1 --> GoogleTranslate
    Worker1 --> DeepLAPI
    Worker2 --> GoogleTranslate
    Worker2 --> DeepLAPI
    
    %% Monitoring
    API1 --> Prometheus
    API2 --> Prometheus
    Worker1 --> Prometheus
    Worker2 --> Prometheus
    Prometheus --> Grafana
    
    %% Message Flow Example
    WebApp -.->|1. Send Message| API1
    API1 -.->|2. Store Message| AppwriteDB
    API1 -.->|3. Queue Translation| RedisQueue
    RedisQueue -.->|4. Process Translation| Worker1
    Worker1 -.->|5. Cache Result| RedisCache
    API1 -.->|6. Trigger Realtime Event| AppwriteRealtime
    AppwriteRealtime -.->|7. Deliver to Recipients| MobileApp
    
    classDef client fill:#e1f5fe
    classDef go fill:#00ADD8
    classDef appwrite fill:#f02e65
    classDef storage fill:#e8f5e8
    classDef external fill:#fff3e0
    classDef monitoring fill:#f3e5f5
    
    class WebApp,MobileApp,DesktopApp client
    class API1,API2,Worker1,Worker2 go
    class AppwriteAuth,AppwriteDB,AppwriteStorage,AppwriteRealtime appwrite
    class RedisCache,RedisQueue storage
    class GoogleTranslate,DeepLAPI external
    class Prometheus,Grafana monitoring
```

### Phase 1 Appwrite Database Schema
```javascript
// Appwrite Collections Configuration

// Users Collection (handled by Appwrite Auth + custom attributes)
const usersAttributes = [
  { key: 'displayName', type: 'string', size: 100, required: true },
  { key: 'nativeLanguage', type: 'string', size: 10, required: true },
  { key: 'targetLanguages', type: 'string', array: true, required: false },
  { key: 'lastActiveAt', type: 'datetime', required: false }
];

// Chats Collection
const chatsCollection = {
  collectionId: 'chats',
  name: 'Chats',
  attributes: [
    { key: 'type', type: 'enum', elements: ['direct', 'group'], required: true },
    { key: 'name', type: 'string', size: 100, required: false },
    { key: 'participants', type: 'string', array: true, required: true },
    { key: 'createdBy', type: 'string', size: 36, required: true },
    { key: 'settings', type: 'string', size: 1000, required: false } // JSON string
  ],
  indexes: [
    { key: 'participants', type: 'key', attributes: ['participants'] },
    { key: 'createdBy', type: 'key', attributes: ['createdBy'] }
  ]
};

// Messages Collection  
const messagesCollection = {
  collectionId: 'messages',
  name: 'Messages',
  attributes: [
    { key: 'chatId', type: 'string', size: 36, required: true },
    { key: 'senderId', type: 'string', size: 36, required: true },
    { key: 'text', type: 'string', size: 10000, required: true },
    { key: 'originalLanguage', type: 'string', size: 10, required: false },
    { key: 'translations', type: 'string', size: 50000, required: false }, // JSON string
    { key: 'deliveryStatus', type: 'enum', elements: ['sent', 'delivered', 'failed'], required: true },
    { key: 'replyToId', type: 'string', size: 36, required: false }
  ],
  indexes: [
    { key: 'chatId_createdAt', type: 'key', attributes: ['chatId', '$createdAt'], orders: ['ASC', 'DESC'] },
    { key: 'senderId', type: 'key', attributes: ['senderId'] },
    { key: 'text_search', type: 'fulltext', attributes: ['text'] }
  ]
};

// Chat Participants Collection (for efficient queries)
const chatParticipantsCollection = {
  collectionId: 'chat_participants', 
  name: 'Chat Participants',
  attributes: [
    { key: 'chatId', type: 'string', size: 36, required: true },
    { key: 'userId', type: 'string', size: 36, required: true },
    { key: 'role', type: 'enum', elements: ['member', 'admin'], required: true },
    { key: 'lastReadMessageId', type: 'string', size: 36, required: false }
  ],
  indexes: [
    { key: 'chatId', type: 'key', attributes: ['chatId'] },
    { key: 'userId', type: 'key', attributes: ['userId'] },
    { key: 'unique_chat_user', type: 'unique', attributes: ['chatId', 'userId'] }
  ]
};
```

### Phase 1 Go API Structure
```go
// main.go - Go server structure
package main

import (
    "log"
    "github.com/gin-gonic/gin"
    "github.com/your-org/multilingual-messenger/internal/handlers"
    "github.com/your-org/multilingual-messenger/internal/services"
    "github.com/your-org/multilingual-messenger/internal/middleware"
)

func main() {
    // Initialize services
    appwriteClient := services.NewAppwriteClient()
    translationService := services.NewTranslationService()
    redisClient := services.NewRedisClient()
    
    // Initialize handlers
    chatHandler := handlers.NewChatHandler(appwriteClient, translationService)
    messageHandler := handlers.NewMessageHandler(appwriteClient, translationService, redisClient)
    
    // Setup Gin router
    r := gin.Default()
    
    // Middleware
    r.Use(middleware.CORS())
    r.Use(middleware.AppwriteAuth(appwriteClient))
    r.Use(middleware.RateLimit())
    
    // API routes
    api := r.Group("/api/v1")
    {
        // Chat routes
        api.GET("/chats", chatHandler.GetUserChats)
        api.POST("/chats", chatHandler.CreateChat)
        api.GET("/chats/:chatId", chatHandler.GetChat)
        api.PUT("/chats/:chatId", chatHandler.UpdateChat)
        api.POST("/chats/:chatId/participants", chatHandler.AddParticipant)
        api.DELETE("/chats/:chatId/participants/:userId", chatHandler.RemoveParticipant)
        
        // Message routes
        api.GET("/chats/:chatId/messages", messageHandler.GetMessages)
        api.POST("/chats/:chatId/messages", messageHandler.SendMessage)
        api.PUT("/chats/:chatId/read", messageHandler.MarkAsRead)
        api.GET("/messages/search", messageHandler.SearchMessages)
    }
    
    // Health check
    r.GET("/health", func(c *gin.Context) {
        c.JSON(200, gin.H{"status": "healthy"})
    })
    
    log.Println("Server starting on :8080")
    r.Run(":8080")
}

// Example handler structure
package handlers

type MessageHandler struct {
    appwrite    *services.AppwriteClient
    translator  *services.TranslationService
    redis       *services.RedisClient
}

func (h *MessageHandler) SendMessage(c *gin.Context) {
    var req SendMessageRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": "Invalid request"})
        return
    }
    
    // Get user from Appwrite auth context
    userID := c.GetString("userID")
    
    // Store message in Appwrite
    message, err := h.appwrite.CreateMessage(req.ChatID, userID, req.Text)
    if err != nil {
        c.JSON(500, gin.H{"error": "Failed to create message"})
        return
    }
    
    // Queue translation job
    go h.translator.TranslateAsync(message.ID, req.Text, req.ChatID)
    
    // Trigger Appwrite Realtime event
    h.appwrite.TriggerRealtimeEvent("messages", "create", message)
    
    c.JSON(201, message)
}
```

### Phase 1 Docker Deployment Architecture

**Development Environment (docker-compose.yml):**
```yaml
version: '3.8'

services:
  # Application Services
  api:
    build:
      context: .
      dockerfile: Dockerfile.api
    ports:
      - "3000:3000"
    environment:
      - NODE_ENV=development
      - DATABASE_URL=postgresql://messenger:password@postgres:5432/messenger_dev
      - REDIS_URL=redis://redis:6379
      - JWT_SECRET=dev_secret_key
      - GOOGLE_TRANSLATE_API_KEY=${GOOGLE_TRANSLATE_API_KEY}
    depends_on:
      - postgres
      - redis
    volumes:
      - ./src:/app/src
      - ./package.json:/app/package.json
    command: npm run dev

  websocket:
    build:
      context: .
      dockerfile: Dockerfile.websocket
    ports:
      - "3001:3001"
    environment:
      - NODE_ENV=development
      - REDIS_URL=redis://redis:6379
      - JWT_SECRET=dev_secret_key
    depends_on:
      - redis
    volumes:
      - ./src:/app/src

  # Database Services
  postgres:
    image: postgres:15-alpine
    environment:
      - POSTGRES_DB=messenger_dev
      - POSTGRES_USER=messenger
      - POSTGRES_PASSWORD=password
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./database/init.sql:/docker-entrypoint-initdb.d/init.sql

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data

  # Load Balancer
  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/nginx.conf
      - ./nginx/ssl:/etc/nginx/ssl
    depends_on:
      - api
      - websocket

  # Monitoring
  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./monitoring/prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus_data:/prometheus

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3002:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
    volumes:
      - grafana_data:/var/lib/grafana
      - ./monitoring/grafana/dashboards:/etc/grafana/provisioning/dashboards

volumes:
  postgres_data:
  redis_data:
  prometheus_data:
  grafana_data:
```

**Production Dockerfile (Multi-stage):**
```dockerfile
# Build stage
FROM node:18-alpine AS builder

WORKDIR /app
COPY package*.json ./
RUN npm ci --only=production

COPY . .
RUN npm run build

# Production stage
FROM node:18-alpine AS production

RUN addgroup -g 1001 -S nodejs
RUN adduser -S messenger -u 1001

WORKDIR /app

# Copy built application
COPY --from=builder --chown=messenger:nodejs /app/dist ./dist
COPY --from=builder --chown=messenger:nodejs /app/node_modules ./node_modules
COPY --from=builder --chown=messenger:nodejs /app/package.json ./package.json

USER messenger

EXPOSE 3000

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:3000/health || exit 1

CMD ["node", "dist/server.js"]
```

### Phase 1 Key Features
- **Complete messaging system** - Direct and group chats (2-100 participants)
- **Real-time translation** - Multiple target languages per user
- **Production database** - PostgreSQL with proper indexing and relationships
- **Caching layer** - Redis for sessions and translation results
- **Async processing** - Redis Streams for translation queue
- **Docker deployment** - Complete containerized environment
- **Load balancing** - NGINX for HTTP and WebSocket traffic
- **Monitoring** - Prometheus metrics and Grafana dashboards
- **Security** - JWT authentication with refresh tokens
- **Scalability** - Horizontal scaling ready with stateless services
  senderId: string;
  text: string;
  originalLanguage: string;
  translation?: string; // Single translation to recipient's target language
  timestamp: Date;
}
```

### Phase 1 API Design
```typescript
// Core messaging operations
POST /api/createChat
{
  "participantId": "user_456" // Create direct chat with this user
}
→ { "chatId": "chat_123" }

POST /api/sendMessage  
{
  "chatId": "chat_123",
  "text": "Hello world"
}
→ { "messageId": "msg_789", "translation": "Hola mundo" }

GET /api/messages/{chatId}
→ [{ "id": "msg_789", "text": "Hello world", "translation": "Hola mundo", ... }]

// Real-time updates via WebSocket
← newMessage
{
  "chatId": "chat_123",
  "messageId": "msg_789", 
  "text": "Hello world",
  "translation": "Hola mundo"
}
```

### Phase 1 Architecture

```mermaid
graph TB
    subgraph "Client Layer"
        UserA[User A - English]
        UserB[User B - Spanish]
    end
    
    subgraph "Single Server"
        ChatServer[Chat Server]
        WebSocket[WebSocket Manager]
    end
    
    subgraph "Services"
        Translation[Translation Service]
    end
    
    subgraph "Storage"
        SQLite[(SQLite Database)]
        Cache[(In-Memory Cache)]
    end
    
    subgraph "External"
        GoogleTranslate[Google Translate API]
    end
    
    UserA <-->|WebSocket| WebSocket
    UserB <-->|WebSocket| WebSocket
    WebSocket --> ChatServer
    
    ChatServer --> Translation
    ChatServer --> SQLite
    ChatServer --> Cache
    
    Translation --> GoogleTranslate
    
    %% Message Flow
    UserA -.->|1. Send "Hello"| ChatServer
    ChatServer -.->|2. Translate to Spanish| Translation
    Translation -.->|3. "Hola"| ChatServer
    ChatServer -.->|4. Store & Forward| UserB
    
    classDef client fill:#e1f5fe
    classDef server fill:#f3e5f5
    classDef storage fill:#e8f5e8
    classDef external fill:#fff3e0
    
    class UserA,UserB client
    class ChatServer,WebSocket,Translation server
    class SQLite,Cache storage
    class GoogleTranslate external
```

### Phase 1 Database Schema
```sql
-- Simple SQLite schema for POC
CREATE TABLE users (
    id TEXT PRIMARY KEY,
    username TEXT UNIQUE,
    native_language TEXT,
    target_language TEXT
);

CREATE TABLE chats (
    id TEXT PRIMARY KEY,
    participant1_id TEXT,
    participant2_id TEXT,
    created_at TIMESTAMP,
    FOREIGN KEY (participant1_id) REFERENCES users(id),
    FOREIGN KEY (participant2_id) REFERENCES users(id)
);

CREATE TABLE messages (
    id TEXT PRIMARY KEY,
    chat_id TEXT,
    sender_id TEXT,
    text TEXT,
    original_language TEXT,
    translation TEXT,
    timestamp TIMESTAMP,
    FOREIGN KEY (chat_id) REFERENCES chats(id),
    FOREIGN KEY (sender_id) REFERENCES users(id)
);
```

### Phase 1 Key Limitations
- **Single server** - No horizontal scaling
- **Direct chats only** - No group messaging
- **No offline support** - Users must be online to receive messages
- **Basic translation** - Single target language per user
- **No persistence guarantees** - Messages may be lost on server restart

---

## Phase 2: Multi-Device & Scaling

## Phase 2: Multi-Device & Scaling

### Phase 2: Multi-Device & Scaling

### Phase 2 Goals
- **Primary:** Horizontal scaling beyond single VPS limitations
- **Secondary:** Geographic distribution and high availability
- **Delivery:** Enterprise-scale messaging infrastructure supporting 100K+ users

### Scaling Beyond Single Appwrite Instance

**Scaling Challenges with Single VPS Appwrite:**
- **Resource limits:** Bounded by single server CPU, RAM, and storage
- **Single point of failure:** VPS downtime affects entire system
- **Database bottleneck:** MariaDB concurrent connection limits
- **Geographic latency:** Single location serves global users
- **Storage constraints:** Limited by VPS disk space

**Phase 2 Scaling Architecture:**

```mermaid
graph TB
    subgraph "Global Load Balancer"
        CloudFlare[CloudFlare CDN + Load Balancer]
    end
    
    subgraph "Region 1 - Primary VPS"
        subgraph "Go Services Cluster"
            GoAPI1[Go API Server 1]
            GoAPI2[Go API Server 2]
            GoWS1[Go WebSocket Server 1]
            GoWS2[Go WebSocket Server 2]
        end
        
        subgraph "Appwrite Services"
            AppwriteAuth[Appwrite Auth]
            AppwriteStorage[Appwrite Storage]
            AppwriteRealtime[Appwrite Realtime]
        end
        
        NGINX1[NGINX Load Balancer]
    end
    
    subgraph "Region 2 - Secondary VPS"
        subgraph "Go Services Cluster 2"
            GoAPI3[Go API Server 3]
            GoAPI4[Go API Server 4]
        end
        
        AppwriteAuth2[Appwrite Auth Replica]
        NGINX2[NGINX Load Balancer]
    end
    
    subgraph "Managed Database Layer"
        PostgresPrimary[(PostgreSQL Primary)]
        PostgresReplica[(PostgreSQL Read Replica)]
        RedisCluster[(Redis Cluster)]
    end
    
    subgraph "External Services"
        S3Storage[(AWS S3 / CloudFlare R2)]
        TranslateAPI[Translation APIs]
    end
    
    %% Traffic flow
    CloudFlare --> NGINX1
    CloudFlare --> NGINX2
    
    NGINX1 --> GoAPI1
    NGINX1 --> GoAPI2
    NGINX1 --> GoWS1
    NGINX1 --> GoWS2
    
    NGINX2 --> GoAPI3
    NGINX2 --> GoAPI4
    
    %% Database connections
    GoAPI1 --> PostgresPrimary
    GoAPI2 --> PostgresPrimary
    GoAPI3 --> PostgresReplica
    GoAPI4 --> PostgresReplica
    
    %% Appwrite integration
    GoAPI1 --> AppwriteAuth
    GoAPI2 --> AppwriteAuth
    GoWS1 --> AppwriteRealtime
    GoWS2 --> AppwriteRealtime
    
    %% Caching
    GoAPI1 --> RedisCluster
    GoAPI2 --> RedisCluster
    GoAPI3 --> RedisCluster
    GoAPI4 --> RedisCluster
    
    %% External services
    GoAPI1 --> S3Storage
    GoAPI1 --> TranslateAPI
    
    classDef go fill:#00ADD8
    classDef appwrite fill:#FD366E
    classDef database fill:#336791
    classDef external fill:#FF9900
    
    class GoAPI1,GoAPI2,GoAPI3,GoAPI4,GoWS1,GoWS2 go
    class AppwriteAuth,AppwriteAuth2,AppwriteStorage,AppwriteRealtime appwrite
    class PostgresPrimary,PostgresReplica,RedisCluster database
    class S3Storage,TranslateAPI external
```

**Migration Strategy from Single VPS:**

**Step 1: Database Migration (Week 1-2)**
```bash
# Migrate from Appwrite MariaDB to managed PostgreSQL
# Export data from Appwrite
appwrite databases export --database-id=main --output=backup.json

# Set up managed PostgreSQL (DigitalOcean, AWS RDS, or Supabase)
# Import data to new PostgreSQL instance
# Update Go services to use PostgreSQL instead of Appwrite DB
```

**Step 2: Horizontal Go Services (Week 3-4)**
```yaml
# Docker Compose for multiple Go instances
version: '3.8'
services:
  go-api-1:
    image: multilingual-messenger/api:latest
    environment:
      - DATABASE_URL=postgresql://user:pass@managed-postgres:5432/messenger
      - REDIS_URL=redis://redis-cluster:6379
      - APPWRITE_ENDPOINT=http://appwrite:80
    
  go-api-2:
    image: multilingual-messenger/api:latest
    environment:
      - DATABASE_URL=postgresql://user:pass@managed-postgres:5432/messenger
      - REDIS_URL=redis://redis-cluster:6379
      
  nginx-lb:
    image: nginx:alpine
    ports:
      - "80:80"
    volumes:
      - ./nginx-lb.conf:/etc/nginx/nginx.conf
```

**Step 3: Geographic Distribution (Week 5-6)**
- **Primary region:** Keep existing VPS with Appwrite
- **Secondary region:** Deploy Go services + Appwrite replica
- **Global CDN:** CloudFlare for static assets and geographic routing
- **Database replication:** PostgreSQL read replicas in each region

**Step 4: Appwrite Scaling Options**

**Option A: Appwrite Cloud (Easiest)**
```typescript
// Switch to Appwrite Cloud for managed scaling
const client = new Client()
  .setEndpoint('https://cloud.appwrite.io/v1')
  .setProject('your-project-id');

// Benefits:
// - Automatic scaling and high availability
// - Global edge locations
// - Managed database and storage
// - Built-in monitoring and backups
```

**Option B: Multiple Appwrite Instances**
```yaml
# Deploy Appwrite in multiple regions
# Region 1 (Primary)
appwrite-primary:
  image: appwrite/appwrite:latest
  environment:
    - _APP_DB_HOST=postgres-primary
    - _APP_REDIS_HOST=redis-primary
    
# Region 2 (Replica)  
appwrite-replica:
  image: appwrite/appwrite:latest
  environment:
    - _APP_DB_HOST=postgres-replica
    - _APP_REDIS_HOST=redis-replica
```

**Option C: Hybrid Approach (Recommended)**
- **Keep Appwrite Auth** for user management (works well at scale)
- **Migrate database** to managed PostgreSQL for better scaling
- **Use external storage** (S3/CloudFlare R2) instead of Appwrite Storage
- **Custom Go WebSocket** servers for messaging (better control)

### Phase 2 Scaling Metrics & Triggers

**When to Scale (Monitoring Thresholds):**
- **CPU usage** > 70% sustained for 10+ minutes
- **Memory usage** > 80% sustained
- **Database connections** > 80% of max connections
- **Response time** > 500ms for 95th percentile
- **Concurrent users** > 5K active connections

### Database Sharding Strategy

**Sharding Approach for Messaging System:**

**1. Shard Key Strategy:**
```go
// Primary: Shard by Chat ID (keeps all messages together)
func GetChatShard(chatID string) int {
    return int(crc32.ChecksumIEEE([]byte(chatID))) % numShards
}

// Secondary: User data sharded by User ID  
func GetUserShard(userID string) int {
    return int(crc32.ChecksumIEEE([]byte(userID))) % numShards
}
```

**Benefits:**
- All messages for a chat stay on same shard (efficient queries)
- Chat history retrieval requires single shard access
- Group operations (add/remove participants) are local to one shard

**2. Data Distribution:**
```sql
-- Shard 0: Chats with hash(chatID) % 4 == 0
CREATE TABLE chats_shard_0 (
    id UUID PRIMARY KEY,
    type VARCHAR(10),
    participants UUID[],
    created_at TIMESTAMP
);

CREATE TABLE messages_shard_0 (
    id UUID PRIMARY KEY,
    chat_id UUID REFERENCES chats_shard_0(id),
    sender_id UUID,
    text TEXT,
    translations JSONB,
    created_at TIMESTAMP
);

-- Shard 1, 2, 3: Same schema, different data
```

**3. Cross-Shard Query Solutions:**

**User's Chats (Cross-Shard Challenge):**
```go
// Problem: User participates in chats across multiple shards
// Solution: Denormalized user_chats table on user's shard

type UserChatService struct {
    userShards map[int]*sql.DB
    chatShards map[int]*sql.DB
}

func (s *UserChatService) GetUserChats(userID string) ([]Chat, error) {
    // 1. Get user's chat list from their shard (fast)
    userShard := s.getUserShard(userID)
    chatRefs, err := s.getUserChatReferences(userShard, userID)
    
    // 2. Group chat IDs by their shards
    shardGroups := make(map[int][]string)
    for _, ref := range chatRefs {
        shardID := s.getChatShardID(ref.ChatID)
        shardGroups[shardID] = append(shardGroups[shardID], ref.ChatID)
    }
    
    // 3. Parallel fetch from relevant shards only
    var allChats []Chat
    for shardID, chatIDs := range shardGroups {
        chatShard := s.chatShards[shardID]
        chats, err := s.getChatsByIDs(chatShard, chatIDs)
        if err != nil {
            return nil, err
        }
        allChats = append(allChats, chats...)
    }
    
    return allChats, nil
}

// Maintain denormalized user_chats table
func (s *UserChatService) AddUserToChat(userID, chatID string) error {
    // 1. Add to chat participants (chat's shard)
    chatShard := s.getChatShard(chatID)
    err := s.addParticipantToChat(chatShard, chatID, userID)
    if err != nil {
        return err
    }
    
    // 2. Add to user's chat list (user's shard) 
    userShard := s.getUserShard(userID)
    return s.addChatToUserList(userShard, userID, chatID)
}
```

**4. Hot Shard Management:**
```go
type ShardMonitor struct {
    metrics map[int]*ShardMetrics
    threshold ShardThresholds
}

type ShardMetrics struct {
    QPS float64
    CPUUsage float64
    MemoryUsage float64
    ConnectionCount int
}

func (m *ShardMonitor) DetectHotShards() []int {
    var hotShards []int
    
    for shardID, metrics := range m.metrics {
        if metrics.QPS > m.threshold.MaxQPS ||
           metrics.CPUUsage > m.threshold.MaxCPU {
            hotShards = append(hotShards, shardID)
        }
    }
    
    return hotShards
}

// Solution: Split hot shards
func (m *ShardManager) SplitShard(hotShardID int) error {
    // 1. Create new shard
    newShardID := m.createNewShard()
    
    // 2. Migrate half the data (by consistent hashing)
    return m.migrateDataToNewShard(hotShardID, newShardID)
}
```

**5. Global Search Index:**
```go
// Separate service for cross-shard search
type SearchIndexService struct {
    indexDB *sql.DB  // Elasticsearch or PostgreSQL with full-text search
    shardRouter *ShardRouter
}

func (s *SearchIndexService) IndexMessage(msg Message) error {
    // Index in separate search database
    doc := SearchDocument{
        MessageID: msg.ID,
        ChatID: msg.ChatID,
        Content: msg.Text,
        Translations: msg.Translations,
        ShardID: s.shardRouter.GetChatShardID(msg.ChatID),
        CreatedAt: msg.CreatedAt,
    }
    
    return s.indexDB.IndexDocument(doc)
}

func (s *SearchIndexService) SearchMessages(userID, query string, limit int) ([]Message, error) {
    // 1. Get user's accessible chats
    userChats, err := s.getUserChats(userID)
    if err != nil {
        return nil, err
    }
    
    // 2. Search index with chat filter
    results, err := s.indexDB.Search(SearchQuery{
        Text: query,
        ChatIDs: extractChatIDs(userChats),
        Limit: limit,
    })
    
    // 3. Fetch full messages from respective shards
    return s.fetchMessagesFromShards(results)
}
```

**6. Sharding Migration Strategy:**
```go
type ShardMigration struct {
    oldTopology map[int]*sql.DB
    newTopology map[int]*sql.DB
    migrationLog *sql.DB
}

func (m *ShardMigration) MigrateWithZeroDowntime() error {
    // 1. Dual-write phase: Write to both old and new shards
    m.enableDualWrite()
    
    // 2. Migrate existing data in background
    go m.migrateHistoricalData()
    
    // 3. Switch reads to new topology gradually
    m.gradualReadMigration()
    
    // 4. Stop dual-write, use new topology only
    return m.completeMigration()
}
```

**Sharding Challenges & Solutions Summary:**

| Challenge | Solution | Implementation |
|-----------|----------|----------------|
| **Cross-shard queries** | Denormalization + lookup tables | user_chats table on user's shard |
| **Hot shards** | Consistent hashing + shard splitting | Monitor QPS, split when threshold exceeded |
| **Global search** | Separate search index service | Elasticsearch/PostgreSQL with full-text |
| **Distributed transactions** | Saga pattern + eventual consistency | Compensating actions for rollback |
| **Shard rebalancing** | Dual-write migration + gradual cutover | Zero-downtime migration strategy |
| **Data locality** | Smart shard key selection | Chat ID sharding keeps messages together |

### Database Sharding Terminology & Concepts Explained

**What is Database Sharding?**
Imagine you have a huge library with millions of books. Instead of putting all books on one giant shelf (which would be slow to search), you split them across multiple smaller shelves:
- Shelf A: Books A-F
- Shelf B: Books G-M  
- Shelf C: Books N-S
- Shelf D: Books T-Z

Database sharding works the same way - split one large database into multiple smaller databases (shards) for better performance.

**What is a Shard Key?**
The shard key is like the filing system that decides which shelf (shard) each book (data) goes to. In our messenger:
- **Chat ID as shard key:** All messages for "study-group" chat go to the same shard
- **User ID as shard key:** All data for user "alice" goes to the same shard

**What is Denormalization?**
In normal database design, you avoid duplicate data. But with sharding, sometimes you intentionally duplicate data for performance.

**Example - Normalized (traditional):**
```sql
-- Users table
users: [alice, bob, charlie]

-- Chats table  
chats: [study-group, work-team, family]

-- Participants table (links users to chats)
participants: [
  (alice, study-group),
  (alice, work-team), 
  (bob, family)
]
```

**Denormalized (for sharding performance):**
```sql
-- User alice's shard contains HER copy of chat info
user_chats_alice: [
  {chat_id: "study-group", shard_location: 0},
  {chat_id: "work-team", shard_location: 2}
]

-- User bob's shard contains HIS copy of chat info  
user_chats_bob: [
  {chat_id: "family", shard_location: 1}
]
```

**Why denormalize?** Instead of searching all shards to find alice's chats, we look at alice's shard only - much faster!

**What is Consistent Hashing?**
Regular hashing has a problem when you add/remove servers:

**Regular Hashing Problem:**
```go
// With 4 servers
server = hash("chat-123") % 4  // chat-123 goes to server 2

// Add 1 server (now 5 total)  
server = hash("chat-123") % 5  // chat-123 now goes to server 3!
// Problem: Data moved! Need to migrate everything
```

**Consistent Hashing Solution:**
Think of servers arranged in a circle (like a clock). Data goes to the next server clockwise:

```
    Server A (12 o'clock)
         |
Server D  |  Server B  
(9)      |      (3)
         |
    Server C (6 o'clock)
```

```go
// Chat "study-group" hashes to 2 o'clock position
// Goes to next server clockwise = Server B

// Add new Server E at 1 o'clock
// "study-group" still goes to Server B (unchanged!)
// Only data between 12-1 o'clock moves to Server E
```

**Benefits:** Adding/removing servers only affects nearby data, not everything!

**What are Virtual Nodes? (Detailed Explanation)**

**The Problem with Basic Consistent Hashing:**

Imagine you have 3 servers arranged on a circle (like a clock):
```
    Server A (12 o'clock)
         |
         |
Server C  |  Server B  
(8)      |      (4)
         |
    (6 o'clock position empty)
```

**Problem 1: Uneven Distribution**
```go
// Data distribution with basic consistent hashing
hash("user-123") = 1 o'clock → goes to Server B (next clockwise)
hash("user-456") = 2 o'clock → goes to Server B  
hash("user-789") = 3 o'clock → goes to Server B
hash("user-abc") = 5 o'clock → goes to Server B
hash("user-def") = 7 o'clock → goes to Server C
hash("user-ghi") = 9 o'clock → goes to Server C
hash("user-jkl") = 11 o'clock → goes to Server A

// Result: Very uneven!
Server A: 1 user (8% of data)
Server B: 4 users (67% of data) ← Overloaded!
Server C: 2 users (25% of data)
```

**Problem 2: Hot Spots**
If many users hash to positions 1-4 o'clock, Server B gets overwhelmed while Server A sits mostly idle.

**Virtual Nodes Solution:**

Instead of each server having 1 position, give each server multiple positions (virtual nodes):

```
// Each server gets 4 virtual positions
Server A virtual nodes: A1(1), A2(5), A3(9), A4(11)  
Server B virtual nodes: B1(2), B2(6), B3(10), B4(12)
Server C virtual nodes: C1(3), C2(7), C3(8), C4(4)

Circle looks like:
    A1  B4  A4
     \  |  /
      \ | /
   C2--+--B1
      / | \
     /  |  \
   C3  A2  B2
      C1
```

**Better Distribution with Virtual Nodes:**
```go
// Same users, but now distributed across virtual nodes
hash("user-123") = 1 o'clock → A1 → Server A
hash("user-456") = 2 o'clock → B1 → Server B  
hash("user-789") = 3 o'clock → C1 → Server C
hash("user-abc") = 5 o'clock → A2 → Server A
hash("user-def") = 7 o'clock → C2 → Server C
hash("user-ghi") = 9 o'clock → A3 → Server A
hash("user-jkl") = 11 o'clock → A4 → Server A

// Much better distribution!
Server A: 4 users (57% of data)
Server B: 1 user (14% of data)  
Server C: 2 users (29% of data)
```

**Real-World Example with More Virtual Nodes:**

```go
// Production setup: 3 servers, 100 virtual nodes each
type VirtualNodeRing struct {
    ring map[uint32]string  // hash position → server name
    servers []string
    virtualNodes int
}

func NewVirtualNodeRing(servers []string, virtualNodes int) *VirtualNodeRing {
    ring := make(map[uint32]string)
    
    for _, server := range servers {
        // Create 100 virtual nodes per server
        for i := 0; i < virtualNodes; i++ {
            // Virtual node identifier: "server-A-vnode-0", "server-A-vnode-1", etc.
            vnodeKey := fmt.Sprintf("%s-vnode-%d", server, i)
            hash := crc32.ChecksumIEEE([]byte(vnodeKey))
            ring[hash] = server
        }
    }
    
    return &VirtualNodeRing{
        ring: ring,
        servers: servers,
        virtualNodes: virtualNodes,
    }
}

// Result: 300 positions total (3 servers × 100 virtual nodes)
// Much more even distribution across the circle!
```

**Distribution Analysis:**
```go
// Test with 10,000 users
func TestDistribution() {
    ring := NewVirtualNodeRing([]string{"server-A", "server-B", "server-C"}, 100)
    
    serverCounts := make(map[string]int)
    
    for i := 0; i < 10000; i++ {
        userID := fmt.Sprintf("user-%d", i)
        server := ring.GetServer(userID)
        serverCounts[server]++
    }
    
    // Results with virtual nodes:
    // server-A: 3,334 users (33.34%)
    // server-B: 3,331 users (33.31%)  
    // server-C: 3,335 users (33.35%)
    // Nearly perfect distribution!
    
    // Without virtual nodes (basic consistent hashing):
    // server-A: 1,245 users (12.45%)
    // server-B: 6,891 users (68.91%) ← Very uneven!
    // server-C: 1,864 users (18.64%)
}
```

**Key Benefits:**
1. **Even Distribution:** ~33.33% per server instead of 12%-69% range
2. **Smooth Scaling:** Adding servers moves only necessary data
3. **Load Balancing:** Hot data spreads across multiple servers
4. **Fault Tolerance:** Removing servers redistributes load evenly

**What is the Saga Pattern?**
When you need to do multiple operations across different databases, but one might fail:

**The Problem (Bank Transfer Example):**
```go
// Transfer $100 from Alice to Bob
// Step 1: Subtract $100 from Alice's account (Bank A)
// Step 2: Add $100 to Bob's account (Bank B)

// What if Step 2 fails? Alice loses money but Bob doesn't get it!
```

**Saga Pattern Solution:**
For each step, define how to undo it:

```go
type BankTransferSaga struct {
    steps []Step
    compensations []UndoStep
}

// Step 1: Subtract from Alice
step1 := func() error { 
    return bankA.Subtract("alice", 100) 
}
undo1 := func() error { 
    return bankA.Add("alice", 100)  // Put money back
}

// Step 2: Add to Bob  
step2 := func() error { 
    return bankB.Add("bob", 100) 
}
undo2 := func() error { 
    return bankB.Subtract("bob", 100)  // Take money back
}

// Execute with automatic rollback
saga.AddStep(step1, undo1)
saga.AddStep(step2, undo2)

err := saga.Execute()
// If step2 fails, automatically runs undo1 to fix Alice's account
```

**What is a Hot Shard?**
When one shard gets way more traffic than others:

**Example:**
```
Shard 0: 100 requests/second (normal)
Shard 1: 150 requests/second (normal)  
Shard 2: 5000 requests/second (HOT! 🔥)
Shard 3: 80 requests/second (normal)
```

**Why does this happen?**
- Celebrity creates popular chat → all messages go to same shard
- Viral content in one chat → everyone messaging at once
- Popular user → all their chats get heavy traffic

**What is Shard Splitting?**
When a shard gets too hot, split it in half:

**Before:**
```
Shard 2: [chat-A, chat-B, chat-C, chat-D] → 5000 QPS
```

**After:**
```  
Shard 2: [chat-A, chat-C] → 2500 QPS
Shard 4: [chat-B, chat-D] → 2500 QPS (new shard)
```

**How to decide what goes where?**
Use the hash function: chats with even hash stay, odd hash moves to new shard.

**What is a Global Secondary Index (GSI)?**
When you need to search by something other than your shard key:

**Example Problem:**
- Data sharded by Chat ID
- But you want to search messages by content: "find all messages containing 'hello'"
- Problem: Content is not the shard key, so you'd have to search ALL shards

**GSI Solution:**
Create a separate "search database" that indexes by content:

```sql
-- Main shards (by chat_id)
shard_0: messages for chats A-F
shard_1: messages for chats G-M
shard_2: messages for chats N-S  
shard_3: messages for chats T-Z

-- Global search index (by content)
search_db: [
  {content: "hello world", message_id: "msg_123", shard: 0},
  {content: "hello there", message_id: "msg_456", shard: 2},
  {content: "world peace", message_id: "msg_789", shard: 1}
]
```

**Search Process:**
1. Search the index: "find messages with 'hello'" → returns msg_123 (shard 0), msg_456 (shard 2)
2. Fetch full messages from only those shards (not all 4!)

**What is Eventual Consistency?**
In distributed systems, sometimes data takes time to sync everywhere:

**Example:**
```
Time 0: Alice sends message "Hello" to group chat
Time 1: Message stored in Chat shard
Time 2: Alice's user shard updated (she has new message)  
Time 3: Bob's user shard updated (he has new message)
Time 4: Charlie's user shard updated (he has new message)
```

Between Time 1-4, the system is "eventually consistent" - it will be consistent eventually, but not immediately everywhere.

**Strong Consistency:** All updates happen at exactly the same time (slower)
**Eventual Consistency:** Updates propagate over time (faster, but temporary inconsistencies)

Now let me add these explanations to the detailed solutions...

**1. Cross-Shard Query Problem & Solution**

**The Problem:**
```go
// User "alice" participates in multiple chats across different shards
// Chat "study-group" → Shard 0 (hash("study-group") % 4 = 0)  
// Chat "work-team" → Shard 2 (hash("work-team") % 4 = 2)
// Chat "family" → Shard 1 (hash("family") % 4 = 1)

// Naive approach: Query ALL shards (expensive!)
func GetUserChats(userID string) ([]Chat, error) {
    var allChats []Chat
    
    // BAD: Hit every shard even if user has no chats there
    for shardID := 0; shardID < 4; shardID++ {
        shard := getShardDB(shardID)
        chats, _ := shard.Query(`
            SELECT * FROM chats 
            WHERE $1 = ANY(participants)
        `, userID)
        allChats = append(allChats, chats...)
    }
    return allChats // Slow: 4 database queries every time!
}
```

**The Solution: Denormalized Lookup Tables**
```go
// Step 1: Create user_chats table on user's shard
// User "alice" → Shard 3 (hash("alice") % 4 = 3)
CREATE TABLE user_chats_shard_3 (
    user_id UUID,
    chat_id UUID,
    chat_shard_id INT,  -- Which shard contains this chat
    joined_at TIMESTAMP,
    last_read_message_id UUID,
    PRIMARY KEY (user_id, chat_id)
);

// Data for alice:
INSERT INTO user_chats_shard_3 VALUES 
('alice', 'study-group', 0, '2024-01-01', 'msg_123'),
('alice', 'work-team', 2, '2024-01-02', 'msg_456'), 
('alice', 'family', 1, '2024-01-03', 'msg_789');

// Step 2: Optimized query (only hits relevant shards)
func GetUserChats(userID string) ([]Chat, error) {
    // 1. Get user's shard and chat references (1 query)
    userShard := getUserShard(userID)
    refs, err := userShard.Query(`
        SELECT chat_id, chat_shard_id FROM user_chats 
        WHERE user_id = $1
    `, userID)
    
    // 2. Group by shard (in-memory operation)
    shardGroups := make(map[int][]string)
    for _, ref := range refs {
        shardGroups[ref.ChatShardID] = append(
            shardGroups[ref.ChatShardID], 
            ref.ChatID
        )
    }
    // Result: {0: ["study-group"], 1: ["family"], 2: ["work-team"]}
    
    // 3. Parallel queries to only relevant shards (3 queries instead of 4)
    var allChats []Chat
    var wg sync.WaitGroup
    
    for shardID, chatIDs := range shardGroups {
        wg.Add(1)
        go func(sID int, cIDs []string) {
            defer wg.Done()
            shard := getShardDB(sID)
            chats, _ := shard.Query(`
                SELECT * FROM chats WHERE id = ANY($1)
            `, pq.Array(cIDs))
            
            mutex.Lock()
            allChats = append(allChats, chats...)
            mutex.Unlock()
        }(shardID, chatIDs)
    }
    
    wg.Wait()
    return allChats, nil
}

// Step 3: Keep lookup table in sync
func AddUserToChat(userID, chatID string) error {
    chatShardID := getChatShardID(chatID)
    userShardID := getUserShardID(userID)
    
    // Transaction 1: Add to chat (chat's shard)
    chatShard := getShardDB(chatShardID)
    err := chatShard.Exec(`
        UPDATE chats SET participants = array_append(participants, $1)
        WHERE id = $2
    `, userID, chatID)
    
    // Transaction 2: Add to user's lookup table (user's shard)
    userShard := getShardDB(userShardID)
    return userShard.Exec(`
        INSERT INTO user_chats (user_id, chat_id, chat_shard_id, joined_at)
        VALUES ($1, $2, $3, NOW())
    `, userID, chatID, chatShardID)
}
```

**Performance Improvement:**
- **Before:** 4 queries to all shards = ~400ms
- **After:** 1 query + 3 parallel queries = ~120ms (3x faster!)

**2. Hot Shard Problem & Solution**

**The Problem:**
```go
// Celebrity user "taylor_swift" creates viral chat
// Chat "taylor-fans" gets 1M messages/hour
// All messages hash to Shard 2: hash("taylor-fans") % 4 = 2
// Shard 2 becomes overloaded while others are idle

// Monitoring shows:
shardMetrics := map[int]ShardMetrics{
    0: {QPS: 100, CPUUsage: 20%},   // Normal
    1: {QPS: 150, CPUUsage: 25%},   // Normal  
    2: {QPS: 5000, CPUUsage: 95%},  // HOT! 🔥
    3: {QPS: 80, CPUUsage: 15%},    // Normal
}
```

**Solution 1: Consistent Hashing with Virtual Nodes**
```go
type ConsistentHashRing struct {
    ring map[uint32]int  // hash position → shard ID
    virtualNodes int     // Multiple positions per shard
}

func NewConsistentHashRing(numShards, virtualNodes int) *ConsistentHashRing {
    ring := make(map[uint32]int)
    
    // Create multiple hash positions for each shard
    for shardID := 0; shardID < numShards; shardID++ {
        for vnode := 0; vnode < virtualNodes; vnode++ {
            // Create virtual node identifier
            vnodeKey := fmt.Sprintf("shard-%d-vnode-%d", shardID, vnode)
            hash := crc32.ChecksumIEEE([]byte(vnodeKey))
            ring[hash] = shardID
        }
    }
    
    return &ConsistentHashRing{ring: ring, virtualNodes: virtualNodes}
}

func (r *ConsistentHashRing) GetShard(key string) int {
    hash := crc32.ChecksumIEEE([]byte(key))
    
    // Find next shard in ring (clockwise)
    for i := hash; ; i++ {
        if shardID, exists := r.ring[i]; exists {
            return shardID
        }
        if i == math.MaxUint32 {
            i = 0  // Wrap around
        }
    }
}

// Better distribution with virtual nodes:
// "taylor-fans" → hash=1234567 → Shard 2
// "taylor-updates" → hash=2345678 → Shard 0  
// "taylor-news" → hash=3456789 → Shard 1
// Load spreads across multiple shards!
```

**Solution 2: Dynamic Shard Splitting**
```go
type HotShardDetector struct {
    metrics map[int]*ShardMetrics
    thresholds ShardThresholds
}

type ShardThresholds struct {
    MaxQPS float64
    MaxCPU float64
    MaxMemory float64
}

func (d *HotShardDetector) MonitorAndSplit() {
    for {
        time.Sleep(30 * time.Second)  // Check every 30s
        
        for shardID, metrics := range d.metrics {
            if d.isHotShard(metrics) {
                log.Printf("Hot shard detected: %d (QPS: %.0f)", shardID, metrics.QPS)
                d.splitShard(shardID)
            }
        }
    }
}

func (d *HotShardDetector) splitShard(hotShardID int) error {
    // 1. Create new shard
    newShardID := d.createNewShard()
    
    // 2. Migrate half the chats to new shard
    hotShard := getShardDB(hotShardID)
    newShard := getShardDB(newShardID)
    
    // Get all chats from hot shard
    chats, err := hotShard.Query("SELECT id, participants FROM chats")
    if err != nil {
        return err
    }
    
    // Migrate chats with odd hash to new shard
    for _, chat := range chats {
        newHash := crc32.ChecksumIEEE([]byte(chat.ID))
        if newHash%2 == 1 {  // Migrate ~50% of chats
            // Move chat and all its messages
            d.migrateChatToShard(chat.ID, hotShardID, newShardID)
        }
    }
    
    // 3. Update routing table
    d.updateShardRouting(hotShardID, newShardID)
    
    log.Printf("Split shard %d → created shard %d", hotShardID, newShardID)
    return nil
}

// Result: Load distributed across more shards
// Before: Shard 2 = 5000 QPS
// After: Shard 2 = 2500 QPS, Shard 4 = 2500 QPS
```

**3. Global Search Problem & Solution**

**The Problem:**
```go
// User searches "hello world" across all their chats
// Messages scattered across all 4 shards:
// Shard 0: "hello world from alice" (study-group chat)
// Shard 1: "world hello from bob" (family chat)  
// Shard 2: "hello there world" (work-team chat)
// Shard 3: No matches

// Naive approach: Search all shards
func SearchMessages(userID, query string) ([]Message, error) {
    var allResults []Message
    
    // BAD: Query every shard with full-text search (very slow!)
    for shardID := 0; shardID < 4; shardID++ {
        shard := getShardDB(shardID)
        results, _ := shard.Query(`
            SELECT * FROM messages 
            WHERE to_tsvector('english', text) @@ plainto_tsquery('english', $1)
            AND chat_id IN (SELECT chat_id FROM user_chats WHERE user_id = $2)
        `, query, userID)
        allResults = append(allResults, results...)
    }
    return allResults  // Slow: 4 full-text searches!
}
```

**The Solution: Dedicated Search Index Service**
```go
// Step 1: Separate search service with Elasticsearch/PostgreSQL
type SearchIndexService struct {
    searchDB *elasticsearch.Client  // Or PostgreSQL with full-text
    shardRouter *ShardRouter
}

// Step 2: Index messages as they're created
func (s *MessageService) SendMessage(chatID, userID, text string) error {
    // 1. Store message in appropriate shard
    shard := s.router.GetChatShard(chatID)
    messageID := uuid.New().String()
    
    err := shard.Exec(`
        INSERT INTO messages (id, chat_id, sender_id, text, created_at)
        VALUES ($1, $2, $3, $4, NOW())
    `, messageID, chatID, userID, text)
    
    // 2. Index in search service (async)
    go s.searchIndex.IndexMessage(SearchDocument{
        MessageID: messageID,
        ChatID: chatID,
        SenderID: userID,
        Text: text,
        ShardID: s.router.GetChatShardID(chatID),
        CreatedAt: time.Now(),
    })
    
    return err
}

// Step 3: Fast search with shard routing
func (s *SearchIndexService) SearchMessages(userID, query string, limit int) ([]Message, error) {
    // 1. Get user's accessible chats (from user's shard)
    userChats, err := s.getUserChats(userID)
    if err != nil {
        return nil, err
    }
    chatIDs := extractChatIDs(userChats)
    
    // 2. Search index (single fast query)
    searchResults, err := s.searchDB.Search(elasticsearch.SearchRequest{
        Index: "messages",
        Body: map[string]interface{}{
            "query": map[string]interface{}{
                "bool": map[string]interface{}{
                    "must": []map[string]interface{}{
                        {"match": map[string]interface{}{"text": query}},
                        {"terms": map[string]interface{}{"chat_id": chatIDs}},
                    },
                },
            },
            "size": limit,
            "_source": []string{"message_id", "shard_id"},
        },
    })
    
    // 3. Group results by shard
    shardGroups := make(map[int][]string)
    for _, hit := range searchResults.Hits.Hits {
        messageID := hit.Source["message_id"].(string)
        shardID := int(hit.Source["shard_id"].(float64))
        shardGroups[shardID] = append(shardGroups[shardID], messageID)
    }
    
    // 4. Fetch full messages from relevant shards (parallel)
    var allMessages []Message
    var wg sync.WaitGroup
    
    for shardID, messageIDs := range shardGroups {
        wg.Add(1)
        go func(sID int, mIDs []string) {
            defer wg.Done()
            shard := getShardDB(sID)
            messages, _ := shard.Query(`
                SELECT * FROM messages WHERE id = ANY($1)
            `, pq.Array(mIDs))
            
            mutex.Lock()
            allMessages = append(allMessages, messages...)
            mutex.Unlock()
        }(shardID, messageIDs)
    }
    
    wg.Wait()
    return allMessages, nil
}
```

**Performance Improvement:**
- **Before:** 4 full-text searches across shards = ~2000ms
- **After:** 1 index search + targeted shard queries = ~150ms (13x faster!)

**4. Distributed Transaction Problem & Solution**

**The Problem:**
```go
// Creating group chat "project-alpha" with users from different shards:
// alice (Shard 3), bob (Shard 1), charlie (Shard 0)
// Chat itself goes to Shard 2: hash("project-alpha") % 4 = 2

// Need ACID guarantees across 4 different shards!
func CreateGroupChat(participants []string, name string) error {
    chatID := uuid.New().String()
    chatShardID := getChatShardID(chatID)
    
    // This approach can fail partially! ❌
    // 1. Create chat record (Shard 2)
    chatShard := getShardDB(chatShardID)
    err := chatShard.Exec(`
        INSERT INTO chats (id, name, participants, created_at)
        VALUES ($1, $2, $3, NOW())
    `, chatID, name, pq.Array(participants))
    
    // 2. Add to each user's chat list (different shards)
    for _, userID := range participants {
        userShard := getShardDB(getUserShardID(userID))
        err := userShard.Exec(`
            INSERT INTO user_chats (user_id, chat_id, chat_shard_id)
            VALUES ($1, $2, $3)
        `, userID, chatID, chatShardID)
        
        if err != nil {
            // PROBLEM: Chat created but some users not added!
            // Inconsistent state across shards
            return err
        }
    }
    
    return nil
}
```

**The Solution: Saga Pattern with Compensations**
```go
type Saga struct {
    steps []SagaStep
    compensations []CompensationStep
    executedSteps []int
}

type SagaStep func() error
type CompensationStep func() error

func (s *Saga) AddStep(step SagaStep, compensation CompensationStep) {
    s.steps = append(s.steps, step)
    s.compensations = append(s.compensations, compensation)
}

func (s *Saga) Execute() error {
    // Execute steps one by one
    for i, step := range s.steps {
        err := step()
        if err != nil {
            // Rollback all executed steps
            s.rollback()
            return fmt.Errorf("saga failed at step %d: %w", i, err)
        }
        s.executedSteps = append(s.executedSteps, i)
    }
    return nil
}

func (s *Saga) rollback() {
    // Execute compensations in reverse order
    for i := len(s.executedSteps) - 1; i >= 0; i-- {
        stepIndex := s.executedSteps[i]
        compensation := s.compensations[stepIndex]
        
        err := compensation()
        if err != nil {
            log.Printf("Compensation failed for step %d: %v", stepIndex, err)
            // Log for manual intervention
        }
    }
}

// Improved group chat creation with saga
func CreateGroupChatSaga(participants []string, name string) error {
    chatID := uuid.New().String()
    chatShardID := getChatShardID(chatID)
    saga := &Saga{}
    
    // Step 1: Create chat record
    saga.AddStep(
        func() error {
            chatShard := getShardDB(chatShardID)
            return chatShard.Exec(`
                INSERT INTO chats (id, name, participants, created_at)
                VALUES ($1, $2, $3, NOW())
            `, chatID, name, pq.Array(participants))
        },
        func() error {  // Compensation: Delete chat
            chatShard := getShardDB(chatShardID)
            return chatShard.Exec("DELETE FROM chats WHERE id = $1", chatID)
        },
    )
    
    // Step 2-N: Add each user to chat
    for _, userID := range participants {
        userID := userID  // Capture for closure
        saga.AddStep(
            func() error {
                userShard := getShardDB(getUserShardID(userID))
                return userShard.Exec(`
                    INSERT INTO user_chats (user_id, chat_id, chat_shard_id, joined_at)
                    VALUES ($1, $2, $3, NOW())
                `, userID, chatID, chatShardID)
            },
            func() error {  // Compensation: Remove user from chat
                userShard := getShardDB(getUserShardID(userID))
                return userShard.Exec(`
                    DELETE FROM user_chats 
                    WHERE user_id = $1 AND chat_id = $2
                `, userID, chatID)
            },
        )
    }
    
    // Execute saga with automatic rollback on failure
    return saga.Execute()
}
```

**Example Execution:**
```
✅ Step 1: Create chat "project-alpha" in Shard 2
✅ Step 2: Add alice to chat (Shard 3)  
✅ Step 3: Add bob to chat (Shard 1)
❌ Step 4: Add charlie to chat (Shard 0) - FAILS!

🔄 Rollback:
✅ Compensation 3: Remove bob from chat (Shard 1)
✅ Compensation 2: Remove alice from chat (Shard 3)  
✅ Compensation 1: Delete chat "project-alpha" (Shard 2)

Result: Clean rollback, no inconsistent state! ✨
```

These detailed solutions ensure your messaging system can scale from single VPS to enterprise level while maintaining data consistency, performance, and reliability across all shards.

**Production Kubernetes Architecture:**
```yaml
# Namespace
apiVersion: v1
kind: Namespace
metadata:
  name: multilingual-messenger

---
# ConfigMap for application configuration
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
  namespace: multilingual-messenger
data:
  NODE_ENV: "production"
  REDIS_URL: "redis://redis-service:6379"
  DATABASE_URL: "postgresql://messenger:${DB_PASSWORD}@postgres-service:5432/messenger_prod"

---
# API Server Deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api-server
  namespace: multilingual-messenger
spec:
  replicas: 3
  selector:
    matchLabels:
      app: api-server
  template:
    metadata:
      labels:
        app: api-server
    spec:
      containers:
      - name: api-server
        image: multilingual-messenger/api:latest
        ports:
        - containerPort: 3000
        envFrom:
        - configMapRef:
            name: app-config
        env:
        - name: JWT_SECRET
          valueFrom:
            secretKeyRef:
              name: app-secrets
              key: jwt-secret
        - name: GOOGLE_TRANSLATE_API_KEY
          valueFrom:
            secretKeyRef:
              name: app-secrets
              key: google-translate-key
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 3000
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 3000
          initialDelaySeconds: 5
          periodSeconds: 5

---
# WebSocket Server Deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: websocket-server
  namespace: multilingual-messenger
spec:
  replicas: 3
  selector:
    matchLabels:
      app: websocket-server
  template:
    metadata:
      labels:
        app: websocket-server
    spec:
      containers:
      - name: websocket-server
        image: multilingual-messenger/websocket:latest
        ports:
        - containerPort: 3001
        envFrom:
        - configMapRef:
            name: app-config
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"

---
# PostgreSQL StatefulSet
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: postgres
  namespace: multilingual-messenger
spec:
  serviceName: postgres-service
  replicas: 1
  selector:
    matchLabels:
      app: postgres
  template:
    metadata:
      labels:
        app: postgres
    spec:
      containers:
      - name: postgres
        image: postgres:15-alpine
        ports:
        - containerPort: 5432
        env:
        - name: POSTGRES_DB
          value: "messenger_prod"
        - name: POSTGRES_USER
          value: "messenger"
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: postgres-secret
              key: password
        volumeMounts:
        - name: postgres-storage
          mountPath: /var/lib/postgresql/data
        resources:
          requests:
            memory: "1Gi"
            cpu: "500m"
          limits:
            memory: "2Gi"
            cpu: "1000m"
  volumeClaimTemplates:
  - metadata:
      name: postgres-storage
    spec:
      accessModes: ["ReadWriteOnce"]
      resources:
        requests:
          storage: 20Gi

---
# Redis Deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: redis
  namespace: multilingual-messenger
spec:
  replicas: 1
  selector:
    matchLabels:
      app: redis
  template:
    metadata:
      labels:
        app: redis
    spec:
      containers:
      - name: redis
        image: redis:7-alpine
        ports:
        - containerPort: 6379
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"

---
# Load Balancer Service
apiVersion: v1
kind: Service
metadata:
  name: api-service
  namespace: multilingual-messenger
spec:
  selector:
    app: api-server
  ports:
  - port: 80
    targetPort: 3000
  type: LoadBalancer

---
# WebSocket Service
apiVersion: v1
kind: Service
metadata:
  name: websocket-service
  namespace: multilingual-messenger
spec:
  selector:
    app: websocket-server
  ports:
  - port: 3001
    targetPort: 3001
  type: LoadBalancer

---
# Horizontal Pod Autoscaler
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: api-server-hpa
  namespace: multilingual-messenger
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: api-server
  minReplicas: 3
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```
- **Primary:** Production-ready architecture with multi-device support
- **Secondary:** Horizontal scaling and offline message delivery
- **Delivery:** WhatsApp-scale messaging infrastructure

### Phase 3 Complete Entities
```typescript
interface User {
  id: string;
  username: string;
  displayName: string;
  email: string;
  nativeLanguage: string;
  targetLanguages: string[];
  privacySettings: {
    messageRetentionDays: number;
  };
  createdAt: Date;
  lastActiveAt: Date;
}

interface Client {
  id: string;
  userId: string;
  deviceType: 'mobile' | 'web' | 'desktop';
  deviceInfo: {
    platform: string;
    version: string;
  };
  connectionStatus: 'online' | 'offline';
  lastActive: Date;
}

interface Chat {
  id: string;
  type: 'direct' | 'group';
  participants: string[];
  name?: string;
  createdBy: string;
  settings: {
    translationEnabled: boolean;
  };
  createdAt: Date;
}

interface Message {
  id: string;
  chatId: string;
  senderId: string;
  text: string;
  originalLanguage: string;
  translations: Map<string, string>;
  timestamp: Date;
}

interface InboxEntry {
  clientId: string;
  messageId: string;
  chatId: string;
  deliveryAttempts: number;
  createdAt: Date;
  ttl: Date; // 30 days
}
```

### Phase 3 Production API
```typescript
// Multi-device aware operations
POST /api/auth/login
{
  "username": "john_doe",
  "password": "...",
  "deviceInfo": {
    "type": "mobile",
    "platform": "iOS 17.0"
  }
}
→ { 
  "token": "jwt_token",
  "clientId": "client_123",
  "userId": "user_456"
}

// Enhanced messaging with client tracking
POST /api/sendMessage
{
  "chatId": "chat_123",
  "text": "Hello from mobile"
}
→ { "messageId": "msg_789", "status": "queued" }

// WebSocket connection per client
WebSocket /ws/{clientId}
← newMessage
{
  "messageId": "msg_789",
  "chatId": "chat_123", 
  "text": "Hello from mobile",
  "translations": { "es": "Hola desde móvil" }
}
→ "RECEIVED" // Client acknowledgment

// Offline message sync on reconnect
GET /api/sync/{clientId}
→ [
  { "messageId": "msg_790", "chatId": "chat_123", ... },
  { "messageId": "msg_791", "chatId": "chat_456", ... }
]
```

### Phase 3 Production Architecture

```mermaid
graph TB
    subgraph "Multi-Device Clients"
        Mobile1[User A Mobile]
        Desktop1[User A Desktop]
        Mobile2[User B Mobile]
        Web2[User B Web]
    end
    
    subgraph "Load Balancing"
        LB[Load Balancer]
        Gateway[API Gateway]
    end
    
    subgraph "Chat Server Cluster"
        ChatServer1[Chat Server 1]
        ChatServer2[Chat Server 2]
        ChatServer3[Chat Server 3]
    end
    
    subgraph "Message Routing"
        PubSub[Redis Pub/Sub]
        MessageQueue[Message Queue Cluster]
    end
    
    subgraph "Services Cluster"
        Translation1[Translation Service 1]
        Translation2[Translation Service 2]
        Auth[Auth Service]
    end
    
    subgraph "Database Cluster - Sharded"
        subgraph "Shard 1"
            ChatDB1[(Chat/Message DB 1)]
            InboxDB1[(Inbox DB 1)]
        end
        subgraph "Shard 2"  
            ChatDB2[(Chat/Message DB 2)]
            InboxDB2[(Inbox DB 2)]
        end
        UserDB[(User/Client DB)]
        RedisCluster[(Redis Cache Cluster)]
    end
    
    subgraph "External Services"
        TranslateAPI[Translation API]
        Monitoring[Monitoring & Alerts]
    end
    
    %% Client Connections
    Mobile1 -.->|WebSocket| LB
    Desktop1 -.->|WebSocket| LB
    Mobile2 -.->|WebSocket| LB
    Web2 -.->|WebSocket| LB
    
    LB --> Gateway
    Gateway --> ChatServer1
    Gateway --> ChatServer2
    Gateway --> ChatServer3
    
    %% Pub/Sub Architecture
    ChatServer1 <-->|Subscribe user:A| PubSub
    ChatServer2 <-->|Subscribe user:A| PubSub
    ChatServer3 <-->|Subscribe user:B| PubSub
    
    %% Message Processing
    ChatServer1 --> MessageQueue
    ChatServer2 --> MessageQueue
    MessageQueue --> Translation1
    MessageQueue --> Translation2
    
    %% Database Sharding
    ChatServer1 --> ChatDB1
    ChatServer1 --> InboxDB1
    ChatServer2 --> ChatDB2
    ChatServer2 --> InboxDB2
    ChatServer3 --> UserDB
    
    %% Caching
    Translation1 --> RedisCluster
    Translation2 --> RedisCluster
    ChatServer1 --> RedisCluster
    
    %% External Services
    Translation1 --> TranslateAPI
    Translation2 --> TranslateAPI
    ChatServer1 --> Monitoring
    
    %% Multi-Device Flow Example
    Mobile1 -.->|1. Send Message| ChatServer1
    ChatServer1 -.->|2. Store + Publish| PubSub
    PubSub -.->|3. Route to user:A| ChatServer2
    PubSub -.->|3. Route to user:B| ChatServer3
    ChatServer2 -.->|4. Deliver| Desktop1
    ChatServer3 -.->|4. Deliver| Mobile2
    ChatServer3 -.->|4. Queue for offline| InboxDB2
    
    classDef client fill:#e1f5fe
    classDef server fill:#f3e5f5
    classDef storage fill:#e8f5e8
    classDef external fill:#fff3e0
    
    class Mobile1,Desktop1,Mobile2,Web2 client
    class ChatServer1,ChatServer2,ChatServer3,PubSub,MessageQueue,Translation1,Translation2,Auth server
    class ChatDB1,ChatDB2,InboxDB1,InboxDB2,UserDB,RedisCluster storage
    class TranslateAPI,Monitoring external
```

### Phase 3 Production Database Schema
```sql
-- Sharded PostgreSQL with proper indexing
CREATE TABLE users (
    id UUID PRIMARY KEY,
    username VARCHAR(50) UNIQUE,
    display_name VARCHAR(100),
    email VARCHAR(255) UNIQUE,
    native_language VARCHAR(10),
    target_languages TEXT[],
    privacy_settings JSONB,
    created_at TIMESTAMP DEFAULT NOW(),
    last_active_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE clients (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    device_type VARCHAR(20),
    device_info JSONB,
    connection_status VARCHAR(20) DEFAULT 'offline',
    last_active TIMESTAMP DEFAULT NOW(),
    created_at TIMESTAMP DEFAULT NOW()
);

-- Sharded by chat_id
CREATE TABLE chats (
    id UUID PRIMARY KEY,
    type VARCHAR(10),
    participants UUID[],
    name VARCHAR(100),
    created_by UUID,
    settings JSONB,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE chat_participants (
    chat_id UUID,
    user_id UUID,
    joined_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (chat_id, user_id)
);

-- Sharded by chat_id  
CREATE TABLE messages (
    id UUID PRIMARY KEY,
    chat_id UUID,
    sender_id UUID,
    text TEXT,
    original_language VARCHAR(10),
    translations JSONB,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Per-client message delivery tracking
CREATE TABLE inbox (
    client_id UUID,
    message_id UUID,
    chat_id UUID,
    delivery_attempts INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    ttl TIMESTAMP DEFAULT (NOW() + INTERVAL '30 days'),
    PRIMARY KEY (client_id, message_id)
);

-- Performance indexes
CREATE INDEX idx_clients_user_status ON clients(user_id, connection_status);
CREATE INDEX idx_inbox_client_pending ON inbox(client_id, created_at) WHERE ttl > NOW();
CREATE INDEX idx_messages_chat_time ON messages(chat_id, created_at);
CREATE INDEX idx_chat_participants_user ON chat_participants(user_id);
```

### Phase 3 Key Features
- **Multi-device support** - Up to 3 clients per user
- **Horizontal scaling** - Stateless Chat Server cluster
- **Redis Pub/Sub routing** - Efficient message distribution
- **Offline message delivery** - 30-day message retention
- **Database sharding** - Scalable data distribution
- **Delivery guarantees** - Per-client acknowledgment tracking
- **Production monitoring** - Comprehensive observability

---

## Phase 3: Advanced Features

### Phase 3 Goals
- **Primary:** Add language learning and advanced communication features
- **Secondary:** Voice calls, grammar analysis, vocabulary management
- **Delivery:** Full-featured multilingual communication platform

### Phase 3 Advanced Entities
```typescript
// Enhanced with learning features
interface User {
  id: string;
  username: string;
  displayName: string;
  email: string;
  nativeLanguage: string;
  targetLanguages: string[];
  learningSettings: {
    grammarEnabled: boolean;
    vocabularyEnabled: boolean;
    difficultyLevel: 'beginner' | 'intermediate' | 'advanced';
  };
  privacySettings: {
    transcriptRecording: boolean;
    messageRetentionDays: number;
  };
  createdAt: Date;
  lastActiveAt: Date;
}

interface Message {
  id: string;
  chatId: string;
  senderId: string;
  text: string;
  originalLanguage: string;
  translations: Map<string, string>;
  grammarAnalysis?: {
    difficulty: string; // CEFR level
    patterns: string[];
    explanations: string[];
  };
  timestamp: Date;
}

interface VocabularyEntry {
  id: string;
  userId: string;
  term: string;
  language: string;
  translation: string;
  definition: string;
  context: {
    messageId: string;
    sentence: string;
  };
  learningData: {
    reviewCount: number;
    correctCount: number;
    nextReview: Date;
    interval: number;
  };
  createdAt: Date;
}

interface CallSession {
  id: string;
  chatId: string;
  participants: string[];
  type: 'audio' | 'video';
  status: 'active' | 'ended';
  startedAt: Date;
  endedAt?: Date;
}

interface CallTranscript {
  id: string;
  callId: string;
  segments: {
    speakerId: string;
    startTime: number;
    endTime: number;
    originalText: string;
    originalLanguage: string;
    translations: Map<string, string>;
  }[];
  createdAt: Date;
}
```

### Phase 3 Advanced API
```typescript
// Grammar analysis
POST /api/analyzeGrammar
{
  "messageId": "msg_123",
  "targetLanguage": "es"
}
→ {
  "difficulty": "B1",
  "patterns": ["present_tense", "direct_object"],
  "explanations": ["This sentence uses present tense...", ...]
}

// Vocabulary management
POST /api/vocabulary/save
{
  "messageId": "msg_123",
  "term": "hola",
  "language": "es"
}
→ { "vocabularyId": "vocab_456" }

GET /api/vocabulary/due
→ [
  { "term": "hola", "translation": "hello", "nextReview": "2024-01-15" }
]

// Voice calls with real-time subtitles
POST /api/calls/initiate
{
  "chatId": "chat_123",
  "type": "video"
}
→ { "callId": "call_789", "signalingToken": "..." }

WebSocket /calls/{callId}/subtitles
← subtitle
{
  "speakerId": "user_123",
  "originalText": "Hello everyone",
  "originalLanguage": "en", 
  "translations": { "es": "Hola a todos" }
}
```

### Phase 3 Complete Architecture

```mermaid
graph TB
    subgraph "Multi-Modal Clients"
        Mobile1[Mobile + Voice]
        Desktop1[Desktop + Video]
        Web2[Web + Learning]
    end
    
    subgraph "Gateway Layer"
        LB[Load Balancer]
        Gateway[API Gateway]
        MediaGW[Media Gateway]
    end
    
    subgraph "Core Services"
        ChatCluster[Chat Server Cluster]
        CallCluster[Call Server Cluster]
        AuthService[Auth Service]
    end
    
    subgraph "Language Services"
        TranslationCluster[Translation Service Cluster]
        GrammarEngine[Grammar Analysis Engine]
        VocabManager[Vocabulary Manager]
        STTService[Speech-to-Text Service]
    end
    
    subgraph "Real-Time Infrastructure"
        PubSub[Redis Pub/Sub Cluster]
        MessageQueue[Message Queue Cluster]
        WebRTC[WebRTC Signaling]
        STUNTurn[STUN/TURN Servers]
    end
    
    subgraph "Data Layer"
        subgraph "Chat Data - Sharded"
            ChatDB1[(Chat/Message Shard 1)]
            ChatDB2[(Chat/Message Shard 2)]
        end
        subgraph "Learning Data"
            VocabDB[(Vocabulary DB)]
            ProgressDB[(Learning Progress DB)]
        end
        subgraph "Call Data"
            CallDB[(Call Sessions DB)]
            TranscriptDB[(Transcript DB)]
        end
        UserDB[(User/Client DB)]
        CacheCluster[(Redis Cache Cluster)]
    end
    
    subgraph "External Services"
        TranslateAPI[Translation APIs]
        STTAPI[Speech-to-Text APIs]
        TTSAPI[Text-to-Speech APIs]
        CDN[Media CDN]
    end
    
    %% Client connections
    Mobile1 <--> LB
    Desktop1 <--> LB
    Web2 <--> LB
    
    LB --> Gateway
    LB --> MediaGW
    
    %% Service routing
    Gateway --> ChatCluster
    Gateway --> AuthService
    MediaGW --> CallCluster
    
    %% Real-time messaging
    ChatCluster <--> PubSub
    ChatCluster --> MessageQueue
    MessageQueue --> TranslationCluster
    MessageQueue --> GrammarEngine
    
    %% Voice/Video calls
    CallCluster --> WebRTC
    CallCluster --> STUNTurn
    CallCluster --> STTService
    STTService --> TranslationCluster
    
    %% Learning features
    TranslationCluster --> GrammarEngine
    GrammarEngine --> VocabManager
    VocabManager --> VocabDB
    VocabManager --> ProgressDB
    
    %% Data access
    ChatCluster --> ChatDB1
    ChatCluster --> ChatDB2
    CallCluster --> CallDB
    CallCluster --> TranscriptDB
    AuthService --> UserDB
    
    %% Caching
    TranslationCluster --> CacheCluster
    GrammarEngine --> CacheCluster
    VocabManager --> CacheCluster
    
    %% External APIs
    TranslationCluster --> TranslateAPI
    STTService --> STTAPI
    CallCluster --> TTSAPI
    CallCluster --> CDN
    
    classDef client fill:#e1f5fe
    classDef server fill:#f3e5f5
    classDef storage fill:#e8f5e8
    classDef external fill:#fff3e0
    
    class Mobile1,Desktop1,Web2 client
    class ChatCluster,CallCluster,AuthService,TranslationCluster,GrammarEngine,VocabManager,STTService server
    class ChatDB1,ChatDB2,VocabDB,ProgressDB,CallDB,TranscriptDB,UserDB,CacheCluster storage
    class TranslateAPI,STTAPI,TTSAPI,CDN external
```

### Phase 3 Complete Feature Set
- **Advanced language learning** - Grammar analysis, vocabulary management, spaced repetition
- **Voice/Video calls** - WebRTC with real-time translation subtitles
- **Call transcription** - Speech-to-text with translation and storage
- **Learning analytics** - Progress tracking and personalized recommendations
- **Advanced search** - Full-text search across messages and transcripts
- **Privacy controls** - Granular data retention and sharing settings

---

## Evolution Summary

| Phase | Duration | Core Focus | Key Technologies | Complexity Level |
|-------|----------|------------|------------------|------------------|
| **Phase 1** | 4-5 weeks | Simple messaging + translation | Go + Appwrite, React (built-in state), Google Translate | **Simple** - Single VPS, in-memory cache |
| **Phase 2** | 4-5 weeks | Multi-device + horizontal scaling | Redis, advanced state management, rate limiting | **Moderate** - Multiple instances, caching |
| **Phase 3** | 5-6 weeks | Advanced features + enterprise scale | Desktop app, ML services, database sharding | **Complex** - Full enterprise features |

**Phase 1 Simplifications:**
- **No Redis:** Simple in-memory caching in Go
- **No complex state management:** React's built-in useState/useContext
- **No rate limiting:** Basic throttling in Go (move advanced rate limiting to Phase 2)
- **No desktop app:** Focus on web + mobile first
- **No advanced UI libraries:** Tailwind CSS + basic React components
- **No complex deployment:** Single Docker container + Appwrite

**Translation Processing (All On-VPS):**
```
User sends message → Go service on VPS → Google Translate API → Cache result → Broadcast to recipients
```

**Benefits of Simplified Phase 1:**
- **Faster development:** 4-5 weeks instead of 6-8 weeks
- **Easier debugging:** Less moving parts, simpler architecture
- **Lower complexity:** Junior developers can contribute immediately
- **Rapid iteration:** Quick feedback loop for core features
- **Cost effective:** $0 infrastructure + minimal translation API usage

**What Gets Added in Later Phases:**
- **Phase 2:** Redis caching, advanced state management, rate limiting, horizontal scaling
- **Phase 3:** Desktop app, database sharding, ML features, enterprise monitoring

This iterative approach ensures:
- **MVP in 4-5 weeks** with core messaging + translation working
- **No premature optimization** - add complexity only when needed
- **User feedback early** to guide feature development
- **Maintainable codebase** that scales with team and user growth

## Tech Stack Summary

### Backend Technologies
- **Runtime:** Go 1.21+ for high performance and low memory usage
- **Framework:** Gin for fast HTTP routing and middleware
- **Database:** Appwrite Database (self-hosted on your VPS)
- **Authentication:** Appwrite Auth with JWT and OAuth providers
- **Real-time:** Appwrite Realtime (WebSocket) - no Socket.io needed
- **File Storage:** Appwrite Storage for media and attachments
- **Cache:** Redis 7+ for translation caching
- **Message Queue:** Go channels + Redis for async processing
- **Testing:** Go standard testing + testify for assertions

### Frontend Technologies

**Web Application:**
- **Framework:** React 18+ with TypeScript
- **State Management:** Zustand + TanStack Query for optimal performance
- **UI Library:** Tailwind CSS + Headless UI for modern design
- **Appwrite SDK:** Appwrite Web SDK for backend integration
- **Build Tool:** Vite for fast development and optimized builds
- **Real-time:** Appwrite Realtime (no Socket.io needed)
- **Forms:** React Hook Form + Zod validation
- **Testing:** Vitest + React Testing Library

**Mobile Application:**
- **Framework:** React Native 0.72+ with TypeScript
- **Navigation:** React Navigation v6
- **State Management:** Zustand + TanStack Query (shared with web)
- **UI Components:** NativeBase or Tamagui for cross-platform design
- **Appwrite SDK:** Appwrite React Native SDK
- **Storage:** Appwrite Storage + AsyncStorage for offline
- **Push Notifications:** Appwrite Messaging or FCM
- **Testing:** Detox for E2E, Jest for unit tests

**Desktop Application:**
- **Framework:** Tauri (Rust-based, 90% smaller than Electron)
- **Frontend:** Same React components as web application
- **Appwrite Integration:** Appwrite Web SDK works in Tauri
- **Performance:** Much faster and more secure than Electron

### Infrastructure & Deployment
- **Containerization:** Docker with multi-stage builds
- **Orchestration:** Docker Compose (dev) → Kubernetes (production)
- **Load Balancing:** NGINX for HTTP/WebSocket traffic
- **SSL/TLS:** Let's Encrypt with automatic renewal
- **Monitoring:** Prometheus + Grafana + Sentry error tracking
- **CI/CD:** GitHub Actions with automated testing and deployment

### External Services & Costs
- **Translation:** Google Translate API (~$20/1M characters), DeepL (fallback)
- **Language Detection:** Google Cloud Translation API
- **Push Notifications:** Firebase Cloud Messaging (free tier generous)
- **Self-hosted Backend:** Appwrite on your VPS (already installed - $0 cost)
- **Total Infrastructure Cost:** $0 (self-hosted) + translation API usage only
- **Socket.io:** Not needed - Appwrite Realtime provides WebSocket functionality

### Development Tools
- **Code Quality:** ESLint + Prettier + Husky git hooks
- **API Documentation:** OpenAPI 3.0 with Swagger UI
- **Package Manager:** npm with package-lock.json
- **Environment Management:** dotenv for configuration
- **Database Migrations:** Prisma migrations for schema changes

```mermaid
graph TB
    subgraph "Client Layer"
        Mobile1[Mobile Client 1]
        Mobile2[Mobile Client 2] 
        Web1[Web Client 1]
        Desktop1[Desktop Client 1]
    end
    
    subgraph "Load Balancing"
        LB[Load Balancer]
    end
    
    subgraph "Chat Server Cluster"
        ChatServer1[Chat Server 1]
        ChatServer2[Chat Server 2]
        ChatServer3[Chat Server 3]
    end
    
    subgraph "Message Routing"
        PubSub[Redis Pub/Sub]
        Queue[Message Queue]
    end
    
    subgraph "Language Services"
        Translation[Translation Service]
        Grammar[Grammar Engine]
        Vocabulary[Vocabulary Manager]
    end
    
    subgraph "Media Services"
        MediaAPI[Media Upload API]
        CDN[Content Delivery Network]
        MediaStore[(Media Storage)]
    end
    
    subgraph "Database Layer - Sharded"
        subgraph "Shard 1"
            ChatDB1[(Chat Tables)]
            MessageDB1[(Message Tables)]
            InboxDB1[(Inbox Tables)]
        end
        subgraph "Shard 2"
            ChatDB2[(Chat Tables)]
            MessageDB2[(Message Tables)]
            InboxDB2[(Inbox Tables)]
        end
        UserDB[(User & Client Tables)]
        Cache[(Redis Cache)]
    end
    
    subgraph "External APIs"
        TranslateAPI[Translation API]
        STTAPI[Speech-to-Text API]
    end
    
    %% Client Connections
    Mobile1 -.->|WebSocket| LB
    Mobile2 -.->|WebSocket| LB
    Web1 -.->|WebSocket| LB
    Desktop1 -.->|WebSocket| LB
    
    %% Load Balancing
    LB --> ChatServer1
    LB --> ChatServer2
    LB --> ChatServer3
    
    %% Chat Server to Pub/Sub
    ChatServer1 <-->|Subscribe/Publish| PubSub
    ChatServer2 <-->|Subscribe/Publish| PubSub
    ChatServer3 <-->|Subscribe/Publish| PubSub
    
    %% Message Flow
    ChatServer1 --> Queue
    ChatServer2 --> Queue
    ChatServer3 --> Queue
    Queue --> Translation
    
    %% Database Access
    ChatServer1 --> ChatDB1
    ChatServer1 --> MessageDB1
    ChatServer1 --> InboxDB1
    ChatServer2 --> ChatDB2
    ChatServer2 --> MessageDB2
    ChatServer2 --> InboxDB2
    ChatServer3 --> UserDB
    
    %% Caching
    ChatServer1 --> Cache
    ChatServer2 --> Cache
    ChatServer3 --> Cache
    
    %% Language Services
    Translation --> Grammar
    Translation --> Vocabulary
    Translation --> TranslateAPI
    
    %% Media Handling
    Mobile1 -->|Upload| MediaAPI
    Web1 -->|Upload| MediaAPI
    MediaAPI --> MediaStore
    MediaStore --> CDN
    CDN --> Mobile2
    CDN --> Desktop1
    
    %% Multi-Device Message Flow Example
    Mobile1 -.->|1. Send Message| ChatServer1
    ChatServer1 -.->|2. Store & Publish| PubSub
    PubSub -.->|3. Route to Subscribers| ChatServer2
    PubSub -.->|3. Route to Subscribers| ChatServer3
    ChatServer2 -.->|4. Deliver| Mobile2
    ChatServer3 -.->|4. Deliver| Desktop1
    
    %% Styling
    classDef client fill:#e1f5fe
    classDef server fill:#f3e5f5
    classDef database fill:#e8f5e8
    classDef external fill:#fff3e0
    
    class Mobile1,Mobile2,Web1,Desktop1 client
    class ChatServer1,ChatServer2,ChatServer3,PubSub,Queue server
    class ChatDB1,ChatDB2,MessageDB1,MessageDB2,InboxDB1,InboxDB2,UserDB,Cache,MediaStore database
    class TranslateAPI,STTAPI,CDN external
```

## Core Entities and API Design

### Core Entities

The system is built around four fundamental entities that mirror WhatsApp's proven architecture:

- **Users**: Registered individuals with language preferences and learning settings
- **Chats**: Conversation threads supporting 2-100 participants  
- **Messages**: Text, media, or voice content with translation metadata
- **Clients**: Device instances (mobile, web, desktop) for multi-device support

### Primary API Commands

**Chat Management:**
```typescript
// Create new chat
POST /api/createChat
{
    "participants": ["userId1", "userId2"],
    "name": "Optional Group Name"
} 
→ { "chatId": "chat_123" }

// Modify chat participants  
POST /api/modifyChatParticipants
{
    "chatId": "chat_123",
    "userId": "user_456", 
    "operation": "ADD" | "REMOVE"
}
→ "SUCCESS" | "FAILURE"
```

**Message Operations:**
```typescript
// Send message
POST /api/sendMessage
{
    "chatId": "chat_123",
    "message": "Hello world",
    "attachments": ["attachment_789"]
}
→ "SUCCESS" | "FAILURE"

// Create media attachment
POST /api/createAttachment  
{
    "body": <binary_data>,
    "hash": "sha256_hash"
}
→ { "attachmentId": "attachment_789" }
```

### Client-Bound Commands (WebSocket)

**Real-time Updates:**
```typescript
// New message received
← newMessage
{
    "chatId": "chat_123",
    "userId": "user_456",
    "message": "Hello world", 
    "attachments": ["attachment_789"],
    "originalLanguage": "en",
    "translations": {"es": "Hola mundo"}
}
→ "RECEIVED" // Client acknowledgment

// Chat metadata updated
← chatUpdate  
{
    "chatId": "chat_123",
    "participants": ["user_123", "user_456"],
    "name": "Updated Group Name"
}
→ "RECEIVED" // Client acknowledgment
```

### Message Delivery Architecture

**Acknowledgment-Based Delivery:**
- All client-bound commands require explicit "RECEIVED" acknowledgment
- Unacknowledged messages remain in per-client Inbox tables
- Automatic retry with exponential backoff for failed deliveries
- 30-day message retention for offline clients

**Multi-Client Support:**
- Each user can have multiple active clients (mobile, web, desktop)
- Messages delivered to all active clients for a user
- Per-client delivery tracking and acknowledgment
- Client registration and deactivation management

## Components and Interfaces

### Chat Server
**Responsibility:** Core message routing, WebSocket connection management, and real-time delivery coordination.

**Key Interfaces:**
- `POST /api/createChat` - Create new chat with participants
- `POST /api/sendMessage` - Send message with delivery guarantees  
- `POST /api/modifyChatParticipants` - Add/remove chat participants
- `POST /api/createAttachment` - Upload media attachments
- `WebSocket /ws/{userId}` - Real-time message delivery and acknowledgments

**Internal APIs:**
- `routeMessage(message, recipients)` - Route messages to appropriate clients
- `manageConnection(userId, clientId)` - WebSocket lifecycle management
- `handleAcknowledgment(messageId, clientId)` - Process delivery confirmations
- `syncOfflineMessages(userId, clientId)` - Deliver queued messages on reconnect

### Database Tables

**Chat Table:**
- Primary Key: `chatId`
- Attributes: `participants[]`, `name`, `createdAt`, `lastActivity`
- Access Pattern: Direct lookup by chatId

**ChatParticipant Table:**  
- Composite Primary Key: `chatId` + `participantId`
- GSI: `participantId` (partition) + `chatId` (sort) for user's chats lookup
- Attributes: `joinedAt`, `role`, `lastReadMessageId`

**Message Table:**
- Primary Key: `messageId` 
- Attributes: `chatId`, `senderId`, `content`, `attachments[]`, `timestamp`
- GSI: `chatId` (partition) + `timestamp` (sort) for chat history

**Inbox Table (Per-Client Delivery):**
- Composite Primary Key: `clientId` + `messageId`
- Attributes: `chatId`, `deliveryAttempts`, `createdAt`
- TTL: 30 days for automatic cleanup

**Clients Table:**
- Composite Primary Key: `userId` + `clientId`
- Attributes: `deviceType`, `lastActive`, `connectionStatus`
- Limit: 3 active clients per user

### Message Delivery Flow

**Sending a Message:**
1. Client sends `sendMessage` command to Chat Server
2. Chat Server looks up all participants via ChatParticipant table  
3. Chat Server creates transaction:
   - Writes message to Message table
   - Creates Inbox entry for each recipient client
4. Chat Server returns SUCCESS/FAILURE with messageId
5. Chat Server publishes to Redis Pub/Sub topics for each participant
6. Connected Chat Servers receive pub/sub messages
7. Chat Servers send `newMessage` to connected clients
8. Clients send "RECEIVED" acknowledgment
9. Chat Server removes message from Inbox table upon acknowledgment

**Offline Message Delivery:**
1. Client connects and authenticates
2. Chat Server subscribes to Redis Pub/Sub topic for userId
3. Chat Server queries Inbox table for undelivered messages
4. Chat Server looks up message details from Message table
5. Chat Server sends `newMessage` commands to client
6. Client acknowledges with "RECEIVED"
7. Chat Server removes messages from Inbox table

**Multi-Client Synchronization:**
- Each client maintains separate Inbox entries
- Messages delivered independently to each active client
- Read status synchronized across all user's clients
- Client deactivation removes pending Inbox entries

## Redis Pub/Sub Architecture for Multi-Device Support

### Connection Management

**Client-to-Server Mapping:**
- Each client (mobile, web, desktop) connects to a specific Chat Server via WebSocket
- Load balancer distributes connections across available Chat Server instances
- Chat Servers maintain hash map: `userId → [WebSocket connections]`
- Multiple clients per user supported (up to 3 active clients)

**Pub/Sub Topic Structure:**
- Each user has a dedicated Redis Pub/Sub topic: `user:{userId}`
- Chat Servers subscribe to topics for all connected users
- Lightweight message routing without persistent storage in Redis
- "At most once" delivery semantics (durability handled by database)

### Multi-Device Message Flow

**Sending a Message (Multi-Device Scenario):**
```
User A (Mobile) → Chat Server 1
├── 1. Receive sendMessage command
├── 2. Store in Message table + create Inbox entries for all recipient clients
├── 3. Publish to Redis topic "user:UserB" 
└── 4. Return SUCCESS to User A Mobile

Redis Pub/Sub → All Chat Servers subscribed to "user:UserB"
├── Chat Server 2 (User B Desktop connected)
│   └── Forward newMessage to User B Desktop → Receive ACK → Delete from Inbox
├── Chat Server 3 (User B Mobile connected)  
│   └── Forward newMessage to User B Mobile → Receive ACK → Delete from Inbox
└── Chat Server 1 (User B Web offline)
    └── No connection, message remains in Inbox for later delivery
```

**Offline Message Delivery:**
```
User B Web connects to Chat Server 1
├── 1. WebSocket connection established
├── 2. Chat Server subscribes to "user:UserB" topic
├── 3. Query Inbox table for undelivered messages to this clientId
├── 4. Fetch message details from Message table
├── 5. Send newMessage commands to User B Web
├── 6. Receive ACK confirmations
└── 7. Delete delivered messages from Inbox table
```

### Database Schema for Multi-Device

**Clients Table:**
```sql
CREATE TABLE Clients (
    userId VARCHAR(50),
    clientId VARCHAR(50),
    deviceType ENUM('mobile', 'web', 'desktop'),
    connectionStatus ENUM('online', 'offline'),
    lastActive TIMESTAMP,
    PRIMARY KEY (userId, clientId),
    INDEX idx_user_active (userId, connectionStatus, lastActive)
);
```

**Inbox Table (Per-Client Delivery Tracking):**
```sql
CREATE TABLE Inbox (
    clientId VARCHAR(50),
    messageId VARCHAR(50), 
    chatId VARCHAR(50),
    deliveryAttempts INT DEFAULT 0,
    createdAt TIMESTAMP,
    ttl TIMESTAMP, -- 30 days from creation
    PRIMARY KEY (clientId, messageId),
    INDEX idx_client_pending (clientId, createdAt)
);
```

### Scaling and Reliability

**Horizontal Scaling:**
- Chat Servers are stateless and can be added/removed dynamically
- Redis Pub/Sub handles message routing across server instances
- Database sharding distributes load across multiple database instances
- Client connections automatically rebalanced on server restarts

**Fault Tolerance:**
- Redis Pub/Sub failures don't lose messages (durability in database)
- Chat Server failures trigger client reconnection to other servers
- Database replication ensures data durability
- Inbox table provides delivery guarantees even during outages

**Performance Optimizations:**
- Connection pooling for database access
- Message batching for high-volume chats
- Intelligent client routing based on geographic location
- Caching of frequently accessed chat metadata

### Translation Service
**Responsibility:** Language detection, translation, and caching of translation results.

**Key Interfaces:**
- `POST /translate` - Translate text with language detection
- `POST /translate/batch` - Batch translation for multiple texts
- `GET /translate/languages` - List supported languages
- `GET /translate/cache/{hash}` - Retrieve cached translation

**Internal APIs:**
- `detectLanguage(text)` - Language detection
- `translateText(text, sourceLang, targetLang)` - Core translation
- `getCachedTranslation(textHash, targetLang)` - Cache retrieval
- `cacheTranslation(textHash, translation, targetLang)` - Cache storage

### Grammar Engine
**Responsibility:** Sentence analysis, grammar explanations, and CEFR difficulty assessment.

**Key Interfaces:**
- `POST /grammar/analyze` - Analyze sentence structure
- `POST /grammar/explain` - Generate grammar explanations
- `GET /grammar/difficulty/{text}` - Assess CEFR difficulty level

**Internal APIs:**
- `parsePartOfSpeech(text, language)` - POS tagging
- `identifyGrammarPatterns(text, language)` - Pattern recognition
- `generateExplanation(patterns, userLevel)` - Explanation generation

### Vocabulary Manager
**Responsibility:** Personal vocabulary lists, spaced repetition scheduling, and learning progress tracking.

**Key Interfaces:**
- `POST /vocabulary` - Save word/phrase to personal list
- `GET /vocabulary` - Retrieve personal vocabulary list
- `GET /vocabulary/due` - Get words due for review
- `POST /vocabulary/practice` - Submit practice session results
- `GET /vocabulary/progress` - Retrieve learning statistics

**Internal APIs:**
- `scheduleReview(wordId, difficulty, performance)` - Spaced repetition scheduling
- `updateProgress(userId, language, metrics)` - Progress tracking
- `generatePracticeSession(userId, language)` - Practice session creation

### Call Service
**Responsibility:** WebRTC call management, real-time transcription, and subtitle generation.

**Key Interfaces:**
- `POST /calls/initiate` - Start new call
- `POST /calls/{callId}/join` - Join existing call
- `POST /calls/{callId}/end` - End call
- `GET /calls/{callId}/subtitles` - Real-time subtitle stream
- `WebSocket /calls/{callId}/signaling` - WebRTC signaling

**Event Publishing:**
- `call.started` - Call initiated
- `call.joined` - Participant joined
- `call.speech.detected` - Speech-to-text result
- `call.ended` - Call terminated

### Transcript Service
**Responsibility:** Call transcript storage, search, and privacy management.

**Key Interfaces:**
- `GET /transcripts/{callId}` - Retrieve call transcript
- `GET /transcripts/search` - Search transcript content
- `DELETE /transcripts/{callId}` - Delete transcript (privacy)
- `PUT /transcripts/{callId}/settings` - Update transcript privacy settings

### Notification Service
**Responsibility:** Multi-device message delivery, push notifications, presence management, and connection state tracking.

**Key Interfaces:**
- `WebSocket /notifications` - Real-time notification stream with heartbeat
- `POST /notifications/register` - Register device for push notifications with token management
- `GET /presence/{userId}` - Get user presence status with last seen information
- `PUT /presence` - Update own presence status with activity indicators
- `GET /notifications/history` - Retrieve missed notifications for offline devices
- `POST /notifications/ack` - Acknowledge notification delivery

**Event Consumption:**
- `message.sent` - Deliver message to recipients with priority routing
- `message.read` - Update read status across devices with conflict resolution
- `call.started` - Notify call participants with urgency flags
- `user.online` - Update presence status and trigger pending deliveries
- `user.offline` - Queue messages for offline delivery

**Internal APIs:**
- `maintainConnection(userId, deviceId)` - WebSocket connection lifecycle management
- `routeNotification(notification, devices)` - Intelligent notification routing
- `handleOfflineQueue(userId)` - Manage offline message queuing
- `syncDeviceState(userId, devices)` - Multi-device state synchronization

### Media Service
**Responsibility:** File upload, processing, storage, and delivery with CDN integration.

**Key Interfaces:**
- `POST /media/upload` - Upload media file
- `GET /media/{fileId}` - Retrieve media file
- `GET /media/{fileId}/thumbnail` - Get media thumbnail
- `DELETE /media/{fileId}` - Delete media file

**Internal APIs:**
- `processImage(file)` - Image optimization and thumbnail generation
- `processVideo(file)` - Video compression and thumbnail extraction
- `processAudio(file)` - Audio compression and format conversion

## Data Models

### Database Sharding Strategy

The system implements horizontal sharding to handle WhatsApp-scale traffic:

**User Data Sharding:**
- Shard key: `hash(userId) % num_shards`
- Each shard contains complete user profiles, settings, and device information
- Cross-shard queries minimized through denormalization

**Message Data Sharding:**
- Shard key: `hash(chatId) % num_shards` 
- Messages for a chat always reside on the same shard for consistency
- Group chats distributed based on chat creation hash
- Hot shard detection and rebalancing for popular chats

**Vocabulary Data Sharding:**
- Shard key: `hash(userId) % num_shards`
- Co-located with user data for efficient queries
- Learning progress and spaced repetition data kept together

### Caching Strategy

**Multi-Level Caching:**
- L1: Application-level cache (in-memory, 5-minute TTL)
- L2: Redis cluster (distributed, 1-hour TTL)
- L3: CDN for media content (24-hour TTL)

**Cache Invalidation:**
- Write-through for critical data (user profiles, chat membership)
- Write-behind for analytics and learning progress
- Event-driven invalidation for real-time updates

### Core Data Models

### User Model
```typescript
interface User {
  id: string;
  email: string;
  username: string;
  displayName: string;
  avatar?: string;
  nativeLanguage: string;
  targetLanguages: string[];
  learningSettings: {
    grammarEnabled: boolean;
    vocabularyEnabled: boolean;
    difficultyLevel: 'beginner' | 'intermediate' | 'advanced';
  };
  privacySettings: {
    transcriptRecording: boolean;
    messageRetentionDays: number;
    dataSharing: boolean;
  };
  createdAt: Date;
  lastActiveAt: Date;
}
```

### Chat Model
```typescript
interface Chat {
  id: string;
  participants: string[]; // Array of userIds (2-100 participants)
  name?: string; // Optional for group chats
  metadata: {
    createdBy: string;
    createdAt: Date;
    lastActivity: Date;
  };
  settings: {
    translationEnabled: boolean;
    learningFeaturesEnabled: boolean;
    messageRetentionDays: number;
  };
}
```

### Message Model
```typescript
interface Message {
  id: string;
  chatId: string;
  senderId: string;
  content: {
    text?: string;
    attachments?: string[]; // Array of attachmentIds
  };
  originalLanguage?: string;
  translations: Map<string, string>; // targetLanguage -> translatedText
  metadata: {
    replyToId?: string;
    mentions: string[];
    grammarAnalysis?: GrammarAnalysis;
  };
  timestamp: Date;
}
```

### Client Model
```typescript
interface Client {
  id: string; // Unique client identifier
  userId: string;
  deviceType: 'mobile' | 'web' | 'desktop';
  deviceInfo: {
    platform: string;
    version: string;
    userAgent?: string;
  };
  connectionStatus: 'online' | 'offline';
  lastActive: Date;
  createdAt: Date;
}
```

### Inbox Model (Per-Client Message Delivery)
```typescript
interface InboxEntry {
  clientId: string;
  messageId: string;
  chatId: string;
  deliveryAttempts: number;
  createdAt: Date;
  ttl: Date; // 30 days from creation
}
```

### ChatParticipant Model
```typescript
interface ChatParticipant {
  chatId: string;
  participantId: string;
  role: 'member' | 'admin';
  joinedAt: Date;
  lastReadMessageId?: string;
  notificationSettings: {
    muted: boolean;
    muteUntil?: Date;
  };
}
```

### Vocabulary Entry Model
```typescript
interface VocabularyEntry {
  id: string;
  userId: string;
  term: string;
  language: string;
  translation: string;
  definition: string;
  context: {
    messageId: string;
    sentence: string;
    chatId: string;
  };
  learningData: {
    difficulty: number;
    reviewCount: number;
    correctCount: number;
    lastReviewed?: Date;
    nextReview: Date;
    interval: number; // days
  };
  createdAt: Date;
}
```

### Call Transcript Model
```typescript
interface CallTranscript {
  id: string;
  callId: string;
  participants: string[];
  segments: TranscriptSegment[];
  metadata: {
    duration: number;
    languages: string[];
    quality: 'high' | 'medium' | 'low';
  };
  privacySettings: {
    retentionDays: number;
    searchable: boolean;
  };
  createdAt: Date;
}

interface TranscriptSegment {
  speakerId: string;
  startTime: number;
  endTime: number;
  originalText: string;
  originalLanguage: string;
  translations: Map<string, string>;
  confidence: number;
}
```

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

Before defining the correctness properties, I need to analyze the acceptance criteria to determine which ones are testable as properties.

After reviewing the acceptance criteria, I need to perform property reflection to eliminate redundancy before writing the correctness properties.

**Property Reflection:**

Looking at the testable properties from the prework analysis, I can identify several areas where properties can be combined or where one property implies another:

1. **User Registration and Profile (1.1, 1.2, 1.3)** - These can be combined into a comprehensive user data management property
2. **Message Delivery and Status (2.2, 3.4, 3.5)** - These are related to message delivery guarantees and can be combined
3. **Translation Caching (4.5, 12.2)** - These are essentially the same property about caching translation results
4. **Privacy Controls (7.6, 9.4)** - These are duplicate requirements about not storing content when recording is disabled
5. **Multi-device Synchronization (2.1, 2.3)** - These can be combined into a single property about device state sync
6. **Search Functionality (8.1, 8.2)** - These can be combined into a comprehensive search property
7. **Translation Display (4.3, 7.4)** - These are similar properties about showing original and translated text

Based on this reflection, here are the consolidated correctness properties:

### Property 1: User Data Management Round Trip
*For any* valid user registration data and profile updates, storing the data and then retrieving it should return equivalent information with all language preferences preserved
**Validates: Requirements 1.1, 1.2, 1.3**

### Property 2: Language Validation Consistency  
*For any* language selection, the system should accept only languages supported by the Translation_Service and reject unsupported languages consistently
**Validates: Requirements 1.4**

### Property 3: Multi-Device State Synchronization
*For any* user action (login, message read, presence update) performed on one device, the state change should be reflected across all active devices for that user
**Validates: Requirements 1.5, 2.1, 2.3**

### Property 4: Message Delivery Guarantees
*For any* message sent to online recipients, the system should deliver it to all active devices and update delivery status correctly with timestamps
**Validates: Requirements 2.2, 3.4, 3.5, 10.5**

### Property 5: Offline Message Persistence
*For any* message sent while a device is offline, the message should be queued and delivered when the device reconnects, with history maintained for at least 30 days
**Validates: Requirements 2.4, 2.5, 10.3**

### Property 6: Real-Time Communication Performance
*For any* text message sent under normal network conditions, the delivery latency to online recipients should be under 500ms
**Validates: Requirements 3.1**

### Property 7: Media Processing Completeness
*For any* supported media file uploaded, the Media_Service should process it successfully and make it available to recipients with appropriate retry mechanisms on failure
**Validates: Requirements 3.2, 10.4**

### Property 8: Typing Indicator Propagation
*For any* typing event in a chat, typing indicators should be broadcast to all other participants in real-time
**Validates: Requirements 3.3**

### Property 9: Group Chat Capacity
*For any* group chat with up to 100 participants, all messaging functionality should work correctly without degradation
**Validates: Requirements 3.6**

### Property 10: Translation Pipeline Consistency
*For any* message received in a language different from the recipient's preference, the system should detect the language, generate a translation, and cache the result for reuse
**Validates: Requirements 4.1, 4.2, 4.5, 12.2**

### Property 11: Translation Display Completeness
*For any* translated content (messages or call subtitles), both original and translated text should be displayed with clear visual distinction
**Validates: Requirements 4.3, 7.4**

### Property 12: Translation Preference Persistence
*For any* translation setting change in a chat, the preference should persist across sessions and apply to future messages
**Validates: Requirements 4.4**

### Property 13: Translation Error Handling
*For any* translation failure, the system should display the original message with a clear error indicator and continue functioning
**Validates: Requirements 4.6, 10.2**

### Property 14: Grammar Analysis Completeness
*For any* message in a user's target language, grammar analysis should provide sentence breakdown, explanations, and CEFR difficulty level when requested
**Validates: Requirements 5.1, 5.2, 5.3**

### Property 15: Learning Feature Visibility Control
*For any* user with learning features disabled, grammar analysis options should be hidden while maintaining normal message functionality
**Validates: Requirements 5.4, 5.5**

### Property 16: Vocabulary Management Round Trip
*For any* word or phrase saved to vocabulary, all required data (term, language, translation, definition, context) should be stored and retrievable
**Validates: Requirements 6.1, 6.2**

### Property 17: Spaced Repetition Scheduling
*For any* vocabulary due for review, the system should present it in practice exercises and update learning progress based on performance
**Validates: Requirements 6.3, 6.4**

### Property 18: Learning Progress Tracking
*For any* learning activity, the system should accurately track and update metrics including words learned, review streaks, and study time
**Validates: Requirements 6.5**

### Property 19: WebRTC Call Establishment
*For any* call initiation, the Call_Service should establish a WebRTC connection and enable real-time communication
**Validates: Requirements 7.1**

### Property 20: Real-Time Speech Processing
*For any* speech detected during a call, the system should convert it to text and translate it to the other participant's language in real-time
**Validates: Requirements 7.2, 7.3**

### Property 21: Call Transcript Completeness
*For any* completed call with recording enabled, the system should store a complete transcript with timestamps, speaker labels, and translations
**Validates: Requirements 7.5**

### Property 22: Privacy-Respecting Call Recording
*For any* call with recording disabled, the system should not store any call content, transcripts, or audio data
**Validates: Requirements 7.6, 9.4**

### Property 23: Comprehensive Search Functionality
*For any* search query, the system should search across messages, transcripts, contact names, and metadata, returning results with highlighted matches and context
**Validates: Requirements 8.1, 8.2, 8.3**

### Property 24: Language-Filtered Search
*For any* search with language-specific filters, results should contain only content in the specified language
**Validates: Requirements 8.4**

### Property 25: Search Performance
*For any* search operation, the system should return results within acceptable time limits using proper indexing
**Validates: Requirements 8.5**

### Property 26: End-to-End Encryption
*For any* message or call, the system should encrypt communication in transit with TLS and implement end-to-end encryption with multi-device key management
**Validates: Requirements 9.1, 9.2**

### Property 27: Data-at-Rest Encryption
*For any* stored data (media, transcripts, learning data), the system should encrypt it at rest
**Validates: Requirements 9.3**

### Property 28: Data Retention Compliance
*For any* user-specified retention period, the system should automatically delete content after the specified time while providing clear privacy controls
**Validates: Requirements 9.5, 9.6**

### Property 29: Privacy-Preserving Analytics
*For any* analytics collection, the system should track usage metrics and patterns without storing message content
**Validates: Requirements 11.1, 11.2**

### Property 30: Feature Flag Management
*For any* feature flag change, the system should enable or disable features for the correct user segments and track API usage for cost control
**Validates: Requirements 11.3, 11.4**

### Property 31: A/B Testing Infrastructure
*For any* A/B test configuration, the system should serve different variations to appropriate user segments
**Validates: Requirements 11.5**

### Property 32: Tiered Feature Enforcement
*For any* free-tier user, premium features should be disabled while core messaging remains fully functional
**Validates: Requirements 12.4**

### Property 33: System Monitoring
*For any* system operation, appropriate metrics should be collected and alerts triggered when thresholds are exceeded
**Validates: Requirements 12.5**

### Property 34: Message Ordering Consistency
*For any* sequence of messages sent in a chat, they should appear in the same chronological order for all recipients across all devices
**Validates: Requirements 13.1, 13.2, 13.4**

### Property 35: Conflict Resolution Determinism
*For any* concurrent message editing scenario, the last-writer-wins resolution should produce consistent results across all devices
**Validates: Requirements 13.3, 13.5**

### Property 36: Connection Heartbeat Reliability
*For any* active WebSocket connection, heartbeat messages should be sent every 30 seconds and connection failures should trigger automatic reconnection
**Validates: Requirements 14.1, 14.2**

### Property 37: Message Synchronization on Reconnect
*For any* device reconnection, all missed messages and status updates should be synchronized correctly
**Validates: Requirements 14.3**

### Property 38: Rate Limiting Enforcement
*For any* user sending messages, the system should enforce the 100 messages per minute limit and queue excess messages appropriately
**Validates: Requirements 15.1, 15.2**

### Property 39: Abuse Detection and Response
*For any* detected spam pattern, the system should apply progressive penalties while maintaining an appeals process
**Validates: Requirements 15.3, 15.4, 15.5**

### Property 40: Data Backup Completeness
*For any* user data backup operation, all messages, learning progress, and settings should be included and verifiable through restoration testing
**Validates: Requirements 16.1, 16.2, 16.5**

### Property 41: Cross-Region Data Replication
*For any* critical data stored in the system, it should be replicated across multiple geographic regions for disaster recovery
**Validates: Requirements 16.3**

### Property 42: Data Export Completeness
*For any* user data export request, the exported data should include complete conversation history and learning data in a usable format
**Validates: Requirements 16.4**

## Reliability and Fault Tolerance

### Message Delivery Guarantees

**At-Least-Once Delivery:**
- Messages stored durably before acknowledgment
- Retry mechanism with exponential backoff (1s, 2s, 4s, 8s, 16s)
- Dead letter queue for permanently failed messages
- Duplicate detection using message IDs

**Ordering Guarantees:**
- Per-chat message ordering maintained through sequence numbers
- Vector clocks for multi-device synchronization
- Conflict resolution for concurrent edits

### Circuit Breaker Implementation

**External Service Protection:**
- Translation API: 50% failure rate triggers circuit breaker
- Speech-to-Text API: 5-second timeout with 3 retries
- Graceful degradation when services unavailable

**Database Protection:**
- Connection pooling with health checks
- Read replica failover for query load
- Write queue buffering during outages

### Disaster Recovery

**Data Replication:**
- Cross-region database replication with 5-minute RPO
- Media storage replicated across 3 availability zones
- Configuration and schema versioning for rollback capability

**Service Recovery:**
- Automated failover for stateless services
- Message queue persistence across restarts
- Connection state recovery for WebSocket clients

## Performance Optimizations

### Message Routing Optimization

**Intelligent Routing:**
- Geographic routing based on user location
- Device preference routing (mobile vs desktop priority)
- Load-based routing to prevent hot spots
- Connection affinity for active conversations

**Batch Processing:**
- Group message delivery batching (up to 100 recipients)
- Translation request batching for efficiency
- Notification delivery batching to reduce push notification costs

### Database Query Optimization

**Indexing Strategy:**
- Composite indexes on (chatId, timestamp) for message retrieval
- Partial indexes for unread messages
- Full-text search indexes for message content

**Query Patterns:**
- Cursor-based pagination for infinite scroll
- Prefetching for predictive loading
- Denormalized data for common queries (last message preview)

### Caching Optimizations

**Smart Cache Warming:**
- Predictive caching based on user activity patterns
- Translation cache pre-warming for common language pairs
- Media thumbnail pre-generation and caching

**Cache Coherence:**
- Event-driven cache invalidation
- Distributed cache consistency using Redis Cluster
- Cache stampede prevention using distributed locks

## Error Handling

The system implements comprehensive error handling patterns inspired by WhatsApp's reliability requirements:

### Message Delivery Error Handling

**Network Partition Handling:**
- Message queuing with persistent storage during network splits
- Automatic retry with exponential backoff (1s, 2s, 4s, 8s, 16s, 32s)
- Message deduplication using unique message IDs
- Ordering preservation through sequence numbers
- Dead letter queue for messages that fail after all retries

**Delivery Status Management:**
- Detailed delivery tracking: sent → delivered → read
- Per-device delivery confirmation for multi-device users
- Timeout handling for undelivered messages (mark as failed after 24 hours)
- Retry queue prioritization based on message age and importance

### Translation Service Error Handling

**API Failure Resilience:**
- Circuit breaker pattern: open after 50% failure rate over 1 minute
- Fallback to cached translations when available
- Graceful degradation: show original text with translation unavailable indicator
- Rate limiting protection with request queuing
- Multiple translation provider failover (primary → secondary → tertiary)

**Language Detection Failures:**
- Fallback to user's declared native language
- Confidence threshold enforcement (minimum 70% confidence)
- Manual language override option for users
- Learning from user corrections to improve detection

### Real-Time Communication Error Handling

**WebRTC Connection Failures:**
- ICE candidate gathering timeout handling (30 seconds)
- STUN/TURN server failover for NAT traversal
- Automatic reconnection with exponential backoff
- Codec negotiation fallback (prefer VP8 → H.264 → VP9)
- Network quality adaptation (bitrate adjustment)

**Call Quality Management:**
- Automatic quality adjustment based on network conditions
- Packet loss detection and compensation
- Echo cancellation and noise suppression
- Graceful degradation from video to audio-only

### Database and Storage Error Handling

**Database Resilience:**
- Connection pool management with health checks
- Read replica failover for query operations
- Write operation retry with conflict resolution
- Distributed transaction handling across shards
- Backup and restore procedures with point-in-time recovery

**Media Storage Failures:**
- Multi-region storage replication
- Checksum verification for data integrity
- Automatic retry for failed uploads with progress resumption
- Thumbnail generation fallback and error recovery
- CDN failover for media delivery

### System-Wide Error Patterns

**Cascading Failure Prevention:**
- Service isolation through bulkhead pattern
- Request timeout enforcement at all service boundaries
- Resource exhaustion protection (memory, CPU, connections)
- Graceful service degradation under high load

**Monitoring and Alerting:**
- Real-time error rate monitoring with automated alerts
- Distributed tracing for error root cause analysis
- Error categorization and prioritization
- Automated incident response for critical failures

## Testing Strategy

The testing strategy employs a dual approach combining unit tests for specific scenarios and property-based tests for comprehensive coverage:

### Unit Testing
Unit tests focus on:
- **Specific Examples**: Concrete test cases that demonstrate correct behavior
- **Edge Cases**: Boundary conditions, empty inputs, maximum limits
- **Error Conditions**: Invalid inputs, service failures, network issues
- **Integration Points**: Component interactions and data flow validation

### Property-Based Testing
Property-based tests validate universal properties using **fast-check** library for JavaScript/TypeScript:
- **Minimum 100 iterations** per property test to ensure comprehensive input coverage
- **Randomized input generation** to discover edge cases not covered by unit tests
- **Property validation** across all possible inputs within defined constraints
- **Regression prevention** through consistent property verification

Each property test must be tagged with:
**Feature: multilingual-messenger, Property {number}: {property_text}**

### Testing Configuration
- **Unit Test Framework**: Jest with TypeScript support
- **Property Test Library**: fast-check for comprehensive input generation
- **Integration Testing**: Supertest for API endpoint testing
- **End-to-End Testing**: Playwright for full user journey validation
- **Load Testing**: Artillery for performance and scalability validation

### Test Data Management
- **Synthetic Data Generation**: Create realistic test data without using real user information
- **Language Test Coverage**: Include tests for major language pairs and edge cases
- **Media Test Assets**: Maintain test files for various media formats and sizes
- **Mock External Services**: Stub translation and speech-to-text APIs for consistent testing

The combination of unit and property-based testing ensures both specific functionality correctness and universal behavior validation across the entire input space.