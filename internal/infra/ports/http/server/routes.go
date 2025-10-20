package server

import (
	"github.com/labstack/echo/v4"

	"github.com/qrave1/RoomSpeak/internal/application/config"
	"github.com/qrave1/RoomSpeak/internal/infra/ports/http/handlers"
	"github.com/qrave1/RoomSpeak/internal/infra/ports/http/middleware"
)

func New(
	cfg *config.Config,
	authHandler *handlers.AuthHandler,
	channelHandler *handlers.ChannelHandler,
	iceHandler *handlers.IceHandler,
	wsHandler *handlers.WebSocketHandler,
) *echo.Echo {
	e := echo.New()

	//e.Use(middleware.SlogLogger())

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

			v1.GET("/ice", iceHandler.IceServers)

			v1.GET("/ws", wsHandler.Handle)

			v1.GET("/channels", channelHandler.ListChannelsHandler)
			v1.POST("/channels", channelHandler.CreateChannelHandler)
			v1.DELETE("/channels/:id", channelHandler.DeleteChannelHandler)

			v1.GET("/users/online", authHandler.GetOnlineUsers)
		}
	}

	e.Static("/", "web")

	return e
}
