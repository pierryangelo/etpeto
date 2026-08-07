package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func main() {
	databaseURL := os.Getenv("ETPETO_DATABASE_URL")
	if databaseURL == "" {
		slog.Error("ETPETO_DATABASE_URL is required")
		os.Exit(1)
	}
	migrationsDirectory := os.Getenv("ETPETO_MIGRATIONS_DIR")
	if migrationsDirectory == "" {
		migrationsDirectory = "db/migrations"
	}

	database, err := sql.Open("pgx", databaseURL)
	if err != nil {
		slog.Error("open migration database failed")
		os.Exit(1)
	}
	defer database.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := database.PingContext(ctx); err != nil {
		slog.Error("migration database unavailable")
		os.Exit(1)
	}
	provider, err := goose.NewProvider(
		goose.DialectPostgres,
		database,
		os.DirFS(migrationsDirectory),
	)
	if err != nil {
		slog.Error("initialize migrations failed")
		os.Exit(1)
	}
	results, err := provider.Up(ctx)
	if err != nil {
		slog.Error("apply migrations failed")
		os.Exit(1)
	}
	slog.Info("database migrations complete", "applied", len(results))
}
