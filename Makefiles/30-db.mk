# Миграции БД (golang-migrate).

MIGRATIONS_DIR := backend/db/migrations

.PHONY: migrate-create
migrate-create: ## Создать новую миграцию (использование: make migrate-create NAME=название)
	@if [ -z "$(NAME)" ]; then \
		echo "Usage: make migrate-create NAME=<migration_name>"; \
		exit 1; \
	fi
	@mkdir -p $(MIGRATIONS_DIR)
	@$(MIGRATE) create -ext sql -dir $(MIGRATIONS_DIR) -seq $(NAME)
	@echo "Migration created"

.PHONY: migrate-up
migrate-up: ## Применить миграции
	@$(MIGRATE) -path=$(MIGRATIONS_DIR) -database "$(DB_DSN)" up
	@echo "Migrations applied"

.PHONY: migrate-down
migrate-down: ## Откатить последнюю миграцию
	@$(MIGRATE) -path=$(MIGRATIONS_DIR) -database "$(DB_DSN)" down 1
	@echo "Migration rolled back"

.PHONY: migrate-force
migrate-force: ## Проставить версию без запуска (использование: make migrate-force VERSION=1)
	@$(MIGRATE) -path=$(MIGRATIONS_DIR) -database "$(DB_DSN)" force $(VERSION)
