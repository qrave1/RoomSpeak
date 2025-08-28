package repository

import (
	"github.com/qrave1/RoomSpeak/internal/domain"
)

// InMemoryClientRepository implements ClientRepository.
type InMemoryClientRepository struct {
}

func (InMemoryClientRepository) AddClient(room *domain.Room, c *domain.Client) {
	room.AddClient(c)
}

func (InMemoryClientRepository) RemoveClient(room *domain.Room, clientID string) {
	room.RemoveClient(clientID)
}

func (InMemoryClientRepository) GetClients(room *domain.Room) map[string]*domain.Client {
	return room.GetClients()
}
