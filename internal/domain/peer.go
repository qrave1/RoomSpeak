package domain

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/pion/webrtc/v4"
	"github.com/qrave1/RoomSpeak/internal/application/config"
)

type Peer struct {
	UserID     uuid.UUID
	ChannelID  uuid.UUID
	Conn       *webrtc.PeerConnection
	AudioTrack *webrtc.TrackLocalStaticRTP
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

	audioTrack, err := webrtc.NewTrackLocalStaticRTP(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus}, "audio", "RoomSpeak",
	)
	if err != nil {
		return nil, fmt.Errorf("create audio track: %w", err)
	}

	if _, err = pc.AddTrack(audioTrack); err != nil {
		return nil, fmt.Errorf("add audio track: %w", err)
	}

	return &Peer{
		UserID:     userID,
		ChannelID:  channelID,
		Conn:       pc,
		AudioTrack: audioTrack,
	}, nil
}
