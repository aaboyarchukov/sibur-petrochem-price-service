# Общие переменные и установка инструментов.

# Переменные окружения из .env
-include .env
export

# Каталоги
TOOLS_DIR := ./bin

# DSN БД (для локального запуска migrate CLI)
DB_DSN := postgresql://$(PG_DB_USER):$(PG_DB_PASSWORD)@$(PG_DB_HOST):$(PG_DB_PORT)/$(PG_DB_NAME)?sslmode=$(PG_DB_SSLMODE)

# Инструменты
MIGRATE := $(TOOLS_DIR)/migrate

.PHONY: install-tools
install-tools: ## Установить инструменты (migrate CLI)
	@mkdir -p $(TOOLS_DIR)
	@echo "Installing tools..."
	# db: migrations
	@GOBIN=$(PWD)/$(TOOLS_DIR) go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.19.0
	@echo "All tools installed"
