// React Native's Metro supplies a `process` polyfill with `process.env` at
// runtime, but the tsconfig doesn't include Node types. Declare it minimally so
// `process.env.X` reads type-check (Expo/CLI inlines EXPO_PUBLIC_* at build).
declare const process: {
  env: Record<string, string | undefined>
}
