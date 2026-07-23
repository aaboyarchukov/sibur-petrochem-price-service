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
OAPI_CODEGEN := $(TOOLS_DIR)/oapi-codegen
SQLC := $(TOOLS_DIR)/sqlc

.PHONY: install-tools
install-tools: ## Установить инструменты (migrate, oapi-codegen, sqlc)
	@mkdir -p $(TOOLS_DIR)
	@echo "Installing tools..."
	# db: migrations
	@GOBIN=$(PWD)/$(TOOLS_DIR) go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.19.0
	# api: codegen из OpenAPI контракта
	@GOBIN=$(PWD)/$(TOOLS_DIR) go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.5.0
	# db: sqlc
	@GOBIN=$(PWD)/$(TOOLS_DIR) go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0
	@echo "All tools installed"

.PHONY: gen-all
gen-all: api-generate sqlc-gen ## Сгенерировать весь код (api + sqlc)
