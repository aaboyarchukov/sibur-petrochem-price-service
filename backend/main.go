package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"sibur-petrochem-price-service/internal/app"
	"sibur-petrochem-price-service/internal/config"
)

// Миграции и seed применяются отдельно (docker-сервис migrate / `make migrate-up`),
// а не приложением. Здесь — подключение к готовой БД и запуск HTTP-сервиса.
func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Parse()
	if err != nil {
		slog.Error("config parse failed", "error", err)
		os.Exit(1)
	}

	application, err := app.New(ctx, cfg)
	if err != nil {
		slog.Error("app init failed", "error", err)
		os.Exit(1)
	}

	slog.Info("connected to database")
	application.Run()

	<-ctx.Done()
	slog.Info("shutdown signal received")
	application.Shutdown(context.Background())
}
