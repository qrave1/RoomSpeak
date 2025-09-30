package dto

import (
	"github.com/google/uuid"
	"github.com/qrave1/RoomSpeak/internal/domain/models"
	"time"
)

type CreateChannelRequest struct {
	Name     string `json:"name"`
	IsPublic bool   `json:"is_public"`
}

type ActiveUserInfo struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

type ChannelResponse struct {
	ID          uuid.UUID       `json:"id"`
	CreatorID   uuid.UUID       `json:"creator_id"`
	Name        string          `json:"name"`
	IsPublic    bool            `json:"is_public"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	ActiveUsers []ActiveUserInfo `json:"active_users"`
}

func NewChannelResponseFromModel(ch *models.Channel, activeUsers []ActiveUserInfo) ChannelResponse {
	if activeUsers == nil {
		activeUsers = []ActiveUserInfo{}
	}
	return ChannelResponse{
		ID:          ch.ID,
		CreatorID:   ch.CreatorID,
		Name:        ch.Name,
		IsPublic:    ch.IsPublic,
		CreatedAt:   ch.CreatedAt,
		UpdatedAt:   ch.UpdatedAt,
		ActiveUsers: activeUsers,
	}
}

type ListChannelsResponse struct {
	Channels []ChannelResponse `json:"channels"`
}
