package memory

import (
	"context"
	"sync"

	"github.com/google/uuid"
)

type ChannelMembersRepository interface {
	// Add a member to a channel
	AddMember(ctx context.Context, channelID uuid.UUID, memberID uuid.UUID)

	// Remove a member from a channel
	RemoveMember(ctx context.Context, channelID uuid.UUID, memberID uuid.UUID)

	// Get all members of a channel
	GetMembers(ctx context.Context, channelID uuid.UUID) []uuid.UUID
}

type channelMembersRepository struct {
	members map[uuid.UUID]map[uuid.UUID]struct{}
	mu      sync.RWMutex
}

func NewChannelMembersRepository() ChannelMembersRepository {
	return &channelMembersRepository{
		members: make(map[uuid.UUID]map[uuid.UUID]struct{}),
	}
}

func (r *channelMembersRepository) AddMember(ctx context.Context, channelID uuid.UUID, memberID uuid.UUID) {
	if _, ok := r.members[channelID]; !ok {
		r.members[channelID] = make(map[uuid.UUID]struct{})
	}

	r.members[channelID][memberID] = struct{}{}
}

func (r *channelMembersRepository) RemoveMember(ctx context.Context, channelID uuid.UUID, memberID uuid.UUID) {
	if _, ok := r.members[channelID]; !ok {
		return
	}

	delete(r.members[channelID], memberID)
}

func (r *channelMembersRepository) GetMembers(ctx context.Context, channelID uuid.UUID) []uuid.UUID {
	if _, ok := r.members[channelID]; !ok {
		return nil
	}

	var members []uuid.UUID

	for memberID := range r.members[channelID] {
		members = append(members, memberID)
	}

	return members
}
