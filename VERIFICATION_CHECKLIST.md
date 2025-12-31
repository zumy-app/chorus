# Chorus Phase 1 - Verification Checklist

Use this checklist to verify your installation and test all features.

## ✅ Installation Verification

### Prerequisites
- [ ] Go 1.21+ installed (`go version`)
- [ ] Node.js 18+ installed (`node --version`)
- [ ] PostgreSQL 15+ installed (`psql --version`)
- [ ] Redis 7+ installed or running (`redis-cli ping` returns PONG)
- [ ] Docker installed (optional) (`docker --version`)

### Database Setup
- [ ] PostgreSQL service running
- [ ] Database `messenger_dev` created
- [ ] User `messenger` with password created
- [ ] Can connect: `psql -U messenger -d messenger_dev`

### Redis Setup
- [ ] Redis service running
- [ ] Can ping: `redis-cli ping` returns PONG

### Backend Setup
- [ ] Navigated to `backend/` directory
- [ ] File `go.mod` exists
- [ ] File `.env` exists (copied from `.env.example`)
- [ ] Dependencies downloaded: `go mod download` (no errors)
- [ ] Backend compiles: `go build ./cmd/server` (no errors)

### Frontend Setup
- [ ] Navigated to `frontend/` directory
- [ ] File `package.json` exists
- [ ] Dependencies installed: `npm install` (no errors)
- [ ] Frontend builds: `npm run build` (no errors)

## 🚀 Application Launch

### Starting Backend
- [ ] Backend started: `go run cmd/server/main.go`
- [ ] No error messages in console
- [ ] See "Server starting on port 8080"
- [ ] See "Database connected successfully"
- [ ] See "Database migrations completed successfully"
- [ ] See "Redis connected successfully"
- [ ] Health check works: http://localhost:8080/health returns `{"status":"healthy"}`

### Starting Frontend
- [ ] Frontend started: `npm run dev`
- [ ] No error messages in console
- [ ] See "Local: http://localhost:3000"
- [ ] Browser opens or can manually navigate to http://localhost:3000
- [ ] Login page loads correctly

## 🧪 Functional Testing

### User Registration
- [ ] Navigate to http://localhost:3000/register
- [ ] Fill in registration form:
  - [ ] Username (min 3 characters)
  - [ ] Email (valid email format)
  - [ ] Display Name
  - [ ] Password (min 8 characters)
  - [ ] Select Native Language
  - [ ] Select at least one Target Language
- [ ] Click "Register"
- [ ] Registration succeeds (redirects to chat page)

### User Login
- [ ] Navigate to http://localhost:3000/login
- [ ] Enter username and password
- [ ] Click "Login"
- [ ] Login succeeds (redirects to chat page)
- [ ] See user info in sidebar (display name, username, languages)

### Create Test User 2 (for testing chat)
- [ ] Logout from first user
- [ ] Register second user with different credentials
- [ ] Different target language preferred
- [ ] Login succeeds

### User Search
- [ ] Login as User 1
- [ ] Click "+ New Chat"
- [ ] Search for User 2 by username
- [ ] User 2 appears in search results
- [ ] Can select User 2

### Direct Chat Creation
- [ ] In New Chat modal, select "Direct Chat"
- [ ] Search and select another user
- [ ] Click "Create Chat"
- [ ] Chat appears in chat list
- [ ] Chat is selected automatically

### Group Chat Creation
- [ ] Click "+ New Chat"
- [ ] Select "Group Chat"
- [ ] Enter group name
- [ ] Search and select multiple users (2+)
- [ ] Click "Create Chat"
- [ ] Group chat appears in chat list
- [ ] Shows participant count

### Send Message
- [ ] Select a chat
- [ ] Type message in input box
- [ ] Press Enter or click Send
- [ ] Message appears in chat area
- [ ] Message has timestamp
- [ ] Message shows as sent

### Receive Message (Real-time)
- [ ] Open app in second browser/window
- [ ] Login as User 2
- [ ] See the chat with User 1
- [ ] User 1 sends a message
- [ ] Message appears in User 2's chat (without refresh)
- [ ] Real-time update works

### Translation Feature
- [ ] User 1 (English → Spanish) sends "Hello"
- [ ] User 2 (Spanish → English) receives message
- [ ] User 2 sees:
  - [ ] Original text: "Hello"
  - [ ] Translation badge showing target language
  - [ ] Translated text (if translation available)

### Typing Indicators
- [ ] User 1 starts typing
- [ ] User 2 sees typing indicator (if implemented)
- [ ] User 1 stops typing
- [ ] Typing indicator disappears

### Message History
- [ ] Send multiple messages (5+)
- [ ] Scroll up in chat
- [ ] All messages visible
- [ ] Messages in correct order (newest at bottom)
- [ ] Timestamps correct

### Chat List
- [ ] Multiple chats visible in sidebar
- [ ] Last message preview shown
- [ ] Time of last message shown
- [ ] Can switch between chats
- [ ] Active chat highlighted

### Profile Update
- [ ] Click on user profile area
- [ ] Update display name
- [ ] Add/remove target languages
- [ ] Save changes
- [ ] Changes reflected in UI

