package runtime

import "github.com/google/uuid"

type ActiveUser struct {
	ID        uuid.UUID `json:"id"`
	ChannelID uuid.UUID `json:"channel_id"`
}
