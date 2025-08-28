package domain

// TODO пока помойка, надо разнести по файлам

import (
	"sync"

	"github.com/gorilla/websocket"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
)

/*
	TODO
	получается что у нас в домене есть зависимости от webrtc и websocket,
	надо подумать как это разнести,
	возможно тут хранить только clientsIDs, а из другой репы доставать самих клиентов
*/

// Room struct.
type Room struct {
	id      string
	clients map[string]*Client
	mu      sync.RWMutex
}

func (r *Room) AddClient(c *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.clients[c.ID] = c
}

func (r *Room) RemoveClient(clientID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.clients, clientID)
}

func (r *Room) GetClients() map[string]*Client {
	r.mu.RLock()
	defer r.mu.RUnlock()

	clients := make(map[string]*Client, len(r.clients))

	for k, v := range r.clients {
		clients[k] = v
	}

	return clients
}

func NewRoom(id string) *Room {
	return &Room{id: id, clients: make(map[string]*Client)}
}

// Client struct.
type Client struct {
	ID     string
	Name   string
	RoomID string

	WsConn     *websocket.Conn
	PeerConn   *webrtc.PeerConnection
	AudioTrack *webrtc.TrackLocalStaticRTP
}

func NewClientFromCmd(cmd *JoinRoomCommand) *Client {
	return &Client{
		ID:         cmd.ClientID,
		Name:       cmd.Name,
		RoomID:     cmd.RoomID,
		WsConn:     cmd.WSConn,
		PeerConn:   cmd.PC,
		AudioTrack: cmd.Track,
	}
}

// Command interface for CQRS commands.
type Command interface{}

// Query interface for CQRS queries.
type Query interface{}

// JoinRoomCommand for joining a room.
type JoinRoomCommand struct {
	ClientID string
	Name     string
	RoomID   string
	WSConn   *websocket.Conn
	PC       *webrtc.PeerConnection
	Track    *webrtc.TrackLocalStaticRTP
}

// LeaveRoomCommand for leaving a room.
type LeaveRoomCommand struct {
	ClientID string
	RoomID   string
}

// BroadcastRTPCommand for broadcasting RTP packets.
type BroadcastRTPCommand struct {
	Packet   *rtp.Packet
	SenderID string
	RoomID   string
}

// ParticipantsQuery for getting room participants.
type ParticipantsQuery struct {
	RoomID string
}
