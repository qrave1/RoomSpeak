import type { User, LoginRequest, RegisterRequest, Channel, CreateChannelRequest, OnlineUser } from '@/types'
import { env } from '@/config/env'

const API_BASE = `${env.backendUrl}/api`

class ApiService {
  // Auth endpoints
  async checkAuth(): Promise<{ isAuthenticated: boolean; user?: User }> {
    try {
      const response = await fetch(`${API_BASE}/v1/me`, {
        credentials: 'include'
      })
      if (response.ok) {
        const user = await response.json()
        return { isAuthenticated: true, user }
      }
      return { isAuthenticated: false }
    } catch (err) {
      console.error('Auth check error:', err)
      return { isAuthenticated: false }
    }
  }

  async login(credentials: LoginRequest): Promise<boolean> {
    try {
      const response = await fetch(`${API_BASE}/auth/login`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(credentials),
        credentials: 'include'
      })
      return response.ok
    } catch (err) {
      console.error('Login error:', err)
      return false
    }
  }

  async register(credentials: RegisterRequest): Promise<boolean> {
    try {
      const response = await fetch(`${API_BASE}/auth/register`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(credentials),
        credentials: 'include'
      })
      return response.ok
    } catch (err) {
      console.error('Registration error:', err)
      return false
    }
  }

  logout(): void {
    document.cookie = 'jwt=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;'
  }

  // Channel endpoints
  async getChannels(): Promise<Channel[]> {
    try {
      const response = await fetch(`${API_BASE}/v1/channels`, {
        credentials: 'include'
      })
      if (!response.ok) {
        throw new Error('Network response was not ok')
      }
      const data = await response.json()
      return data.channels || []
    } catch (err) {
      console.error('Error getting channels:', err)
      return []
    }
  }

  async createChannel(channel: CreateChannelRequest): Promise<boolean> {
    try {
      const response = await fetch(`${API_BASE}/v1/channels`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(channel),
        credentials: 'include'
      })
      return response.ok
    } catch (err) {
      console.error('Error creating channel:', err)
      return false
    }
  }

  async deleteChannel(channelId: string): Promise<boolean> {
    try {
      const response = await fetch(`${API_BASE}/v1/channels/${channelId}`, {
        method: 'DELETE',
        credentials: 'include'
      })
      return response.ok
    } catch (err) {
      console.error('Error deleting channel:', err)
      return false
    }
  }

  // Online users endpoints
  async getOnlineUsers(): Promise<OnlineUser[]> {
    try {
      const response = await fetch(`${API_BASE}/v1/users/online`, {
        credentials: 'include'
      })
      if (!response.ok) {
        throw new Error('Failed to fetch online users')
      }
      const data = await response.json()
      return data || []
    } catch (err) {
      console.error('Error getting online users:', err)
      return []
    }
  }
}

export const apiService = new ApiService()

