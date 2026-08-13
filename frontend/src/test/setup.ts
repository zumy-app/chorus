import { vi } from 'vitest'
import '@testing-library/jest-dom'

// jsdom does not implement scrollIntoView; ChatArea scrolls the message list.
Object.defineProperty(Element.prototype, 'scrollIntoView', {
  writable: true,
  value: vi.fn(),
})
