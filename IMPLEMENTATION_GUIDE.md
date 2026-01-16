# Chorus - Complete Implementation Guide
## Phase 2 & Phase 3 Features

This guide provides comprehensive instructions for running and testing the fully-implemented Chorus messenger with all Phase 2 and Phase 3 features.

## 🎯 Implemented Features

### Phase 1 (Completed)
✅ User authentication (JWT)  
✅ Direct and group chats  
✅ Real-time messaging via WebSocket  
✅ Automatic translation  
✅ Message search  
✅ Typing indicators  

### Phase 2 (Backend Complete)
✅ Multi-device support (up to 3 devices per user)  
✅ Offline message delivery (30-day inbox)  
✅ Client device registration and tracking  
✅ Per-device message acknowledgment  
✅ Enhanced WebSocket with heartbeat  
✅ Presence tracking system  
✅ Advanced search service  

### Phase 3 (Backend Complete)
✅ Grammar analysis with CEFR levels  
✅ Vocabulary management with spaced repetition  
✅ Voice/video calls with WebRTC  
✅ Speech-to-Text integration (Google Cloud)  
✅ Call transcription with translations  
✅ Call history and transcript search  

## 📋 Prerequisites

- Go 1.21+ 
- PostgreSQL 15+
- Redis 7+
- Node.js 18+ (for frontend)
- Docker & Docker Compose (optional)
- Google Cloud API key (for Speech-to-Text, optional)
- Google Translate API key

## 🚀 Quick Start

### Option 1: Using Docker (Recommended)

```bash
# Clone and navigate
cd /home/uhsarp/dev/chorus

# Start all services
docker-compose up --build

# Access the application
# Frontend: http://localhost:3000
# Backend API: http://localhost:8080
# Health check: http://localhost:8080/health
```

### Option 2: Manual Setup

#### 1. Start PostgreSQL and Redis

```bash
# PostgreSQL
sudo systemctl start postgresql
createdb chorus_dev

# Redis
sudo systemctl start redis
```

#### 2. Configure Environment

```bash
cd backend

# Create .env file
cat > .env << EOF
DATABASE_URL=postgresql://postgres:password@localhost:5432/chorus_dev
REDIS_URL=redis://localhost:6379
JWT_SECRET=your-secret-key-change-this-in-production
GOOGLE_TRANSLATE_API_KEY=your-google-translate-api-key
GOOGLE_APPLICATION_CREDENTIALS=/path/to/google-cloud-credentials.json
PORT=8080
ENVIRONMENT=development
EOF
```

#### 3. Start Backend

```bash
cd backend

# Install dependencies
go mod download

# Run database migrations (automatic on startup)
# Start server
go run cmd/server/main.go
```

#### 4. Start Frontend

```bash
cd frontend

# Install dependencies
npm install

# Start development server
npm run dev
```

## 🔌 API Endpoints

### Authentication
- `POST /api/v1/auth/register` - Register new user
- `POST /api/v1/auth/login` - User login
- `POST /api/v1/auth/refresh` - Refresh access token

### Users
- `GET /api/v1/users/me` - Get current user
- `PUT /api/v1/users/me` - Update user profile
- `GET /api/v1/users/search` - Search users

### Chats
- `GET /api/v1/chats` - Get user's chats
- `POST /api/v1/chats` - Create new chat
- `GET /api/v1/chats/:chatId` - Get chat details
- `PUT /api/v1/chats/:chatId` - Update chat
- `POST /api/v1/chats/:chatId/participants` - Add participant
- `DELETE /api/v1/chats/:chatId/participants/:userId` - Remove participant
- `DELETE /api/v1/chats/:chatId/leave` - Leave chat

### Messages
- `GET /api/v1/chats/:chatId/messages` - Get messages
- `POST /api/v1/chats/:chatId/messages` - Send message
- `PUT /api/v1/chats/:chatId/read` - Mark messages as read

### Search (Phase 2)
- `GET /api/v1/messages/search` - Search messages
- `GET /api/v1/chats/:chatId/search` - Search within chat
- `GET /api/v1/search/suggestions` - Get search suggestions
- `GET /api/v1/search/recent` - Get recent searches
- `DELETE /api/v1/search/history` - Clear search history

### Presence (Phase 2)
- `GET /api/v1/presence/:userId` - Get user presence
- `POST /api/v1/presence/batch` - Get multiple presences
- `PUT /api/v1/presence` - Update presence
- `POST /api/v1/presence/heartbeat` - Send heartbeat
- `GET /api/v1/presence/online/count` - Get online user count

### Grammar Analysis (Phase 3)
- `POST /api/v1/grammar/analyze` - Analyze message grammar
- `POST /api/v1/grammar/analyze-text` - Analyze arbitrary text
- `GET /api/v1/grammar/suggestions` - Get learning suggestions
- `GET /api/v1/grammar/report` - Get grammar learning report

### Vocabulary (Phase 3)
- `POST /api/v1/vocabulary` - Save new vocabulary
- `GET /api/v1/vocabulary` - Get all vocabulary
- `GET /api/v1/vocabulary/due` - Get vocabulary due for review
- `GET /api/v1/vocabulary/:id` - Get specific vocabulary
- `POST /api/v1/vocabulary/practice` - Record practice result
- `GET /api/v1/vocabulary/progress` - Get learning progress
- `DELETE /api/v1/vocabulary/:id` - Delete vocabulary
- `GET /api/v1/vocabulary/search` - Search vocabulary

