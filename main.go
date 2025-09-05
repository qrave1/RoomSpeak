package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
	"github.com/qrave1/RoomSpeak/internal/dto"
	"github.com/qrave1/RoomSpeak/internal/middleware"
	"github.com/qrave1/RoomSpeak/internal/signaling"

	"github.com/qrave1/RoomSpeak/internal/config"
	"github.com/qrave1/RoomSpeak/internal/constant"
)

type RoomManager struct {
	rooms map[string]*Room
	mu    sync.RWMutex
}

func NewRoomManager() *RoomManager {
	return &RoomManager{
		rooms: make(map[string]*Room),
	}
}

func (rm *RoomManager) GetOrCreate(roomID string) *Room {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if room, exists := rm.rooms[roomID]; exists {
		return room
	}

	room := NewRoom(roomID)

	rm.rooms[roomID] = room

	return room
}

func (rm *RoomManager) Remove(roomID string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	delete(rm.rooms, roomID)
}

type Room struct {
	id       string
	sessions map[string]*Session
	mu       sync.RWMutex
}

func NewRoom(id string) *Room {
	return &Room{
		id:       id,
		sessions: make(map[string]*Session),
	}
}

func (r *Room) AddSession(c *Session) {
	slog.Info(
		"Session joined room",
		slog.String(constant.SessionID, c.id),
		slog.String(constant.SessionName, c.name),
		slog.String(constant.RoomID, r.id),
	)

	r.mu.Lock()
	defer r.mu.Unlock()

	r.sessions[c.id] = c

	r.broadcastParticipants()
}

func (r *Room) RemoveSession(sessionID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	slog.Info(
		"Session left room",
		slog.String(constant.SessionID, sessionID),
		slog.String(constant.RoomID, r.id),
	)

	delete(r.sessions, sessionID)

	r.broadcastParticipants()
}

func (r *Room) broadcastParticipants() {
	parts := make([]string, 0, len(r.sessions))

	for _, session := range r.sessions {
		parts = append(parts, session.name)
	}

	for _, session := range r.sessions {
		session.wsConn.WriteJSON(map[string]interface{}{"type": "participants", "list": parts})
	}
}

func (r *Room) BroadcastRTP(pkt *rtp.Packet, senderID string, trackType string) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, session := range r.sessions {
		if session.id == senderID {
			continue
		}

		var err error
		if trackType == "audio" {
			err = session.audioTrack.WriteRTP(pkt)
		} else if trackType == "video" {
			err = session.videoTrack.WriteRTP(pkt)
		}

		if err != nil {
			slog.Error(
				"write RTP",
				slog.Any(constant.Error, err),
				slog.String("track_type", trackType),
			)
		}
	}
}

type Session struct {
	id   string
	name string

	// TODO: move to roomID
	room *Room

	wsConn *websocket.Conn

	peerConn   *webrtc.PeerConnection
	audioTrack *webrtc.TrackLocalStaticRTP
	videoTrack *webrtc.TrackLocalStaticRTP
}

// TODO move to peerConnectionFactory
func createPeerConnection(cfg *config.Config) (*webrtc.PeerConnection, *webrtc.TrackLocalStaticRTP, *webrtc.TrackLocalStaticRTP, error) {
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
		return nil, nil, nil, err
	}

	audioTrack, err := webrtc.NewTrackLocalStaticRTP(
		webrtc.RTPCodecCapability{MimeType: "audio/opus"}, "audio", "RoomSpeak",
	)
	if err != nil {
		slog.Error(
			"create audio track",
			slog.Any(constant.Error, err),
		)

		return nil, nil, nil, err
	}

	if _, err = pc.AddTrack(audioTrack); err != nil {
		slog.Error(
			"add audio track",
			slog.Any(constant.Error, err),
		)

		return nil, nil, nil, err
	}

	videoTrack, err := webrtc.NewTrackLocalStaticRTP(
		webrtc.RTPCodecCapability{MimeType: "video/vp8"}, "video", "RoomSpeak",
	)
	if err != nil {
		slog.Error(
			"create video track",
			slog.Any(constant.Error, err),
		)

		return nil, nil, nil, err
	}

	if _, err = pc.AddTrack(videoTrack); err != nil {
		slog.Error(
			"add video track",
			slog.Any(constant.Error, err),
		)

		return nil, nil, nil, err
	}

	return pc, audioTrack, videoTrack, nil
}

