# Chorus - Multilingual Messenger

A real-time messaging application with built-in translation features for multilingual conversations.

## Features (Phase 1)

- ✅ **User Authentication** - Secure JWT-based authentication
- ✅ **Direct & Group Chats** - Support for one-on-one and group conversations (2-100 participants)
- ✅ **Real-time Messaging** - WebSocket-based instant messaging
- ✅ **Automatic Translation** - Messages automatically translated to user's target languages
- ✅ **Multi-language Support** - Support for English, Spanish, French, German, Italian, Portuguese, Japanese, Korean, Chinese
- ✅ **User Profiles** - Customizable display names and language preferences
- ✅ **Message Search** - Full-text search across messages
- ✅ **Typing Indicators** - See when others are typing
- ✅ **Responsive UI** - Modern React-based interface

## Tech Stack

### Backend
- **Go 1.21+** with Gin framework
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

- Go 1.21 or higher
- Node.js 18 or higher
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

## Quick Start (With Docker)

```powershell
# Build and start all services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down
```

Services will be available at:
- Frontend: http://localhost:3000
- Backend API: http://localhost:8080
- PostgreSQL: localhost:5432
- Redis: localhost:6379

## Running Services Locally

### Prerequisites Checklist
- ✅ Docker Desktop running
- ✅ PostgreSQL 15+ installed and running
- ✅ Redis 7+ installed and running
- ✅ Go 1.21+ installed
- ✅ Node.js 18+ installed

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

#### 5. (Optional) Start Mobile App
```powershell
# Navigate to mobile directory
cd ChorusMobile

# Install dependencies (first time only)
npm install

# For iOS (macOS only)
npm run ios

# For Android (requires Android Studio & emulator/device)
npm run android

# For Web
npm run web
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
   - Username (min 3 characters)
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

## Future Phases

### Phase 2: Multi-Device & Scaling
- Horizontal scaling with Kubernetes
- Redis Pub/Sub for cross-instance messaging
- Offline message delivery
- Advanced monitoring and metrics

### Phase 3: Advanced Features
- Grammar analysis and corrections
- Vocabulary management for language learning
- Voice and video calls
- Message reactions and emoji support
- File sharing (images, documents)
- Desktop application (Tauri)

## License

MIT

## Support

For issues and questions, please create an issue in the repository.
