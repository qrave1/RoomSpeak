package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
)

type Config struct {
	Debug      bool   `env:"DEBUG" envDefault:"false"`
	Port       string `env:"PORT" envDefault:"3000"`
	Domain     string `env:"DOMAIN" envDefault:"https://xxsm.ru"`
	TurnServer TurnServerConfig
}

type TurnServerConfig struct {
	URL      string `env:"TURN_URL,required"`
	Username string `env:"TURN_USERNAME,required"`
	Password string `env:"TURN_PASSWORD,required"`
}

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
	id      string
	clients map[string]*Client
	mu      sync.RWMutex
}

func NewRoom(id string) *Room {
	return &Room{
		id:      id,
		clients: make(map[string]*Client),
	}
}

func (r *Room) AddClient(c *Client) {
	slog.Info("Client joined room", "client_id", c.id, "client_name", c.name, "room_id", r.id)

	r.mu.Lock()
	defer r.mu.Unlock()

	r.clients[c.id] = c

	r.broadcastParticipants()
}

func (r *Room) RemoveClient(clientID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	slog.Info("Client left room", "client_id", clientID, "room_id", r.id)

	delete(r.clients, clientID)

	r.broadcastParticipants()
}

func (r *Room) broadcastParticipants() {
	parts := make([]string, 0, len(r.clients))

	for _, client := range r.clients {
		parts = append(parts, client.name)
	}

	for _, client := range r.clients {
		client.wsConn.WriteJSON(map[string]interface{}{"type": "participants", "list": parts})
	}
}

func (r *Room) BroadcastRTP(pkt *rtp.Packet, senderID string, trackType string) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, client := range r.clients {
		if client.id == senderID {
			continue
		}

		var err error
		if trackType == "audio" {
			err = client.audioTrack.WriteRTP(pkt)
		} else if trackType == "video" {
			err = client.videoTrack.WriteRTP(pkt)
		}

		if err != nil {
			slog.Error("write RTP", "error", err, "track_type", trackType)
		}
	}
}

type Client struct {
	id         string
	name       string
	wsConn     *websocket.Conn
	peerConn   *webrtc.PeerConnection
	room       *Room
	audioTrack *webrtc.TrackLocalStaticRTP
	videoTrack *webrtc.TrackLocalStaticRTP
}

// TODO: refactor shit
func createPeerConnection(cfg *Config) (*webrtc.PeerConnection, *webrtc.TrackLocalStaticRTP, *webrtc.TrackLocalStaticRTP, error) {
	pc, err := webrtc.NewPeerConnection(
		webrtc.Configuration{
			ICEServers: []webrtc.ICEServer{
				{
					URLs: []string{"stun:stun.l.google.com:19302"},
				},
				{
					URLs:       []string{fmt.Sprintf("turn:%s", cfg.TurnServer.URL)},
					Username:   cfg.TurnServer.Username,
					Credential: cfg.TurnServer.Password,
				},
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
		slog.Error("create audio track", "error", err)
		return nil, nil, nil, err
	}

	videoTrack, err := webrtc.NewTrackLocalStaticRTP(
		webrtc.RTPCodecCapability{MimeType: "video/vp8"}, "video", "RoomSpeak",
	)
	if err != nil {
		slog.Error("create video track", "error", err)
		return nil, nil, nil, err
	}

	if _, err = pc.AddTrack(audioTrack); err != nil {
		slog.Error("add audio track", "error", err)
		return nil, nil, nil, err
	}

	if _, err = pc.AddTrack(videoTrack); err != nil {
		slog.Error("add video track", "error", err)
		return nil, nil, nil, err
	}

	return pc, audioTrack, videoTrack, nil
}

func (h *HttpHandler) handleWebSocket(c echo.Context) error {
	ws, err := h.upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		slog.Error("WebSocket upgrade error", "error", err)
		return err
	}
	defer ws.Close()

	err = ws.SetReadDeadline(time.Now().Add(60 * time.Second))
	if err != nil {
		return err
	}
	ws.SetPongHandler(func(string) error {
		slog.Info("Получен pong")
		ws.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-ticker.C:
				if err := ws.WriteMessage(websocket.PingMessage, nil); err != nil {
					slog.Error("ping failed", slog.Any("err", err))
					return
				}
			case <-c.Request().Context().Done():
				slog.Info("client context done")
				return
			}
		}
	}()

	pc, audioTrack, videoTrack, err := createPeerConnection(h.cfg)
	if err != nil {
		slog.Error("create peer connection", "error", err)
		return nil
	}

	client := &Client{
		id:         uuid.NewString(),
		wsConn:     ws,
		peerConn:   pc,
		audioTrack: audioTrack,
		videoTrack: videoTrack,
	}

	slog.Info("WebSocket connection established", "client_id", client.id)

	defer func() {
		if client.room != nil {
			if len(client.room.clients) == 0 {
				h.roomManager.Remove(client.room.id)
			} else {
				client.room.RemoveClient(client.id)
			}
		}
		ws.Close()
		pc.Close()
	}()

	pc.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
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
					client.room.BroadcastRTP(pkt, client.id, "audio")
				} else if track.Kind() == webrtc.RTPCodecTypeVideo {
					client.room.BroadcastRTP(pkt, client.id, "video")
				}
			}
		}()
	})

	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		ws.WriteJSON(map[string]interface{}{"type": "candidate", "candidate": c.ToJSON()})
	})

	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		slog.Info("PeerConnection state change", "state", state.String(), "client_id", client.id)
		if state == webrtc.PeerConnectionStateFailed || state == webrtc.PeerConnectionStateDisconnected {
			slog.Error("PeerConnection bad status", "client_id", client.id, "state", state.String())
			return
		}
	})

	for {
		_, msg, err := ws.ReadMessage()
		if err != nil {
			slog.Error("WebSocket read error", "error", err)
			return nil
		}
		if err := h.handleClientMessage(client, msg); err != nil {
			slog.Error("Message handling error", "error", err)
		}
	}
}

