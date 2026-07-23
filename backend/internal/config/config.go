// Package config — конфигурация сервиса из yaml и переменных окружения.
package config

import (
	"time"

	"sibur-petrochem-price-service/internal/repository/postgres"
)

// App — корневая конфигурация приложения. Подключаемые группы имеют собственные
// yaml-секции; env перекрывает yaml, env-default — фолбэк для необязательных полей.
type App struct {
	Postgres postgres.Config `yaml:"postgres" validate:"required"`
	Server   Server          `yaml:"server"   validate:"required"`
}

// Server — параметры HTTP-сервера.
type Server struct {
	Addr              string        `yaml:"addr"                env:"SERVER_ADDR"                 env-default:":8080" validate:"required"`
	ReadHeaderTimeout time.Duration `yaml:"read_header_timeout"  env:"SERVER_READ_HEADER_TIMEOUT" env-default:"5s"    validate:"required,gt=0"`
	ShutdownTimeout   time.Duration `yaml:"shutdown_timeout"     env:"SERVER_SHUTDOWN_TIMEOUT"    env-default:"10s"   validate:"required,gt=0"`
}
