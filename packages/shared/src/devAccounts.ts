// Dev-only test accounts — mirrors backend/internal/services/dev_seed.go:16
// This file is imported only inside `if (import.meta.env.DEV)` / `if (__DEV__)` branches
// so Vite/RN minifiers dead-code-eliminate it from production bundles.
// Never add real user credentials here.

export type DevAccount = {
  label: string
  email: string
  username: string
  password: string
  role: string
  note: string
}

export const DEV_ACCOUNTS: DevAccount[] = [
  {
    label: 'Alice — learner (en→es)',
    email: 'alice.dev@chorus.test',
    username: 'alice.dev',
    password: 'ChorusDev123!',
    role: 'learner',
    note: 'has trial credit, 1★ for Sofia',
  },
  {
    label: 'Bob — learner (es→en)',
    email: 'bob.dev@chorus.test',
    username: 'bob.dev',
    password: 'ChorusDev123!',
    role: 'learner',
    note: 'second learner',
  },
  {
    label: 'Sofia — approved tutor',
    email: 'sofia.tutor@chorus.test',
    username: 'sofia.tutor',
    password: 'ChorusDev123!',
    role: 'tutor',
    note: 'approved, 4 availability slots, 4.5★',
  },
]

// Well-known invite token for register flow (dev_seed.go:22)
export const DEV_INVITE = {
  email: 'invite.dev@chorus.test',
  token: 'chorus-dev-invite-2026',
} as const
