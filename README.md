# Chorus - Multilingual Messenger

A real-time messaging application with built-in translation features for multilingual conversations.

### Option 1: Hot Reload (Recommended for Frontend) ⚡

Run the frontend in **Vite dev mode** with Hot Module Replacement — changes appear instantly in the browser, no rebuild or redeploy needed.

**Prerequisites:** Backend + DB must be running in Docker.

```powershell
# Terminal 1: Start backend services (one time)
docker-compose up -d

# Terminal 2: Start frontend in dev mode with hot reload
cd frontend
npm run dev
```

Now open http://localhost:5173 — every time you save a `.tsx`/`.ts` file, the browser updates instantly.

### Option 2: Docker Rebuild (Slower, Full Production Build)
```powershell
# Frontend changes (.tsx, .ts, .css):
docker-compose up -d --build frontend

# Backend changes (.go):
docker-compose up -d --build backend

# Both:
docker-compose up -d --build
```

Then refresh your browser at http://localhost:3000

## �📌 Current Development Snapshot (Feb 2026)

### Architecture
- **Backend**: Go + Gin monolith with layered structure (`handlers` → `services` → `database/models`) and REST + WebSocket interfaces.
- **Data layer**: PostgreSQL for durable state and Redis for cache/pub-sub/presence runtime state.
- **Frontend (Web)**: React + TypeScript SPA (Vite, Zustand, Axios) with route-based auth flow.
- **Frontend (Mobile)**: Capacitor Android shell around the React app; emulator API access uses `10.0.2.2`.
- **Deployment**: Docker Compose orchestration for `frontend`, `backend`, `postgres`, and `redis`.

### Design & Implementation Shape
- Backend exposes **Phase 1–3 APIs** (auth, chat, messaging, search, presence, grammar, vocabulary, calls).
- Web UI currently focuses on **core messaging UX**: login/register, chat list, chat area, new chat creation, message send/read flow.
- Registration UX is simplified: UI does **not** ask for username; backend derives `username` from `email` when omitted.
- Translation, typing indicators, and chat updates are integrated through HTTP + WebSocket store handlers.

### Data Model (Current)
- **Core entities**: `users`, `chats`, `chat_participants`, `messages`, `refresh_tokens`.
- **Phase 2 entities**: `clients`, `inbox`, `user_settings`, `presence_log`, `rate_limits`, `media_attachments`.
- **Phase 3 entities**: `vocabulary`, `call_sessions`, `call_participants`, `call_transcripts`.
- Migrations are automatic on backend startup (`database.Migrate`).

### Feature Implementation Status
- ✅ **Running stack**: backend + frontend + postgres + redis via Docker Compose.
- ✅ **Core app**: auth, chats, realtime messaging, translation display, typing indicators.
- ✅ **Extended APIs**: search/presence/grammar/vocabulary/calls available on backend.
- ⚠️ **Frontend parity gap**: many Phase 2/3 APIs are backend-ready but not fully exposed in current React UI.

## 🌐 Live Demo

- **Marketing Website**: http://localhost:5000 (Landing page showcasing features)
- **Web Application**: http://localhost:3000 (Chat application)
- **API Backend**: http://localhost:8080 (RESTful API + WebSocket)

## Features

### User-Facing (Current UI)
- ✅ **User Authentication** - JWT-based register/login/refresh flow
- ✅ **Direct & Group Chats** - One-on-one and group conversations (2-100 participants)
- ✅ **Real-time Messaging** - WebSocket-driven updates in chat flows
- ✅ **Translation Display** - Message translation support rendered in message bubbles
- ✅ **Typing Indicators** - Typing start/stop events wired through WebSocket
- ✅ **User Profiles** - Display name and language preference support

### Backend-Available (Not Fully Surfaced in UI Yet)
- ✅ **Search APIs** - Message/chat/contact search endpoints
- ✅ **Presence APIs** - Presence status and activity update endpoints
- ✅ **Grammar APIs** - Text/message grammar analysis endpoints
- ✅ **Vocabulary APIs** - Vocabulary save/review/progress endpoints
- ✅ **Call APIs** - Call session and transcript endpoints

## Tech Stack

### Backend
- **Go 1.23+** with Gin framework
- **PostgreSQL 15** for data persistence
- **Redis 7** for caching and session management
- **WebSocket** for real-time communication
- **JWT** for authentication
- **Google Translate API** for translations (optional, works with mock translations)

### Frontend
- **React 18** with TypeScript
- **Vite** for fast development
- **Tailwind CSS** for styling
- **Zustand** for state management
- **Axios** for API calls

## Prerequisites

- Go 1.23 or higher
- Node.js 20 or higher (**Node 22+ recommended** for Capacitor 8 tooling)
- PostgreSQL 15 or higher
- Redis 7 or higher
- Docker & Docker Compose (optional, for containerized deployment)