### Calls (Phase 3)
- `POST /api/v1/calls/initiate` - Initiate call
- `POST /api/v1/calls/:callId/end` - End call
- `GET /api/v1/calls/:callId` - Get call session
- `GET /api/v1/calls/:callId/transcript` - Get call transcript
- `GET /api/v1/calls/history` - Get call history
- `DELETE /api/v1/calls/:callId/transcript` - Delete transcript
- `GET /api/v1/calls/transcripts/search` - Search transcripts
- `POST /api/v1/calls/:callId/signal` - WebRTC signaling

### WebSocket
- `GET /ws` - WebSocket connection (requires Bearer token)

## 🧪 Testing

### Run All Tests
```bash
./build-and-test.sh
```

### Run Specific Tests
```bash
cd backend

# Unit tests
go test ./internal/services/... -v

# Integration tests
go test ./internal/handlers/... -v

# With coverage
go test ./... -cover
```

## 🏗️ Architecture

### Backend Services
- **AuthService** - Authentication and JWT management
- **UserService** - User profile management
- **ChatService** - Chat operations
- **MessageService** - Message CRUD and delivery
- **TranslationService** - Message translation with caching
- **WebSocketHub** - Real-time message broadcasting
- **ClientService** - Multi-device management
- **InboxService** - Offline message queuing
- **PresenceService** - User presence tracking
- **SearchService** - Full-text search
- **GrammarService** - Grammar analysis and CEFR levels
- **VocabularyService** - Vocabulary with spaced repetition
- **CallService** - WebRTC call management
- **SpeechToTextService** - Voice transcription

### Database Schema

**Core Tables:**
- `users` - User accounts
- `chats` - Chat metadata
- `chat_participants` - Chat membership
- `messages` - Message content
- `refresh_tokens` - JWT refresh tokens

**Phase 2 Tables:**
- `clients` - Device registration
- `inbox` - Offline message queue
- `user_settings` - Extended preferences
- `media_attachments` - File metadata
- `presence_log` - Presence history
- `rate_limits` - Rate limiting data

**Phase 3 Tables:**
- `vocabulary` - Vocabulary entries
- `call_sessions` - Call metadata
- `call_participants` - Call membership
- `call_transcripts` - Transcriptions

## 📱 Mobile App (Coming Soon)

The React Native mobile app is in development with the following structure:

```
ChorusMobile/
├── src/
│   ├── screens/     # Screen components
│   ├── components/  # Reusable UI components
│   ├── navigation/  # Navigation setup
│   ├── services/    # API and WebSocket
│   ├── store/       # State management
│   └── types/       # TypeScript types
```

To initialize:
```bash
npx react-native init ChorusMobile --template react-native-template-typescript
```

## 🔧 Configuration

### Environment Variables

**Backend (.env)**
```env
DATABASE_URL=postgresql://user:pass@host:5432/dbname
REDIS_URL=redis://host:6379
JWT_SECRET=your-secret-key
GOOGLE_TRANSLATE_API_KEY=your-api-key
GOOGLE_APPLICATION_CREDENTIALS=/path/to/credentials.json
PORT=8080
ENVIRONMENT=development
```

**Frontend (.env)**
```env
VITE_API_URL=http://localhost:8080
VITE_WS_URL=ws://localhost:8080
```

## 🐳 Docker Deployment

### Build Images
```bash
docker-compose build
```

### Start Services
```bash
docker-compose up -d
```

### View Logs
```bash
docker-compose logs -f
```

### Stop Services
```bash
docker-compose down
```

## 📊 Monitoring

### Health Check
```bash
curl http://localhost:8080/health
```

### Database Status
```bash
psql -h localhost -U postgres -d chorus_dev -c "SELECT COUNT(*) FROM users;"
```

### Redis Status
```bash
redis-cli ping
```

## 🔐 Security

- JWT-based authentication
- Password hashing with bcrypt
- Rate limiting (100 messages/minute)
- SQL injection prevention
- XSS protection
- CORS configuration
- WebSocket authentication

## 🐛 Troubleshooting

### Backend Won't Start
```bash
# Check database connection
psql -h localhost -U postgres -d chorus_dev

# Check Redis
redis-cli ping

# View logs
tail -f backend/logs/app.log
```

### WebSocket Connection Fails
```bash
# Check if server is running
curl http://localhost:8080/health

# Test WebSocket (requires valid JWT)
wscat -c ws://localhost:8080/ws -H "Authorization: Bearer YOUR_TOKEN"
```

### Database Migration Issues
```bash
# Reset database
dropdb chorus_dev
createdb chorus_dev

# Restart backend (migrations run automatically)
go run cmd/server/main.go
```

## 📚 Additional Resources

- [API Documentation](./API_DOCUMENTATION.md)
- [Phase 2 & 3 Implementation Status](./PHASE_2_3_IMPLEMENTATION.md)
- [Design Document](./.kiro/specs/multilingual-messenger/design.md)
- [Testing Guide](./TESTING_GUIDE.md)

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Write/update tests
5. Submit a pull request

## 📝 License

[Add your license here]

## 👥 Support

For issues and questions:
- Create an issue on GitHub
- Check existing documentation
- Review the design document

---

**Last Updated**: December 31, 2025  
**Status**: Phase 2 & 3 Backend Complete, Mobile App Pending
