package handlers

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/pion/webrtc/v4"
	"github.com/qrave1/RoomSpeak/internal/application/config"
)

type IceHandler struct {
	cfg *config.Config
}

func NewIceHandler(cfg *config.Config) *IceHandler {
	return &IceHandler{cfg: cfg}
}

// Handler для выдачи ICE серверов
func (h *IceHandler) IceServers(c echo.Context) error {
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
