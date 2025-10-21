package appctx

import (
	"context"

	"github.com/google/uuid"
)

const userIDKey ctxKey = "userID"

// WithUserID добавляет userID в контекст
func WithUserID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, userIDKey, id)
}

// UserID извлекает userID из контекста
func UserID(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(userIDKey).(uuid.UUID)
	return id, ok
}
