// Bare React Native project — Expo is only used for the optional `web` target
// (`npx expo start --web`). Its native modules must NOT be autolinked into the
// Android/iOS builds, which are plain RN 0.83 projects.
module.exports = {
  dependencies: {
    expo: { platforms: { android: null, ios: null } },
    'expo-modules-core': { platforms: { android: null, ios: null } },
    'expo-modules-autolinking': { platforms: { android: null, ios: null } },
  },
};
