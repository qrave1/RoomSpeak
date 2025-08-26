package models

import (
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

// Client представляет подключенного клиента
type Client struct {
	ID          string                 `json:"id"`
	UserName    string                 `json:"userName"`
	RoomID      string                 `json:"roomId"`
	Conn        *websocket.Conn        `json:"-"`
	PeerConn    *webrtc.PeerConnection `json:"-"`
	IsMuted     bool                   `json:"isMuted"`
	IsConnected bool                   `json:"isConnected"`
}

// WSMessage представляет сообщение WebSocket
type WSMessage struct {
	Action string      `json:"action"`
	Data   interface{} `json:"data"`
}

// WebRTCMessage сообщения для WebRTC сигналинга
type WebRTCMessage struct {
	Type          string `json:"type"`
	SDP           string `json:"sdp,omitempty"`
	Candidate     string `json:"candidate,omitempty"`
	SDPMid        string `json:"sdpMid,omitempty"`
	SDPMLineIndex int    `json:"sdpMLineIndex,omitempty"`
}

// UserEvent события пользователей
type UserEvent struct {
	UserID   string `json:"userId"`
	UserName string `json:"userName"`
	IsMuted  bool   `json:"isMuted"`
}

// ToggleMuteRequest запрос на изменение состояния микрофона
type ToggleMuteRequest struct {
	IsMuted bool `json:"isMuted"`
}
