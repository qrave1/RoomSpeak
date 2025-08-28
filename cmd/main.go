package main

import (
	"log/slog"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/qrave1/RoomSpeak/internal/application"
	"github.com/qrave1/RoomSpeak/internal/infrastructure/http/middleware"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	app := application.NewApp()

	e := echo.New()
	e.Use(middleware.SlogLogger())
	e.Static("/", "web")
	e.GET("/ws", app.HandleWebSocket)
	err := e.Start(":3000")
	if err != nil {
		slog.Error("HTTP server failed", "error", err)
		os.Exit(1)
	}
}
