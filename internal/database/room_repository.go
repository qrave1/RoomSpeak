package database

import (
	"context"
	"fmt"

	"github.com/qrave1/RoomSpeak/internal/models"
)

// RoomRepository интерфейс для работы с комнатами
type RoomRepository interface {
	CreateRoom(ctx context.Context, room *models.Room) error
	GetRoomByID(ctx context.Context, roomID string) (*models.Room, error)
	GetAllRooms(ctx context.Context) ([]*models.Room, error)
}

// roomRepository реализация RoomRepository
type roomRepository struct {
	db *DB
}

// NewRoomRepository создает новый экземпляр roomRepository
func NewRoomRepository(db *DB) RoomRepository {
	return &roomRepository{db: db}
}

// CreateRoom создает новую комнату
func (r *roomRepository) CreateRoom(ctx context.Context, room *models.Room) error {
	query := `
		INSERT INTO rooms (room_id, name) 
		VALUES ($1, $2)
		RETURNING id, created_at
	`

	err := r.db.Pool.QueryRow(ctx, query, room.RoomID, room.Name).
		Scan(&room.ID, &room.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create room: %w", err)
	}

	return nil
}

// GetRoomByID получает комнату по ID
func (r *roomRepository) GetRoomByID(ctx context.Context, roomID string) (*models.Room, error) {
	room := &models.Room{}
	query := `
		SELECT id, room_id, name, created_at 
		FROM rooms 
		WHERE room_id = $1
	`

	err := r.db.Pool.QueryRow(ctx, query, roomID).
		Scan(&room.ID, &room.RoomID, &room.Name, &room.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get room: %w", err)
	}

	return room, nil
}

// GetAllRooms получает все комнаты
func (r *roomRepository) GetAllRooms(ctx context.Context) ([]*models.Room, error) {
	query := `
		SELECT id, room_id, name, created_at 
		FROM rooms 
		ORDER BY created_at DESC
	`

	rows, err := r.db.Pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query rooms: %w", err)
	}
	defer rows.Close()

	var rooms []*models.Room
	for rows.Next() {
		room := &models.Room{}
		if err := rows.Scan(&room.ID, &room.RoomID, &room.Name, &room.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan room: %w", err)
		}
		rooms = append(rooms, room)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return rooms, nil
}