## Quick Start (Without Docker)

### 1. Setup PostgreSQL

```powershell
# Install PostgreSQL (if not already installed)
# Create database
createdb messenger_dev

# Or using psql
psql -U postgres
CREATE DATABASE messenger_dev;
CREATE USER messenger WITH PASSWORD 'password';
GRANT ALL PRIVILEGES ON DATABASE messenger_dev TO messenger;
```

### 2. Setup Redis

```powershell
# Install Redis (if not already installed)
# Start Redis server
redis-server
```

### 3. Setup Backend

```powershell
cd backend

# Copy environment variables
Copy-Item .env.example .env

# Install Go dependencies
go mod download

# Run database migrations (automatic on first run)
go run cmd/server/main.go
```

The backend will start on `http://localhost:8080`

### 4. Setup Frontend

```powershell
cd frontend

# Install dependencies
npm install

# Start development server
npm run dev
```

The frontend will start on `http://localhost:3000`

### 5. Run Marketing Website (Optional)

```powershell
cd landing

# Using Node.js
node server.js

# Or using Python
python -m http.server 5000
```

The marketing website will start on `http://localhost:5000`

## Quick Start (With Docker)

### First Time Setup

```powershell
# Build all services
docker-compose build

# Start everything
docker-compose up -d
```

### 🔄 Development Workflow — Reloading After Code Changes

When you edit a `.tsx`, `.ts`, `.go`, or any source file, you need to rebuild and restart the affected service. Here's how:

#### Frontend Changes (`.tsx`, `.ts`, `.css` files)

```powershell
# 1. Rebuild the frontend image with your changes
docker-compose build frontend

# 2. Restart just the frontend container
docker-compose up -d frontend

# 3. Verify it's running
docker-compose ps

# Or do it in one command:
docker-compose up -d --build frontend
```

#### Backend Changes (`.go` files)

```powershell
# 1. Rebuild the backend image
docker-compose build backend

# 2. Restart the backend container
docker-compose up -d backend

# 3. Check the logs to make sure it started correctly
docker-compose logs --tail=10 backend

# One command:
docker-compose up -d --build backend
```

#### Both Frontend + Backend

```powershell
# Rebuild and restart everything
docker-compose up -d --build
```

#### Quick Reload (No Rebuild Needed)

If you only changed static assets or nginx config (not TypeScript/Go source):

```powershell
# Just restart the container (uses existing image)
docker-compose restart frontend
```

### Build Services

```powershell
# Build all services (first time or after code changes)
docker-compose build

# Build specific service
docker-compose build backend
docker-compose build frontend
```

### Start Services

```powershell
# Start all services
docker-compose up -d

# Start specific services
docker-compose up -d postgres redis
docker-compose up -d backend frontend

# Start with logs visible (without detached mode)
docker-compose up
```

### Stop Services

```powershell
# Stop all services (preserves data)
docker-compose stop

# Stop and remove containers (preserves volumes/data)
docker-compose down

# Stop and remove everything including volumes (WARNING: deletes data)
docker-compose down -v
```

### View Logs

```powershell
# View all logs
docker-compose logs -f

# View specific service logs
docker-compose logs -f backend
docker-compose logs -f frontend

# View last 50 lines
docker-compose logs --tail=50 backend
```

### Restart Services

```powershell
# Restart all services
docker-compose restart

# Restart specific service
docker-compose restart backend
```

Services will be available at:
- Frontend: http://localhost:3000
- Backend API: http://localhost:8080
- Backend Health: http://localhost:8080/health
- PostgreSQL: localhost:5432
- Redis: localhost:6379

## Running Services Locally

### Prerequisites Checklist
- ✅ Docker Desktop running
- ✅ PostgreSQL 15+ installed and running
- ✅ Redis 7+ installed and running
- ✅ Go 1.23+ installed
- ✅ Node.js 20+ installed (22+ recommended for Capacitor 8)

### Step-by-Step Service Startup

#### 1. Start PostgreSQL
```powershell
# Check if PostgreSQL is running
Get-Service postgresql*

# If not running, start it
Start-Service postgresql-x64-15  # Adjust version as needed

# Create database (first time only)
psql -U postgres -c "CREATE DATABASE messenger_dev;"
psql -U postgres -c "CREATE USER messenger WITH PASSWORD 'password';"
psql -U postgres -c "GRANT ALL PRIVILEGES ON DATABASE messenger_dev TO messenger;"
```

#### 2. Start Redis
```powershell
# If installed via Chocolatey or MSI
redis-server

# Or if using Docker for Redis only
docker run -d -p 6379:6379 redis:7-alpine

# Verify Redis is running
redis-cli ping  # Should return "PONG"
```

