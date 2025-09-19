package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/pion/webrtc/v4"
	"github.com/qrave1/RoomSpeak/internal/application/constant"
	"github.com/qrave1/RoomSpeak/internal/domain/events"
	"github.com/qrave1/RoomSpeak/internal/infra/adapters/memory"
	postrepo "github.com/qrave1/RoomSpeak/internal/infra/adapters/postgres/repository"
)

type SignalingUsecase interface {
	HandleJoin(context.Context, uuid.UUID, events.JoinEvent) error
	HandleLeave(context.Context, uuid.UUID) error

	HandleOffer(context.Context, uuid.UUID, string) error
	HandleAnswer(context.Context, uuid.UUID, string) error
	HandleCandidate(context.Context, uuid.UUID, webrtc.ICECandidateInit) error

	HandlePing(context.Context, uuid.UUID)
}

type signalingUsecase struct {
	channelRepo        postrepo.ChannelRepository
	pcRepo             memory.PeerConnectionRepository
	wsRepo             memory.WebsocketConnectionRepository
	channelMembersRepo memory.ChannelMembersRepository

	peerUsecase PeerUsecase
}

func NewSignalingUsecase(
	channelRepo postrepo.ChannelRepository,
	pcRepo memory.PeerConnectionRepository,
	wsRepo memory.WebsocketConnectionRepository,
	channelMembersRepo memory.ChannelMembersRepository,
	peerUsecase PeerUsecase,
) SignalingUsecase {
	return &signalingUsecase{
		channelRepo:        channelRepo,
		pcRepo:             pcRepo,
		wsRepo:             wsRepo,
		channelMembersRepo: channelMembersRepo,
		peerUsecase:        peerUsecase,
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

	s.channelMembersRepo.AddMember(ctx, channelID, userID)

	return nil
}

func (s *signalingUsecase) HandleLeave(ctx context.Context, userID uuid.UUID) error {
	peer, ok := s.pcRepo.Get(userID)
	if !ok {
		return fmt.Errorf("peer connection not found")
	}

	s.channelMembersRepo.RemoveMember(ctx, peer.ChannelID, userID)

	s.pcRepo.Remove(userID)

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
