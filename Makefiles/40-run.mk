# Запуск сервисов через Docker.

COMPOSE := docker compose

.PHONY: run
run: ## Поднять весь стек (db + migrate + backend + frontend)
	@$(COMPOSE) up -d --build

.PHONY: run-fresh
run-fresh: ## Пересоздать стек, снеся volumes (чистая БД + миграции заново)
	@$(COMPOSE) down -v
	@$(COMPOSE) up -d --build --force-recreate

.PHONY: stop
stop: ## Остановить стек
	@$(COMPOSE) down

.PHONY: logs
logs: ## Логи migrate-сервиса (применение миграций)
	@$(COMPOSE) logs migrate

.PHONY: psql
psql: ## psql в контейнере БД
	@$(COMPOSE) exec db psql -U $(PG_DB_USER) -d $(PG_DB_NAME)
