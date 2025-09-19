package cmd

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/qrave1/RoomSpeak/internal/application/config"
	"github.com/qrave1/RoomSpeak/internal/application/constant"
	"github.com/qrave1/RoomSpeak/internal/infra/adapters/memory"
	"github.com/qrave1/RoomSpeak/internal/infra/adapters/postgres"
	"github.com/qrave1/RoomSpeak/internal/infra/adapters/postgres/repository"
	"github.com/qrave1/RoomSpeak/internal/infra/ports/http/handlers"
	"github.com/qrave1/RoomSpeak/internal/infra/ports/http/server"
	"github.com/qrave1/RoomSpeak/internal/usecase"
)

func runApp() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

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
	wsConnRepo := memory.NewWSConnectionRepository()
	pcConnRepo := memory.NewPeerConnectionRepository()
	channelMembersRepo := memory.NewChannelMembersRepository()

	userUsecase := usecase.NewUserUsecase([]byte(cfg.JWTSecret), userRepo)
	channelUsecase := usecase.NewChannelUsecase(channelRepo)
	peerUsecase := usecase.NewPeerUsecase(cfg, pcConnRepo, wsConnRepo, channelMembersRepo)
	signalingUsecase := usecase.NewSignalingUsecase(channelRepo, pcConnRepo, wsConnRepo, channelMembersRepo, peerUsecase)

	authHandler := handlers.NewAuthHandler(userUsecase)
	channelHandler := handlers.NewChannelHandler(channelUsecase)
	iceHandler := handlers.NewIceHandler(cfg)
	wsHandler := handlers.NewWebSocketHandler(cfg, signalingUsecase)

	echoSrv := server.New(cfg, authHandler, channelHandler, iceHandler, wsHandler)

	srvCh := make(chan (error), 1)
	go func() {
		srvCh <- echoSrv.Start(":" + cfg.Port)
	}()

	select {
	case <-ctx.Done():
		slog.Info("Shutting down server due to context cancel")
	case <-srvCh:
		slog.Error(
			"HTTP server failed",
			slog.Any(constant.Error, err),
		)

		os.Exit(1)
	}

	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, 5*time.Second)
	defer timeoutCancel()

	if err := echoSrv.Shutdown(timeoutCtx); err != nil {
		slog.Error("Failed to gracefully shutdown server", slog.Any(constant.Error, err))
	}
}
