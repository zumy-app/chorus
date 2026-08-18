import { WEBSOCKET_URL } from './config'
import { getAccessToken } from './session'
import type { WebSocketEvent } from '../types'

export type WebSocketStatus = 'disconnected' | 'connecting' | 'connected'

interface Socket {
  readyState: number
  onopen: (() => void) | null
  onclose: (() => void) | null
  onerror: (() => void) | null
  onmessage: ((event: { data: string }) => void) | null
  send(data: string): void
  close(): void
}

interface SocketConstructor {
  new (url: string): Socket
  OPEN: number
}

export interface WebSocketManagerOptions {
  url: string
  getAccessToken: () => Promise<string | null>
  WebSocket?: SocketConstructor
  onEvent: (event: WebSocketEvent) => void
  onStatusChange?: (status: WebSocketStatus) => void
  baseReconnectDelayMs?: number
  maxReconnectDelayMs?: number
}

export class WebSocketManager {
  private socket: Socket | null = null
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  private reconnectAttempts = 0
  private status: WebSocketStatus = 'disconnected'
  private stopped = true
  private connecting: Promise<void> | null = null
  private readonly listeners = new Set<(event: WebSocketEvent) => void>()
  private readonly Socket: SocketConstructor
  private readonly baseReconnectDelayMs: number
  private readonly maxReconnectDelayMs: number

  constructor(private readonly options: WebSocketManagerOptions) {
    this.Socket = options.WebSocket ?? (globalThis.WebSocket as unknown as SocketConstructor)
    this.baseReconnectDelayMs = options.baseReconnectDelayMs ?? 1_000
    this.maxReconnectDelayMs = options.maxReconnectDelayMs ?? 30_000
  }

  getStatus(): WebSocketStatus {
    return this.status
  }

  subscribe(listener: (event: WebSocketEvent) => void): () => void {
    this.listeners.add(listener)
    return () => this.listeners.delete(listener)
  }

  async connect(): Promise<void> {
    this.stopped = false
    if (this.socket?.readyState === this.Socket.OPEN || this.connecting) return this.connecting ?? Promise.resolve()

    this.connecting = this.open()
    try {
      await this.connecting
    } finally {
      this.connecting = null
    }
  }

  disconnect(): void {
    this.stopped = true
    if (this.reconnectTimer) clearTimeout(this.reconnectTimer)
    this.reconnectTimer = null
    const socket = this.socket
    this.socket = null
    socket?.close()
    this.setStatus('disconnected')
  }

  send(event: WebSocketEvent): boolean {
    if (!this.socket || this.socket.readyState !== this.Socket.OPEN) return false
    this.socket.send(JSON.stringify(event))
    return true
  }

  private async open(): Promise<void> {
    const token = await this.options.getAccessToken()
    if (!token || this.stopped) return

    this.setStatus('connecting')
    const url = new URL(this.options.url)
    url.searchParams.set('token', token)
    const socket = new this.Socket(url.toString())
    this.socket = socket

    socket.onopen = () => {
      if (this.socket !== socket) return
      this.reconnectAttempts = 0
      this.setStatus('connected')
    }
    socket.onmessage = (message) => this.handleMessage(message.data)
    socket.onerror = () => undefined
    socket.onclose = () => {
      if (this.socket !== socket) return
      this.socket = null
      this.setStatus('disconnected')
      if (!this.stopped) this.scheduleReconnect()
    }
  }

  private handleMessage(raw: string): void {
    try {
      const event = JSON.parse(raw) as WebSocketEvent
      if (typeof event.type === 'string' && event.data !== undefined) {
        this.options.onEvent(event)
        this.listeners.forEach((listener) => listener(event))
      }
    } catch {
      // Ignore malformed messages; the next REST snapshot will restore state.
    }
  }

  private scheduleReconnect(): void {
    if (this.reconnectTimer) return
    const delay = Math.min(this.baseReconnectDelayMs * 2 ** this.reconnectAttempts, this.maxReconnectDelayMs)
    this.reconnectAttempts += 1
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null
      void this.connect()
    }, delay)
  }

  private setStatus(status: WebSocketStatus): void {
    if (this.status === status) return
    this.status = status
    this.options.onStatusChange?.(status)
  }
}

export const websocketManager = new WebSocketManager({
  url: WEBSOCKET_URL,
  getAccessToken,
  onEvent: () => undefined,
})
