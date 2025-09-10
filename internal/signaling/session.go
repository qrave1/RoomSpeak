package signaling

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
	"github.com/qrave1/RoomSpeak/internal/constant"
)

type Session struct {
	ID     string
	Name   string
	RoomID string

	// WS
	wsMu   sync.Mutex
	wsConn *websocket.Conn

	// WebRTC
	peerConn   *webrtc.PeerConnection
	audioTrack *webrtc.TrackLocalStaticRTP
}

func NewSession() *Session {
	return &Session{
		ID: uuid.NewString(),
	}
}

func (s *Session) SetWebsocketConnection(ws *websocket.Conn) {
	s.wsConn = ws
}

func (s *Session) SetPeerConnection(pc *webrtc.PeerConnection) {
	s.peerConn = pc
}

func (s *Session) SetAudioTrack(track *webrtc.TrackLocalStaticRTP) {
	s.audioTrack = track
}

func (s *Session) WriteWS(v any) error {
	s.wsMu.Lock()
	defer s.wsMu.Unlock()

	return s.wsConn.WriteJSON(v)
}

func (s *Session) WriteRTP(pkt *rtp.Packet) error {
	return s.audioTrack.WriteRTP(pkt)
}

func (s *Session) Close() error {
	s.wsMu.Lock()
	defer s.wsMu.Unlock()

	if s.wsConn != nil {
		if err := s.wsConn.Close(); err != nil {
			return fmt.Errorf("close websocket: %w", err)
		}
	}

	if s.peerConn != nil {
		if err := s.peerConn.Close(); err != nil {
			return fmt.Errorf("close peer: %w", err)
		}
	}

	return nil
}

func (s *Session) TrackHandler(track *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
	go func() {
		for {
			pkt, _, err := track.ReadRTP()
			if err != nil {
				if !errors.Is(err, io.EOF) {
					slog.Error("RTP read error", slog.Any(constant.Error, err))
				}

				return
			}

			if track.Kind() == webrtc.RTPCodecTypeAudio {
				session.room.BroadcastRTP(pkt, session.id)
			}
		}
	}()
}
