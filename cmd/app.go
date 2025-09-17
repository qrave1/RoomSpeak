package cmd

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/pion/rtp"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/pion/webrtc/v4"

	"github.com/qrave1/RoomSpeak/internal/auth"
	"github.com/qrave1/RoomSpeak/internal/config"
	"github.com/qrave1/RoomSpeak/internal/constant"
	"github.com/qrave1/RoomSpeak/internal/domain/events"
	"github.com/qrave1/RoomSpeak/internal/infra/http/middleware"
	"github.com/qrave1/RoomSpeak/internal/infra/postgres"
	"github.com/qrave1/RoomSpeak/internal/infra/postgres/repository"
	"github.com/qrave1/RoomSpeak/internal/usecase"
)

func runApp() {
	ctx := context.Background()

	slog.SetDefault(
		slog.New(
			slog.NewJSONHandler(
				os.Stdout,
				&slog.HandlerOptions{Level: slog.LevelInfo},
			),
		),
	)

	cfg, err := config.New()
	if err != nil {
		slog.Error("parse config", slog.Any(constant.Error, err))
		os.Exit(1)
	}

	slog.Info("Running app", slog.Bool("debug", cfg.Debug))

	dbConn, err := postgres.NewPostgres(ctx, cfg.Postgres.DSN())
	if err != nil {
		slog.Error("connect to postgres", slog.Any(constant.Error, err))
		os.Exit(1)
	}
	defer dbConn.Close()

	userRepo := repository.NewUserRepo(dbConn)
	channelRepo := repository.NewChannelRepo(dbConn)

	userUsecase := usecase.NewUserUsecase([]byte(cfg.JWTSecret), userRepo)
	channelUsecase := usecase.NewChannelUsecase(channelRepo)
	authHandler := auth.NewAuthHandler(userUsecase)

	httpHandler := NewHttpHandler(
		cfg,
		channelUsecase,
		&websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				if cfg.Debug {
					return true
				}

				return r.Header.Get("Origin") == cfg.Domain
			},
		},
		dbConn,
	)

	e := echo.New()

	e.Use(middleware.SlogLogger())

	api := e.Group("/api")
	{
		authGroup := api.Group("/auth")
		{
			authGroup.POST("/register", authHandler.Register)
			authGroup.POST("/login", authHandler.Login)
		}

		v1 := api.Group("/v1")
		v1.Use(middleware.JWTAuthMiddleware(cfg.JWTSecret))
		{
			v1.GET("/me", authHandler.GetMe)
			v1.GET("/channels", httpHandler.listChannelsHandler)
			v1.POST("/channels", httpHandler.createChannelHandler)
			v1.DELETE("/channels/:id", httpHandler.deleteChannelHandler)
		}
	}

	e.Static("/", "web")
	e.GET("/ws", httpHandler.handleWebSocket)

	e.GET("/ice", httpHandler.iceServersHandler)

	err = e.Start(":" + cfg.Port)
	if err != nil {
		slog.Error(
			"HTTP server failed",
			slog.Any(constant.Error, err),
		)

		os.Exit(1)
	}
}

type Channel struct {
	id       string
	sessions map[string]*Session
	mu       sync.RWMutex
}

func NewChannel(id string) *Channel {
	return &Channel{
		id:       id,
		sessions: make(map[string]*Session),
	}
}

func (c *Channel) AddSession(s *Session) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.sessions[s.id] = s

	c.broadcastParticipants()
}

func (c *Channel) RemoveSession(sessionID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.sessions, sessionID)

	c.broadcastParticipants()
}

func (c *Channel) broadcastParticipants() {
	parts := make([]string, 0, len(c.sessions))

	for _, session := range c.sessions {
		parts = append(parts, session.name)
	}

	for _, session := range c.sessions {
		session.WriteWS(map[string]interface{}{"type": "participants", "list": parts})
	}
}

func (c *Channel) BroadcastRTP(pkt *rtp.Packet, senderID string) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, session := range c.sessions {
		if session.id == senderID || session.peer == nil || session.peer.audioTrack == nil {
			continue
		}

		err := session.peer.audioTrack.WriteRTP(pkt)

		if err != nil {
			slog.Error(
				"write RTP",
				slog.Any(constant.Error, err),
				slog.Any(constant.SessionID, senderID),
			)
		}
	}
}

type Session struct {
	id   string
	name string

	channel *Channel

	wsMu   sync.Mutex
	wsConn *websocket.Conn

	peer     *Peer
	joinedAt time.Time
}

func NewSession(ws *websocket.Conn) *Session {
	return &Session{
		id:       uuid.NewString(),
		wsConn:   ws,
		joinedAt: time.Now(),
	}
}

func (s *Session) WriteWS(v any) error {
	s.wsMu.Lock()
	defer s.wsMu.Unlock()

	return s.wsConn.WriteJSON(v)
}

type Peer struct {
	peerConn   *webrtc.PeerConnection
	audioTrack *webrtc.TrackLocalStaticRTP
}

