package auth

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/qrave1/RoomSpeak/internal/constant"
	"github.com/qrave1/RoomSpeak/internal/infra/postgres/repository"
	"github.com/qrave1/RoomSpeak/internal/models"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	userRepo  repository.UserRepository
	jwtSecret []byte
}

func NewAuthHandler(userRepo repository.UserRepository, jwtSecret string) *AuthHandler {
	return &AuthHandler{
		userRepo:  userRepo,
		jwtSecret: []byte(jwtSecret),
	}
}

func (h *AuthHandler) Register(c echo.Context) error {
	var req RegisterRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("generate password hash failed", slog.Any(constant.Error, err))

		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "could not hash password"})
	}

	user := models.NewUser()

	user.Username = req.Username
	user.Password = string(hashedPassword)

	err = h.userRepo.CreateUser(user)
	if err != nil {
		slog.Error("user repo create failed", slog.Any(constant.Error, err))

		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "could not create user"})
	}

	user.Password = ""

	return c.JSON(http.StatusCreated, user)
}

func (h *AuthHandler) Login(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	user, err := h.userRepo.GetUserByUsername(req.Username)
	if err != nil {
		slog.Error("get user from repo failed", slog.Any(constant.Error, err))

		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		slog.Error("wrong password", slog.String(constant.UserName, user.Username), slog.Any(constant.Error, err))

		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
	}

	claims := &jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 72)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		Subject:   user.ID.String(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	ss, err := token.SignedString(h.jwtSecret)
	if err != nil {
		slog.Error("sign token failed", slog.Any(constant.Error, err))

		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "could not create token"})
	}

	c.SetCookie(&http.Cookie{
		Name:     "jwt",
		Value:    ss,
		Expires:  time.Now().Add(time.Hour * 72),
		Path:     "/",
		Secure:   true,
		HttpOnly: true,
	})

	return c.NoContent(http.StatusOK)
}

func (h *AuthHandler) GetMe(c echo.Context) error {
	userIDStr, ok := c.Request().Context().Value(constant.UserID).(string)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid token"})
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid user id in token"})
	}

	user, err := h.userRepo.GetUserByID(userID)
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
