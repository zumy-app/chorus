# Android Setup for Expo-Managed Chorus

Chorus mobile lives in `mobile/` and is an Expo-managed React Native app. Use Expo commands from that directory; do not use legacy `react-native run-android`, Metro-only commands, Gradle wrappers, or Capacitor commands.

## Prerequisites

- Node.js 22 (the repository CI version)
- Android Studio with the Android SDK, Platform-Tools, Android Emulator, and an emulator system image
- A recent Android emulator or a USB-debuggable physical device
- The Chorus backend running locally or an accessible HTTPS development API

Install the JavaScript dependencies:

```bash
cd mobile
npm ci
```

## Android SDK setup

Install Android Studio from <https://developer.android.com/studio>. In **Tools > SDK Manager**, install Android SDK Platform-Tools, Android Emulator, and at least one current Android SDK platform and Google APIs system image. In **Tools > Device Manager**, create a device such as a Pixel profile and start it.

Configure the SDK for your shell if Android Studio did not do this automatically:

```bash
# macOS
export ANDROID_HOME="$HOME/Library/Android/sdk"
export PATH="$PATH:$ANDROID_HOME/emulator:$ANDROID_HOME/platform-tools"
```

```powershell
# Windows PowerShell (current session)
$env:ANDROID_HOME = "$env:LOCALAPPDATA\Android\Sdk"
$env:Path += ";$env:ANDROID_HOME\emulator;$env:ANDROID_HOME\platform-tools"
```

Confirm that a device is visible:

```bash
adb devices
emulator -list-avds
```

## Configure the API endpoint

For the Android emulator, the app defaults to:

```env
EXPO_PUBLIC_API_URL=http://10.0.2.2:8080/api/v1
```

`10.0.2.2` reaches the host machine from an Android emulator. To use another API, set `EXPO_PUBLIC_API_URL` in the environment that starts Expo or in the appropriate EAS environment. The value is public build configuration, not a secret. Production builds must use an HTTPS API endpoint; the corresponding WebSocket endpoint is derived as WSS.

Physical devices cannot use `10.0.2.2`. Use an HTTPS endpoint reachable from the device, or a temporary LAN/tunnel endpoint appropriate for local development.

## Run the app

Start the backend first, then launch the Android target:

```bash
cd mobile
npm run android
```

Expo starts its development server and opens the running emulator or a connected device. If it does not select a target, start an emulator in Android Studio or connect a device with USB debugging enabled, then rerun the command.

To inspect the resolved public configuration:

```bash
npx expo config --type public
```

Do not place server credentials, signing material, `EXPO_TOKEN`, or push-provider credentials in `EXPO_PUBLIC_*` variables; public Expo configuration is bundled into the app.

## Build distributable Android artifacts

EAS Build creates the binary remotely using the project profiles in `mobile/eas.json`:

```bash
cd mobile
npx eas build --platform android --profile preview
npx eas build --platform android --profile production
```

The `preview` profile creates an internally distributable APK. The `production` profile creates an Android App Bundle for Play Store delivery and increments the Android version code through EAS. Use `npx eas credentials` or the EAS dashboard to manage the Android signing key. Keep signing keys and the EAS access token out of the repository.

## Troubleshooting

**No Android device found**

- Start an AVD from Android Studio Device Manager, or reconnect the physical device.
- Run `adb devices`; authorize the device if it is listed as unauthorized.

**The app cannot reach the local API**

- Confirm the backend listens on port 8080.
- On an Android emulator, use `10.0.2.2`, not `localhost`.
- On a physical device, use a device-reachable HTTPS/LAN endpoint rather than `10.0.2.2`.

**Configuration did not change**

- Restart Expo after changing environment variables.
- Run `npx expo config --type public` to verify the resolved public value.

**A production build fails**

- Run `npm ci`, `npm run typecheck`, and `npm test` locally.
- Check that the EAS project has access to its Android signing credentials and that CI has a valid `EXPO_TOKEN` secret.

## Release checks

Before Play submission, test a preview build on real hardware and verify authentication, token refresh/logout, chat recovery after network loss, account deletion, privacy-policy links, and any enabled push-notification flow. See [docs/MOBILE_STRATEGY.md](docs/MOBILE_STRATEGY.md) for the full Android/iOS release gate and credential guidance.
