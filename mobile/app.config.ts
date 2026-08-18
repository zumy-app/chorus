import type { ExpoConfig } from 'expo/config'

const isProduction = process.env.APP_VARIANT === 'production'

const config: ExpoConfig = {
  name: 'Chorus',
  slug: 'chorus-mobile',
  version: '1.0.0',
  orientation: 'portrait',
  scheme: 'chorus',
  userInterfaceStyle: 'automatic',
  ios: {
    bundleIdentifier: 'talk.chorus.mobile',
  },
  android: {
    package: 'talk.chorus.mobile',
    usesCleartextTraffic: !isProduction,
  } as ExpoConfig['android'],
  plugins: ['expo-router', 'expo-secure-store'],
  experiments: {
    typedRoutes: true,
    reactCompiler: true,
  },
}

export default config
