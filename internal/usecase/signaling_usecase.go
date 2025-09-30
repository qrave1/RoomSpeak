package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/pion/webrtc/v4"
	"github.com/qrave1/RoomSpeak/internal/application/constant"
	"github.com/qrave1/RoomSpeak/internal/domain/events"
	"github.com/qrave1/RoomSpeak/internal/domain/runtime"
	"github.com/qrave1/RoomSpeak/internal/infra/adapters/memory"
	postrepo "github.com/qrave1/RoomSpeak/internal/infra/adapters/postgres/repository"
)

type SignalingUsecase interface {
	BroadcastActiveMembers(ctx context.Context, channelID uuid.UUID) error

	HandleJoin(context.Context, uuid.UUID, events.JoinEvent) error
	HandleLeave(context.Context, uuid.UUID) error

	HandleOffer(context.Context, uuid.UUID, string) error
	HandleAnswer(context.Context, uuid.UUID, string) error
	HandleCandidate(context.Context, uuid.UUID, webrtc.ICECandidateInit) error

	HandlePing(context.Context, uuid.UUID)
	HandleMute(ctx context.Context, userID uuid.UUID, isMuted bool) error
}

type signalingUsecase struct {
	channelRepo postrepo.ChannelRepository
	userRepo    postrepo.UserRepository

	pcRepo         memory.PeerConnectionRepository
	wsRepo         memory.WebsocketConnectionRepository
	activeUserRepo memory.ActiveUserRepository

	peerUsecase PeerUsecase
}

func NewSignalingUsecase(
	channelRepo postrepo.ChannelRepository,
	userRepo postrepo.UserRepository,
	pcRepo memory.PeerConnectionRepository,
	wsRepo memory.WebsocketConnectionRepository,
	activeUserRepo memory.ActiveUserRepository,
	peerUsecase PeerUsecase,
) SignalingUsecase {
	return &signalingUsecase{
		channelRepo:    channelRepo,
		userRepo:       userRepo,
		pcRepo:         pcRepo,
		wsRepo:         wsRepo,
		activeUserRepo: activeUserRepo,
		peerUsecase:    peerUsecase,
	}
}

func (s *signalingUsecase) HandleJoin(ctx context.Context, userID uuid.UUID, joinEvent events.JoinEvent) error {
	if joinEvent.ChannelID == "" {
		s.wsRepo.Write(userID, map[string]any{"type": constant.Error, "message": "channel_id is required"})
		return nil
	}

	channelID, err := uuid.Parse(joinEvent.ChannelID)
	if err != nil {
		s.wsRepo.Write(userID, map[string]any{"type": constant.Error, "message": "invalid channel_id"})
		return nil
	}

	// Проверяем, что канал существует в базе данных
	_, err = s.channelRepo.GetByID(ctx, channelID)
	if err != nil {
		slog.Error("get channel", slog.Any(constant.Error, err))
		s.wsRepo.Write(userID, map[string]any{"type": constant.Error, "message": "channel not found"})
		return nil
	}

	peer, err := s.peerUsecase.CreateWebrtcPeer(ctx, userID, channelID)
	if err != nil {
		slog.Error("create peer connection", slog.Any(constant.Error, err))

		return nil
	}

	s.pcRepo.Add(userID, peer)

	activeUser := &runtime.ActiveUser{
		ID:        userID,
		ChannelID: channelID,
	}
	s.activeUserRepo.Add(ctx, activeUser)

	if err = s.BroadcastActiveMembers(ctx, channelID); err != nil {
		return fmt.Errorf("broadcast active members: %w", err)
	}

	return nil
}

