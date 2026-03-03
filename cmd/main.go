package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/gutsavosouza/transactions-api/internal/env"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx := context.Background()
	cfg := config{
		addr: ":8080",
		db: dbConfig{
			dsn: env.GetString("GOOSE_DBSTRING", "host=localhost user=postgres password=postgres dbname=transactions sslmode=disable"),
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// database connection
	pool, err := pgxpool.New(ctx, cfg.db.dsn)
	// conn, err := pgx.Connect(ctx, cfg.db.dsn)
	if err != nil {
		logger.Error("error connecting to database: %v", "error", err)
		panic(1)
	}
	// defer conn.Close(ctx)
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		logger.Error("error pinging database", "error", err)
		os.Exit(1)
	}

	logger.Info("connected to database", "dsn", cfg.db.dsn)

	api := app{
		config: cfg,
		db:     pool,
	}

	if error := api.run(api.mount()); error != nil {
		slog.Error("server failed to start, error: %v", "error", error)
		os.Exit(1)
	}
}
