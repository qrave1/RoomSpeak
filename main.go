package main

import (
	"context"
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
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
	"github.com/qrave1/RoomSpeak/internal/infrastructure/postgres"

	"github.com/qrave1/RoomSpeak/internal/config"
	"github.com/qrave1/RoomSpeak/internal/constant"
	"github.com/qrave1/RoomSpeak/internal/middleware"
	"github.com/qrave1/RoomSpeak/internal/signaling"
)

type ChannelManager struct {
	channels map[string]*Channel
	mu       sync.RWMutex
}

func NewChannelManager() *ChannelManager {
	return &ChannelManager{
		channels: make(map[string]*Channel),
	}
}

func (cm *ChannelManager) GetOrCreate(channelID string) *Channel {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if channel, exists := cm.channels[channelID]; exists {
		return channel
	}

	channel := NewChannel(channelID)

	cm.channels[channelID] = channel

	return channel
}

func (cm *ChannelManager) Remove(channelID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	delete(cm.channels, channelID)
}

func (cm *ChannelManager) ListIDs() []string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	listIDs := make([]string, 0, len(cm.channels))

	for _, ch := range cm.channels {
		listIDs = append(listIDs, ch.id)
	}

	return listIDs
}

type Channel struct {
	id       string
	sessions map[string]*Session
	mu       sync.RWMutex
}

func NewChannel(id string) *Channel {
	return &Channel{
		id:       id,
		sessions: make(map[string]*Session),
	}
}

func (c *Channel) AddSession(s *Session) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.sessions[s.id] = s

	c.broadcastParticipants()
}

func (c *Channel) RemoveSession(sessionID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.sessions, sessionID)

	c.broadcastParticipants()
}

func (c *Channel) broadcastParticipants() {
	parts := make([]string, 0, len(c.sessions))

	for _, session := range c.sessions {
		parts = append(parts, session.name)
	}

	for _, session := range c.sessions {
		session.WriteWS(map[string]interface{}{"type": "participants", "list": parts})
	}
}

func (c *Channel) BroadcastRTP(pkt *rtp.Packet, senderID string) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, session := range c.sessions {
		if session.id == senderID || session.peer == nil || session.peer.audioTrack == nil {
			continue
		}

		err := session.peer.audioTrack.WriteRTP(pkt)

		if err != nil {
			slog.Error(
				"write RTP",
				slog.Any(constant.Error, err),
				slog.Any(constant.SessionID, senderID),
			)
		}
	}
}

type Session struct {
	id   string
	name string

	channel *Channel

	wsMu   sync.Mutex
	wsConn *websocket.Conn

	peer     *Peer
	joinedAt time.Time
}

func NewSession(ws *websocket.Conn) *Session {
	return &Session{
		id:       uuid.NewString(),
		wsConn:   ws,
		joinedAt: time.Now(),
	}
}

func (s *Session) WriteWS(v any) error {
	s.wsMu.Lock()
	defer s.wsMu.Unlock()

	return s.wsConn.WriteJSON(v)
}

type Peer struct {
	peerConn   *webrtc.PeerConnection
	audioTrack *webrtc.TrackLocalStaticRTP
}

// TODO move to peerConnectionFactory
func createPeerConnection(cfg *config.Config) (*Peer, error) {
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

	return &Peer{peerConn: pc, audioTrack: audioTrack}, nil
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

	session := NewSession(ws)

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

	defer func() {
		if session.channel != nil {
			session.channel.RemoveSession(session.id)
		}
		if session.peer != nil && session.peer.peerConn != nil {
			session.peer.peerConn.Close()
		}
	}()

	for {
		select {
		case <-c.Request().Context().Done():
			return nil
		default:
			_, msg, err := session.wsConn.ReadMessage()
			if err != nil {
				slog.Error(
					"webSocket read error",
					slog.Any(constant.Error, err),
				)

				return nil
			}

			signalMessage := new(signaling.Message)

			if err = json.Unmarshal(msg, &signalMessage); err != nil {
				slog.Error("unmarshal websocket message", slog.Any(constant.Error, err))

				return nil
			}

			if err = h.handleMessage(session, signalMessage); err != nil {
				slog.Error("handle message", slog.Any(constant.Error, err))
			}
		}
	}
}

