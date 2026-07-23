package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Config — параметры подключения к PostgreSQL. Значения берутся из yaml/env
// (env перекрывает yaml), env-default используется, когда ни то, ни другое не задано.
type Config struct {
	User     string `yaml:"user"      env:"PG_DB_USER"     validate:"required"`
	Password string `yaml:"password"  env:"PG_DB_PASSWORD" validate:"required"`
	DB       string `yaml:"name"      env:"PG_DB_NAME"     validate:"required"`
	Host     string `yaml:"host"      env:"PG_DB_HOST"     env-default:"localhost" validate:"required"`
	Port     string `yaml:"port"      env:"PG_DB_PORT"     env-default:"5432"      validate:"required"`
	SSLMode  string `yaml:"sslmode"   env:"PG_DB_SSLMODE"  env-default:"disable"   validate:"required"`
	MaxConns int32  `yaml:"max_conns" env:"PG_DB_MAX_CONNS" env-default:"25" validate:"required,gt=0"`
	MinConns int32  `yaml:"min_conns" env:"PG_DB_MIN_CONNS" env-default:"5"  validate:"gte=0"`
}

// DSN — строка подключения формата postgres://user:pass@host:port/db?sslmode=...
func (c Config) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.DB, c.SSLMode)
}

// NewPool — пул подключений с проверкой доступности базы (Ping).
func NewPool(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("parse pool config: %w", err)
	}
	poolConfig.MaxConns = cfg.MaxConns
	poolConfig.MinConns = cfg.MinConns

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create pgx pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()

		return nil, fmt.Errorf("ping database: %w", err)
	}

	return pool, nil
}
