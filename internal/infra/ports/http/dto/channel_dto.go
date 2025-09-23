package dto

import (
	"github.com/qrave1/RoomSpeak/internal/domain/models"
)

type CreateChannelRequest struct {
	Name     string `json:"name"`
	IsPublic bool   `json:"is_public"`
}

type ListChannelsResponse struct {
	Channels []*models.Channel `json:"channels"`
}