func (s *signalingUsecase) BroadcastActiveMembers(ctx context.Context, channelID uuid.UUID) error {
	activeUsers := s.activeUserRepo.GetInChannel(ctx, channelID)

	activeMembers := events.ParticipantListEvent{List: make([]string, 0, len(activeUsers))}

	for _, activeUser := range activeUsers {
		user, err := s.userRepo.GetUserByID(activeUser.ID)
		if err != nil {
			continue // Skip users that can't be found
		}
		activeMembers.List = append(activeMembers.List, user.Username)
	}

	activeMembersDataJSON, err := json.Marshal(activeMembers)
	if err != nil {
		return fmt.Errorf("marshal active members: %w", err)
	}

	for _, activeUser := range activeUsers {
		s.wsRepo.Write(activeUser.ID, events.Message{Type: "participants", Data: activeMembersDataJSON})
	}

	return nil
}

func (s *signalingUsecase) HandleLeave(ctx context.Context, userID uuid.UUID) error {
	peer, ok := s.pcRepo.Get(userID)
	if !ok {
		return fmt.Errorf("peer connection not found")
	}

	s.activeUserRepo.Remove(ctx, userID)

	s.pcRepo.Remove(userID)

	if err := s.BroadcastActiveMembers(ctx, peer.ChannelID); err != nil {
		return fmt.Errorf("broadcast active members: %w", err)
	}

	return nil
}

func (s *signalingUsecase) HandleOffer(ctx context.Context, userID uuid.UUID, offer string) error {
	peer, ok := s.pcRepo.Get(userID)
	if !ok {
		return fmt.Errorf("peer connection not found")
	}

	if err := peer.Conn.SetRemoteDescription(
		webrtc.SessionDescription{
			Type: webrtc.SDPTypeOffer,
			SDP:  offer,
		},
	); err != nil {
		return fmt.Errorf("set remote description: %w", err)
	}

	answer, err := peer.Conn.CreateAnswer(nil)
	if err != nil {
		return fmt.Errorf("create answer: %w", err)
	}

	if err = peer.Conn.SetLocalDescription(answer); err != nil {
		return fmt.Errorf("set local description: %w", err)
	}

	s.wsRepo.Write(userID, map[string]any{"type": "answer", "sdp": answer.SDP})

	return nil
}

func (s *signalingUsecase) HandleAnswer(ctx context.Context, userID uuid.UUID, answer string) error {
	peer, ok := s.pcRepo.Get(userID)
	if !ok {
		return fmt.Errorf("peer connection not found")
	}

	err := peer.Conn.SetRemoteDescription(webrtc.SessionDescription{Type: webrtc.SDPTypeAnswer, SDP: answer})
	if err != nil {
		return fmt.Errorf("set remote description: %w", err)
	}

	return nil
}

func (s *signalingUsecase) HandleCandidate(ctx context.Context, userID uuid.UUID, candidate webrtc.ICECandidateInit) error {
	peer, ok := s.pcRepo.Get(userID)
	if !ok {
		return fmt.Errorf("peer connection not found")
	}

	err := peer.Conn.AddICECandidate(candidate)
	if err != nil {
		return err
	}

	return nil
}

func (s *signalingUsecase) HandlePing(ctx context.Context, userID uuid.UUID) {
	s.wsRepo.Write(userID, map[string]any{"type": "pong"})
}

func (s *signalingUsecase) HandleMute(ctx context.Context, userID uuid.UUID, isMuted bool) error {
	peer, ok := s.pcRepo.Get(userID)
	if !ok {
		return fmt.Errorf("peer connection not found")
	}

	activeUsers := s.activeUserRepo.GetInChannel(ctx, peer.ChannelID)

	user, err := s.userRepo.GetUserByID(userID)
	if err != nil {
		return fmt.Errorf("get user from postgres: %w", err)
	}

	userActionEvent, err := json.Marshal(events.UserActionEvent{
		UserName: user.Username,
		IsMuted:  isMuted,
	})
	if err != nil {
		return fmt.Errorf("marshal user action event: %w", err)
	}

	for _, activeUser := range activeUsers {
		if activeUser.ID == userID {
			continue
		}
		s.wsRepo.Write(activeUser.ID, events.Message{Type: "user_action", Data: userActionEvent})
	}

	return nil
}
