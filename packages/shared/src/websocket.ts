// Shared WebSocket client for Chorus. Platform-agnostic: the consuming app
// injects a token provider and a URL builder (per-platform ws/wss base URL).

import type { WebSocketMessage } from './types'

export interface WebSocketServiceOptions {
  /** Returns the current access token (used on connect/reconnect). */
  getToken: () => Promise<string | null>
  /** Builds the WebSocket URL including the auth token. */
  createUrl: (token: string) => string
}

interface MessageHandler {
  (message: WebSocketMessage): void
}

interface ReconnectHandler {
  (): void
}

export function createWebSocketService(options: WebSocketServiceOptions) {
  const { getToken, createUrl } = options
  let ws: WebSocket | null = null
  let reconnectAttempts = 0
  const maxReconnectAttempts = 5
  const reconnectDelay = 1000
  let messageHandlers: MessageHandler[] = []
  let reconnectHandlers: ReconnectHandler[] = []
  let isConnecting = false
  let intentionalClose = false

  async function connect(token?: string) {
    if (isConnecting || (ws !== null && ws.readyState === WebSocket.OPEN)) {
      return
    }

    isConnecting = true
    intentionalClose = false
    const resolvedToken = token || (await getToken())

    if (!resolvedToken) {
      console.log('No access token available for WebSocket')
      isConnecting = false
      return
    }

    const isReconnect = reconnectAttempts > 0

    try {
      ws = new WebSocket(createUrl(resolvedToken))

      ws.onopen = () => {
        console.log('WebSocket connected')
        reconnectAttempts = 0
        isConnecting = false
        // Re-fetch data after reconnect to catch any missed events.
        if (isReconnect) {
          reconnectHandlers.forEach(handler => handler())
        }
      }

      ws.onmessage = event => {
        try {
          const message: WebSocketMessage = JSON.parse(event.data)
          messageHandlers.forEach(handler => handler(message))
        } catch {
          // Malformed message — skip silently.
        }
      }

      ws.onerror = () => {
        // Handled by onclose — avoid console.error which triggers the RN dev overlay.
        isConnecting = false
      }

      ws.onclose = () => {
        isConnecting = false
        ws = null
        if (!intentionalClose) {
          reconnect()
        }
      }
    } catch {
      // Connection failed (e.g. backend unreachable) — non-fatal, will reconnect.
      isConnecting = false
      reconnect()
    }
  }

  async function reconnect() {
    if (reconnectAttempts >= maxReconnectAttempts) {
      console.log('Max reconnection attempts reached')
      return
    }

    reconnectAttempts++
    const delay = reconnectDelay * reconnectAttempts

    console.log(`Reconnecting in ${delay}ms...`)
    setTimeout(() => {
      connect()
    }, delay)
  }

  function disconnect() {
    intentionalClose = true
    if (ws) {
      ws.close()
      ws = null
    }
    isConnecting = false
    reconnectAttempts = 0
  }

  function onMessage(handler: MessageHandler) {
    messageHandlers.push(handler)
    return () => {
      messageHandlers = messageHandlers.filter(h => h !== handler)
    }
  }

  function onReconnect(handler: ReconnectHandler) {
    reconnectHandlers.push(handler)
    return () => {
      reconnectHandlers = reconnectHandlers.filter(h => h !== handler)
    }
  }

  function send(message: WebSocketMessage) {
    if (ws !== null && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify(message))
    }
  }

  function sendTyping(chatId: string, isTyping: boolean) {
    send({
      type: isTyping ? 'typing_start' : 'typing_stop',
      data: { chatId },
    })
  }

  function sendReceipt(chatId: string, messageId: string, status: 'received' | 'read') {
    send({
      type: 'message_ack',
      data: { chatId, messageId, status },
    })
  }

  return {
    connect,
    disconnect,
    onMessage,
    onReconnect,
    send,
    sendTyping,
    sendReceipt,
  }
}