### User Search in Chat
- [ ] Create new chat
- [ ] Search for users with partial username
- [ ] Results appear dynamically
- [ ] Can select from results

### WebSocket Persistence
- [ ] Send a message
- [ ] Close browser tab
- [ ] Reopen application
- [ ] Login again
- [ ] Messages still visible
- [ ] Can send new messages

### Token Refresh
- [ ] Login
- [ ] Wait for access token to expire (or manually expire it)
- [ ] Make an API call (send message)
- [ ] Token refreshes automatically
- [ ] Request succeeds

## 🔒 Security Testing

### Authentication
- [ ] Cannot access /chat without login (redirects to /login)
- [ ] Cannot access API without token (401 error)
- [ ] Invalid credentials rejected
- [ ] Password not visible in network requests
- [ ] JWT token in Authorization header

### Input Validation
- [ ] Short username rejected (< 3 chars)
- [ ] Short password rejected (< 8 chars)
- [ ] Invalid email format rejected
- [ ] Empty messages cannot be sent
- [ ] SQL injection attempts fail (try `'; DROP TABLE users;--`)

### Authorization
- [ ] Cannot read messages from chats you're not in
- [ ] Cannot add users to chats you're not in
- [ ] Cannot delete other users' messages

## 🐳 Docker Testing (Optional)

### Docker Compose
- [ ] Run `docker-compose up -d`
- [ ] All containers start:
  - [ ] postgres
  - [ ] redis
  - [ ] backend
  - [ ] frontend
- [ ] Check health: `docker-compose ps` (all healthy)
- [ ] Access frontend: http://localhost:3000
- [ ] Access backend: http://localhost:8080/health
- [ ] All features work same as manual setup
- [ ] Logs visible: `docker-compose logs -f`
- [ ] Can stop: `docker-compose down`

## 🔧 Error Handling

### Network Errors
- [ ] Stop backend while frontend running
- [ ] Try to send message
- [ ] Error displayed to user
- [ ] App doesn't crash

### Database Errors
- [ ] Stop PostgreSQL
- [ ] Backend shows error message
- [ ] Restart PostgreSQL
- [ ] Backend reconnects

### WebSocket Errors
- [ ] Disconnect WebSocket
- [ ] App attempts reconnection
- [ ] WebSocket reconnects automatically
- [ ] Messages sync after reconnect

## 📊 Performance Testing

### Message Load
- [ ] Send 50+ messages
- [ ] UI remains responsive
- [ ] Scrolling is smooth
- [ ] No memory leaks

### Multiple Chats
- [ ] Create 10+ chats
- [ ] Switch between chats
- [ ] Fast switching (< 1 second)
- [ ] No lag

### Translation Performance
- [ ] Send message with translation
- [ ] Translation appears within 2 seconds
- [ ] Cached translations instant on repeat

## 📱 Browser Compatibility

### Desktop Browsers
- [ ] Chrome (latest)
- [ ] Firefox (latest)
- [ ] Edge (latest)
- [ ] Safari (if available)

### Mobile Browsers (Responsive)
- [ ] Chrome Mobile
- [ ] Safari Mobile (if available)
- [ ] UI adapts to mobile screen

## 📝 Documentation Review

- [ ] README.md present and clear
- [ ] INSTALLATION.md has all steps
- [ ] QUICK_START.md helpful
- [ ] IMPLEMENTATION_SUMMARY.md complete
- [ ] GET_STARTED.md welcoming
- [ ] Code comments present
- [ ] API endpoints documented

## 🎯 Phase 1 Requirements

All Phase 1 requirements from design.md:
- [x] User authentication (register, login, JWT)
- [x] Direct messaging (1-to-1 chats)
- [x] Group messaging (2-100 participants)
- [x] Real-time message delivery (WebSocket)
- [x] Automatic translation
- [x] Multiple target languages per user
- [x] Message persistence (PostgreSQL)
- [x] Message search (full-text)
- [x] User profiles with language preferences
- [x] User search
- [x] Chat creation and management
- [x] Typing indicators
- [x] Docker deployment
- [x] Production-ready infrastructure

## ✨ Final Checks

- [ ] No console errors in browser
- [ ] No errors in backend logs
- [ ] All features working
- [ ] Performance acceptable
- [ ] UI looks good
- [ ] Ready for demo/use

---

## 🎉 Verification Complete

If all items are checked, congratulations! Your Chorus Phase 1 implementation is fully functional and ready to use.

### Issues Found?

1. Check error messages in:
   - Browser console (F12)
   - Backend terminal
   - PostgreSQL logs
   - Redis logs

2. Refer to troubleshooting in:
   - README.md
   - INSTALLATION.md
   - QUICK_START.md

3. Verify:
   - All services running
   - Environment variables correct
   - Database accessible
   - Redis accessible

### Everything Working?

Great! Now you can:
1. Demo the application
2. Create real accounts
3. Invite others to test
4. Start planning Phase 2 features
5. Deploy to production

---

**Status:** Phase 1 Complete ✅
