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
	memory2 "github.com/qrave1/RoomSpeak/internal/infra/adapters/memory"
)

type PeerUsecase interface {
	CreateWebrtcPeer(ctx context.Context, userID uuid.UUID, channelID uuid.UUID) (*domain.Peer, error)
}

type peerUsecase struct {
	cfg *config.Config

	pcRepo         memory2.PeerConnectionRepository
	wsRepo         memory2.WebsocketConnectionRepository
	activeUserRepo memory2.ActiveUserRepository
}

func NewPeerUsecase(
	cfg *config.Config,
	pcRepo memory2.PeerConnectionRepository,
	wsRepo memory2.WebsocketConnectionRepository,
	activeUserRepo memory2.ActiveUserRepository,
) *peerUsecase {
	return &peerUsecase{
		cfg:            cfg,
		pcRepo:         pcRepo,
		wsRepo:         wsRepo,
		activeUserRepo: activeUserRepo,
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

	return peer, nil
}

func (p *peerUsecase) broadcastRTP(ctx context.Context, pkt *rtp.Packet, userID uuid.UUID, channelID uuid.UUID) {
	activeUsers := p.activeUserRepo.GetInChannel(ctx, channelID)

	for _, activeUser := range activeUsers {
		if activeUser.ID == userID {
			continue
		}

		pc, ok := p.pcRepo.Get(activeUser.ID)
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
