# sibur-petrochem-price-service

Сервис для расчёта цен на нефтехимическую продукцию: загрузка источников →
подбор договорной формулы для каждой строки прогноза → подстановка котировок и
курсов (каскад Факт → ОФ → ППР) → цена + расшифровка → ручные правки → выгрузка Excel.

## Стек

- **Backend**: Go, Clean Architecture (domain → service → delivery → repository),
  oapi-codegen strict-mode из контракта, sqlc + pgx, PostgreSQL
- **Frontend**: Vue 3 + TypeScript, Pinia, типы из OpenAPI-контракта
- **Контракт**: `api/openapi.yaml` — единый источник правды
- **Эталон алгоритма**: `pricing_pipeline_fixed.py` — Go-движок сверен с ним 1:1
  (статусы и цены на полном демо-наборе, приёмочный тест)

## Быстрый старт (Docker)

```bash
cp .env.example .env        # задать PG_DB_PASSWORD
make run                    # db + миграции (схема + seed) + backend + frontend
```

Фронтенд: http://localhost:8080 · API: http://localhost:8080/api/v1

Миграции применяет отдельный сервис `migrate` (golang-migrate), приложение
подключается к готовой БД. Демо-данные (`documents/*.csv`) заливаются seed-миграциями.

```bash
make run-fresh              # пересоздать стек со сносом volumes
make stop                   # остановить
```

## Разработка

```bash
make install-tools          # migrate, oapi-codegen, sqlc в ./bin
make gen-all                # кодогенерация: strict-server из контракта + sqlc

# backend
cd backend
go test ./...               # юнит + поведенческие (groat) + приёмка с эталоном
golangci-lint run ./...

# frontend
cd frontend
npm ci
npm run dev                 # vite dev-server, прокси /api → localhost:8080
npm run typecheck && npm run lint && npm run test && npm run build
```

Фронтенд по умолчанию ходит в реальный backend; режим моков без сервера:
`VITE_API_MODE=mock npm run dev`.

## Структура

```
├── api/openapi.yaml        # контракт (16 операций) + конфиг кодогена
├── backend/
│   ├── db/migrations/      # golang-migrate: схема + seed из documents/*.csv
│   ├── db/queries/         # sqlc-запросы
│   └── internal/
│       ├── domain/         # модели + sentinel-ошибки
│       ├── service/pricing # движок расчёта (порт эталона: подбор, каскады, safe-eval)
│       ├── service/calculations # жизненный цикл расчёта, KPI, правки, сводный документ
│       ├── repository/postgres  # чтение источников (sqlc)
│       └── delivery/http   # strict-хендлеры по доменам, SSE, экспорт xlsx
├── frontend/               # Vue SPA (5 экранов сценария)
├── documents/              # демо-данные (8 источников .csv)
└── openspec/               # спецификации и changes
```

## Особенности MVP

- Состояние расчёта живёт в памяти процесса: рестарт backend — новый расчёт.
- Загрузка .xlsx для ssp и formulas выполняет полный ingest: файл валидируется
  (строгая схема колонок) и замещает данные источника в БД; справочники — из seed-миграций.
- Ошибка одной строки не прерывает расчёт: проблемные строки помечаются статусом
  (`component_error`, `invalid_formula`, `no_formula`) с текстом причины.
