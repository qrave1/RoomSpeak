package sfu

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/pion/webrtc/v3"
	"github.com/qrave1/RoomSpeak/internal/models"
)

// SFU представляет Selective Forwarding Unit
type SFU struct {
	api        *webrtc.API
	config     webrtc.Configuration
	rooms      map[string]*Room
	roomsMutex sync.RWMutex
}

// Room представляет комнату в SFU
type Room struct {
	ID           string
	Clients      map[string]*Client
	clientsMutex sync.RWMutex
}

// Client представляет клиента в SFU
type Client struct {
	ID       string
	PeerConn *webrtc.PeerConnection
	Room     *Room
	OnTrack  func(*webrtc.TrackRemote, *webrtc.RTPReceiver)
}

// NewSFU создает новый экземпляр SFU
func NewSFU() *SFU {
	// Создаем MediaEngine с кодеками
	mediaEngine := &webrtc.MediaEngine{}

	// Добавляем поддержку Opus аудио кодека
	if err := mediaEngine.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType:    webrtc.MimeTypeOpus,
			ClockRate:   48000,
			Channels:    2,
			SDPFmtpLine: "minptime=10;useinbandfec=1",
		},
		PayloadType: 111,
	}, webrtc.RTPCodecTypeAudio); err != nil {
		log.Printf("Error registering Opus codec: %v", err)
	}

	// Создаем API
	api := webrtc.NewAPI(webrtc.WithMediaEngine(mediaEngine))

	// Конфигурация WebRTC
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	}

	return &SFU{
		api:    api,
		config: config,
		rooms:  make(map[string]*Room),
	}
}

// CreateRoom создает новую комнату
func (s *SFU) CreateRoom(roomID string) *Room {
	s.roomsMutex.Lock()
	defer s.roomsMutex.Unlock()

	if room, exists := s.rooms[roomID]; exists {
		return room
	}

	room := &Room{
		ID:      roomID,
		Clients: make(map[string]*Client),
	}

	s.rooms[roomID] = room
	log.Printf("Created room: %s", roomID)
	return room
}

// GetRoom получает комнату по ID
func (s *SFU) GetRoom(roomID string) *Room {
	s.roomsMutex.RLock()
	defer s.roomsMutex.RUnlock()
	return s.rooms[roomID]
}

// JoinRoom присоединяет клиента к комнате
func (s *SFU) JoinRoom(roomID, clientID string) (*Client, error) {
	room := s.GetRoom(roomID)
	if room == nil {
		room = s.CreateRoom(roomID)
	}

	// Создаем PeerConnection для клиента
	peerConn, err := s.api.NewPeerConnection(s.config)
	if err != nil {
		return nil, fmt.Errorf("failed to create peer connection: %w", err)
	}

	client := &Client{
		ID:       clientID,
		PeerConn: peerConn,
		Room:     room,
	}

	// Обработчик входящих треков
	peerConn.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		log.Printf("Track received from client %s: %s", clientID, track.ID())

		// Ретранслируем трек всем остальным клиентам в комнате
		go s.forwardTrack(room, clientID, track)
	})

	// Обработчик изменения состояния соединения
	peerConn.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Printf("Client %s connection state: %s", clientID, state.String())
		if state == webrtc.PeerConnectionStateClosed || state == webrtc.PeerConnectionStateFailed {
			s.LeaveRoom(roomID, clientID)
		}
	})

	// Добавляем клиента в комнату
	room.clientsMutex.Lock()
	room.Clients[clientID] = client
	room.clientsMutex.Unlock()

	log.Printf("Client %s joined room %s", clientID, roomID)
	return client, nil
}

// LeaveRoom удаляет клиента из комнаты
func (s *SFU) LeaveRoom(roomID, clientID string) {
	room := s.GetRoom(roomID)
	if room == nil {
		return
	}

	room.clientsMutex.Lock()
	defer room.clientsMutex.Unlock()

	client, exists := room.Clients[clientID]
	if !exists {
		return
	}

	// Закрываем PeerConnection
	if client.PeerConn != nil {
		client.PeerConn.Close()
	}

	delete(room.Clients, clientID)
	log.Printf("Client %s left room %s", clientID, roomID)

	// Удаляем комнату, если она пустая
	if len(room.Clients) == 0 {
		s.roomsMutex.Lock()
		delete(s.rooms, roomID)
		s.roomsMutex.Unlock()
		log.Printf("Removed empty room: %s", roomID)
	}
}

// forwardTrack ретранслирует трек всем остальным клиентам в комнате
func (s *SFU) forwardTrack(room *Room, senderID string, track *webrtc.TrackRemote) {
	room.clientsMutex.RLock()
	defer room.clientsMutex.RUnlock()

	for clientID, client := range room.Clients {
		if clientID == senderID {
			continue
		}

		// Добавляем трек для отправки клиенту
		sender, err := client.PeerConn.AddTrack(track)
		if err != nil {
			log.Printf("Error adding track to client %s: %v", clientID, err)
			continue
		}

		// Обработка RTCP пакетов
		go func(sender *webrtc.RTPSender, clientID string) {
			rtcpBuf := make([]byte, 1500)
			for {
				if _, _, rtcpErr := sender.Read(rtcpBuf); rtcpErr != nil {
					log.Printf("RTCP read error for client %s: %v", clientID, rtcpErr)
					return
				}
			}
		}(sender, clientID)
	}
}

// CreateOffer создает offer для клиента
func (c *Client) CreateOffer(ctx context.Context) (*webrtc.SessionDescription, error) {
	offer, err := c.PeerConn.CreateOffer(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create offer: %w", err)
	}

	if err := c.PeerConn.SetLocalDescription(offer); err != nil {
		return nil, fmt.Errorf("failed to set local description: %w", err)
	}

	return &offer, nil
}

// CreateAnswer создает answer для клиента
func (c *Client) CreateAnswer(ctx context.Context, offer *webrtc.SessionDescription) (*webrtc.SessionDescription, error) {
	if err := c.PeerConn.SetRemoteDescription(*offer); err != nil {
		return nil, fmt.Errorf("failed to set remote description: %w", err)
	}

	answer, err := c.PeerConn.CreateAnswer(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create answer: %w", err)
	}

	if err := c.PeerConn.SetLocalDescription(answer); err != nil {
		return nil, fmt.Errorf("failed to set local description: %w", err)
	}

	return &answer, nil
}

// AddICECandidate добавляет ICE кандидата
func (c *Client) AddICECandidate(candidate *webrtc.ICECandidateInit) error {
	return c.PeerConn.AddICECandidate(*candidate)
}

// GetClients возвращает список клиентов в комнате
func (r *Room) GetClients() []*models.Client {
	r.clientsMutex.RLock()
	defer r.clientsMutex.RUnlock()

	clients := make([]*models.Client, 0, len(r.Clients))
	for _, client := range r.Clients {
		clients = append(clients, &models.Client{
			ID:          client.ID,
			RoomID:      r.ID,
			IsConnected: client.PeerConn.ConnectionState() == webrtc.PeerConnectionStateConnected,
		})
	}

	return clients
}
