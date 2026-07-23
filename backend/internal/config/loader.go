package config

import (
	"flag"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

// Parse — загрузка конфигурации: .env → yaml (по флагу -cfg) → env-переменные (перекрывают yaml).
// Флаг -env-only парсит только окружение, игнорируя yaml-файл (режим контейнера).
func Parse() (App, error) {
	var cfg App

	_ = godotenv.Load()

	cfgPath, envOnly := configFlags()

	if err := load(&cfg, cfgPath, envOnly); err != nil {
		return App{}, err
	}

	if err := validate(cfg); err != nil {
		return App{}, err
	}

	return cfg, nil
}

func configFlags() (cfgPath string, envOnly bool) {
	const (
		cfgArgKey      = "cfg"
		defaultCfgPath = "./configs/local.yaml"
		envOnlyArgKey  = "env-only"
	)

	cfgPathPtr := flag.String(cfgArgKey, defaultCfgPath, "path to config file")
	envOnlyPtr := flag.Bool(envOnlyArgKey, false, "use only environment variables, ignore yaml config")
	flag.Parse()

	return *cfgPathPtr, *envOnlyPtr
}

func load(cfg *App, cfgPath string, envOnly bool) error {
	if envOnly {
		if err := cleanenv.ReadEnv(cfg); err != nil {
			return fmt.Errorf("read env: %w", err)
		}

		return nil
	}

	// yaml как база, поверх — env (env перекрывает значения из файла)
	if err := cleanenv.ReadConfig(cfgPath, cfg); err != nil {
		return fmt.Errorf("read config %s: %w", cfgPath, err)
	}

	return nil
}

func validate(cfg App) error {
	v := validator.New(validator.WithRequiredStructEnabled())
	if err := v.Struct(cfg); err != nil {
		return fmt.Errorf("validate config: %w", err)
	}

	return nil
}