#### 3. Start Backend Server
```powershell
# Navigate to backend directory
cd backend

# Ensure .env file exists with correct settings
# Copy from .env.example if needed
if (-Not (Test-Path .env)) { Copy-Item .env.example .env }

# Install dependencies (first time)
go mod download

# Run backend server
go run cmd/server/main.go
```

The backend will:
- Auto-create database tables on first run
- Start HTTP server on `:8080`
- Start WebSocket server on `/ws`
- Connect to PostgreSQL and Redis

#### 4. Start Frontend Development Server
```powershell
# Open new terminal, navigate to frontend
cd frontend

# Install dependencies (first time only)
npm install

# Start Vite dev server
npm run dev
```

Frontend will be available at http://localhost:3000

#### 5. Start Android Mobile App (Capacitor)

```powershell
# Navigate to frontend app (mobile shell uses this web app)
cd frontend

# Install dependencies (first time only)
npm install

# Build web assets
npm run build

# Sync assets/plugins to Android project
npx cap sync android

# Run on connected emulator/device
npx cap run android --target=<deviceId>
```

Notes:
- Use `adb devices` to list target IDs.
- For Android emulator networking, API host mapping uses `10.0.2.2`.

### Build Commands

#### Build Backend Binary

```powershell
cd backend
go build -o chorus.exe ./cmd/server
```

#### Build Frontend for Production

```powershell
cd frontend
npm run build
# Output will be in dist/ folder
```

#### Build Android Mobile App

```powershell
cd frontend
npm run build
npx cap sync android

# Build debug APK
cd android
.\gradlew.bat assembleDebug
```

### Stop Services

```powershell
# Stop backend (Ctrl+C in the terminal where it's running)

# Stop frontend (Ctrl+C in the terminal where it's running)

# Stop mobile app (Ctrl+C in the terminal where it's running)

# Stop PostgreSQL service
Stop-Service postgresql-x64-15  # Adjust version as needed

# Stop Redis
# If running in terminal: Ctrl+C
# If running as Docker container:
docker stop <redis-container-id>
```

### Verify All Services Are Running

```powershell
# Check PostgreSQL
psql -U messenger -d messenger_dev -c "SELECT version();"

# Check Redis
redis-cli ping

# Check Backend
curl http://localhost:8080/health  # Or visit in browser

# Check Frontend
# Visit http://localhost:3000 in browser
```

### Common Startup Issues

**PostgreSQL Connection Failed:**
```powershell
# Check if PostgreSQL is running
Get-Service postgresql*
# Check connection string in backend/.env
# Verify credentials: psql -U messenger -d messenger_dev
```

**Redis Connection Failed:**
```powershell
# Check if Redis is running
redis-cli ping
# If not installed, use Docker: docker run -d -p 6379:6379 redis:7-alpine
```

**Backend Port Already in Use:**
```powershell
# Find process using port 8080
netstat -ano | findstr :8080
# Kill the process
taskkill /PID <PID> /F
```

**Frontend Build Errors:**
```powershell
# Clear node_modules and reinstall
rm -r node_modules
rm package-lock.json
npm install
```

## Usage

### 1. Register a New Account

1. Open http://localhost:3000
2. Click "Register"
3. Fill in your details:
   - Email
   - Display Name
   - Password (min 8 characters)
   - Native Language
   - Target Languages (languages you want to learn/see translations in)
4. Click "Register"

### 2. Create a Chat

1. Click "+ New Chat"
2. Choose "Direct Chat" or "Group Chat"
3. Search for users by username or display name
4. Select one or more users
5. For group chats, optionally add a group name
6. Click "Create Chat"

### 3. Send Messages

1. Select a chat from the left sidebar
2. Type your message in the input field
3. Press Enter or click "Send"
4. Messages will be automatically translated for recipients based on their target languages

### 4. View Translations

- Your messages appear in your native language
- Received messages show the original text
- If you have target languages set, you'll see translations below the original text

## API Documentation

Core auth/chat/message APIs are used by the current frontend. Additional Phase 2/3 APIs are implemented on backend and available for integration.

### Authentication

```http
POST /api/v1/auth/register
POST /api/v1/auth/login
POST /api/v1/auth/refresh
GET /api/v1/users/me
PUT /api/v1/users/me
GET /api/v1/users/search?q=username
```

### Chats

```http
GET /api/v1/chats
POST /api/v1/chats
GET /api/v1/chats/:chatId
PUT /api/v1/chats/:chatId
POST /api/v1/chats/:chatId/participants
DELETE /api/v1/chats/:chatId/participants/:userId
DELETE /api/v1/chats/:chatId/leave
```

### Messages