func (h *HttpHandler) handleMessage(
	session *Session,
	msg *signaling.Message,
) error {
	switch msg.Type {
	case "join":
		var joinEvent signaling.JoinEvent

		if err := json.Unmarshal(msg.Data, &joinEvent); err != nil {
			return err
		}

		if joinEvent.Name == "" {
			joinEvent.Name = "Anonymous"
		}

		if joinEvent.ChannelID == "" {
			session.WriteWS(map[string]interface{}{"type": constant.Error, "message": "channel_id is required"})
			return nil
		}

		session.name = joinEvent.Name

		channel := h.channelManager.GetOrCreate(joinEvent.ChannelID)

		channel.AddSession(session)

		session.channel = channel

		var err error
		session.peer, err = createPeerConnection(h.cfg)
		if err != nil {
			slog.Error("create peer connection", slog.Any(constant.Error, err))

			return nil
		}

		session.peer.peerConn.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
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
						session.channel.BroadcastRTP(pkt, session.id)
					}
				}
			}()
		})

		session.peer.peerConn.OnICECandidate(func(c *webrtc.ICECandidate) {
			if c == nil {
				return
			}
			session.WriteWS(map[string]interface{}{"type": "candidate", "candidate": c.ToJSON()})
		})

		session.peer.peerConn.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
			if state == webrtc.PeerConnectionStateFailed || state == webrtc.PeerConnectionStateDisconnected {
				slog.Error("PeerConnection bad status",
					slog.String(constant.State, state.String()),
					slog.String(constant.SessionID, session.id),
				)

				err := session.WriteWS(map[string]interface{}{
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

	case "offer":
		var offer signaling.SdpEvent

		if err := json.Unmarshal(msg.Data, &offer); err != nil {
			return err
		}

		if err := session.peer.peerConn.SetRemoteDescription(
			webrtc.SessionDescription{
				Type: webrtc.SDPTypeOffer,
				SDP:  offer.SDP,
			},
		); err != nil {
			return err
		}

		answer, err := session.peer.peerConn.CreateAnswer(nil)
		if err != nil {
			return err
		}

		if err = session.peer.peerConn.SetLocalDescription(answer); err != nil {
			slog.Error("set local description", slog.Any(constant.Error, err))

			return err
		}

		return session.WriteWS(map[string]interface{}{"type": "answer", "sdp": answer.SDP})

	case "answer":
		var answer signaling.SdpEvent

		if err := json.Unmarshal(msg.Data, &answer); err != nil {
			return err
		}

		if err := session.peer.peerConn.SetRemoteDescription(
			webrtc.SessionDescription{
				Type: webrtc.SDPTypeAnswer,
				SDP:  answer.SDP,
			},
		); err != nil {
			slog.Error("set remote description", slog.Any(constant.Error, err))

			return err
		}

	case "candidate":
		var candidate signaling.IceCandidateEvent

		if err := json.Unmarshal(msg.Data, &candidate); err != nil {
			return err
		}

		return session.peer.peerConn.AddICECandidate(candidate.Candidate)

	case "leave":
		if session.channel != nil {
			session.channel.RemoveSession(session.id)
		}
		if session.peer != nil && session.peer.peerConn != nil {
			session.peer.peerConn.Close()
		}
	case "ping":
		return session.WriteWS(map[string]interface{}{"type": "pong"})
	default:
		return errors.New("unknown message type")
	}

	return nil
}

func (h *HttpHandler) listChannelsHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, h.channelManager.ListIDs())
}

type CreateChannelRequest struct {
	ChannelID string `json:"channel_id"`
}

func (h *HttpHandler) createChannelHandler(c echo.Context) error {
	var req CreateChannelRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	if req.ChannelID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "channel_id is required"})
	}

	h.channelManager.GetOrCreate(req.ChannelID)

	return c.NoContent(http.StatusCreated)
}

func (h *HttpHandler) deleteChannelHandler(c echo.Context) error {
	channelID := c.Param("id")

	h.channelManager.Remove(channelID)

	return c.NoContent(http.StatusOK)
}

// Handler для выдачи ICE серверов
func (h *HttpHandler) iceServersHandler(c echo.Context) error {
	expiration := time.Now().Add(time.Hour).Unix()
	username := fmt.Sprintf("%d", expiration)

	// Создаём HMAC-SHA1 с использованием static-auth-secret
	mac := hmac.New(sha1.New, []byte(h.cfg.CoturnServer.Secret))
	mac.Write([]byte(username))
	password := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	response := webrtc.ICEServer{
		URLs: []string{
			h.cfg.TurnUDPServer.URLs[0],
			h.cfg.TurnTCPServer.URLs[0],
		},
		Username:   username,
		Credential: password,
	}

	return c.JSON(http.StatusOK, response)
}

type HttpHandler struct {
	cfg            *config.Config
	upgrader       *websocket.Upgrader
	channelManager *ChannelManager
	db             *sqlx.DB
}

func NewHttpHandler(
	cfg *config.Config,
	channelManager *ChannelManager,
	upgrader *websocket.Upgrader,
	db *sqlx.DB,
) *HttpHandler {
	return &HttpHandler{
		cfg:            cfg,
		channelManager: channelManager,
		upgrader:       upgrader,
		db:             db,
	}
}

func main() {
	ctx := context.Background()

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

	dbConn, err := postgres.NewPostgres(ctx, cfg.PostgresURL)
	if err != nil {
		slog.Error("connect to postgres", slog.Any(constant.Error, err))
		os.Exit(1)
	}
	defer dbConn.Close()

	httpHandler := NewHttpHandler(
		cfg,
		NewChannelManager(),
		&websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				if cfg.Debug {
					return true
				}

				return r.Header.Get("Origin") == cfg.Domain
			},
		},
		dbConn,
	)

	e := echo.New()

	e.Use(middleware.SlogLogger())

	api := e.Group("/api")
	api.GET("/channels", httpHandler.listChannelsHandler)
	api.POST("/channels", httpHandler.createChannelHandler)
	api.DELETE("/channels/:id", httpHandler.deleteChannelHandler)

	e.Static("/", "web")
	e.GET("/ws", httpHandler.handleWebSocket)

	e.GET("/ice", httpHandler.iceServersHandler)

	err = e.Start(":3000")
	if err != nil {
		slog.Error(
			"HTTP server failed",
			slog.Any(constant.Error, err),
		)

		os.Exit(1)
	}
}
