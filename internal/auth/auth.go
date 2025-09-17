package auth

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/qrave1/RoomSpeak/internal/constant"
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
	var req RegisterRequest
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
	var req LoginRequest
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
		Expires:  time.Now().Add(time.Hour * 72),
		Path:     "/",
		Secure:   true,
		HttpOnly: true,
	})

	return c.NoContent(http.StatusOK)
}

func (h *AuthHandler) GetMe(c echo.Context) error {
	userID, ok := c.Request().Context().Value(constant.UserID).(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid token in context"})
	}

	user, err := h.userUsecase.GetUserByID(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
	}

	resp := GetMeResponse{
		ID:       user.ID,
		Username: user.Username,
	}

	return c.JSON(http.StatusOK, resp)
}

type GetMeResponse struct {
	ID       uuid.UUID `json:"id"`
	Username string    `json:"username"`
}
