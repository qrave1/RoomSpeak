package domain

import (
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/pion/webrtc/v4"
	"github.com/qrave1/RoomSpeak/internal/application/config"
)

type Peer struct {
	UserID      uuid.UUID
	ChannelID   uuid.UUID
	Conn        *webrtc.PeerConnection
	AudioTracks map[uuid.UUID]*webrtc.TrackLocalStaticRTP // map[senderID]track
	mu          sync.RWMutex
}

func NewPeer(userID, channelID uuid.UUID, cfg *config.Config) (*Peer, error) {
	pc, err := webrtc.NewPeerConnection(
		webrtc.Configuration{
			ICEServers: []webrtc.ICEServer{
				{
					URLs: []string{"stun:stun.l.google.com:19302"},
				},
				cfg.TurnUDPServer,
				cfg.TurnTCPServer,
			},
		},
	)
	if err != nil {
		return nil, err
	}

	return &Peer{
		UserID:      userID,
		ChannelID:   channelID,
		Conn:        pc,
		AudioTracks: make(map[uuid.UUID]*webrtc.TrackLocalStaticRTP),
	}, nil
}

// AddAudioTrack создает и добавляет новый аудио трек для указанного отправителя
func (p *Peer) AddAudioTrack(senderID uuid.UUID) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Проверяем, не существует ли уже трек для этого отправителя
	if _, exists := p.AudioTracks[senderID]; exists {
		return nil
	}

	audioTrack, err := webrtc.NewTrackLocalStaticRTP(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus},
		fmt.Sprintf("audio-%s", senderID.String()),
		fmt.Sprintf("RoomSpeak-%s", senderID.String()),
	)
	if err != nil {
		return fmt.Errorf("create audio track for sender %s: %w", senderID, err)
	}

	if _, err = p.Conn.AddTrack(audioTrack); err != nil {
		return fmt.Errorf("add audio track for sender %s: %w", senderID, err)
	}

	p.AudioTracks[senderID] = audioTrack
	return nil
}

// RemoveAudioTrack удаляет аудио трек для указанного отправителя
func (p *Peer) RemoveAudioTrack(senderID uuid.UUID) {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.AudioTracks, senderID)
}

// GetAudioTrack возвращает аудио трек для указанного отправителя
func (p *Peer) GetAudioTrack(senderID uuid.UUID) (*webrtc.TrackLocalStaticRTP, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	track, exists := p.AudioTracks[senderID]
	return track, exists
}
