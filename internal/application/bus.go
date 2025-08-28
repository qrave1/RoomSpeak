package application

import (
	"context"
	"errors"

	"github.com/qrave1/RoomSpeak/internal/domain"
)

// CommandBus dispatches commands.
type CommandBus interface {
	Dispatch(ctx context.Context, cmd domain.Command) error
}

// QueryBus dispatches queries.
type QueryBus interface {
	Dispatch(ctx context.Context, q domain.Query) (any, error)
}

// SimpleCommandBus implements CommandBus.
type SimpleCommandBus struct {
	Handler *RoomCommandHandler
}

func (b *SimpleCommandBus) Dispatch(ctx context.Context, cmd domain.Command) error {
	switch c := cmd.(type) {
	case *domain.JoinRoomCommand:
		return b.Handler.HandleJoin(ctx, c)
	case *domain.LeaveRoomCommand:
		return b.Handler.HandleLeave(ctx, c)
	case *domain.BroadcastRTPCommand:
		return b.Handler.HandleBroadcastRTP(ctx, c)
	default:
		return errors.New("unknown command")
	}
}

// SimpleQueryBus implements QueryBus.
type SimpleQueryBus struct {
	Handler *RoomQueryHandler
}

func (b *SimpleQueryBus) Dispatch(ctx context.Context, q domain.Query) (any, error) {
	switch qq := q.(type) {
	case *domain.ParticipantsQuery:
		return b.Handler.HandleParticipants(ctx, qq)
	default:
		return nil, errors.New("unknown query")
	}
}
