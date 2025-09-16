package repository

import (
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/qrave1/RoomSpeak/internal/models"
)

type ChannelRepository interface {
	CreateChannel(channel *models.Channel) error
	GetChannelByID(id uuid.UUID) (*models.Channel, error)
	GetChannelByCreatorID(creatorID uuid.UUID) (*models.Channel, error)
	DeleteChannel(id uuid.UUID) error
}

type channelRepo struct {
	db *sqlx.DB
}

func NewChannelRepo(db *sqlx.DB) ChannelRepository {
	return &channelRepo{db: db}
}

func (r *channelRepo) CreateChannel(channel *models.Channel) error {
	_, err := r.db.Exec("INSERT INTO channels (id, creator_id, name) VALUES ($1, $2, $3)", channel.ID, channel.CreatorID, channel.Name)

	return err
}

func (r *channelRepo) GetChannelByID(id uuid.UUID) (*models.Channel, error) {
	var channel models.Channel

	err := r.db.Get(&channel, "SELECT * FROM channels WHERE id = $1", id)
	if err != nil {
		return nil, err
	}

	return &channel, nil
}

func (r *channelRepo) GetChannelByCreatorID(creatorID uuid.UUID) (*models.Channel, error) {
	var channel models.Channel

	err := r.db.Get(&channel, "SELECT * FROM channels WHERE creator_id = $1", creatorID)
	if err != nil {
		return nil, err
	}

	return &channel, nil
}

func (r *channelRepo) DeleteChannel(id uuid.UUID) error {
	_, err := r.db.Exec("DELETE FROM channels WHERE id = $1", id)

	return err
}
