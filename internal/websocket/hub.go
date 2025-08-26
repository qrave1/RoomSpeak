package websocket

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/qrave1/RoomSpeak/internal/database"
	"github.com/qrave1/RoomSpeak/internal/models"
	"github.com/qrave1/RoomSpeak/internal/sfu"
)

// Hub управляет WebSocket соединениями и комнатами
type Hub struct {
	// Активные клиенты: clientID -> Client
	clients map[string]*models.Client

	// Комнаты: roomID -> []clientID
	rooms map[string][]string

	// Mutex для безопасного доступа
	mutex sync.RWMutex

	// Каналы для регистрации и отключения клиентов
	register   chan *models.Client
	unregister chan *models.Client

	// Канал для широковещательных сообщений
	broadcast chan BroadcastMessage

	// SFU для WebRTC
	sfuServer *sfu.SFU

	// Репозиторий комнат
	roomRepo database.RoomRepository
}

// BroadcastMessage сообщение для отправки в комнату
type BroadcastMessage struct {
	RoomID  string
	Message models.WSMessage
	Exclude string // исключить клиента из рассылки
}

// NewHub создает новый Hub
func NewHub(sfuServer *sfu.SFU, roomRepo database.RoomRepository) *Hub {
	return &Hub{
		clients:    make(map[string]*models.Client),
		rooms:      make(map[string][]string),
		register:   make(chan *models.Client),
		unregister: make(chan *models.Client),
		broadcast:  make(chan BroadcastMessage),
		sfuServer:  sfuServer,
		roomRepo:   roomRepo,
	}
}

// Run запускает Hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.handleRegister(client)

		case client := <-h.unregister:
			h.handleUnregister(client)

		case message := <-h.broadcast:
			h.handleBroadcast(message)
		}
	}
}

// RegisterClient регистрирует нового клиента
func (h *Hub) RegisterClient(conn *websocket.Conn) *models.Client {
	client := &models.Client{
		ID:          uuid.New().String(),
		Conn:        conn,
		IsConnected: true,
		IsMuted:     false,
	}

	h.register <- client
	return client
}

// UnregisterClient отключает клиента
func (h *Hub) UnregisterClient(client *models.Client) {
	h.unregister <- client
}

// BroadcastToRoom отправляет сообщение всем клиентам в комнате
func (h *Hub) BroadcastToRoom(roomID string, message models.WSMessage, excludeClientID string) {
	h.broadcast <- BroadcastMessage{
		RoomID:  roomID,
		Message: message,
		Exclude: excludeClientID,
	}
}

// handleRegister обрабатывает регистрацию клиента
func (h *Hub) handleRegister(client *models.Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.clients[client.ID] = client
	log.Printf("Client registered: %s", client.ID)
}

// handleUnregister обрабатывает отключение клиента
func (h *Hub) handleUnregister(client *models.Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	// Удаляем клиента из списка
	delete(h.clients, client.ID)

	// Закрываем WebSocket соединение
	if client.Conn != nil {
		client.Conn.Close()
	}

	// Если клиент был в комнате, удаляем его оттуда
	if client.RoomID != "" {
		h.leaveRoomInternal(client)

		// Уведомляем других участников
		h.broadcast <- BroadcastMessage{
			RoomID: client.RoomID,
			Message: models.WSMessage{
				Action: "user_left",
				Data: models.UserEvent{
					UserID:   client.ID,
					UserName: client.UserName,
				},
			},
			Exclude: client.ID,
		}

		// Удаляем из SFU
		h.sfuServer.LeaveRoom(client.RoomID, client.ID)
	}

	log.Printf("Client unregistered: %s", client.ID)
}

// handleBroadcast обрабатывает широковещательную рассылку
func (h *Hub) handleBroadcast(broadcastMsg BroadcastMessage) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	clientIDs, exists := h.rooms[broadcastMsg.RoomID]
	if !exists {
		return
	}

	messageBytes, err := json.Marshal(broadcastMsg.Message)
	if err != nil {
		log.Printf("Error marshaling broadcast message: %v", err)
		return
	}

	for _, clientID := range clientIDs {
		if clientID == broadcastMsg.Exclude {
			continue
		}

		client, exists := h.clients[clientID]
		if !exists || client.Conn == nil {
			continue
		}

		if err := client.Conn.WriteMessage(websocket.TextMessage, messageBytes); err != nil {
			log.Printf("Error sending message to client %s: %v", clientID, err)
			// Отключаем клиента при ошибке отправки
			go func() {
				h.unregister <- client
			}()
		}
	}
}

// JoinRoom присоединяет клиента к комнате
func (h *Hub) JoinRoom(client *models.Client, roomID string, userName string) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	// Обновляем информацию о клиенте
	client.RoomID = roomID
	client.UserName = userName

	// Добавляем клиента в комнату
	h.rooms[roomID] = append(h.rooms[roomID], client.ID)

	// Создаем клиента в SFU
	_, err := h.sfuServer.JoinRoom(roomID, client.ID)
	if err != nil {
		return err
	}

	log.Printf("Client %s (%s) joined room %s", client.ID, userName, roomID)

	// Уведомляем других участников
	go func() {
		h.broadcast <- BroadcastMessage{
			RoomID: roomID,
			Message: models.WSMessage{
				Action: "user_joined",
				Data: models.UserEvent{
					UserID:   client.ID,
					UserName: userName,
					IsMuted:  client.IsMuted,
				},
			},
			Exclude: client.ID,
		}
	}()

	return nil
}

// leaveRoomInternal внутренняя функция для выхода из комнаты (без блокировки)
func (h *Hub) leaveRoomInternal(client *models.Client) {
	if client.RoomID == "" {
		return
	}

	// Удаляем клиента из списка комнаты
	clientIDs := h.rooms[client.RoomID]
	for i, id := range clientIDs {
		if id == client.ID {
			h.rooms[client.RoomID] = append(clientIDs[:i], clientIDs[i+1:]...)
			break
		}
	}

	// Удаляем комнату, если она пустая
	if len(h.rooms[client.RoomID]) == 0 {
		delete(h.rooms, client.RoomID)
	}

	client.RoomID = ""
}

// GetRoomClients возвращает список клиентов в комнате
func (h *Hub) GetRoomClients(roomID string) []*models.Client {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	clientIDs, exists := h.rooms[roomID]
	if !exists {
		return []*models.Client{}
	}

	clients := make([]*models.Client, 0, len(clientIDs))
	for _, clientID := range clientIDs {
		if client, exists := h.clients[clientID]; exists {
			// Создаем копию без WebSocket соединения для безопасности
			clientCopy := &models.Client{
				ID:          client.ID,
				UserName:    client.UserName,
				RoomID:      client.RoomID,
				IsMuted:     client.IsMuted,
				IsConnected: client.IsConnected,
			}
			clients = append(clients, clientCopy)
		}
	}

	return clients
}

// CreateRoom создает новую комнату
func (h *Hub) CreateRoom(ctx context.Context, roomName string) (*models.Room, error) {
	room := &models.Room{
		RoomID: uuid.New().String(),
		Name:   roomName,
	}

	// Сохраняем в базу данных
	if err := h.roomRepo.CreateRoom(ctx, room); err != nil {
		return nil, err
	}

	log.Printf("Room created: %s (%s)", room.RoomID, room.Name)
	return room, nil
}
