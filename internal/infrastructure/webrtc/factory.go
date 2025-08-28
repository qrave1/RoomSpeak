package webrtc

import (
	"github.com/pion/webrtc/v4"
)

// PeerConnectionFactory creates PeerConnections.
type PeerConnectionFactory interface {
	Create() (*webrtc.PeerConnection, *webrtc.TrackLocalStaticRTP, error)
}

// DefaultPeerConnectionFactory implements PeerConnectionFactory.
type DefaultPeerConnectionFactory struct{}

func (DefaultPeerConnectionFactory) Create() (*webrtc.PeerConnection, *webrtc.TrackLocalStaticRTP, error) {
	pc, err := webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{{URLs: []string{"stun:stun.l.google.com:19302"}}},
	})
	if err != nil {
		return nil, nil, err
	}
	audioTrack, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: "audio/opus"}, "audio", "RoomSpeak")
	if err != nil {
		return nil, nil, err
	}
	if _, err = pc.AddTrack(audioTrack); err != nil {
		return nil, nil, err
	}
	return pc, audioTrack, nil
}