```http
GET /api/v1/chats/:chatId/messages?limit=50&before=messageId
POST /api/v1/chats/:chatId/messages
PUT /api/v1/chats/:chatId/read
GET /api/v1/messages/search?q=query&chatId=chatId
```

### Search (Phase 2)

```http
GET /api/v1/chats/search
GET /api/v1/contacts/search
```

### Presence (Phase 2)

```http
GET /api/v1/presence/:userId
PUT /api/v1/presence
POST /api/v1/presence/activity
```

### Grammar (Phase 3)

```http
POST /api/v1/grammar/analyze
POST /api/v1/grammar/analyze-text
GET /api/v1/grammar/suggestions
GET /api/v1/grammar/report
```

### Vocabulary (Phase 3)

```http
POST /api/v1/vocabulary
GET /api/v1/vocabulary
GET /api/v1/vocabulary/due
GET /api/v1/vocabulary/:id
POST /api/v1/vocabulary/practice
GET /api/v1/vocabulary/progress
DELETE /api/v1/vocabulary/:id
GET /api/v1/vocabulary/search
```

### Calls (Phase 3)

```http
POST /api/v1/calls/initiate
POST /api/v1/calls/:callId/end
GET /api/v1/calls/:callId
GET /api/v1/calls/:callId/transcript
GET /api/v1/calls/history
DELETE /api/v1/calls/:callId/transcript
GET /api/v1/calls/transcripts/search
POST /api/v1/calls/:callId/signal
```

### WebSocket

```
ws://localhost:8080/ws
```

Events:
- `new_message` - New message received
- `message_updated` - Message translation completed
- `chat_updated` - Chat participants or settings changed
- `user_typing` - User typing indicator

## Configuration

### Backend (.env)

```env
ENVIRONMENT=development
DATABASE_URL=postgres://messenger:password@localhost:5432/messenger_dev?sslmode=disable
REDIS_URL=localhost:6379
JWT_SECRET=your-secret-key-change-in-production
GOOGLE_TRANSLATE_API_KEY=  # Optional
PORT=8080
```

### Google Translate API (Optional)

To enable real translations (instead of mock translations):

1. Create a Google Cloud Platform account
2. Enable the Cloud Translation API
3. Create an API key
4. Add the API key to your `.env` file:
   ```
   GOOGLE_TRANSLATE_API_KEY=your-api-key-here
   ```

Without an API key, the app will use mock translations for common phrases.

## Project Structure

```
chorus/
├── backend/
│   ├── cmd/
│   │   └── server/
│   │       └── main.go          # Application entry point
│   ├── internal/
│   │   ├── config/              # Configuration
│   │   ├── database/            # Database connections & migrations
│   │   ├── handlers/            # HTTP handlers
│   │   ├── middleware/          # Middleware (auth, etc.)
│   │   ├── models/              # Data models
│   │   └── services/            # Business logic
│   ├── .env                     # Environment variables
│   ├── Dockerfile
│   ├── go.mod
│   └── go.sum
├── frontend/
│   ├── src/
│   │   ├── components/          # React components
│   │   ├── pages/               # Page components
│   │   ├── services/            # API & WebSocket services
│   │   ├── store/               # State management
│   │   ├── types/               # TypeScript types
│   │   ├── App.tsx              # Main app component
│   │   ├── main.tsx             # Entry point
│   │   └── index.css            # Global styles
│   ├── Dockerfile
│   ├── nginx.conf
│   ├── package.json
│   ├── tailwind.config.js
│   ├── tsconfig.json
│   └── vite.config.ts
└── docker-compose.yml
```

## Development

### Run Backend Tests

```powershell
cd backend
go test ./...
```

### Build Backend

```powershell
cd backend
go build -o chorus.exe ./cmd/server
```

### Build Frontend for Production

```powershell
cd frontend
npm run build
```

## Troubleshooting

### Backend won't start

- Check PostgreSQL is running: `psql -U messenger -d messenger_dev`
- Check Redis is running: `redis-cli ping`
- Verify `.env` file exists and has correct values
- Check port 8080 is not in use

### Frontend can't connect to backend

- Verify backend is running on port 8080
- Check browser console for errors
- Verify CORS settings in backend allow localhost:3000

### WebSocket connection fails

- Verify backend WebSocket endpoint is accessible
- Check browser console for WebSocket errors
- Ensure firewall/antivirus isn't blocking connections

### Translations not working

- If using Google Translate API, verify API key is correct
- Check backend logs for translation errors
- Without API key, mock translations will be used for common phrases

## Roadmap (Next Priorities)

- Frontend parity for backend-available features (presence/search/grammar/vocabulary/calls)
- Mobile UX hardening and emulator/device runtime polish
- Better observability and health/diagnostic reporting
- Optional scale-out and infrastructure automation improvements

## License

MIT

## Support

For issues and questions, please create an issue in the repository.
