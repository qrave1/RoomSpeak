package usecase

import (
	"context"

	"github.com/google/uuid"

	"github.com/qrave1/RoomSpeak/internal/domain/models"
	"github.com/qrave1/RoomSpeak/internal/infra/postgres/repository"
)

type ChannelUsecase interface {
	CreateChannel(ctx context.Context, creatorID uuid.UUID, name string) (*models.Channel, error)
	GetChannel(ctx context.Context, id uuid.UUID) (*models.Channel, error)
	UpdateChannel(ctx context.Context, id uuid.UUID, name string) (*models.Channel, error)
	DeleteChannel(ctx context.Context, id uuid.UUID) error
	AddUserToChannel(ctx context.Context, userID, channelID uuid.UUID) error
	RemoveUserFromChannel(ctx context.Context, userID, channelID uuid.UUID) error
	GetChannelsByUserID(ctx context.Context, userID uuid.UUID) ([]*models.Channel, error)
}

type channelUsecase struct {
	channelRepo repository.ChannelRepository
}

func NewChannelUsecase(channelRepo repository.ChannelRepository) ChannelUsecase {
	return &channelUsecase{channelRepo: channelRepo}
}

func (uc *channelUsecase) CreateChannel(ctx context.Context, creatorID uuid.UUID, name string) (*models.Channel, error) {
	channel := models.NewChannel(creatorID, name)
	if err := uc.channelRepo.Create(ctx, channel); err != nil {
		return nil, err
	}
	return channel, nil
}

func (uc *channelUsecase) GetChannel(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
	return uc.channelRepo.GetByID(ctx, id)
}

func (uc *channelUsecase) UpdateChannel(ctx context.Context, id uuid.UUID, name string) (*models.Channel, error) {
	channel, err := uc.channelRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	channel.Name = name

	if err := uc.channelRepo.Update(ctx, channel); err != nil {
		return nil, err
	}

	return channel, nil
}

func (uc *channelUsecase) DeleteChannel(ctx context.Context, id uuid.UUID) error {
	return uc.channelRepo.Delete(ctx, id)
}

func (uc *channelUsecase) AddUserToChannel(ctx context.Context, userID, channelID uuid.UUID) error {
	return uc.channelRepo.AddUserToChannel(ctx, userID, channelID)
}

func (uc *channelUsecase) RemoveUserFromChannel(ctx context.Context, userID, channelID uuid.UUID) error {
	return uc.channelRepo.RemoveUserFromChannel(ctx, userID, channelID)
}

func (uc *channelUsecase) GetChannelsByUserID(ctx context.Context, userID uuid.UUID) ([]*models.Channel, error) {
	return uc.channelRepo.GetChannelsByUserID(ctx, userID)
}
