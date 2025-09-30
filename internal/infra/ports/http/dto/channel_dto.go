package dto

import (
	"github.com/google/uuid"
	"github.com/qrave1/RoomSpeak/internal/domain/models"
	"github.com/qrave1/RoomSpeak/internal/domain/runtime"
	"time"
)

type CreateChannelRequest struct {
	Name     string `json:"name"`
	IsPublic bool   `json:"is_public"`
}

type ChannelResponse struct {
	ID          uuid.UUID            `json:"id"`
	CreatorID   uuid.UUID            `json:"creator_id"`
	Name        string               `json:"name"`
	IsPublic    bool                 `json:"is_public"`
	CreatedAt   time.Time            `json:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at"`
	ActiveUsers []runtime.ActiveUser `json:"active_users"`
}

func NewChannelResponseFromModel(ch *models.Channel, activeUsers []runtime.ActiveUser) ChannelResponse {
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
