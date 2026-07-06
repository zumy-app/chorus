import type { WebSocketMessage } from '../types'

class WebSocketService {
  private ws: WebSocket | null = null
  private reconnectAttempts = 0
  private maxReconnectAttempts = 5
  private reconnectDelay = 1000
  private messageHandlers: ((message: WebSocketMessage) => void)[] = []
  private reconnectHandlers: (() => void)[] = []

  connect(token: string) {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    // Pass token as query parameter since WebSocket API doesn't support custom headers
    const wsUrl = `${protocol}//${window.location.host}/ws?token=${encodeURIComponent(token)}`

    this.ws = new WebSocket(wsUrl)

    this.ws.onopen = () => {
      const isReconnect = this.reconnectAttempts > 0
      console.log('WebSocket connected')
      this.reconnectAttempts = 0
      // Re-fetch data after reconnect to catch any missed events
      if (isReconnect) {
        this.reconnectHandlers.forEach((handler) => handler())
      }
    }

    this.ws.onmessage = (event) => {
      try {
        const message: WebSocketMessage = JSON.parse(event.data)
        this.messageHandlers.forEach((handler) => handler(message))
      } catch (error) {
        console.error('Failed to parse WebSocket message:', error)
      }
    }

    this.ws.onerror = (error) => {
      console.error('WebSocket error:', error)
    }

    this.ws.onclose = () => {
      console.log('WebSocket disconnected')
      this.attemptReconnect()
    }
  }

  private attemptReconnect() {
    if (this.reconnectAttempts < this.maxReconnectAttempts) {
      this.reconnectAttempts++
      setTimeout(() => {
        console.log(`Reconnecting... Attempt ${this.reconnectAttempts}`)
        // Always get a fresh token from localStorage — the old one may have expired
        const freshToken = localStorage.getItem('accessToken')
        if (freshToken) {
          this.connect(freshToken)
        }
      }, this.reconnectDelay * this.reconnectAttempts)
    }
  }

  disconnect() {
    if (this.ws) {
      this.ws.close()
      this.ws = null
    }
  }

  send(message: WebSocketMessage) {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message))
    }
  }

  onMessage(handler: (message: WebSocketMessage) => void) {
    this.messageHandlers.push(handler)
    return () => {
      this.messageHandlers = this.messageHandlers.filter((h) => h !== handler)
    }
  }

  onReconnect(handler: () => void) {
    this.reconnectHandlers.push(handler)
    return () => {
      this.reconnectHandlers = this.reconnectHandlers.filter((h) => h !== handler)
    }
  }

  sendTyping(chatId: string, isTyping: boolean) {
    this.send({
      type: isTyping ? 'typing_start' : 'typing_stop',
      data: { chatId },
    })
  }
}

export const wsService = new WebSocketService()
