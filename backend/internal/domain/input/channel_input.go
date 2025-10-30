package input

import "github.com/google/uuid"

type CreateChannelInput struct {
	CreatorID uuid.UUID `json:"creator_id"`
	Name      string    `json:"name"`
	IsPublic  bool      `json:"is_public"`
}

type UpdateChannelInput struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	IsPublic bool      `json:"is_public"`
}
