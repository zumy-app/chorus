/**
 * Test user credentials for Chorus E2E tests.
 *
 * Canonical dev accounts seeded via backend/internal/services/dev_seed.go:16
 * with `go run ./cmd/server --seed-dev`. All share password ChorusDev123!
 * Legacy Gmail accounts kept as fallback aliases for local runs without seed.
 */

export interface TestUser {
  email: string
  password: string
  nativeLanguage: string
  displayName: string
}

// Canonical DEV accounts — must match packages/shared/src/devAccounts.ts:15 + dev_seed.go:17
export const DEV_ALICE: TestUser = {
  email: 'alice.dev@chorus.test',
  password: 'ChorusDev123!',
  nativeLanguage: 'en',
  displayName: 'Alice Dev',
}

export const DEV_BOB: TestUser = {
  email: 'bob.dev@chorus.test',
  password: 'ChorusDev123!',
  nativeLanguage: 'es',
  displayName: 'Bob Dev',
}

export const DEV_SOFIA: TestUser = {
  email: 'sofia.tutor@chorus.test',
  password: 'ChorusDev123!',
  nativeLanguage: 'en',
  displayName: 'Sofia Tutor',
}

// Aliases for backward compat — new tests should use DEV_* directly
export const ENGLISH_USER: TestUser = DEV_ALICE
export const SPANISH_USER: TestUser = DEV_BOB
export const ENGLISH_USER_LEGACY: TestUser = {
  email: 'uhsarp@gmail.com',
  password: 'Demor@cer1',
  nativeLanguage: 'en',
  displayName: 'Prashanth',
}
export const SPANISH_USER_LEGACY: TestUser = {
  email: 'avcxafefwer@gmail.com',
  password: 'Demor@cer1',
  nativeLanguage: 'es',
  displayName: 'avcxafefwer',
}

/**
 * Sample English message used in cross-language translation tests.
 * Chosen to be simple enough for ALMA-7B to translate reliably.
 */
export const SAMPLE_ENGLISH_MESSAGE = 'Hello, how are you doing today?'

/**
 * Sample Spanish message for the reverse direction.
 */
export const SAMPLE_SPANISH_MESSAGE = 'Hola, ¿cómo estás hoy?'