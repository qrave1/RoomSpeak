package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/qrave1/RoomSpeak/internal/config"
	"github.com/qrave1/RoomSpeak/internal/database"
	"github.com/qrave1/RoomSpeak/internal/sfu"
	"github.com/qrave1/RoomSpeak/internal/websocket"
)

func main() {
	// Загружаем конфигурацию
	cfg := config.Load()

	// Подключаемся к базе данных
	db, err := database.New(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	log.Println("Connected to PostgreSQL database")

	// Создаем репозиторий комнат
	roomRepo := database.NewRoomRepository(db)

	// Создаем SFU сервер
	sfuServer := sfu.NewSFU()
	log.Println("SFU server initialized")

	// Создаем WebSocket hub
	hub := websocket.NewHub(sfuServer, roomRepo)
	go hub.Run()
	log.Println("WebSocket hub started")

	// Создаем WebSocket обработчик
	wsHandler := websocket.NewHandler(hub)

	// Настраиваем HTTP роуты
	mux := http.NewServeMux()

	// WebSocket endpoint
	mux.HandleFunc("/ws", wsHandler.HandleWebSocket)

	// API endpoints
	mux.HandleFunc("/api/rooms", handleRoomsAPI(roomRepo))

	// Статические файлы
	fileServer := http.FileServer(http.Dir(cfg.Server.StaticDir))
	mux.Handle("/", fileServer)

	// Создаем HTTP сервер
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Запускаем сервер в горутине
	go func() {
		log.Printf("Server starting on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Ожидаем сигнал завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

// handleRoomsAPI обрабатывает API для комнат
func handleRoomsAPI(roomRepo database.RoomRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			// Получить список всех комнат
			rooms, err := roomRepo.GetAllRooms(r.Context())
			if err != nil {
				http.Error(w, "Failed to get rooms", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(rooms); err != nil {
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)
				return
			}

		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}
