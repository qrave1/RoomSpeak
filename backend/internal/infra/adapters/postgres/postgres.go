package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

func NewPostgres(ctx context.Context, url string) (*sqlx.DB, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	db, err := sqlx.ConnectContext(dbCtx, "pgx", url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	if err = db.PingContext(dbCtx); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	slog.Info("connected to postgres")

	return db, nil
}
