/**
 * Test user credentials for Chorus E2E tests.
 *
 * Both users are expected to already exist in the database.
 * If they don't, the first login attempt will fail — run the
 * registration flow manually once before the test suite.
 */

export interface TestUser {
  email: string
  password: string
  nativeLanguage: string
  displayName: string
}

export const ENGLISH_USER: TestUser = {
  email: 'uhsarp@gmail.com',
  password: 'Demor@cer1',
  nativeLanguage: 'en',
  displayName: 'Prashanth',
}

export const SPANISH_USER: TestUser = {
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