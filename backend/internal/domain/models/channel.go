package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/qrave1/RoomSpeak/backend/internal/domain/input"
)

type Channel struct {
	ID        uuid.UUID `json:"id" db:"id"`
	CreatorID uuid.UUID `json:"creator_id" db:"creator_id"`
	Name      string    `json:"name" db:"name"`
	IsPublic  bool      `json:"is_public" db:"is_public"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

func NewChannel(input *input.CreateChannelInput) *Channel {
	return &Channel{
		ID:        uuid.New(),
		CreatorID: input.CreatorID,
		Name:      input.Name,
		IsPublic:  input.IsPublic,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}
