package memory

import (
	"log/slog"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/qrave1/RoomSpeak/internal/application/constant"
)

// WebsocketConnectionRepository интерфейс для работы с активными сессиями в памяти
type WebsocketConnectionRepository interface {
	Add(uuid.UUID, *websocket.Conn)
	Remove(uuid uuid.UUID)

	Write(uuid.UUID, any)
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
}

func (w *wsConnectionRepository) Remove(userID uuid.UUID) {
	w.mu.Lock()
	defer w.mu.Unlock()

	delete(w.wsConns, userID)
}

func (w *wsConnectionRepository) Write(userID uuid.UUID, payload any) {
	safews, ok := w.getSafeWS(userID)
	if !ok {
		slog.Error("get websocket", slog.Any(constant.UserID, userID))
		return
	}

	safews.mu.Lock()
	defer safews.mu.Unlock()

	err := safews.conn.WriteJSON(payload)
	if err != nil {
		slog.Error("write to websocket", slog.Any(constant.UserID, userID))
		return
	}
}

func (w *wsConnectionRepository) getSafeWS(userID uuid.UUID) (*safeWS, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	conn, ok := w.wsConns[userID]
	return conn, ok
}
