import { vi } from 'vitest'
import '@testing-library/jest-dom'

// jsdom does not implement scrollIntoView; ChatArea scrolls the message list.
Object.defineProperty(Element.prototype, 'scrollIntoView', {
  writable: true,
  value: vi.fn(),
})

// jsdom does not provide a functional localStorage by default; several modules
// read it (e.g. src/i18n/index.ts resolveInitialLanguage). Polyfill a working
// in-memory implementation so those modules behave in the test environment.
const storeMap: Record<string, string> = {}
const localStorageMock = {
  getItem: (key: string) => (key in storeMap ? storeMap[key] : null),
  setItem: (key: string, value: string) => {
    storeMap[key] = String(value)
  },
  removeItem: (key: string) => {
    delete storeMap[key]
  },
  clear: () => {
    for (const k of Object.keys(storeMap)) delete storeMap[k]
  },
  key: (i: number) => Object.keys(storeMap)[i] ?? null,
  get length() {
    return Object.keys(storeMap).length
  },
}
Object.defineProperty(globalThis, 'localStorage', {
  value: localStorageMock,
  writable: true,
  configurable: true,
})