func (h *HttpHandler) handleClientMessage(c *Client, msg []byte) error {
	var base struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(msg, &base); err != nil {
		return err
	}

	switch base.Type {
	case "join":
		var data struct {
			Name   string `json:"name"`
			RoomID string `json:"room"`
		}
		if err := json.Unmarshal(msg, &data); err != nil {
			return err
		}

		c.name = data.Name

		room := h.roomManager.GetOrCreate(data.RoomID)

		room.AddClient(c)

		c.room = room

	case "offer":
		var data struct{ SDP string }
		if err := json.Unmarshal(msg, &data); err != nil {
			return err
		}

		if err := c.peerConn.SetRemoteDescription(
			webrtc.SessionDescription{
				Type: webrtc.SDPTypeOffer, SDP: data.SDP,
			},
		); err != nil {
			return err
		}

		answer, err := c.peerConn.CreateAnswer(nil)
		if err != nil {
			return err
		}

		if err = c.peerConn.SetLocalDescription(answer); err != nil {
			return err
		}

		return c.wsConn.WriteJSON(map[string]interface{}{"type": "answer", "sdp": answer.SDP})

	case "candidate":
		var data struct{ Candidate webrtc.ICECandidateInit }
		if err := json.Unmarshal(msg, &data); err != nil {
			return err
		}

		return c.peerConn.AddICECandidate(data.Candidate)

	case "ping":
		slog.Info(
			"pong",
			slog.Any("client_id", c.id),
			slog.Any("room_id", c.room.id),
		)

		return c.wsConn.WriteJSON(map[string]interface{}{"type": "pong"})
	default:
		return errors.New("unknown message type")
	}

	return nil
}

// Кастомный логгер через slog
func SlogLogger() echo.MiddlewareFunc {
	return middleware.RequestLoggerWithConfig(
		middleware.RequestLoggerConfig{
			LogStatus: true,
			LogURI:    true,
			LogMethod: true,
			LogError:  true,

			LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
				level := slog.LevelInfo
				if v.Error != nil || v.Status >= http.StatusInternalServerError {
					level = slog.LevelError
				} else if v.Status >= http.StatusBadRequest {
					level = slog.LevelWarn
				}

				slog.LogAttrs(
					c.Request().Context(),
					level,
					"HTTP request",
					slog.Int("status", v.Status),
					slog.String("uri", v.URI),
					slog.String("method", v.Method),
				)

				return nil
			},
		},
	)
}

type HttpHandler struct {
	cfg         *Config
	upgrader    *websocket.Upgrader
	roomManager *RoomManager
}

func NewHttpHandler(
	cfg *Config,
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

	cfg, err := env.ParseAs[Config]()
	if err != nil {
		slog.Error("parse config", "error", err)
		os.Exit(1)
	}

	slog.Info("Running app", slog.Any("debug", cfg.Debug))

	httpHandler := &HttpHandler{
		cfg: &cfg,
		upgrader: &websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				if cfg.Debug {
					return true
				}

				return r.Header.Get("Origin") == cfg.Domain
			},
		},
		roomManager: NewRoomManager(),
	}

	e := echo.New()

	e.Use(SlogLogger())

	e.Static("/", "web")
	e.GET("/ws", httpHandler.handleWebSocket)

	err = e.Start(":3000")
	if err != nil {
		slog.Error(
			"HTTP server failed",
			slog.Any("error", err),
		)

		os.Exit(1)
	}
}