// TODO move to peerConnectionFactory
func createPeerConnection(cfg *config.Config) (*Peer, error) {
	pc, err := webrtc.NewPeerConnection(
		webrtc.Configuration{
			ICEServers: []webrtc.ICEServer{
				{
					URLs: []string{"stun:stun.l.google.com:19302"},
				},
				cfg.TurnUDPServer,
				cfg.TurnTCPServer,
			},
		},
	)
	if err != nil {
		return nil, err
	}

	audioTrack, err := webrtc.NewTrackLocalStaticRTP(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus}, "audio", "RoomSpeak",
	)
	if err != nil {
		return nil, fmt.Errorf("create audio track: %w", err)
	}

	if _, err = pc.AddTrack(audioTrack); err != nil {
		return nil, fmt.Errorf("add audio track: %w", err)
	}

	return &Peer{peerConn: pc, audioTrack: audioTrack}, nil
}

func (h *HttpHandler) handleWebSocket(c echo.Context) error {
	ws, err := h.upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		slog.Error(
			"WebSocket upgrade error",
			slog.Any(constant.Error, err),
		)
		return err
	}
	defer ws.Close()

	err = ws.SetReadDeadline(time.Now().Add(60 * time.Second))
	if err != nil {
		return err
	}
	ws.SetPongHandler(func(string) error {
		ws.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	session := NewSession(ws)

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

	defer func() {
		if session.channel != nil {
			session.channel.RemoveSession(session.id)
		}
		if session.peer != nil && session.peer.peerConn != nil {
			session.peer.peerConn.Close()
		}
	}()

	for {
		select {
		case <-c.Request().Context().Done():
			return nil
		default:
			_, msg, err := session.wsConn.ReadMessage()
			if err != nil {
				slog.Error(
					"webSocket read error",
					slog.Any(constant.Error, err),
				)

				return nil
			}

			signalMessage := new(events.Message)

			if err = json.Unmarshal(msg, &signalMessage); err != nil {
				slog.Error("unmarshal websocket message", slog.Any(constant.Error, err))

				return nil
			}

			if err = h.handleMessage(c.Request().Context(), session, signalMessage); err != nil {
				slog.Error("handle message", slog.Any(constant.Error, err))
			}
		}
	}
}

func (h *HttpHandler) handleMessage(
	ctx context.Context,
	session *Session,
	msg *events.Message,
) error {
	switch msg.Type {
	case "join":
		var joinEvent events.JoinEvent

		if err := json.Unmarshal(msg.Data, &joinEvent); err != nil {
			return err
		}

		if joinEvent.ChannelID == "" {
			session.WriteWS(map[string]interface{}{"type": constant.Error, "message": "channel_id is required"})
			return nil
		}

		// TODO: user from db here
		session.name = "todo"

		// Проверяем, что канал существует в базе данных
		channelID, err := uuid.Parse(joinEvent.ChannelID)
		if err != nil {
			session.WriteWS(map[string]interface{}{"type": constant.Error, "message": "invalid channel_id format"})
			return nil
		}

		_, err = h.channelUsecase.GetChannel(ctx, channelID)
		if err != nil {
			slog.Error("get channel", slog.Any(constant.Error, err))
			session.WriteWS(map[string]interface{}{"type": constant.Error, "message": "channel not found"})
			return nil
		}

		// Получаем userID из контекста (если есть)
		userID, ok := ctx.Value(constant.UserID).(uuid.UUID)
		if ok {
			// Добавляем пользователя в канал, если его там нет
			if err := h.channelUsecase.AddUserToChannel(ctx, userID, channelID); err != nil {
				slog.Error("add user to channel", slog.Any(constant.Error, err))
				// Не блокируем подключение, если не удалось добавить в БД
			}
		}

		// TODO вернуть
		//session.channel = channel

		session.peer, err = createPeerConnection(h.cfg)
		if err != nil {
			slog.Error("create peer connection", slog.Any(constant.Error, err))

			return nil
		}

		session.peer.peerConn.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
			go func() {
				for {
					pkt, _, err := track.ReadRTP()
					if err != nil {
						if !errors.Is(err, io.EOF) {
							slog.Error("RTP read error", "error", err)
						}

						return
					}

					if track.Kind() == webrtc.RTPCodecTypeAudio {
						session.channel.BroadcastRTP(pkt, session.id)
					}
				}
			}()
		})

		session.peer.peerConn.OnICECandidate(func(c *webrtc.ICECandidate) {
			if c == nil {
				return
			}
			session.WriteWS(map[string]interface{}{"type": "candidate", "candidate": c.ToJSON()})
		})

		session.peer.peerConn.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
			if state == webrtc.PeerConnectionStateFailed || state == webrtc.PeerConnectionStateDisconnected {
				slog.Error("PeerConnection bad status",
					slog.String(constant.State, state.String()),
					slog.String(constant.SessionID, session.id),
				)

				err := session.WriteWS(map[string]interface{}{
					"type":    constant.Error,
					"message": fmt.Sprintf("peer connection bad state: %s", state.String()),
				})
				if err != nil {
					slog.Error("send peer connection state", slog.Any(constant.Error, err))
					return
				}
				return
			}
		})

	case "offer":
		var offer events.SdpEvent

		if err := json.Unmarshal(msg.Data, &offer); err != nil {
			return err
		}

		if err := session.peer.peerConn.SetRemoteDescription(
			webrtc.SessionDescription{
				Type: webrtc.SDPTypeOffer,
				SDP:  offer.SDP,
			},
		); err != nil {
			return err
		}

		answer, err := session.peer.peerConn.CreateAnswer(nil)
		if err != nil {
			return err
		}

		if err = session.peer.peerConn.SetLocalDescription(answer); err != nil {
			slog.Error("set local description", slog.Any(constant.Error, err))

			return err
		}

		return session.WriteWS(map[string]interface{}{"type": "answer", "sdp": answer.SDP})

	case "answer":
		var answer events.SdpEvent

		if err := json.Unmarshal(msg.Data, &answer); err != nil {
			return err
		}

		if err := session.peer.peerConn.SetRemoteDescription(
			webrtc.SessionDescription{
				Type: webrtc.SDPTypeAnswer,
				SDP:  answer.SDP,
			},
		); err != nil {
			slog.Error("set remote description", slog.Any(constant.Error, err))

			return err
		}

	case "candidate":
		var candidate events.IceCandidateEvent

		if err := json.Unmarshal(msg.Data, &candidate); err != nil {
			return err
		}

		return session.peer.peerConn.AddICECandidate(candidate.Candidate)

	case "leave":
		if session.channel != nil {
			session.channel.RemoveSession(session.id)
		}
		if session.peer != nil && session.peer.peerConn != nil {
			session.peer.peerConn.Close()
		}
	case "ping":
		return session.WriteWS(map[string]interface{}{"type": "pong"})
	default:
		return errors.New("unknown message type")
	}

	return nil
}

