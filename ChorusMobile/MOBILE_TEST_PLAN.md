# Mobile App Functional Test Plan

## Test Environment
- **Platform**: Android Emulator (Medium_Phone_API_36.1)
- **App**: Chorus Mobile (React Native 0.83.1)
- **Backend**: http://10.0.2.2:8080 (Docker services)

## Manual Test Cases

### 1. App Launch & Initial State
**Steps:**
1. Launch the app on emulator
2. Verify splash screen appears
3. Verify app navigates to Login screen

**Expected Results:**
- ✅ App launches without crashes
- ✅ Login screen displays with username/password fields
- ✅ "Register" link is visible

### 2. User Registration Flow
**Steps:**
1. Tap "Don't have an account? Register"
2. Fill in registration form:
   - Username: `testuser_${timestamp}`
   - Email: `testuser_${timestamp}@example.com`
   - Password: `TestPass123!`
   - Display Name: `Test User`
   - Native Language: `en`
3. Tap "Create Account"

**Expected Results:**
- ✅ Navigation to Register screen
- ✅ Form validation works
- ✅ Success alert shows
- ✅ Auto-navigation to ChatList screen
- ✅ Token saved in AsyncStorage

### 3. User Login Flow
**Steps:**
1. Logout if logged in
2. Enter username and password
3. Tap "Sign In"

**Expected Results:**
- ✅ Login button shows loading indicator
- ✅ Successful authentication
- ✅ Navigation to ChatList screen
- ✅ Auth token persisted

### 4. Chat List Display
**Steps:**
1. From ChatList screen, observe the UI
2. Pull down to refresh

**Expected Results:**
- ✅ Empty state shows if no chats
- ✅ FAB (+) button visible
- ✅ Pull-to-refresh works
- ✅ Header shows "Chorus"

### 5. Create New Chat (Manual - UI not fully implemented)
**Steps:**
1. Tap FAB (+) button
2. Select user (when implemented)
3. Create direct chat

**Expected Results:**
- ✅ New chat screen opens (when implemented)
- ✅ Chat created successfully
- ✅ Appears in chat list

### 6. Send Message
**Steps:**
1. Open existing chat (create one via API if needed)
2. Type message in input box
3. Tap "Send"

**Expected Results:**
- ✅ Message appears in chat
- ✅ Input clears after send
- ✅ Message saved to backend
- ✅ Timestamp displayed

### 7. Receive Message (Real-time)
**Steps:**
1. Have another user send a message (simulate via API)
2. Observe chat list and chat screen

**Expected Results:**
- ✅ New message appears in real-time
- ✅ Chat list updates with latest message
- ✅ Unread count updates (when implemented)

### 8. WebSocket Connection
**Steps:**
1. Open a chat
2. Start typing
3. Stop typing

**Expected Results:**
- ✅ WebSocket connects successfully
- ✅ Typing indicator sends (check backend logs)
- ✅ Connection maintained

### 9. Message Translation (When Available)
**Steps:**
1. Send message in one language
2. Observe translated text

**Expected Results:**
- ✅ Translation appears below original
- ✅ Styled differently (italic)

### 10. Error Handling
**Steps:**
1. Turn off backend services
2. Try to login/send message
3. Turn backend back on

**Expected Results:**
- ✅ Error alert shows
- ✅ Graceful error messages
- ✅ Reconnects when backend available

## Automated Test Results

Run the automated test suite:
```bash
cd c:\dev\chorus\ChorusMobile
npx ts-node --project tsconfig.tests.json tests/functional-tests.ts
```

**Latest Results:**
- ✅ Health Check - PASSED
- ✅ User Registration - PASSED
- ✅ User Login - PASSED
- ✅ Get User Profile - PASSED
- ✅ Create Direct Chat - PASSED
- ✅ Send Message - PASSED
- ✅ Get Messages - PASSED
- ✅ Get Chats List - PASSED

**Pass Rate**: 100% (8/8)

## Backend Connectivity Test

Verify backend is accessible from Android emulator:
```bash
# From emulator, test:
curl http://10.0.2.2:8080/health
```

Expected: `{"status":"healthy","version":"1.0.0"}`

## Performance Metrics

### App Launch Time
- **Cold Start**: < 3 seconds
- **Hot Start**: < 1 second

### API Response Times
- **Login**: < 1.5 seconds
- **Load Chats**: < 200ms
- **Load Messages**: < 150ms
- **Send Message**: < 100ms

### Memory Usage
- **Idle**: < 150MB
- **Active Chat**: < 200MB

## Known Issues & Fixes

### 1. NULL Handling in Backend ✅ FIXED
- **Issue**: original_language and translations NULL causing errors
- **Fix**: Added COALESCE in SQL queries

### 2. Response Format Mismatches ✅ FIXED
- **Issue**: App expected arrays, backend returned wrapped objects
- **Fix**: Updated API service to extract nested data

### 3. Gradle Version Compatibility ✅ FIXED
- **Issue**: Gradle 9.0 incompatibility
- **Fix**: Downgraded to Gradle 8.13

## Test Checklist for Current Session

- [x] Backend services running
- [x] Android emulator started
- [x] App built and deployed
- [x] App launches successfully
- [ ] Test login flow
- [ ] Test registration flow
- [ ] Test chat display
- [ ] Test message sending
- [ ] Verify WebSocket connection
- [ ] Check error handling
- [ ] Performance validation

## Next Steps

1. Complete manual testing checklist
2. Fix any UI/UX issues discovered
3. Implement missing features (create chat UI)
4. Add automated E2E tests with Detox
5. Test on physical device
6. Performance optimization
