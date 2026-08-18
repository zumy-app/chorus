export function validateEmail(email: string): string | undefined {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.trim()) ? undefined : 'Enter a valid email address.'
}

export function validatePassword(password: string): string | undefined {
  return password.length >= 8 ? undefined : 'Password must be at least 8 characters.'
}

export function validateRegistration(input: {
  email: string
  password: string
  confirmPassword: string
  displayName: string
}): string | undefined {
  return (
    validateEmail(input.email) ??
    validatePassword(input.password) ??
    (input.displayName.trim() ? undefined : 'Enter your name.') ??
    (input.password === input.confirmPassword ? undefined : 'Passwords do not match.')
  )
}