func (h *HttpHandler) listChannelsHandler(c echo.Context) error {
	// Получаем userID из JWT токена
	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid user"})
	}

	channels, err := h.channelUsecase.GetChannelsByUserID(c.Request().Context(), userID)
	if err != nil {
		slog.Error("get channels by user id", slog.Any(constant.Error, err))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get channels"})
	}

	return c.JSON(http.StatusOK, channels)
}

type CreateChannelRequest struct {
	Name string `json:"name"`
}

func (h *HttpHandler) createChannelHandler(c echo.Context) error {
	var req CreateChannelRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	if req.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "name is required"})
	}

	// Получаем userID из JWT токена
	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid user"})
	}

	channel, err := h.channelUsecase.CreateChannel(c.Request().Context(), userID, req.Name)
	if err != nil {
		slog.Error("create channel", slog.Any(constant.Error, err))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create channel"})
	}

	// Добавляем создателя в канал
	if err := h.channelUsecase.AddUserToChannel(c.Request().Context(), userID, channel.ID); err != nil {
		slog.Error("add user to channel", slog.Any(constant.Error, err))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to add user to channel"})
	}

	return c.JSON(http.StatusCreated, channel)
}

func (h *HttpHandler) deleteChannelHandler(c echo.Context) error {
	channelIDStr := c.Param("id")
	channelID, err := uuid.Parse(channelIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid channel id"})
	}

	// Получаем userID из JWT токена
	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid user"})
	}

	// Проверяем, что пользователь является создателем канала
	channel, err := h.channelUsecase.GetChannel(c.Request().Context(), channelID)
	if err != nil {
		slog.Error("get channel", slog.Any(constant.Error, err))
		return c.JSON(http.StatusNotFound, map[string]string{"error": "channel not found"})
	}

	if channel.CreatorID != userID {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "only channel creator can delete the channel"})
	}

	// Удаляем канал из базы данных
	if err := h.channelUsecase.DeleteChannel(c.Request().Context(), channelID); err != nil {
		slog.Error("delete channel", slog.Any(constant.Error, err))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to delete channel"})
	}

	return c.NoContent(http.StatusOK)
}

// Handler для выдачи ICE серверов
func (h *HttpHandler) iceServersHandler(c echo.Context) error {
	expiration := time.Now().Add(time.Hour).Unix()
	username := fmt.Sprintf("%d", expiration)

	// Создаём HMAC-SHA1 с использованием static-auth-secret
	mac := hmac.New(sha1.New, []byte(h.cfg.CoturnServer.Secret))
	mac.Write([]byte(username))
	password := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	response := webrtc.ICEServer{
		URLs: []string{
			h.cfg.TurnUDPServer.URLs[0],
			h.cfg.TurnTCPServer.URLs[0],
		},
		Username:   username,
		Credential: password,
	}

	return c.JSON(http.StatusOK, response)
}

type HttpHandler struct {
	cfg            *config.Config
	upgrader       *websocket.Upgrader
	channelUsecase usecase.ChannelUsecase
	db             *sqlx.DB
}

func NewHttpHandler(
	cfg *config.Config,
	channelUsecase usecase.ChannelUsecase,
	upgrader *websocket.Upgrader,
	db *sqlx.DB,
) *HttpHandler {
	return &HttpHandler{
		cfg:            cfg,
		channelUsecase: channelUsecase,
		upgrader:       upgrader,
		db:             db,
	}
}
