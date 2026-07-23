// Package config — конфигурация сервиса из переменных окружения.
package config

import (
	"errors"
	"os"
	"time"
)

// ErrEmptyDatabaseURL — не задан DSN базы данных.
var ErrEmptyDatabaseURL = errors.New("empty DATABASE_URL")

// App — конфигурация приложения.
type App struct {
	DatabaseURL       string
	ServerAddr        string
	ReadHeaderTimeout time.Duration
	ShutdownTimeout   time.Duration
}

func Parse() (App, error) {
	const (
		defaultAddr              = ":8080"
		defaultReadHeaderTimeout = 5 * time.Second
		defaultShutdownTimeout   = 10 * time.Second
	)

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return App{}, ErrEmptyDatabaseURL
	}

	addr := os.Getenv("SERVER_ADDR")
	if addr == "" {
		addr = defaultAddr
	}

	return App{
		DatabaseURL:       databaseURL,
		ServerAddr:        addr,
		ReadHeaderTimeout: defaultReadHeaderTimeout,
		ShutdownTimeout:   defaultShutdownTimeout,
	}, nil
}
