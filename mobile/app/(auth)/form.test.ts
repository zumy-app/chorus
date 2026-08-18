import { describe, expect, it } from 'vitest'

import { validateEmail, validatePassword, validateRegistration } from './form'

describe('auth form validation', () => {
  it('requires a valid email address', () => {
    expect(validateEmail('not-an-email')).toBe('Enter a valid email address.')
    expect(validateEmail(' person@example.com ')).toBeUndefined()
  })

  it('requires passwords to meet the server minimum length', () => {
    expect(validatePassword('short')).toBe('Password must be at least 8 characters.')
    expect(validatePassword('long-enough')).toBeUndefined()
  })

  it('requires matching passwords and a display name during registration', () => {
    expect(
      validateRegistration({
        email: 'person@example.com',
        password: 'long-enough',
        confirmPassword: 'different',
        displayName: '',
      }),
    ).toBe('Enter your name.')
  })

  it('rejects a registration whose password confirmation differs', () => {
    expect(
      validateRegistration({
        email: 'person@example.com',
        password: 'long-enough',
        confirmPassword: 'different',
        displayName: 'Person',
      }),
    ).toBe('Passwords do not match.')
  })
})
