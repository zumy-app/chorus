import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { wsService } from '../../services/websocket'

let activeWsInstances: MockWebSocket[] = []

// Trackable Mock WebSocket
class MockWebSocket {
  static CONNECTING = 0
  static OPEN = 1
  static CLOSING = 2
  static CLOSED = 3

  onopen: (() => void) | null = null
  onclose: (() => void) | null = null
  onmessage: ((event: any) => void) | null = null
  onerror: ((error: any) => void) | null = null
  readyState: number = 1 // OPEN
  url: string = ''
  sentData: string[] = []

  constructor(url: string) {
    this.url = url
    activeWsInstances.push(this)
    queueMicrotask(() => {
      if (this.readyState === 1) {
        this.onopen?.()
      }
    })
  }

  send(data: string) {
    this.sentData.push(data)
  }

  close() {
    this.readyState = 3 // CLOSED
    this.onclose?.()
  }

  // Test helper to simulate incoming socket frame
  simulateMessage(data: any) {
    this.onmessage?.({ data: typeof data === 'string' ? data : JSON.stringify(data) })
  }

  // Test helper to simulate error
  simulateError(err: any) {
    this.onerror?.(err)
  }
}

// Stub global WebSocket
vi.stubGlobal('WebSocket', MockWebSocket)

const localStorageMock = (() => {
  let store: Record<string, string> = { accessToken: 'test-token' }
  return {
    getItem: (key: string) => store[key] || null,
    setItem: (key: string, value: string) => { store[key] = value },
    removeItem: (key: string) => { delete store[key] },
    clear: () => { store = {} },
  }
})()

Object.defineProperty(window, 'localStorage', { value: localStorageMock })
Object.defineProperty(window, 'location', { value: { host: 'localhost:3000', protocol: 'http:' } })

describe('WebSocket Service', () => {
  beforeEach(() => {
    localStorageMock.setItem('accessToken', 'test-token')
    wsService.disconnect()
    activeWsInstances = []
    vi.clearAllMocks()
    vi.useFakeTimers()
  })

  afterEach(() => {
    wsService.disconnect()
    vi.useRealTimers()
  })

  it('should connect to WebSocket with token in query params', async () => {
    await wsService.connect('my-secret-token')
    await vi.runAllTimersAsync()

    expect(activeWsInstances).toHaveLength(1)
    const ws = activeWsInstances[0]
    expect(ws.url).toContain('token=my-secret-token')
    expect(ws.readyState).toBe(1) // OPEN
  })

  it('should handle incoming JSON messages and notify registered handlers', async () => {
    await wsService.connect('test-token')
    await vi.runAllTimersAsync()

    const ws = activeWsInstances[0]
    const handler = vi.fn()
    wsService.onMessage(handler)

    const incomingPayload = { type: 'new_message', data: { id: 'm1', text: 'Hello' } }
    ws.simulateMessage(incomingPayload)

    expect(handler).toHaveBeenCalledTimes(1)
    expect(handler).toHaveBeenCalledWith(incomingPayload)
  })

  it('should send typing events with correct JSON structure', async () => {
    await wsService.connect('test-token')
    await vi.runAllTimersAsync()

    const ws = activeWsInstances[0]

    wsService.sendTyping('chat-1', true)
    expect(ws.sentData).toHaveLength(1)
    expect(JSON.parse(ws.sentData[0])).toEqual({
      type: 'typing_start',
      data: { chatId: 'chat-1' },
    })

    wsService.sendTyping('chat-1', false)
    expect(ws.sentData).toHaveLength(2)
    expect(JSON.parse(ws.sentData[1])).toEqual({
      type: 'typing_stop',
      data: { chatId: 'chat-1' },
    })
  })

  it('should send message receipts with correct JSON structure', async () => {
    await wsService.connect('test-token')
    await vi.runAllTimersAsync()

    const ws = activeWsInstances[0]

    wsService.sendReceipt('chat-100', 'msg-500', 'read')
    expect(ws.sentData).toHaveLength(1)
    expect(JSON.parse(ws.sentData[0])).toEqual({
      type: 'message_ack',
      data: { chatId: 'chat-100', messageId: 'msg-500', status: 'read' },
    })
  })

  it('should handle onMessage unsubscribe correctly', async () => {
    await wsService.connect('test-token')
    await vi.runAllTimersAsync()

    const ws = activeWsInstances[0]
    const handler = vi.fn()
    const unsubscribe = wsService.onMessage(handler)

    ws.simulateMessage({ type: 'test' })
    expect(handler).toHaveBeenCalledTimes(1)

    unsubscribe()
    ws.simulateMessage({ type: 'test2' })
    expect(handler).toHaveBeenCalledTimes(1) // Still 1
  })

  it('should disconnect cleanly and close socket', async () => {
    await wsService.connect('test-token')
    await vi.runAllTimersAsync()

    const ws = activeWsInstances[0]
    expect(ws.readyState).toBe(1) // OPEN

    wsService.disconnect()
    expect(ws.readyState).toBe(3) // CLOSED
  })

  it('should handle connection errors and trigger auto-reconnect on unexpected close', async () => {
    await wsService.connect('test-token')
    await vi.runAllTimersAsync()

    const ws1 = activeWsInstances[0]

    // Simulate socket error and close
    ws1.simulateError(new Error('Network drop'))
    ws1.close()

    // Automatic reconnect delay (1000ms for attempt 1)
    await vi.advanceTimersByTimeAsync(1200)

    // Second WebSocket instance created on reconnect
    expect(activeWsInstances.length).toBeGreaterThan(1)
  })

  it('should invoke reconnect handlers upon successful reconnection', async () => {
    const reconnectHandler = vi.fn()
    wsService.onReconnect(reconnectHandler)

    await wsService.connect('test-token')
    await vi.runAllTimersAsync()

    const ws1 = activeWsInstances[0]
    // Drop first connection
    ws1.close()

    // Reconnect timer runs (1000ms delay)
    await vi.advanceTimersByTimeAsync(1200)
    await vi.runAllTimersAsync()

    expect(reconnectHandler).toHaveBeenCalled()
  })

  it('should not attempt connection if access token is missing', async () => {
    localStorageMock.removeItem('accessToken')

    await wsService.connect() // no explicit token provided, falls back to getToken
    await vi.runAllTimersAsync()

    expect(activeWsInstances).toHaveLength(0)
  })
})
