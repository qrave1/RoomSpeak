package handlers

import (
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/qrave1/RoomSpeak/internal/application/constant"
	"github.com/qrave1/RoomSpeak/internal/domain/input"
	"github.com/qrave1/RoomSpeak/internal/infra/appctx"
	"github.com/qrave1/RoomSpeak/internal/infra/ports/http/dto"
	postrepo "github.com/qrave1/RoomSpeak/internal/infra/adapters/postgres/repository"
	"github.com/qrave1/RoomSpeak/internal/usecase"
)

type ChannelHandler struct {
	channelUsecase usecase.ChannelUsecase
	userRepo       postrepo.UserRepository
}

func NewChannelHandler(channelUsecase usecase.ChannelUsecase, userRepo postrepo.UserRepository) *ChannelHandler {
	return &ChannelHandler{channelUsecase: channelUsecase, userRepo: userRepo}
}

func (h *ChannelHandler) ListChannelsHandler(c echo.Context) error {
	// Получаем userID из JWT токена
	userID, ok := appctx.UserID(c.Request().Context())
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid user"})
	}

	// Получаем все доступные каналы для пользователя (приватные + публичные)
	availableChannels, err := h.channelUsecase.GetAvailableChannelsForUser(c.Request().Context(), userID)
	if err != nil {
		slog.Error("get available channels for user", slog.Any(constant.Error, err))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get available channels"})
	}

	resp := dto.ListChannelsResponse{
		Channels: make([]dto.ChannelResponse, 0, len(availableChannels)),
	}

	for _, ch := range availableChannels {
		activeUsers, err := h.channelUsecase.GetActiveUsersByID(c.Request().Context(), ch.ID)
		if err != nil {
			slog.Error("get active users by id", slog.Any(constant.Error, err))
			continue
		}

		// Преобразуем ActiveUser в ActiveUserInfo
		activeUserInfos := make([]dto.ActiveUserInfo, 0, len(activeUsers))
		for _, activeUser := range activeUsers {
			user, err := h.userRepo.GetUserByID(activeUser.ID)
			if err != nil {
				continue // Пропускаем пользователей, которых не можем найти
			}
			activeUserInfos = append(activeUserInfos, dto.ActiveUserInfo{
				ID:       activeUser.ID.String(),
				Username: user.Username,
			})
		}

		resp.Channels = append(resp.Channels, dto.NewChannelResponseFromModel(ch, activeUserInfos))
	}

	return c.JSON(http.StatusOK, resp)
}

func (h *ChannelHandler) CreateChannelHandler(c echo.Context) error {
	var req dto.CreateChannelRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	if req.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "name is required"})
	}

	// Получаем userID из JWT токена
	userID, ok := appctx.UserID(c.Request().Context())
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid user"})
	}

	createChannelInput := &input.CreateChannelInput{
		Name:      req.Name,
		IsPublic:  req.IsPublic,
		CreatorID: userID,
	}

	channel, err := h.channelUsecase.CreateChannel(c.Request().Context(), createChannelInput)
	if err != nil {
		slog.Error("create channel", slog.Any(constant.Error, err))

		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create channel"})
	}

	// Добавляем создателя в канал, только если канал приватный
	if !channel.IsPublic {
		if err := h.channelUsecase.AddUserToChannel(c.Request().Context(), userID, channel.ID); err != nil {
			slog.Error("add user to channel", slog.Any(constant.Error, err))

			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to add user to channel"})
		}
	}

	return c.JSON(http.StatusCreated, channel)
}

func (h *ChannelHandler) DeleteChannelHandler(c echo.Context) error {
	channelIDStr := c.Param("id")
	channelID, err := uuid.Parse(channelIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid channel id"})
	}

	// Получаем userID из JWT токена
	userID, ok := appctx.UserID(c.Request().Context())
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
