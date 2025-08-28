package application

import (
	"context"
	"log/slog"

	"github.com/qrave1/RoomSpeak/internal/domain"
)

// RoomCommandHandler handles room-related commands.
type RoomCommandHandler struct {
	RoomRepo   RoomRepository
	ClientRepo ClientRepository
}

func (h *RoomCommandHandler) HandleJoin(ctx context.Context, cmd *domain.JoinRoomCommand) error {
	room := h.RoomRepo.GetOrCreate(cmd.RoomID)

	client := domain.NewClientFromCmd(cmd)

	h.ClientRepo.AddClient(room, client)

	h.broadcastParticipants(room)

	return nil
}

func (h *RoomCommandHandler) HandleLeave(ctx context.Context, cmd *domain.LeaveRoomCommand) error {
	room := h.RoomRepo.GetOrCreate(cmd.RoomID) // Assume room exists
	h.ClientRepo.RemoveClient(room, cmd.ClientID)
	h.broadcastParticipants(room)
	if len(h.ClientRepo.GetClients(room)) == 0 {
		h.RoomRepo.Remove(cmd.RoomID)
	}
	return nil
}

func (h *RoomCommandHandler) HandleBroadcastRTP(ctx context.Context, cmd *domain.BroadcastRTPCommand) error {
	room := h.RoomRepo.GetOrCreate(cmd.RoomID) // Assume room exists
	clients := h.ClientRepo.GetClients(room)
	for _, client := range clients {
		if client.ID == cmd.SenderID {
			continue
		}
		if err := client.AudioTrack.WriteRTP(cmd.Packet); err != nil {
			slog.Error("RTP write error", "error", err)
		}
	}
	return nil
}

func (h *RoomCommandHandler) broadcastParticipants(room *domain.Room) {
	clients := h.ClientRepo.GetClients(room)
	parts := make([]string, 0, len(clients))
	for _, client := range clients {
		parts = append(parts, client.Name)
	}
	for _, client := range clients {
		client.WsConn.WriteJSON(map[string]interface{}{"type": "participants", "list": parts})
	}
}

// RoomQueryHandler handles room-related queries.
type RoomQueryHandler struct {
	RoomRepo   RoomRepository
	ClientRepo ClientRepository
}

func (h *RoomQueryHandler) HandleParticipants(ctx context.Context, q *domain.ParticipantsQuery) ([]string, error) {
	room := h.RoomRepo.GetOrCreate(q.RoomID) // Assume room exists
	clients := h.ClientRepo.GetClients(room)
	parts := make([]string, 0, len(clients))
	for _, client := range clients {
		parts = append(parts, client.Name)
	}
	return parts, nil
}

// RoomRepository interface for room storage.
type RoomRepository interface {
	GetOrCreate(roomID string) *domain.Room
	Remove(roomID string)
}

// ClientRepository interface for client operations within rooms.
type ClientRepository interface {
	AddClient(room *domain.Room, client *domain.Client)
	RemoveClient(room *domain.Room, clientID string)
	GetClients(room *domain.Room) map[string]*domain.Client
}
