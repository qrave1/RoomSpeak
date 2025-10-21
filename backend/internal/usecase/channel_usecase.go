package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	
	"github.com/qrave1/RoomSpeak/internal/domain/input"
	"github.com/qrave1/RoomSpeak/internal/domain/models"
	"github.com/qrave1/RoomSpeak/internal/domain/runtime"
	"github.com/qrave1/RoomSpeak/internal/infra/adapters/memory"
	"github.com/qrave1/RoomSpeak/internal/infra/adapters/postgres/repository"
)

type ChannelUsecase interface {
	CreateChannel(ctx context.Context, input *input.CreateChannelInput) (*models.Channel, error)
	GetChannel(ctx context.Context, id uuid.UUID) (*models.Channel, error)
	UpdateChannel(ctx context.Context, update *input.UpdateChannelInput) (*models.Channel, error)
	DeleteChannel(ctx context.Context, id uuid.UUID) error

	AddUserToChannel(ctx context.Context, userID, channelID uuid.UUID) error
	RemoveUserFromChannel(ctx context.Context, userID, channelID uuid.UUID) error
	GetAvailableChannelsForUser(ctx context.Context, userID uuid.UUID) ([]*models.Channel, error)

	GetActiveUsersByID(ctx context.Context, channelID uuid.UUID) ([]runtime.ActiveUser, error)
}

type channelUsecase struct {
	channelRepo    repository.ChannelRepository
	activeUserRepo memory.ActiveUserRepository
}

func NewChannelUsecase(channelRepo repository.ChannelRepository, activeUserRepo memory.ActiveUserRepository) ChannelUsecase {
	return &channelUsecase{channelRepo: channelRepo, activeUserRepo: activeUserRepo}
}

func (uc *channelUsecase) CreateChannel(ctx context.Context, input *input.CreateChannelInput) (*models.Channel, error) {
	channel := models.NewChannel(input)

	if err := uc.channelRepo.Create(ctx, channel); err != nil {
		return nil, fmt.Errorf("create channel: %w", err)
	}

	return channel, nil
}

func (uc *channelUsecase) GetChannel(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
	return uc.channelRepo.GetByID(ctx, id)
}

func (uc *channelUsecase) UpdateChannel(ctx context.Context, update *input.UpdateChannelInput) (*models.Channel, error) {
	channel, err := uc.channelRepo.GetByID(ctx, update.ID)
	if err != nil {
		return nil, fmt.Errorf("get channel by id: %w", err)
	}

	channel.Name = update.Name

	// TODO: на потом
	// channel.IsPublic = update.IsPublic

	if err := uc.channelRepo.Update(ctx, channel); err != nil {
		return nil, fmt.Errorf("update channel: %w", err)
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

func (uc *channelUsecase) GetAvailableChannelsForUser(ctx context.Context, userID uuid.UUID) ([]*models.Channel, error) {
	return uc.channelRepo.GetAvailableChannelsForUser(ctx, userID)
}

func (uc *channelUsecase) GetActiveUsersByID(ctx context.Context, channelID uuid.UUID) ([]runtime.ActiveUser, error) {
	return uc.activeUserRepo.GetInChannel(ctx, channelID), nil
}
