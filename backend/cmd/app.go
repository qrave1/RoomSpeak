package cmd

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/qrave1/RoomSpeak/internal/application/config"
	"github.com/qrave1/RoomSpeak/internal/application/constant"
	"github.com/qrave1/RoomSpeak/internal/application/metric"
	memory2 "github.com/qrave1/RoomSpeak/internal/infra/adapters/memory"
	"github.com/qrave1/RoomSpeak/internal/infra/adapters/postgres"
	repository2 "github.com/qrave1/RoomSpeak/internal/infra/adapters/postgres/repository"
	handlers2 "github.com/qrave1/RoomSpeak/internal/infra/ports/http/handlers"
	"github.com/qrave1/RoomSpeak/internal/infra/ports/http/server"
	usecase2 "github.com/qrave1/RoomSpeak/internal/usecase"
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

	// TODO DI
	dbConn, err := postgres.NewPostgres(ctx, cfg.Postgres.DSN())
	if err != nil {
		slog.Error("connect to postgres", slog.Any(constant.Error, err))
		os.Exit(1)
	}
	defer dbConn.Close()

	userRepo := repository2.NewUserRepo(dbConn)
	channelRepo := repository2.NewChannelRepo(dbConn)
	wsConnRepo := memory2.NewWSConnectionRepository()
	pcConnRepo := memory2.NewPeerConnectionRepository()
	activeUserRepo := memory2.NewActiveUserRepository()

	userUsecase := usecase2.NewUserUsecase([]byte(cfg.JWTSecret), userRepo, channelRepo, wsConnRepo)
	channelUsecase := usecase2.NewChannelUsecase(channelRepo, activeUserRepo)
	peerUsecase := usecase2.NewPeerUsecase(cfg, pcConnRepo, wsConnRepo, activeUserRepo)
	signalingUsecase := usecase2.NewSignalingUsecase(channelRepo, userRepo, pcConnRepo, wsConnRepo, activeUserRepo, peerUsecase)

	authHandler := handlers2.NewAuthHandler(userUsecase)
	channelHandler := handlers2.NewChannelHandler(channelUsecase, userRepo)
	iceHandler := handlers2.NewIceHandler(cfg)
	wsHandler := handlers2.NewWebSocketHandler(cfg, signalingUsecase, wsConnRepo)

	echoSrv := server.New(cfg, authHandler, channelHandler, iceHandler, wsHandler)

	metricsSrv := metric.NewServer()

	echoSrvCh := make(chan error, 1)
	metricsSrvCh := make(chan error, 1)

	// Запускаем HTTP сервер
	go func() {
		echoSrvCh <- echoSrv.Start(":" + cfg.Port)
	}()

	// Запускаем сервер метрик
	go func() {
		metricsSrvCh <- metricsSrv.Start(":" + cfg.MetricPort)
	}()

	// Ожидаем сигнал завершения или ошибку сервера
	select {
	case <-ctx.Done():
		slog.Info("Shutting down servers due to context cancel")
	case err := <-echoSrvCh:
		slog.Error(
			"HTTP server failed",
			slog.Any(constant.Error, err),
		)
		os.Exit(1)
	case err := <-metricsSrvCh:
		slog.Error(
			"Metrics server failed",
			slog.Any(constant.Error, err),
		)
		os.Exit(1)
	}

	// Graceful shutdown
	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, 5*time.Second)
	defer timeoutCancel()

	if err := echoSrv.Shutdown(timeoutCtx); err != nil {
		slog.Error("Failed to gracefully shutdown HTTP server", slog.Any(constant.Error, err))
	}

	if err := metricsSrv.Shutdown(timeoutCtx); err != nil {
		slog.Error("Failed to gracefully shutdown metric server", slog.Any(constant.Error, err))
	}
}
