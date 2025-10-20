import type {WSMessage} from '@/types'
import {env} from '@/config'

export class WebSocketService {
    private ws: WebSocket | null = null
    private pingInterval: number | null = null
    private onMessageCallback: ((message: WSMessage) => void) | null = null
    private onCloseCallback: (() => void) | null = null

    connect(): WebSocket {
        const wsUrl = `${env.backendUrl}/api/v1/ws`

        this.ws = new WebSocket(wsUrl)

        this.ws.onopen = () => {
            console.log('WebSocket connected')
            this.startPing()
        }

        this.ws.onmessage = (event) => {
            const message: WSMessage = JSON.parse(event.data)
            if (this.onMessageCallback) {
                this.onMessageCallback(message)
            }
        }

        this.ws.onclose = () => {
            console.log('WebSocket closed')
            this.stopPing()
            if (this.onCloseCallback) {
                this.onCloseCallback()
            }
        }

        this.ws.onerror = (error) => {
            console.error('WebSocket error:', error)
        }

        return this.ws
    }

    send(message: WSMessage) {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify(message))
        }
    }

    onMessage(callback: (message: WSMessage) => void) {
        this.onMessageCallback = callback
    }

    onClose(callback: () => void) {
        this.onCloseCallback = callback
    }

    disconnect() {
        this.stopPing()
        if (this.ws) {
            this.ws.close()
            this.ws = null
        }
    }

    private startPing() {
        this.pingInterval = window.setInterval(() => {
            if (this.ws && this.ws.readyState === WebSocket.OPEN) {
                this.send({type: 'ping'})
            }
        }, 30000)
    }

    private stopPing() {
        if (this.pingInterval !== null) {
            clearInterval(this.pingInterval)
            this.pingInterval = null
        }
    }

    getSocket(): WebSocket | null {
        return this.ws
    }

    isConnected(): boolean {
        return this.ws !== null && this.ws.readyState === WebSocket.OPEN
    }
}

export const wsService = new WebSocketService()

