import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import path from 'path'

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@chorus/shared': path.resolve(__dirname, '../packages/shared/src/index.ts'),
      // Pin axios to the frontend copy so vi.doMock('axios') in tests also
      // intercepts imports from the shared package (which resolves node
      // modules from the repo root).
      'axios': path.resolve(__dirname, 'node_modules/axios/index.js'),
    },
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
    include: ['src/**/*.test.{ts,tsx}'],
    css: true,
  },
})
