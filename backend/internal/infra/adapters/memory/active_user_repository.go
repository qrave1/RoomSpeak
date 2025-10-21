package memory

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/qrave1/RoomSpeak/backend/internal/domain/runtime"
)

type ActiveUserRepository interface {
	// Add an active user to a channel
	Add(ctx context.Context, activeUser runtime.ActiveUser)

	// Remove an active user from a channel
	Remove(ctx context.Context, userID uuid.UUID)

	// Get all active users in a channel
	GetInChannel(ctx context.Context, channelID uuid.UUID) []runtime.ActiveUser

	// Get active user by ID
	GetByID(ctx context.Context, userID uuid.UUID) (runtime.ActiveUser, bool)
}

type activeUserRepository struct {
	activeUsers map[uuid.UUID]runtime.ActiveUser
	mu          sync.RWMutex
}

func NewActiveUserRepository() ActiveUserRepository {
	return &activeUserRepository{
		activeUsers: make(map[uuid.UUID]runtime.ActiveUser),
	}
}

func (r *activeUserRepository) Add(ctx context.Context, activeUser runtime.ActiveUser) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.activeUsers[activeUser.ID] = activeUser
}

func (r *activeUserRepository) Remove(ctx context.Context, userID uuid.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.activeUsers, userID)
}

func (r *activeUserRepository) GetInChannel(ctx context.Context, channelID uuid.UUID) []runtime.ActiveUser {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var activeUsers []runtime.ActiveUser

	for _, activeUser := range r.activeUsers {
		if activeUser.ChannelID == channelID {
			activeUsers = append(activeUsers, activeUser)
		}
	}

	return activeUsers
}

func (r *activeUserRepository) GetByID(ctx context.Context, userID uuid.UUID) (runtime.ActiveUser, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	activeUser, exists := r.activeUsers[userID]

	return activeUser, exists
}
