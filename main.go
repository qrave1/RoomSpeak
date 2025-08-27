package main

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
)

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
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

	if len(r.clients) == 0 {
		roomManager.Remove(r.id)
	}
}

func (r *Room) broadcastParticipants() {
	parts := make([]string, 0, len(r.clients))
	for _, client := range r.clients {
		parts = append(parts, client.name)
	}

	for _, client := range r.clients {
		client.conn.WriteJSON(map[string]interface{}{"type": "participants", "list": parts})
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
			slog.Error("RTP write error", "error", err)
		}
	}
}

type Client struct {
	id         string
	name       string
	conn       *websocket.Conn
	pc         *webrtc.PeerConnection
	room       *Room
	audioTrack *webrtc.TrackLocalStaticRTP
}

func createPeerConnection() (*webrtc.PeerConnection, *webrtc.TrackLocalStaticRTP, error) {
	pc, err := webrtc.NewPeerConnection(
		webrtc.Configuration{
			ICEServers: []webrtc.ICEServer{{URLs: []string{"stun:stun.l.google.com:19302"}}},
		},
	)
	if err != nil {
		return nil, nil, err
	}

	audioTrack, err := webrtc.NewTrackLocalStaticRTP(
		webrtc.RTPCodecCapability{MimeType: "audio/opus"}, "audio", "RoomSpeak",
	)
	if err != nil {
		return nil, nil, err
	}

	if _, err = pc.AddTrack(audioTrack); err != nil {
		return nil, nil, err
	}

	return pc, audioTrack, nil
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("WebSocket upgrade error", "error", err)
		return
	}
	defer conn.Close()

	pc, audioTrack, err := createPeerConnection()
	if err != nil {
		slog.Error("PeerConnection error", "error", err)
		return
	}

	client := &Client{
		id:         uuid.NewString(),
		conn:       conn,
		pc:         pc,
		audioTrack: audioTrack,
	}
	slog.Info("WebSocket connection established", "client_id", client.id)

	defer func() {
		if client.room != nil {
			client.room.RemoveClient(client.id)
		}
		pc.Close()
	}()

	pc.OnTrack(
		func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
			if track.Kind() == webrtc.RTPCodecTypeAudio {
				go func() {
					for {
						pkt, _, err := track.ReadRTP()
						if err != nil {
							return
						}
						client.room.BroadcastRTP(pkt, client.id)
					}
				}()
			}
		},
	)

	pc.OnICECandidate(
		func(c *webrtc.ICECandidate) {
			if c == nil {
				return
			}
			conn.WriteJSON(map[string]interface{}{"type": "candidate", "candidate": c.ToJSON()})
		},
	)

	pc.OnConnectionStateChange(
		func(state webrtc.PeerConnectionState) {
			switch state {
			case webrtc.PeerConnectionStateFailed:
				slog.Warn("PeerConnection state failed", "client_id", client.id)
				pc.Close()
			case webrtc.PeerConnectionStateDisconnected:
				slog.Warn("PeerConnection state disconnected", "client_id", client.id)
			default:
			}
		},
	)

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			slog.Error("WebSocket read error", "error", err)
			return
		}

		if err := handleClientMessage(client, msg); err != nil {
			slog.Error("Message handling error", "error", err)
			return
		}
	}
}

func handleClientMessage(c *Client, msg []byte) error {
	var base struct{ Type string }
	if err := json.Unmarshal(msg, &base); err != nil {
		return err
	}

	switch base.Type {
	case "join":
		var data struct{ Name, Room string }
		if err := json.Unmarshal(msg, &data); err != nil {
			return err
		}
		c.name = data.Name
		room := roomManager.GetOrCreate(data.Room)
		room.AddClient(c)
		c.room = room

	case "offer":
		var data struct{ SDP string }
		if err := json.Unmarshal(msg, &data); err != nil {
			return err
		}
		if err := c.pc.SetRemoteDescription(
			webrtc.SessionDescription{
				Type: webrtc.SDPTypeOffer, SDP: data.SDP,
			},
		); err != nil {
			return err
		}
		answer, err := c.pc.CreateAnswer(nil)
		if err != nil {
			return err
		}
		if err = c.pc.SetLocalDescription(answer); err != nil {
			return err
		}
		return c.conn.WriteJSON(map[string]interface{}{"type": "answer", "sdp": answer.SDP})

	case "candidate":
		var data struct{ Candidate webrtc.ICECandidateInit }
		if err := json.Unmarshal(msg, &data); err != nil {
			return err
		}
		return c.pc.AddICECandidate(data.Candidate)

	default:
		return errors.New("unknown message type")
	}
	return nil
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

	mux := http.NewServeMux()

	mux.Handle("/", http.FileServer(http.Dir("web")))
	mux.HandleFunc("/ws", handleWebSocket)

	slog.Info(
		"HTTP server starting",
		slog.Any("port", "3000"),
	)

	err := http.ListenAndServe(":3000", mux)
	if err != nil {
		slog.Error(
			"HTTP server failed",
			slog.Any("error", err),
		)

		os.Exit(1)
	}
}
