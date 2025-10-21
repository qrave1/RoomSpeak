package memory

import (
	"log/slog"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	
	"github.com/qrave1/RoomSpeak/internal/application/constant"
	"github.com/qrave1/RoomSpeak/internal/application/metric"
)

// WebsocketConnectionRepository интерфейс для работы с активными сессиями в памяти
type WebsocketConnectionRepository interface {
	Add(uuid.UUID, *websocket.Conn)
	Remove(uuid uuid.UUID)

	Write(uuid.UUID, any)
	GetAllConnected() []uuid.UUID
}

type safeWS struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

type wsConnectionRepository struct {
	// wsConns хранит map[user_id]*ws.conn
	wsConns map[uuid.UUID]*safeWS

	mu sync.RWMutex
}

func NewWSConnectionRepository() WebsocketConnectionRepository {
	return &wsConnectionRepository{
		wsConns: make(map[uuid.UUID]*safeWS, 10),
	}
}

func (w *wsConnectionRepository) Add(userID uuid.UUID, conn *websocket.Conn) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.wsConns[userID] = &safeWS{conn: conn}

	// Увеличиваем счетчик активных WS соединений
	metric.IncrementWSActiveConnections()
}

func (w *wsConnectionRepository) Remove(userID uuid.UUID) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Проверяем, существует ли соединение перед удалением
	if _, exists := w.wsConns[userID]; exists {
		delete(w.wsConns, userID)

		// Уменьшаем счетчик активных WS соединений
		metric.DecrementWSActiveConnections()
	}
}

func (w *wsConnectionRepository) Write(userID uuid.UUID, payload any) {
	safews, ok := w.getSafeWS(userID)
	if !ok {
		return
	}

	safews.mu.Lock()
	defer safews.mu.Unlock()

	err := safews.conn.WriteJSON(payload)
	if err != nil {
		slog.Error(
			"write to websocket",
			slog.Any(constant.Error, err),
			slog.Any(constant.UserID, userID),
		)
		return
	}
}

func (w *wsConnectionRepository) getSafeWS(userID uuid.UUID) (*safeWS, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	conn, ok := w.wsConns[userID]
	return conn, ok
}

func (w *wsConnectionRepository) GetAllConnected() []uuid.UUID {
	w.mu.RLock()
	defer w.mu.RUnlock()

	userIDs := make([]uuid.UUID, 0, len(w.wsConns))

	for userID := range w.wsConns {
		userIDs = append(userIDs, userID)
	}

	return userIDs
}