func (h *HttpHandler) handleWebSocket(c echo.Context) error {
	ws, err := h.upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		slog.Error(
			"WebSocket upgrade error",
			slog.Any(constant.Error, err),
		)
		return err
	}
	defer ws.Close()

	err = ws.SetReadDeadline(time.Now().Add(60 * time.Second))
	if err != nil {
		return err
	}
	ws.SetPongHandler(func(string) error {
		ws.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	session := &Session{
		id:     uuid.NewString(),
		wsConn: ws,
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-ticker.C:
				if err := ws.WriteMessage(websocket.PingMessage, nil); err != nil {
					slog.Error("ping failed", slog.Any(constant.Error, err))
					return
				}
			case <-c.Request().Context().Done():
				return
			}
		}
	}()

	session.peerConn, session.audioTrack, session.videoTrack, err = createPeerConnection(h.cfg)
	if err != nil {
		slog.Error("create peer connection", slog.Any(constant.Error, err))

		return nil
	}

	defer func() {
		if session.room != nil {
			if len(session.room.sessions) == 0 {
				h.roomManager.Remove(session.room.id)
			} else {
				session.room.RemoveSession(session.id)
			}
		}
		session.peerConn.Close()
	}()

	// TODO: вынести все хендлеры для peerConnection в пакет signaling
	session.peerConn.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		go func() {
			for {
				pkt, _, err := track.ReadRTP()
				if err != nil {
					if !errors.Is(err, io.EOF) {
						slog.Error("RTP read error", "error", err)
					}

					return
				}

				if track.Kind() == webrtc.RTPCodecTypeAudio {
					session.room.BroadcastRTP(pkt, session.id, "audio")
				} else if track.Kind() == webrtc.RTPCodecTypeVideo {
					session.room.BroadcastRTP(pkt, session.id, "video")
				}
			}
		}()
	})

	session.peerConn.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		session.wsConn.WriteJSON(map[string]interface{}{"type": "candidate", "candidate": c.ToJSON()})
	})

	session.peerConn.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		if state == webrtc.PeerConnectionStateFailed || state == webrtc.PeerConnectionStateDisconnected {
			slog.Error("PeerConnection bad status",
				slog.String(constant.State, state.String()),
				slog.String(constant.SessionID, session.id),
			)

			err := session.wsConn.WriteJSON(map[string]interface{}{
				"type":    constant.Error,
				"message": fmt.Sprintf("peer connection bad state: %s", state.String()),
			})
			if err != nil {
				slog.Error("send peer connection state", slog.Any(constant.Error, err))
				return
			}
			return
		}
	})

	for {
		select {
		case <-c.Request().Context().Done():
			return nil
		default:
			_, msg, err := session.wsConn.ReadMessage()
			if err != nil && IsConnectionClosed(err) {
				slog.Error(
					"webSocket read error",
					slog.Any(constant.Error, err),
				)

				return nil
			}

			signalMessage := new(signaling.Message)

			if err := json.Unmarshal(msg, &signalMessage); err != nil {
				return err
			}

			if err := h.handleMessage(session, signalMessage); err != nil {
				slog.Error("handle message", slog.Any(constant.Error, err))
			}
		}
	}
}

// IsConnectionClosed проверяет, закрыто ли соединение
func IsConnectionClosed(err error) bool {
	return websocket.IsCloseError(err) ||
		websocket.IsUnexpectedCloseError(err) ||
		strings.Contains(err.Error(), "use of closed network connection")
}

