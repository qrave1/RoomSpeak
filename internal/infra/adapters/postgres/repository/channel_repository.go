package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/qrave1/RoomSpeak/internal/domain/models"
)

type ChannelRepository interface {
	Create(ctx context.Context, channel *models.Channel) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Channel, error)
	Update(ctx context.Context, channel *models.Channel) error
	Delete(ctx context.Context, id uuid.UUID) error

	AddUserToChannel(ctx context.Context, userID, channelID uuid.UUID) error
	RemoveUserFromChannel(ctx context.Context, userID, channelID uuid.UUID) error

	GetChannelsByUserID(ctx context.Context, userID uuid.UUID) ([]*models.Channel, error)
	GetPublicChannels(ctx context.Context) ([]*models.Channel, error)
}

type channelRepo struct {
	db *sqlx.DB
}

func NewChannelRepo(db *sqlx.DB) ChannelRepository {
	return &channelRepo{db: db}
}

func (r *channelRepo) Create(ctx context.Context, channel *models.Channel) error {
	_, err := r.db.ExecContext(
		ctx,
		"INSERT INTO channels (id, creator_id, name, is_public) VALUES ($1, $2, $3, $4)",
		channel.ID,
		channel.CreatorID,
		channel.Name,
		channel.IsPublic,
	)

	return err
}

func (r *channelRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
	var channel models.Channel

	err := r.db.GetContext(ctx, &channel, "SELECT * FROM channels WHERE id = $1", id)
	if err != nil {
		return nil, err
	}

	return &channel, nil
}

func (r *channelRepo) Update(ctx context.Context, channel *models.Channel) error {
	_, err := r.db.ExecContext(
		ctx,
		"UPDATE channels SET name = $1, updated_at = $2 WHERE id = $3",
		channel.Name,
		time.Now(),
		channel.ID,
	)

	return err
}

func (r *channelRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM channels WHERE id = $1", id)

	return err
}

func (r *channelRepo) AddUserToChannel(ctx context.Context, userID, channelID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, "INSERT INTO channel_users (user_id, channel_id) VALUES ($1, $2)", userID, channelID)
	return err
}

func (r *channelRepo) RemoveUserFromChannel(ctx context.Context, userID, channelID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM channel_users WHERE user_id = $1 AND channel_id = $2", userID, channelID)
	return err
}

func (r *channelRepo) GetChannelsByUserID(ctx context.Context, userID uuid.UUID) ([]*models.Channel, error) {
	var channels []*models.Channel

	query := `
		SELECT c.*
		FROM channels c
		INNER JOIN channel_users cu ON c.id = cu.channel_id
		WHERE cu.user_id = $1
	`

	err := r.db.SelectContext(ctx, &channels, query, userID)
	if err != nil {
		return nil, err
	}

	return channels, nil
}

func (r *channelRepo) GetPublicChannels(ctx context.Context) ([]*models.Channel, error) {
	var channels []*models.Channel

	query := `
		SELECT c.*
		FROM channels c
		WHERE c.is_public = true
	`

	err := r.db.SelectContext(ctx, &channels, query)
	if err != nil {
		return nil, err
	}

	return channels, nil
}
