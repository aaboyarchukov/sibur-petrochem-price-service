package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Миграции и seed применяются отдельно (docker-сервис migrate / `make migrate-up`),
// а не приложением. Здесь — подключение к готовой БД и запуск сервиса.
func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		slog.Error("database_url is empty")
		os.Exit(1)
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		slog.Error("connect pool failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		slog.Error("db ping failed", "error", err)
		os.Exit(1)
	}
	slog.Info("connected to database")

	// TODO: здесь стартует HTTP-сервис (хендлеры из api/openapi.yaml).
	// Пока сервиса нет — держим процесс живым до сигнала.
	stop, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	<-stop.Done()
	slog.Info("shutdown signal received")
}
