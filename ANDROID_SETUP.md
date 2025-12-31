# Android Development Setup Guide

## Prerequisites

To run the Chorus mobile app on an Android emulator, you need to install the Android SDK and set up an Android Virtual Device (AVD).

## Installation Steps

### 1. Install Android Studio
Download and install Android Studio from: https://developer.android.com/studio

During installation, make sure to install:
- Android SDK
- Android SDK Platform
- Android Virtual Device

### 2. Configure Environment Variables

Add the following to your system environment variables:

**Windows:**
```powershell
setx ANDROID_HOME "%LOCALAPPDATA%\Android\Sdk"
setx PATH "%PATH%;%LOCALAPPDATA%\Android\Sdk\platform-tools;%LOCALAPPDATA%\Android\Sdk\tools;%LOCALAPPDATA%\Android\Sdk\emulator"
```

**macOS/Linux:**
```bash
export ANDROID_HOME=$HOME/Library/Android/sdk
export PATH=$PATH:$ANDROID_HOME/emulator
export PATH=$PATH:$ANDROID_HOME/platform-tools
```

### 3. Install Required SDK Packages

Open Android Studio and go to: **Tools > SDK Manager**

Install:
- Android SDK Platform 34 (or latest)
- Android SDK Build-Tools 34.0.0 (or latest)
- Android Emulator
- Android SDK Platform-Tools

Or via command line:
```bash
sdkmanager "platform-tools" "platforms;android-34" "build-tools;34.0.0" "system-images;android-34;google_apis;x86_64"
```

### 4. Create an Android Virtual Device (AVD)

**Option A: Using Android Studio**
1. Open Android Studio
2. Go to **Tools > Device Manager**
3. Click **Create Device**
4. Select a device definition (e.g., Pixel 6)
5. Select a system image (e.g., Android 14.0 with Google APIs)
6. Configure AVD settings
7. Click **Finish**

**Option B: Using Command Line**
```bash
avdmanager create avd -n ChorusEmulator -k "system-images;android-34;google_apis;x86_64" -d "pixel_6"
```

### 5. List Available AVDs

```bash
emulator -list-avds
```

### 6. Start the Emulator

**Option A: From Android Studio**
- Open Device Manager
- Click the play button next to your AVD

**Option B: From Command Line**
```bash
emulator -avd ChorusEmulator
```

## Running the Chorus Mobile App

### Prerequisites
Ensure the backend services are running:
```powershell
cd c:\dev\chorus
docker-compose up -d
```

### Start Metro Bundler
```bash
cd c:\dev\chorus\ChorusMobile
npm start
```

### Run on Android Emulator
In a new terminal:
```bash
cd c:\dev\chorus\ChorusMobile
npx react-native run-android
```

Or with npm script (if configured):
```bash
npm run android
```

## Troubleshooting

### "SDK location not found"
Create `android/local.properties` file:
```
sdk.dir=C:\\Users\\YourUsername\\AppData\\Local\\Android\\Sdk
```

### "Unable to load script from assets"
1. Make sure Metro bundler is running (`npm start`)
2. Try clearing cache: `npm start -- --reset-cache`
3. Reload app in emulator: Press `R` twice or `Ctrl+M` > `Reload`

### "Emulator: ERROR: x86_64 emulation currently requires hardware acceleration!"
Enable Intel HAXM or AMD-V/Hyper-V in BIOS

### "Error: spawn ./gradlew ENOENT"
Make sure you're in the ChorusMobile directory before running commands

### Port conflicts
If port 8081 (Metro) is in use:
```bash
npx react-native start --port 8082
npx react-native run-android --port 8082
```

## Testing the App

### 1. Registration Flow
1. Launch app (should show Login screen)
2. Tap "Don't have an account? Register"
3. Fill in:
   - Username: testuser
   - Email: test@example.com
   - Password: TestPass123!
   - Display Name: Test User
4. Tap "Create Account"
5. Should navigate to Chat List screen

### 2. Login Flow
1. Enter username or email
2. Enter password
3. Tap "Sign In"
4. Should navigate to Chat List screen

### 3. Chat Features
1. Chat List: Shows empty state or existing chats
2. Tap FAB (+) button to create new chat (when implemented)
3. Select chat to view messages
4. Send messages using input box
5. Receive real-time updates

### 4. Verify Backend Connection
The app connects to `http://10.0.2.2:8080` which is the special Android emulator address for host machine's localhost.

Check backend logs:
```bash
docker logs chorus-backend --follow
```

## Development Tips

### Live Reload
- Enable Fast Refresh in app: Shake device > Enable Fast Refresh
- Changes to React components will hot-reload automatically

### Debug Menu
- Android Emulator: Press `Ctrl+M` or `Cmd+M`
- Options: Reload, Debug, Show Inspector, etc.

### React Native Debugger
Install standalone debugger for better DX:
```bash
npm install -g react-native-debugger
```

### Viewing Logs
```bash
# All logs
npx react-native log-android

# Filtered
npx react-native log-android | grep "Chorus"
```

## Alternative: Physical Device

### Enable Developer Options
1. Go to Settings > About Phone
2. Tap "Build Number" 7 times
3. Go back to Settings > Developer Options
4. Enable "USB Debugging"

### Connect via USB
```bash
# List devices
adb devices

# Run on device
npx react-native run-android --device
```

### Connect via WiFi (same network as PC)
```bash
# Get device IP
adb shell ip addr show wlan0

# Connect
adb tcpip 5555
adb connect <DEVICE_IP>:5555

# Run
npx react-native run-android
```

## Next Steps

Once the emulator is set up and app is running:
1. Test all authentication flows
2. Create and join chats
3. Send and receive messages
4. Test real-time features (typing indicators, new messages)
5. Test translation features
6. Run comprehensive UI/UX testing

## Resources

- [React Native Environment Setup](https://reactnative.dev/docs/environment-setup)
- [Android Studio Downloads](https://developer.android.com/studio)
- [React Native Debugging](https://reactnative.dev/docs/debugging)
- [Android Emulator Guide](https://developer.android.com/studio/run/emulator)
