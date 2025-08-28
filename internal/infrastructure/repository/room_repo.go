package repository

import (
	"sync"

	"github.com/qrave1/RoomSpeak/internal/domain"
)

// InMemoryRoomRepository implements RoomRepository in memory.
type InMemoryRoomRepository struct {
	rooms map[string]*domain.Room
	mu    sync.RWMutex
}

func NewInMemoryRoomRepository() *InMemoryRoomRepository {
	return &InMemoryRoomRepository{rooms: make(map[string]*domain.Room)}
}

func (r *InMemoryRoomRepository) GetOrCreate(roomID string) *domain.Room {
	r.mu.Lock()
	defer r.mu.Unlock()
	if room, exists := r.rooms[roomID]; exists {
		return room
	}
	room := domain.NewRoom(roomID)
	r.rooms[roomID] = room
	return room
}

func (r *InMemoryRoomRepository) Remove(roomID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.rooms, roomID)
}
