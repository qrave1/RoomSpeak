package models

import (
	"time"

	"github.com/google/uuid"
)

type Channel struct {
	ID        uuid.UUID `json:"id" db:"id"`
	CreatorID uuid.UUID `json:"creator_id" db:"creator_id"`
	Name      string    `json:"name" db:"name"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

func NewChannel(creatorID uuid.UUID, name string) *Channel {
	return &Channel{
		ID:        uuid.New(),
		CreatorID: creatorID,
		Name:      name,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}
