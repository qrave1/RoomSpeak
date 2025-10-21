package cmd

import (
	"log"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/spf13/cobra"

	"github.com/qrave1/RoomSpeak/internal/application/config"
	"github.com/qrave1/RoomSpeak/internal/infra/adapters/postgres/migrations"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run database migrations",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			log.Fatalf("empty args: needed at least one arg")
		}

		cfg, err := config.New()
		if err != nil {
			log.Fatalf("could not load config: %v", err)
		}

		goose.SetBaseFS(migrations.MigrationsFS)

		db, err := goose.OpenDBWithDriver("pgx", cfg.Postgres.DSN())
		if err != nil {
			log.Fatalf("goose: failed to open DB: %v", err)
		}

		defer func() {
			if err := db.Close(); err != nil {
				log.Fatalf("goose: failed to close DB: %v", err)
			}
		}()

		err = goose.RunContext(
			cmd.Context(),
			args[0],
			db,
			".",
			args[1:]...,
		)

		if err != nil {
			log.Fatalf("goose: up failed: %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)
}
