package user

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/qrave1/RoomSpeak/internal/repository"
)

type UserHandler struct {
	userRepo repository.UserRepository
}

func NewUserHandler(userRepo repository.UserRepository) *UserHandler {
	return &UserHandler{
		userRepo: userRepo,
	}
}

func (h *UserHandler) CreateUser(c echo.Context) error {
	return c.String(http.StatusOK, "CreateUser endpoint")
}
