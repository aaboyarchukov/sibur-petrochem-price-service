// Package app — сборка приложения: БД, сервисы, хендлеры, HTTP-сервер.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"sibur-petrochem-price-service/internal/config"
	"sibur-petrochem-price-service/internal/repository/postgres"
	"sibur-petrochem-price-service/internal/service/calculations"
	"sibur-petrochem-price-service/internal/service/pricing"

	delivery_http "sibur-petrochem-price-service/internal/delivery/http"
)

type App struct {
	cfg    config.App
	pool   *pgxpool.Pool
	server *http.Server
}

func New(ctx context.Context, cfg config.App) (*App, error) {
	pool, err := postgres.NewPool(ctx, cfg.Postgres)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}

	repo := postgres.New(pool)
	service := calculations.New(repo, pricing.NewEngine())

	server := &http.Server{
		Addr:              cfg.Server.Addr,
		Handler:           delivery_http.NewRouter(service, repo),
		ReadHeaderTimeout: cfg.Server.ReadHeaderTimeout,
	}

	return &App{cfg: cfg, pool: pool, server: server}, nil
}

func (a *App) Run() {
	go func() {
		slog.Info("http server started", "addr", a.cfg.Server.Addr)
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("http server failed", "error", err)
		}
	}()
}

func (a *App) Shutdown(ctx context.Context) {
	shutdownCtx, cancel := context.WithTimeout(ctx, a.cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := a.server.Shutdown(shutdownCtx); err != nil {
		slog.Error("http server shutdown failed", "error", err)
	}
	a.pool.Close()
	slog.Info("app stopped")
}
