export interface User {
  id: string
  username: string
}

export interface LoginRequest {
  username: string
  password: string
}

export interface RegisterRequest {
  username: string
  password: string
}

export interface Channel {
  id: string
  creator_id: string
  name: string
  is_public: boolean
  created_at: string
  updated_at: string
  active_users: ActiveUser[]
}

export interface ActiveUser {
  id: string
  username: string
}

export interface CreateChannelRequest {
  name: string
  is_public: boolean
}

