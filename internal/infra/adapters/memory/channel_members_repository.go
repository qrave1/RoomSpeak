package memory

import (
	"context"
	"github.com/qrave1/RoomSpeak/internal/domain/models"
	"sync"

	"github.com/google/uuid"
)

type ChannelMembersRepository interface {
	// Add a member to a channel
	AddMember(ctx context.Context, channelID uuid.UUID, user *models.User)

	// Remove a member from a channel
	RemoveMember(ctx context.Context, channelID uuid.UUID, userID uuid.UUID)

	// Get all members of a channel
	GetMembers(ctx context.Context, channelID uuid.UUID) []*models.User
}

type channelMembersRepository struct {
	members map[uuid.UUID]map[uuid.UUID]*models.User
	mu      sync.RWMutex
}

func NewChannelMembersRepository() ChannelMembersRepository {
	return &channelMembersRepository{
		members: make(map[uuid.UUID]map[uuid.UUID]*models.User),
	}
}

func (r *channelMembersRepository) AddMember(ctx context.Context, channelID uuid.UUID, user *models.User) {
	if _, ok := r.members[channelID]; !ok {
		r.members[channelID] = make(map[uuid.UUID]*models.User)
	}

	r.members[channelID][user.ID] = user
}

func (r *channelMembersRepository) RemoveMember(ctx context.Context, channelID uuid.UUID, userID uuid.UUID) {
	if _, ok := r.members[channelID]; !ok {
		return
	}

	delete(r.members[channelID], userID)
}

func (r *channelMembersRepository) GetMembers(ctx context.Context, channelID uuid.UUID) []*models.User {
	if _, ok := r.members[channelID]; !ok {
		return nil
	}

	var members []*models.User

	for _, user := range r.members[channelID] {
		members = append(members, user)
	}

	return members
}
