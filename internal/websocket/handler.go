package websocket

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
	"github.com/qrave1/RoomSpeak/internal/models"
	"github.com/qrave1/RoomSpeak/internal/sfu"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// В продакшене здесь должна быть проверка домена
		return true
	},
}

// Handler обрабатывает WebSocket соединения
type Handler struct {
	hub *Hub
}

// NewHandler создает новый WebSocket обработчик
func NewHandler(hub *Hub) *Handler {
	return &Handler{hub: hub}
}

// HandleWebSocket обрабатывает WebSocket соединения
func (h *Handler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	// Регистрируем клиента
	client := h.hub.RegisterClient(conn)
	defer h.hub.UnregisterClient(client)

	log.Printf("New WebSocket connection: %s", client.ID)

	// Отправляем приветственное сообщение
	welcomeMsg := models.WSMessage{
		Action: "connected",
		Data: map[string]string{
			"clientId": client.ID,
		},
	}
	h.sendMessage(client, welcomeMsg)

	// Читаем сообщения от клиента
	for {
		var message models.WSMessage
		err := conn.ReadJSON(&message)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error for client %s: %v", client.ID, err)
			}
			break
		}

		h.handleMessage(client, message)
	}
}

// handleMessage обрабатывает входящие сообщения от клиента
func (h *Handler) handleMessage(client *models.Client, message models.WSMessage) {
	log.Printf("Message from client %s: %s", client.ID, message.Action)

	switch message.Action {
	case "create_room":
		h.handleCreateRoom(client, message)
	case "join_room":
		h.handleJoinRoom(client, message)
	case "offer":
		h.handleOffer(client, message)
	case "answer":
		h.handleAnswer(client, message)
	case "ice_candidate":
		h.handleICECandidate(client, message)
	case "toggle_mute":
		h.handleToggleMute(client, message)
	case "get_room_users":
		h.handleGetRoomUsers(client, message)
	default:
		log.Printf("Unknown action: %s", message.Action)
	}
}

// handleCreateRoom обрабатывает создание комнаты
func (h *Handler) handleCreateRoom(client *models.Client, message models.WSMessage) {
	var req models.CreateRoomRequest
	if err := h.parseMessageData(message.Data, &req); err != nil {
		h.sendError(client, "Invalid create room request")
		return
	}

	room, err := h.hub.CreateRoom(context.Background(), req.Name)
	if err != nil {
		log.Printf("Error creating room: %v", err)
		h.sendError(client, "Failed to create room")
		return
	}

	response := models.WSMessage{
		Action: "room_created",
		Data:   room,
	}
	h.sendMessage(client, response)
}

// handleJoinRoom обрабатывает присоединение к комнате
func (h *Handler) handleJoinRoom(client *models.Client, message models.WSMessage) {
	var req models.JoinRoomRequest
	if err := h.parseMessageData(message.Data, &req); err != nil {
		h.sendError(client, "Invalid join room request")
		return
	}

	// Проверяем, существует ли комната в базе данных
	_, err := h.hub.roomRepo.GetRoomByID(context.Background(), req.RoomID)
	if err != nil {
		h.sendError(client, "Room not found")
		return
	}

	// Присоединяем к комнате
	if err := h.hub.JoinRoom(client, req.RoomID, client.UserName); err != nil {
		log.Printf("Error joining room: %v", err)
		h.sendError(client, "Failed to join room")
		return
	}

	// Отправляем список участников
	clients := h.hub.GetRoomClients(req.RoomID)
	response := models.WSMessage{
		Action: "joined_room",
		Data: map[string]interface{}{
			"roomId": req.RoomID,
			"users":  clients,
		},
	}
	h.sendMessage(client, response)
}

// handleOffer обрабатывает WebRTC offer
func (h *Handler) handleOffer(client *models.Client, message models.WSMessage) {
	var webRTCMsg models.WebRTCMessage
	if err := h.parseMessageData(message.Data, &webRTCMsg); err != nil {
		h.sendError(client, "Invalid WebRTC offer")
		return
	}

	if client.RoomID == "" {
		h.sendError(client, "Not in a room")
		return
	}

	// Получаем SFU клиента
	room := h.hub.sfuServer.GetRoom(client.RoomID)
	if room == nil {
		h.sendError(client, "Room not found in SFU")
		return
	}

	// Находим SFU клиента
	var sfuClient *sfu.Client
	room.clientsMutex.RLock()
	if c, exists := room.Clients[client.ID]; exists {
		sfuClient = c
	}
	room.clientsMutex.RUnlock()

	if sfuClient == nil {
		h.sendError(client, "SFU client not found")
		return
	}

	// Создаем answer
	offer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  webRTCMsg.SDP,
	}

	answer, err := sfuClient.CreateAnswer(context.Background(), &offer)
	if err != nil {
		log.Printf("Error creating answer: %v", err)
		h.sendError(client, "Failed to create answer")
		return
	}

	// Отправляем answer клиенту
	response := models.WSMessage{
		Action: "answer",
		Data: models.WebRTCMessage{
			Type: "answer",
			SDP:  answer.SDP,
		},
	}
	h.sendMessage(client, response)
}

