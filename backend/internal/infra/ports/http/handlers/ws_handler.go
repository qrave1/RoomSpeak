package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	
	"github.com/qrave1/RoomSpeak/internal/application/config"
	"github.com/qrave1/RoomSpeak/internal/application/constant"
	"github.com/qrave1/RoomSpeak/internal/domain/events"
	"github.com/qrave1/RoomSpeak/internal/infra/adapters/memory"
	"github.com/qrave1/RoomSpeak/internal/infra/appctx"
	"github.com/qrave1/RoomSpeak/internal/usecase"
)

type WebSocketHandler struct {
	upgrader *websocket.Upgrader

	signalingUsecase usecase.SignalingUsecase

	wsConnRepo memory.WebsocketConnectionRepository
}

func NewWebSocketHandler(cfg *config.Config, signalingUsecase usecase.SignalingUsecase, wsConnRepo memory.WebsocketConnectionRepository) *WebSocketHandler {
	return &WebSocketHandler{
		upgrader: &websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				if cfg.Debug {
					return true
				}

				return r.Header.Get("Origin") == cfg.Domain
			},
		},
		signalingUsecase: signalingUsecase,
		wsConnRepo:       wsConnRepo,
	}
}

func (h *WebSocketHandler) Handle(c echo.Context) error {
	ws, err := h.upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		slog.Error(
			"WebSocket upgrade error",
			slog.Any(constant.Error, err),
		)
		return err
	}
	defer ws.Close()

	userID, ok := appctx.UserID(c.Request().Context())
	if !ok {
		return fmt.Errorf("get user id from context")
	}

	h.wsConnRepo.Add(userID, ws)
	defer h.wsConnRepo.Remove(userID)

	err = ws.SetReadDeadline(time.Now().Add(60 * time.Second))
	if err != nil {
		return err
	}
	ws.SetPongHandler(func(string) error {
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
					slog.Error("ping failed", slog.Any(constant.Error, err))
					return
				}
			case <-c.Request().Context().Done():
				return
			}
		}
	}()

	for {
		select {
		case <-c.Request().Context().Done():
			return nil
		default:
			_, msg, err := ws.ReadMessage()
			if err != nil {
				h.handleWebsocketError(c.Request().Context(), err)

				// TODO: пробовать достать channel_id из ctx

				if err = h.signalingUsecase.HandleLeave(c.Request().Context(), userID); err != nil {
					slog.Error(
						"handle leave while reading websocket message",
						slog.Any(constant.Error, err),
						slog.Any(constant.UserID, userID),
					)
				}

				return nil
			}

			signalMessage := new(events.Message)

			if err = json.Unmarshal(msg, &signalMessage); err != nil {
				slog.Error("unmarshal websocket message", slog.Any(constant.Error, err))

				return nil
			}

			if err = h.handleMessage(c.Request().Context(), signalMessage); err != nil {
				slog.Error("handle message", slog.Any(constant.Error, err))
			}
		}
	}
}

func (h *WebSocketHandler) handleMessage(
	ctx context.Context,
	msg *events.Message,
) error {
	userID, ok := appctx.UserID(ctx)
	if !ok {
		return fmt.Errorf("get user id from context")
	}

	switch msg.Type {
	case "join":
		var joinEvent events.JoinEvent

		if err := json.Unmarshal(msg.Data, &joinEvent); err != nil {
			return fmt.Errorf("unmarshal join event: %w", err)
		}

		if err := h.signalingUsecase.HandleJoin(ctx, userID, joinEvent); err != nil {
			return fmt.Errorf("handle join: %w", err)
		}

	case "offer":
		var offer events.SdpEvent

		if err := json.Unmarshal(msg.Data, &offer); err != nil {
			return fmt.Errorf("unmarshal offer: %w", err)
		}

		if err := h.signalingUsecase.HandleOffer(ctx, userID, offer.SDP); err != nil {
			return fmt.Errorf("handle offer: %w", err)
		}

	case "answer":
		var answer events.SdpEvent

		if err := json.Unmarshal(msg.Data, &answer); err != nil {
			return fmt.Errorf("unmarshal answer: %w", err)
		}

		if err := h.signalingUsecase.HandleAnswer(ctx, userID, answer.SDP); err != nil {
			return fmt.Errorf("handle answer: %w", err)
		}

	case "candidate":
		var candidate events.IceCandidateEvent

		if err := json.Unmarshal(msg.Data, &candidate); err != nil {
			return fmt.Errorf("unmarshal ice candidate: %w", err)
		}

		if err := h.signalingUsecase.HandleCandidate(ctx, userID, candidate.Candidate); err != nil {
			return fmt.Errorf("handle ice candidate: %w", err)
		}

	case "leave":
		if err := h.signalingUsecase.HandleLeave(ctx, userID); err != nil {
			return fmt.Errorf("handle leave: %w", err)
		}
	case "mute":
		var muteEvent struct {
			IsMuted bool `json:"is_muted"`
		}

		if err := json.Unmarshal(msg.Data, &muteEvent); err != nil {
			return fmt.Errorf("unmarshal mute event: %w", err)
		}

		if err := h.signalingUsecase.HandleMute(ctx, userID, muteEvent.IsMuted); err != nil {
			return fmt.Errorf("handle mute: %w", err)
		}

	case "ping":
		h.signalingUsecase.HandlePing(ctx, userID)

	default:
		return errors.New("unknown message type")
	}

	return nil
}

func (h *WebSocketHandler) handleWebsocketError(ctx context.Context, err error) {
	userID, ok := appctx.UserID(ctx)
	if !ok {
		userID = uuid.Nil
	}

	var closeErr *websocket.CloseError
	if errors.As(err, &closeErr) {
		switch closeErr.Code {
		case websocket.CloseNormalClosure, websocket.CloseGoingAway:
			slog.Info("user disconnected from websocket", slog.Any(constant.UserID, userID))
		default:
			slog.Error("websocket close error")
		}
	} else {
		slog.Error(
			"websocket read",
			slog.Any(constant.Error, err),
		)
	}
}
