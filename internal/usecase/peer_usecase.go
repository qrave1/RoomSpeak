package usecase

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/google/uuid"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"

	"github.com/qrave1/RoomSpeak/internal/application/config"
	"github.com/qrave1/RoomSpeak/internal/application/constant"
	"github.com/qrave1/RoomSpeak/internal/domain"
	"github.com/qrave1/RoomSpeak/internal/infra/adapters/memory"
)

type PeerUsecase interface {
	CreateWebrtcPeer(ctx context.Context, userID uuid.UUID, channelID uuid.UUID) (*domain.Peer, error)
}

type peerUsecase struct {
	cfg *config.Config

	pcRepo             memory.PeerConnectionRepository
	wsRepo             memory.WebsocketConnectionRepository
	channelMembersRepo memory.ChannelMembersRepository
}

func NewPeerUsecase(
	cfg *config.Config,
	pcRepo memory.PeerConnectionRepository,
	wsRepo memory.WebsocketConnectionRepository,
	channelMembersRepo memory.ChannelMembersRepository,
) *peerUsecase {
	return &peerUsecase{
		cfg:                cfg,
		pcRepo:             pcRepo,
		wsRepo:             wsRepo,
		channelMembersRepo: channelMembersRepo,
	}
}

func (p *peerUsecase) CreateWebrtcPeer(ctx context.Context, userID uuid.UUID, channelID uuid.UUID) (*domain.Peer, error) {
	peer, err := domain.NewPeer(userID, channelID, p.cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create peer: %w", err)
	}

	peer.Conn.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		go func(ctx context.Context, userID uuid.UUID, channelID uuid.UUID) {
			for {
				select {
				case <-ctx.Done():
					return
				default:
					pkt, _, err := track.ReadRTP()
					if err != nil {
						if !errors.Is(err, io.EOF) {
							slog.Error("RTP read error", slog.Any(constant.Error, err))
						}

						return
					}

					if track.Kind() == webrtc.RTPCodecTypeAudio {
						p.broadcastRTP(ctx, pkt, userID, channelID)
					}
				}
			}
		}(ctx, userID, channelID)
	})

	peer.Conn.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}

		p.wsRepo.Write(userID, map[string]any{"type": "candidate", "candidate": c.ToJSON()})
	})

	//peer.Conn.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
	//	if state == webrtc.PeerConnectionStateFailed || state == webrtc.PeerConnectionStateDisconnected {
	//		slog.Error("PeerConnection bad status",
	//			slog.String(constant.State, state.String()),
	//			slog.Any(constant.UserName, userID),
	//		)
	//
	//		p.wsRepo.Write(userID, map[string]any{
	//			"type":    constant.Error,
	//			"message": fmt.Sprintf("peer connection bad state: %s", state.String()),
	//		})
	//	}
	//})

	return peer, nil
}

func (p *peerUsecase) broadcastRTP(ctx context.Context, pkt *rtp.Packet, userID uuid.UUID, channelID uuid.UUID) {
	members := p.channelMembersRepo.GetMembers(ctx, channelID)

	for _, member := range members {
		if member.ID == userID {
			continue
		}

		pc, ok := p.pcRepo.Get(member.ID)
		if !ok {
			slog.Error("get peer connection in broadcast")
			continue
		}

		err := pc.AudioTrack.WriteRTP(pkt)

		if err != nil {
			slog.Error(
				"write RTP",
				slog.Any(constant.Error, err),
				slog.Any(constant.UserID, userID),
				slog.Any(constant.ChannelID, channelID),
			)
		}
	}
}
