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

export interface Participant {
  id: string
  username: string
  is_muted: boolean
}

export interface ICEServer {
  urls: string | string[]
  username?: string
  credential?: string
}

export interface ICEServerResponse {
  urls: string
  username: string
  credential: string
}

export interface WSMessage {
  type: string
  data?: any
  message?: string
  candidate?: RTCIceCandidateInit
  sdp?: string
}

export interface AudioDevice {
  deviceId: string
  label: string
  kind: string
}

export interface OnlineUser {
  id: string
  username: string
  channel_id?: string
  channel_name?: string
}
