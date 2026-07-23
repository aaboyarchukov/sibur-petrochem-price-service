# Главный Makefile — подключает модульные makefiles.

.DEFAULT_GOAL := help

include Makefiles/00-common.mk
include Makefiles/30-db.mk
include Makefiles/40-run.mk

.PHONY: help
help:
	@echo ""
	@echo "Доступные команды:"
	@echo ""
	@echo "=== Инструменты ==="
	@echo "  make install-tools            - установить migrate CLI в ./bin"
	@echo ""
	@echo "=== База данных ==="
	@echo "  make migrate-create NAME=...  - создать новую миграцию"
	@echo "  make migrate-up               - применить миграции"
	@echo "  make migrate-down             - откатить последнюю миграцию"
	@echo "  make migrate-force VERSION=N  - проставить версию без запуска"
	@echo ""
	@echo "=== Запуск ==="
	@echo "  make run                      - поднять весь стек (db + migrate + backend + frontend)"
	@echo "  make run-fresh                - пересоздать стек со сносом volumes"
	@echo "  make stop                     - остановить стек"
	@echo "  make logs                     - логи migrate-сервиса"
	@echo "  make psql                     - psql в контейнере БД"
	@echo ""
