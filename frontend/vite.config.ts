import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@chorus/shared': path.resolve(__dirname, '../packages/shared/src/index.ts'),
      'axios': path.resolve(__dirname, 'node_modules/axios/index.js'),
    },
  },
  server: {
    port: 3000,
    historyApiFallback: true,
    fs: {
      allow: [path.resolve(__dirname, '..')],
    },
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/ws': {
        target: 'ws://localhost:8080',
        ws: true,
      },
    },
  },
  // Mirror the dev proxy for the production preview (`vite preview`). This is
  // what the Playwright e2e suite runs against (baseURL http://localhost:4173)
  // so real API + WebSocket flows work against the local backend.
  preview: {
    port: 4173,
    host: true,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/ws': {
        target: 'ws://localhost:8080',
        ws: true,
      },
    },
  },
})