func (h *HttpHandler) handleMessage(session *Session,
	msg *signaling.Message) error {
	switch msg.Type {
	case "join":
		var joinEvent signaling.JoinEvent

		if err := json.Unmarshal(msg.Data, &joinEvent); err != nil {
			return err
		}

		if joinEvent.Name == "" {
			joinEvent.Name = "Anonymous"
		}

		if joinEvent.RoomID == "" {
			session.wsConn.WriteJSON(map[string]interface{}{"type": constant.Error, "message": "room_id is required"})
			return errors.New("room_id is required")
		}

		session.name = joinEvent.Name

		room := h.roomManager.GetOrCreate(joinEvent.RoomID)

		room.AddSession(session)

		session.room = room

	case "offer":
		var offer signaling.SdpEvent

		if err := json.Unmarshal(msg.Data, &offer); err != nil {
			return err
		}

		if err := session.peerConn.SetRemoteDescription(
			webrtc.SessionDescription{
				Type: webrtc.SDPTypeOffer,
				SDP:  offer.SDP,
			},
		); err != nil {
			return err
		}

		answer, err := session.peerConn.CreateAnswer(nil)
		if err != nil {
			return err
		}

		if err = session.peerConn.SetLocalDescription(answer); err != nil {
			slog.Error("set local description", slog.Any(constant.Error, err))

			return err
		}

		return session.wsConn.WriteJSON(map[string]interface{}{"type": "answer", "sdp": answer.SDP})

	case "answer":
		var answer signaling.SdpEvent

		if err := json.Unmarshal(msg.Data, &answer); err != nil {
			return err
		}

		if err := session.peerConn.SetRemoteDescription(
			webrtc.SessionDescription{
				Type: webrtc.SDPTypeAnswer,
				SDP:  answer.SDP,
			},
		); err != nil {
			slog.Error("set remote description", slog.Any(constant.Error, err))

			return err
		}

	case "candidate":
		var candidate signaling.CandidateEvent

		if err := json.Unmarshal(msg.Data, &candidate); err != nil {
			return err
		}

		return session.peerConn.AddICECandidate(candidate.Candidate)

	case "ping":
		slog.Info(
			"pong",
			slog.Any(constant.SessionID, session.id),
			slog.Any(constant.RoomID, session.room.id),
		)

		return session.wsConn.WriteJSON(map[string]interface{}{"type": "pong"})
	default:
		return errors.New("unknown message type")
	}

	return nil
}

// Handler для выдачи TURN-кредитов
func (h *HttpHandler) turnCredentialsHandler(c echo.Context) error {
	expiration := time.Now().Add(time.Hour).Unix()
	username := fmt.Sprintf("%d", expiration)

	// Создаём HMAC-SHA1 с использованием static-auth-secret
	mac := hmac.New(sha1.New, []byte(h.cfg.CoturnServer.Secret))
	mac.Write([]byte(username))
	password := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	response := dto.TurnCredentialsResponse{
		URLs: []string{
			h.cfg.TurnUDPServer.URLs[0],
			h.cfg.TurnTCPServer.URLs[0],
		},
		Username: username,
		Password: password,
		TTL:      3600,
	}

	return c.JSON(http.StatusOK, response)
}

type HttpHandler struct {
	cfg         *config.Config
	upgrader    *websocket.Upgrader
	roomManager *RoomManager
}

func NewHttpHandler(
	cfg *config.Config,
	roomManager *RoomManager,
	upgrader *websocket.Upgrader,
) *HttpHandler {
	return &HttpHandler{
		cfg:         cfg,
		roomManager: roomManager,
		upgrader:    upgrader,
	}
}

func main() {
	slog.SetDefault(
		slog.New(
			slog.NewJSONHandler(
				os.Stdout,
				&slog.HandlerOptions{Level: slog.LevelInfo},
			),
		),
	)

	cfg, err := config.New()
	if err != nil {
		slog.Error("parse config", slog.Any(constant.Error, err))
		os.Exit(1)
	}

	slog.Info("Running app", slog.Bool("debug", cfg.Debug))

	httpHandler := NewHttpHandler(
		cfg,
		NewRoomManager(),
		&websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				if cfg.Debug {
					return true
				}

				return r.Header.Get("Origin") == cfg.Domain
			},
		},
	)

	e := echo.New()

	e.Use(middleware.SlogLogger())

	e.Static("/", "web")
	e.GET("/ws", httpHandler.handleWebSocket)

	e.GET("/turn/credentials", httpHandler.turnCredentialsHandler)

	err = e.Start(":3000")
	if err != nil {
		slog.Error(
			"HTTP server failed",
			slog.Any(constant.Error, err),
		)

		os.Exit(1)
	}
}
