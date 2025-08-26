package models

import (
	"time"
)

// Room представляет комнату для голосового общения
type Room struct {
	ID        int       `json:"id" db:"id"`
	RoomID    string    `json:"room_id" db:"room_id"`
	Name      string    `json:"name" db:"name"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// CreateRoomRequest запрос на создание комнаты
type CreateRoomRequest struct {
	Name string `json:"roomName"`
}

// JoinRoomRequest запрос на присоединение к комнате
type JoinRoomRequest struct {
	RoomID string `json:"roomId"`
}
