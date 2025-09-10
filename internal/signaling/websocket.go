package signaling

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/qrave1/RoomSpeak/internal/constant"
)

func ConfigureWebsocket(ctx context.Context, ws *websocket.Conn) error {
	if err := ws.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
		return fmt.Errorf("websocket setting read deadline failed: %w", err)
	}

	ws.SetPongHandler(
		func(string) error {
			if err := ws.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
				return fmt.Errorf("websocket update read deadline failed: %w", err)
			}

			return nil
		},
	)

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
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

// IsConnectionClosed проверяет, закрыто ли соединение
func IsConnectionClosed(err error) bool {
	return websocket.IsCloseError(err) ||
		websocket.IsUnexpectedCloseError(err) ||
		strings.Contains(err.Error(), "use of closed network connection")
}
