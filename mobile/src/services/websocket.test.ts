import { afterEach, describe, expect, it, vi } from 'vitest'

vi.mock('./session', () => ({
  getAccessToken: vi.fn(),
}))

import { WebSocketManager } from './websocket'

class FakeWebSocket {
  static instances: FakeWebSocket[] = []
  static readonly OPEN = 1
  readonly url: string
  readyState = 0
  onopen: (() => void) | null = null
  onclose: (() => void) | null = null
  onerror: (() => void) | null = null
  onmessage: ((event: { data: string }) => void) | null = null
  send = vi.fn()
  close = vi.fn(() => this.onclose?.())

  constructor(url: string) {
    this.url = url
    FakeWebSocket.instances.push(this)
  }

  open() {
    this.readyState = FakeWebSocket.OPEN
    this.onopen?.()
  }
}

describe('WebSocketManager', () => {
  afterEach(() => {
    FakeWebSocket.instances = []
    vi.useRealTimers()
  })

  it('connects with the stored access token and dispatches protocol events', async () => {
    const onEvent = vi.fn()
    const manager = new WebSocketManager({
      url: 'ws://localhost:8080/ws',
      getAccessToken: vi.fn().mockResolvedValue('token-123'),
      WebSocket: FakeWebSocket as never,
      onEvent,
    })

    await manager.connect()
    const socket = FakeWebSocket.instances[0]
    expect(socket.url).toBe('ws://localhost:8080/ws?token=token-123')

    socket.open()
    socket.onmessage?.({ data: JSON.stringify({ type: 'new_message', data: { id: 'message-1' } }) })
    expect(onEvent).toHaveBeenCalledWith({ type: 'new_message', data: { id: 'message-1' } })
  })

  it('reconnects with exponential backoff after an unexpected close', async () => {
    vi.useFakeTimers()
    const manager = new WebSocketManager({
      url: 'ws://localhost:8080/ws',
      getAccessToken: vi.fn().mockResolvedValue('token-123'),
      WebSocket: FakeWebSocket as never,
      onEvent: vi.fn(),
      baseReconnectDelayMs: 100,
    })

    await manager.connect()
    FakeWebSocket.instances[0].open()
    FakeWebSocket.instances[0].onclose?.()
    await vi.advanceTimersByTimeAsync(100)

    expect(FakeWebSocket.instances).toHaveLength(2)
  })
})
