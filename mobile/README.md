# Chorus Mobile

React Native (0.83) client for **Chorus** — a chat app with AI translation and language learning, built for Android and iOS.

## Quick Start (one script)

From the repo root, a single command sets up everything — Docker infra, Go backend, React frontend, Android emulator, Metro bundler, and installs the app:

```powershell
.\start-dev.ps1 -Mobile
```

This will:
1. Start Docker (PostgreSQL + Redis)
2. Start the Go backend on `http://localhost:8080` (air hot-reload)
3. Start the React frontend on `http://localhost:3000` (Vite HMR)
4. Detect or start an Android emulator (AVD)
5. Install npm deps and start Metro bundler
6. Build and install the app on the emulator

**Prerequisites:** Docker Desktop, Go, Node.js, Android Studio (SDK + AVD). See `ANDROID_SETUP.md` for first-time setup.

### Other modes

```powershell
.\start-dev.ps1                  # web + backend only (no mobile)
.\start-dev.ps1 -SplitWindows    # separate windows for backend/frontend
```

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

Override by setting `EXPO_PUBLIC_API_URL` / `VITE_API_URL` environment variables, or editing `packages/shared/src/config.ts`. A deployed build should point at the production API (`https://api.chorus.talk/...`).

The backend must be running (see repo root: `docker-compose up -d`, or use `.\start-dev.ps1`).

## Running manually

```sh
cd mobile
npm install
```

### Android

Requires JDK 17–21, the Android SDK, and the backend running on your machine.
See `ANDROID_SETUP.md` at the repo root for first-time SDK/AVD setup.

Start the Metro bundler in one terminal:

```sh
npm start
```

Then build and launch on the connected device/emulator:

```sh
npm run android          # = react-native run-android
```

#### Android emulator (AVD)

1. Start the backend on your machine (see above).
2. Boot an emulator from Android Studio (AVD Manager) or the CLI:

   ```sh
   emulator -list-avds                    # list available AVDs
   emulator -avd <your_avd_name>          # boot one
   ```

3. Confirm it's connected: `adb devices`
4. Install & launch: `npm run android`

   The emulator reaches your machine's backend via the loopback alias
   `10.0.2.2` — already the default in `packages/shared/src/config.ts`.

#### Physical device via USB (ADB)

1. On the phone enable **Developer options** → **USB debugging**, connect it
   with a USB cable, and accept the "Allow USB debugging" prompt.
2. Verify the device is visible: `adb devices`
3. Physical devices cannot use `10.0.2.2`. Point the app at your machine's
   **LAN IP** (phone and PC on the same Wi-Fi):
   - Find your IP: `ipconfig` (Windows) → `IPv4 Address`, e.g. `192.168.1.42`
   - Edit `packages/shared/src/config.ts` → set `EMULATOR_ANDROID_ORIGIN` to `http://<your-ip>:8080`
   - Make sure the backend is reachable on that IP and the firewall allows port 8080.
4. Install & launch:

   ```sh
   npm run android
   # or target a specific device:
   npx react-native run-android --deviceId <serial>
   ```

#### Physical device over Wi-Fi (wireless ADB, Android 11+)

1. Do the USB steps once to enable Developer options / USB debugging.
2. On Android 11+: enable **Wireless debugging**, open **Pair device with
   pairing code**, then:

   ```sh
   adb pair <phone_ip>:<pair_port>       # enter the pairing code shown on device
   adb connect <phone_ip>:<connect_port> # port from the Wireless debugging screen
   ```

   Android 10 and older: connect over USB once, then
   `adb tcpip 5555` and `adb connect <phone_ip>:5555`.
3. Confirm: `adb devices` shows the device.
4. Same as USB — the app must reach the backend via your machine's LAN IP
   (see step 3 of "Physical device via USB").

#### Build a debug APK (no device attached)

```sh
cd android
.\gradlew.bat assembleDebug
# output: android/app/build/outputs/apk/debug/app-debug.apk
```

### iOS (macOS only)

```sh
bundle install && bundle exec pod install
npm run ios
```

### Web (react-native-web, via Expo)

```sh
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
src/services/api.ts      axios client (auth, chats, messages, translate) — thin wrapper over @chorus/shared
src/services/websocket.ts WebSocket client (reconnect, typing, events) — thin wrapper over @chorus/shared
src/utils/storage.ts     AsyncStorage/localStorage adapter
packages/shared (repo root) shared types, config, API + WebSocket clients used by web and mobile
```
