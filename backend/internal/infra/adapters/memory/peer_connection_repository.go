package memory

import (
	"sync"

	"github.com/google/uuid"
	"github.com/qrave1/RoomSpeak/internal/domain"
)

type PeerConnectionRepository interface {
	Add(uuid.UUID, *domain.Peer)
	Get(uuid.UUID) (*domain.Peer, bool)
	Remove(uuid.UUID)
}

type peerConnectionRepository struct {
	// peers хранит map[user_id]*Peer
	peers map[uuid.UUID]*domain.Peer
	mu    sync.RWMutex
}

func NewPeerConnectionRepository() *peerConnectionRepository {
	return &peerConnectionRepository{
		peers: make(map[uuid.UUID]*domain.Peer),
	}
}

func (r *peerConnectionRepository) Add(userID uuid.UUID, peer *domain.Peer) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.peers[userID] = peer
}

func (r *peerConnectionRepository) Get(userID uuid.UUID) (*domain.Peer, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	peer, ok := r.peers[userID]
	return peer, ok
}

func (r *peerConnectionRepository) Remove(userID uuid.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.peers, userID)

}