// handleAnswer обрабатывает WebRTC answer (не используется в SFU архитектуре)
func (h *Handler) handleAnswer(client *models.Client, message models.WSMessage) {
	// В SFU архитектуре клиенты не отправляют answer серверу
	log.Printf("Received unexpected answer from client %s", client.ID)
}

// handleICECandidate обрабатывает ICE кандидатов
func (h *Handler) handleICECandidate(client *models.Client, message models.WSMessage) {
	var webRTCMsg models.WebRTCMessage
	if err := h.parseMessageData(message.Data, &webRTCMsg); err != nil {
		h.sendError(client, "Invalid ICE candidate")
		return
	}

	if client.RoomID == "" {
		h.sendError(client, "Not in a room")
		return
	}

	// Получаем SFU клиента
	room := h.hub.sfuServer.GetRoom(client.RoomID)
	if room == nil {
		return
	}

	var sfuClient *sfu.Client
	room.clientsMutex.RLock()
	if c, exists := room.Clients[client.ID]; exists {
		sfuClient = c
	}
	room.clientsMutex.RUnlock()

	if sfuClient == nil {
		return
	}

	// Добавляем ICE кандидата
	candidate := webrtc.ICECandidateInit{
		Candidate:     webRTCMsg.Candidate,
		SDPMid:        &webRTCMsg.SDPMid,
		SDPMLineIndex: &webRTCMsg.SDPMLineIndex,
	}

	if err := sfuClient.AddICECandidate(&candidate); err != nil {
		log.Printf("Error adding ICE candidate: %v", err)
	}
}

// handleToggleMute обрабатывает изменение состояния микрофона
func (h *Handler) handleToggleMute(client *models.Client, message models.WSMessage) {
	var req models.ToggleMuteRequest
	if err := h.parseMessageData(message.Data, &req); err != nil {
		h.sendError(client, "Invalid mute request")
		return
	}

	client.IsMuted = req.IsMuted

	// Уведомляем других участников комнаты
	if client.RoomID != "" {
		h.hub.BroadcastToRoom(client.RoomID, models.WSMessage{
			Action: "mute_changed",
			Data: models.UserEvent{
				UserID:   client.ID,
				UserName: client.UserName,
				IsMuted:  client.IsMuted,
			},
		}, client.ID)
	}

	// Подтверждаем изменение
	response := models.WSMessage{
		Action: "mute_toggled",
		Data: map[string]bool{
			"isMuted": client.IsMuted,
		},
	}
	h.sendMessage(client, response)
}

// handleGetRoomUsers обрабатывает запрос списка пользователей комнаты
func (h *Handler) handleGetRoomUsers(client *models.Client, message models.WSMessage) {
	if client.RoomID == "" {
		h.sendError(client, "Not in a room")
		return
	}

	clients := h.hub.GetRoomClients(client.RoomID)
	response := models.WSMessage{
		Action: "room_users",
		Data: map[string]interface{}{
			"users": clients,
		},
	}
	h.sendMessage(client, response)
}

// parseMessageData парсит данные сообщения
func (h *Handler) parseMessageData(data interface{}, target interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonData, target)
}

// sendMessage отправляет сообщение клиенту
func (h *Handler) sendMessage(client *models.Client, message models.WSMessage) {
	if client.Conn == nil {
		return
	}

	if err := client.Conn.WriteJSON(message); err != nil {
		log.Printf("Error sending message to client %s: %v", client.ID, err)
	}
}

// sendError отправляет сообщение об ошибке клиенту
func (h *Handler) sendError(client *models.Client, errorMsg string) {
	message := models.WSMessage{
		Action: "error",
		Data: map[string]string{
			"message": errorMsg,
		},
	}
	h.sendMessage(client, message)
}
