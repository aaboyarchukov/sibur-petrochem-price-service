# sibur-petrochem-price — frontend

SPA расчёта цен на нефтехимию. Vue 3 + TypeScript. Работает на **моках за интерфейсами** —
backend не требуется. Переход на реальный API — замена одной реализации.

## Стек

- Vue 3 (Composition API, `<script setup>`), TypeScript strict
- Vite, Pinia, Vue Router, Vitest
- Типы моделей генерируются из контракта `../api/openapi.yaml`

## Запуск

```bash
npm install
npm run dev        # http://localhost:5173
```

## Скрипты

| Скрипт | Действие |
|--------|----------|
| `npm run dev` | dev-сервер |
| `npm run build` | typecheck + production-сборка в `dist/` |
| `npm run typecheck` | `vue-tsc --noEmit` |
| `npm run test` | Vitest |
| `npm run lint` | ESLint |
| `npm run gen:api` | регенерация типов из `../api/openapi.yaml` |

## Структура

```
src/
├── api/schema.d.ts      # сгенерировано из openapi.yaml (не править руками)
├── types/               # доменные алиасы из контракта
├── design/              # CSS-токены (light/dark) + базовые стили
├── components/          # переиспользуемые UI (BlueprintPanel, StatusChip, KpiCard, ...)
├── screens/             # экраны: Upload, Computing, Results, DecodePanel, FormulaModal, Consolidated
├── stores/              # Pinia: sources, calculation, consolidated
├── services/
│   ├── PricingApi.ts    # интерфейс доступа к данным
│   ├── mock/            # MockPricingApi + детерминированный датасет
│   ├── http/            # HttpPricingApi (заготовка на реальный backend)
│   └── provide.ts       # DI: выбор реализации
├── lib/                 # statusMeta, format, csv, formulaParser
└── router/
```

## Переключение mock → http

Единственная точка — [`src/services/provide.ts`](src/services/provide.ts):

```ts
export function createPricingApi(): PricingApi {
  return new MockPricingApi()      // → return new HttpPricingApi()
}
```

Экраны и сторы зависят только от интерфейса `PricingApi` — менять их не нужно.
`HttpPricingApi` уже реализует контракт `/api/v1` (dev-прокси настроен в `vite.config.ts`),
прогресс расчёта — через SSE (`GET /calculations/{id}/events`).

## Данные и статусы

Колонки экранов и статусы строк соответствуют контракту (`RowStatus`) и эталонному
алгоритму `pricing_pipeline_fixed.py`. Полная нормализация: UI оперирует только статусами
контракта, чипы — это вид, статус — это данные (см. `src/lib/statusMeta.ts`).

## Выгрузка

На моках `export` собирает CSV на клиенте и скачивает файл (fallback вместо серверного `.xlsx`).
На реальном backend — бинарный `.xlsx` из `GET /calculations/{id}/export`.
