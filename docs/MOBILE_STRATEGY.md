# Mobile Strategy

## Platform

Chorus mobile is an Expo-managed React Native application in `mobile/`, delivered on Android and iOS from one TypeScript codebase. Expo Router provides navigation, `expo-secure-store` holds session tokens, and the app talks directly to the Chorus REST and WebSocket APIs.

The mobile app is not a Capacitor wrapper and does not require a checked-in native `android/` or `ios/` project for normal development. Use Expo tooling locally and EAS Build for distributable binaries.

## Environments and configuration

The public runtime API endpoint is configured with:

```env
EXPO_PUBLIC_API_URL=https://api.example.com/api/v1
```

`EXPO_PUBLIC_*` values are embedded in the client build. They must contain only public configuration; never put JWT secrets, database credentials, service-account keys, or push-provider credentials in them.

When `EXPO_PUBLIC_API_URL` is absent, the app falls back to `http://10.0.2.2:8080/api/v1` for the Android emulator. This fallback is for local Android development only. iOS simulators can reach a locally running API at `http://localhost:8080/api/v1`; physical devices need an HTTPS endpoint reachable from the device.

The WebSocket URL is derived from the API URL (`https` becomes `wss`, `http` becomes `ws`) and uses `/ws`. Production API endpoints therefore require HTTPS and a working WSS endpoint.

## Build and distribution

`mobile/eas.json` defines three profiles:

- `development`: internal development-client builds.
- `preview`: internally distributed Android APKs for QA.
- `production`: store-ready Android App Bundles with EAS-managed version-code increments.

For iOS, EAS uses the same profile configuration and produces an archive suitable for TestFlight/App Store submission once Apple signing credentials are configured. Build commands should specify the intended platform and profile:

```bash
cd mobile
npx eas build --platform android --profile preview
npx eas build --platform all --profile production
```

Use `eas credentials` or the EAS dashboard to create and manage Android keystores and iOS distribution credentials. Do not commit keystores, Apple certificates, provisioning profiles, Firebase configuration files, or `EXPO_TOKEN`. CI receives `EXPO_TOKEN` only through GitHub Actions secrets.

## Push notifications

Push notifications are not release-ready merely because a build can be produced. Before enabling them, configure the Expo push integration for each platform:

- Android: register an FCM v1 service account in the Expo project and add the Android package configuration required by the chosen push setup.
- iOS: create and upload an APNs authentication key for the production bundle identifier.
- Use real devices to obtain and exercise tokens; simulators/emulators are not sufficient push validation.
- Treat push tokens as personal data: associate them with the authenticated user, rotate or remove them on logout/account deletion, and document their purpose and retention.

## Privacy and account deletion

Before store submission, publish a privacy policy and make it reachable from the app listing and app settings. It must accurately describe account data, messages, translations, diagnostics, secure token storage, and any push-token processing.

The backend exposes authenticated `DELETE /api/v1/users/me`, and the mobile API client clears its locally stored session after a successful response. A release must expose a clear in-app account-deletion path, explain what is deleted or retained and for how long, and provide a support path when deletion cannot be completed in-app. Verify the implemented UI and backend deletion semantics before representing this as complete in store metadata.

## Release gate

Before creating a production build:

1. Set the production `EXPO_PUBLIC_API_URL` in the EAS/CI environment and verify no secret appears in the resolved Expo public config.
2. Run `npm ci`, `npm run typecheck`, `npm test`, and `npx expo config --type public` from `mobile/`.
3. Test registration, login, refresh/logout, API failures, WebSocket reconnection, and account deletion against the production-like API.
4. Verify Android and iOS on physical devices, including network-loss recovery and foreground/background transitions.
5. Validate release signing, app identifiers, versioning, store assets, privacy disclosures, support URL, and account-deletion instructions.
6. If push is enabled, send and receive notifications on real Android and iOS devices and verify token cleanup.

The GitHub Actions mobile workflow runs install, type checking, unit tests, and public Expo-config validation. The manual EAS workflow starts an Android preview build; it is not a production release or store-submission workflow.
