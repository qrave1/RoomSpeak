package application

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/pion/webrtc/v4"
	"github.com/qrave1/RoomSpeak/internal/infrastructure/repository"

	"github.com/qrave1/RoomSpeak/internal/domain"
	webrtcinfra "github.com/qrave1/RoomSpeak/internal/infrastructure/webrtc"
)

// App holds dependencies.
type App struct {
	CommandBus CommandBus
	QueryBus   QueryBus
	Upgrader   *websocket.Upgrader
	PCFactory  webrtcinfra.PeerConnectionFactory
}

func NewApp() *App {
	roomRepo := repository.NewInMemoryRoomRepository()
	clientRepo := repository.InMemoryClientRepository{}
	cmdHandler := &RoomCommandHandler{RoomRepo: roomRepo, ClientRepo: clientRepo}
	queryHandler := &RoomQueryHandler{RoomRepo: roomRepo, ClientRepo: clientRepo}
	return &App{
		CommandBus: &SimpleCommandBus{Handler: cmdHandler},
		QueryBus:   &SimpleQueryBus{Handler: queryHandler},
		Upgrader:   &websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
		PCFactory:  webrtcinfra.DefaultPeerConnectionFactory{},
	}
}

func (app *App) HandleWebSocket(c echo.Context) error {
	ws, err := app.Upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		slog.Error("WebSocket upgrade error", "error", err)
		return c.String(http.StatusBadRequest, "upgrade error")
	}
	defer ws.Close()

	pc, audioTrack, err := app.PCFactory.Create()
	if err != nil {
		slog.Error("PeerConnection error", "error", err)
		return c.String(http.StatusBadRequest, "peer connection error")
	}

	clientID := uuid.NewString()
	slog.Info("WebSocket connection established", "client_id", clientID)

	var roomID string

	defer func() {
		if roomID != "" {
			app.CommandBus.Dispatch(c.Request().Context(), &domain.LeaveRoomCommand{ClientID: clientID, RoomID: roomID})
		}
		pc.Close()
	}()

	pc.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		if track.Kind() == webrtc.RTPCodecTypeAudio {
			go func() {
				for {
					pkt, _, err := track.ReadRTP()
					if err != nil {
						if errors.Is(err, io.EOF) {
							slog.Info("RTP stream ended", "client_id", clientID)
						} else {
							slog.Error("RTP read error", "error", err)
						}
						return
					}
					app.CommandBus.Dispatch(
						context.Background(),
						&domain.BroadcastRTPCommand{Packet: pkt, SenderID: clientID, RoomID: roomID},
					)
				}
			}()
		}
	})

	pc.OnICECandidate(func(cand *webrtc.ICECandidate) {
		if cand == nil {
			return
		}
		ws.WriteJSON(map[string]interface{}{"type": "candidate", "candidate": cand.ToJSON()})
	})

	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		switch state {
		case webrtc.PeerConnectionStateFailed:
			slog.Warn("PeerConnection state failed", "client_id", clientID)
		case webrtc.PeerConnectionStateDisconnected:
			slog.Warn("PeerConnection state disconnected", "client_id", clientID)
		default:
		}
	})

	for {
		_, msg, err := ws.ReadMessage()
		if err != nil {
			slog.Error("WebSocket read error", "error", err)
			return c.String(http.StatusBadRequest, "read ws message error")
		}

		if err := app.handleClientMessage(c.Request().Context(), clientID, ws, pc, audioTrack, msg, &roomID); err != nil {
			slog.Error("Message handling error", "error", err)
		}
	}
}

func (app *App) handleClientMessage(
	ctx context.Context,
	clientID string,
	ws *websocket.Conn,
	pc *webrtc.PeerConnection,
	track *webrtc.TrackLocalStaticRTP,
	msg []byte,
	roomID *string,
) error {
	var base struct{ Type string }
	if err := json.Unmarshal(msg, &base); err != nil {
		return err
	}

	switch base.Type {
	case "join":
		var data struct{ Name, RoomID string }
		if err := json.Unmarshal(msg, &data); err != nil {
			return err
		}
		*roomID = data.RoomID
		return app.CommandBus.Dispatch(
			ctx,
			&domain.JoinRoomCommand{
				ClientID: clientID,
				Name:     data.Name,
				RoomID:   data.RoomID,
				WSConn:   ws,
				PC:       pc,
				Track:    track,
			},
		)

	case "offer":
		var data struct{ SDP string }
		if err := json.Unmarshal(msg, &data); err != nil {
			return err
		}
		if err := pc.SetRemoteDescription(webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: data.SDP}); err != nil {
			return err
		}
		answer, err := pc.CreateAnswer(nil)
		if err != nil {
			return err
		}
		if err = pc.SetLocalDescription(answer); err != nil {
			return err
		}
		return ws.WriteJSON(map[string]interface{}{"type": "answer", "sdp": answer.SDP})

	case "candidate":
		var data struct{ Candidate webrtc.ICECandidateInit }
		if err := json.Unmarshal(msg, &data); err != nil {
			return err
		}
		return pc.AddICECandidate(data.Candidate)

	default:
		return errors.New("unknown message type")
	}
}
