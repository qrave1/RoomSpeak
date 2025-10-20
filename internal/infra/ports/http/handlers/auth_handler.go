package handlers

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/qrave1/RoomSpeak/internal/application/constant"
	"github.com/qrave1/RoomSpeak/internal/infra/appctx"
	"github.com/qrave1/RoomSpeak/internal/infra/ports/http/dto"
	"github.com/qrave1/RoomSpeak/internal/usecase"
)

type AuthHandler struct {
	userUsecase usecase.UserUsecase
}

func NewAuthHandler(userUsecase usecase.UserUsecase) *AuthHandler {
	return &AuthHandler{
		userUsecase: userUsecase,
	}
}

func (h *AuthHandler) Register(c echo.Context) error {
	var req dto.RegisterRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	user, err := h.userUsecase.CreateUser(c.Request().Context(), req.Username, req.Password)
	if err != nil {
		slog.Error("create user failed", slog.Any(constant.Error, err))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "could not create user"})
	}

	return c.JSON(http.StatusCreated, user)
}

func (h *AuthHandler) Login(c echo.Context) error {
	var req dto.LoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	user, err := h.userUsecase.ValidateCredentials(c.Request().Context(), req.Username, req.Password)
	if err != nil {
		slog.Error("validate credentials failed", slog.String(constant.UserName, req.Username), slog.Any(constant.Error, err))
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
	}

	token, err := h.userUsecase.GenerateJWT(user)
	if err != nil {
		slog.Error("generate JWT failed", slog.Any(constant.Error, err))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "could not create token"})
	}

	c.SetCookie(&http.Cookie{
		Name:     "jwt",
		Value:    token,
		Expires:  time.Now().Add(72 * time.Hour),
		Domain:   ".xxsm.ru",
		Path:     "/",
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteNoneMode,
	})

	return c.NoContent(http.StatusOK)
}

func (h *AuthHandler) GetMe(c echo.Context) error {
	userID, ok := appctx.UserID(c.Request().Context())
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid user ID in context"})
	}

	user, err := h.userUsecase.GetUserByID(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
	}

	resp := dto.GetMeResponse{
		ID:       user.ID,
		Username: user.Username,
	}

	return c.JSON(http.StatusOK, resp)
}

func (h *AuthHandler) GetOnlineUsers(c echo.Context) error {
	onlineUsers, err := h.userUsecase.GetOnlineUsers(c.Request().Context())
	if err != nil {
		slog.Error("get online users failed", slog.Any(constant.Error, err))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "could not get online users"})
	}

	return c.JSON(http.StatusOK, onlineUsers)
}
