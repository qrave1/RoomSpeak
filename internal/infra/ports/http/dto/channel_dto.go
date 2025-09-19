package dto

import "github.com/qrave1/RoomSpeak/internal/domain/models"

type CreateChannelRequest struct {
	Name string `json:"name"`
}

type ListChannelsResponse struct {
	Channels []*models.Channel `json:"channels"`
}
