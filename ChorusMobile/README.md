# Chorus Mobile

React Native (0.83) client for **Chorus** — a chat app with AI translation and language learning, built for Android and iOS.

## Features

- **Auth**: register / login with JWT tokens, auto-refresh, token persistence (AsyncStorage on native, localStorage on web)
- **Chat list**: pull-to-refresh, last-message preview, unread badges, real-time updates over WebSocket
- **Chat**: send/receive messages, typing indicators, mark-as-read, translation display in your native language, and a per-message **Translate** button that requests a translation on demand
- **New chat**: search users and start a direct conversation
- **Profile**: edit display name, native language, and target (learning) languages, log out

## Screens & Navigation

| Screen | Route | Purpose |
|--------|-------|---------|
| LoginScreen | `Login` | Sign in |
| RegisterScreen | `Register` | Create account |
| ChatListScreen | `ChatList` | Conversation list + settings button |
| ChatScreen | `Chat` | Message thread with translations |
| NewChatScreen | `NewChat` | Search users → start direct chat |
| ProfileScreen | `Profile` | Profile + language settings + logout |

Navigation uses `@react-navigation/native-stack` (native stack screens).

## Backend connection

Dev endpoints default to the host machine:

- **Android emulator**: `http://10.0.2.2:8080/api/v1` (and `ws://10.0.2.2:8080/ws`)
- **iOS simulator / web**: `http://localhost:8080/api/v1` (and `ws://localhost:8080/ws`)

Override by editing the base URLs in `src/services/api.ts` and `src/services/websocket.ts`. A deployed build should point at the production API (`https://api.chorus.talk/...`).

The backend must be running (see repo root: `docker-compose up -d`, or `backend` + `docker-compose.dev.yml`).

## Running

```sh
npm install
# Android (requires JDK 17–21 + Android SDK; see ANDROID_SETUP.md at repo root)
npm run android
# iOS (macOS only; requires CocoaPods)
bundle install && bundle exec pod install
npm run ios
# Web (react-native-web, via Expo)
npm run web
```

## Validation

```sh
npm test        # jest unit/snapshot tests
npm run lint    # eslint
npx tsc --noEmit
```

## Layout

```
App.tsx                  navigation + auth gating
src/screens/             Login, Register, ChatList, Chat, NewChat, Profile
src/services/api.ts      axios client (auth, chats, messages, translate)
src/services/websocket.ts WebSocket client (reconnect, typing, events)
src/types/index.ts       shared types + supported languages
src/utils/storage.ts     AsyncStorage/localStorage adapter
```
