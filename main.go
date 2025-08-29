package main

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
)

type Config struct {
	Debug bool   `env:"DEBUG" envDefault:"true"`
	Port  string `env:"PORT" envDefault:"3000"`
}

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			//return r.Header.Get("Origin") == "https://xxsm.ru"
			return true
		},
	}
	roomManager = NewRoomManager()
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

func (r *Room) BroadcastRTP(pkt *rtp.Packet, senderID string) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, client := range r.clients {
		if client.id == senderID {
			continue
		}

		if err := client.audioTrack.WriteRTP(pkt); err != nil {
			slog.Error("write RTP", "error", err)
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
}

func createPeerConnection() (*webrtc.PeerConnection, *webrtc.TrackLocalStaticRTP, error) {
	pc, err := webrtc.NewPeerConnection(
		webrtc.Configuration{
			ICEServers: []webrtc.ICEServer{
				{
					URLs: []string{"stun:stun.l.google.com:19302"},
				},
				{
					URLs:       []string{"turn:relay1.expressturn.com:3480"},
					Username:   "000000002071768906",
					Credential: "0hagJJHrDeiDO6zu7+GRqTcMTrc=",
				},
			},
		},
	)
	if err != nil {
		return nil, nil, err
	}

	audioTrack, err := webrtc.NewTrackLocalStaticRTP(
		webrtc.RTPCodecCapability{MimeType: "audio/opus"}, "audio", "RoomSpeak",
	)
	if err != nil {
		slog.Error("create audio track error", "error", err)

		return nil, nil, err
	}

	if _, err = pc.AddTrack(audioTrack); err != nil {
		slog.Error("add audio track error", "error", err)

		return nil, nil, err
	}

	return pc, audioTrack, nil
}

func handleWebSocket(c echo.Context) error {
	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		slog.Error("WebSocket upgrade error", "error", err)
		return err // Let Echo handle HTTP error
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

	pc, audioTrack, err := createPeerConnection()
	if err != nil {
		slog.Error("PeerConnection error", "error", err)
		return nil // Don't return HTTP error after upgrade
	}

	client := &Client{
		id:         uuid.NewString(),
		wsConn:     ws,
		peerConn:   pc,
		audioTrack: audioTrack,
	}

	slog.Info("WebSocket connection established", "client_id", client.id)

	defer func() {
		if client.room != nil {
			if len(client.room.clients) == 0 {
				roomManager.Remove(client.room.id)
			} else {
				client.room.RemoveClient(client.id)
			}
		}
		ws.Close()
		pc.Close()
	}()

	// WebRTC handlers (unchanged)
	pc.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		//slog.Info("Received track", "kind", track.Kind(), "id", track.ID())
		if track.Kind() == webrtc.RTPCodecTypeAudio {
			go func() {
				for {
					pkt, _, err := track.ReadRTP()
					if err != nil {
						slog.Error("RTP read error", "error", err)
						return
					}
					//slog.Info("RTP packet received", "seq", pkt.SequenceNumber)
					client.room.BroadcastRTP(pkt, client.id)
				}
			}()
		}
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
			return nil // Don't return HTTP error
		}
		if err := handleClientMessage(client, msg); err != nil {
			slog.Error("Message handling error", "error", err)
		}
	}
}

func handleClientMessage(c *Client, msg []byte) error {
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

		room := roomManager.GetOrCreate(data.RoomID)

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
					slog.String("method", v.Method),
					slog.String("uri", v.URI),
					slog.Int("status", v.Status),
					slog.String("remote_ip", c.RealIP()),
				)

				return nil
			},
		},
	)
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

	e := echo.New()

	e.Use(SlogLogger())

	e.Static("/", "web")
	e.GET("/ws", handleWebSocket)

	err := e.Start(":3000")
	if err != nil {
		slog.Error(
			"HTTP server failed",
			slog.Any("error", err),
		)

		os.Exit(1)
	}
}
