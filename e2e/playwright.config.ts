import { defineConfig, devices } from '@playwright/test'

/**
 * Playwright configuration for Chorus E2E tests.
 *
 * Tests run against the Dockerized stack (frontend on :3000, backend on :8080).
 * The global-setup starts services and verifies health; global-teardown stops them.
 */
export default defineConfig({
  testDir: './tests',
  fullyParallel: false, // Sequential — tests share state (users, chats)
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  workers: 1, // Single worker — two-user scenarios need coordination
  reporter: [
    ['html', { outputFolder: 'playwright-report' }],
    ['list'],
  ],
  timeout: 300_000, // 5 min per test — translator-engine model download can be slow on first run
  expect: {
    timeout: 30_000, // Generous expect timeout for async translation arrival
  },

  use: {
    baseURL: process.env.E2E_BASE_URL || 'http://localhost:3000',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
    actionTimeout: 30_000,
    navigationTimeout: 30_000,
  },

  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],

  // Start services before tests, stop after
  globalSetup: './global-setup.ts',
  globalTeardown: './global-teardown.ts',
})