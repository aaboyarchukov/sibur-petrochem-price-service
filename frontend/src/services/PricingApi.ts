// Контракт доступа к данным. Экраны и сторы зависят ТОЛЬКО от этого интерфейса.
// Реализации: MockPricingApi (детерминированные данные), HttpPricingApi (реальный backend).
import type {
  Source,
  SourcePreview,
  SourceFacets,
  Calculation,
  CalculationProgressEvent,
  Kpi,
  RowsPage,
  RowDetails,
  ConsolidatedDocument,
  ConsolidatedPart,
  RowsQuery,
  CalcParams,
  ParsedFormula,
} from '@/types'

// Событие прогресса + функция отписки от потока.
export type ProgressHandler = (event: CalculationProgressEvent) => void
export type Unsubscribe = () => void

export interface PricingApi {
  // Источники данных.
  listSources(): Promise<Source[]>
  loadDemoData(): Promise<Source[]>
  // Загрузка пользовательского .xlsx (ssp | formulas): замещает данные источника.
  uploadSource(key: string, file: File): Promise<Source>
  // Сброс в начальное состояние: отметки загрузки сняты, расчёты сессии удалены.
  resetSources(): Promise<Source[]>
  previewSource(key: string, limit?: number): Promise<SourcePreview>
  // Значения для пикеров экрана параметров: продукты, клиенты, границы горизонта.
  getSourceFacets(): Promise<SourceFacets>

  // Присутствие: поток числа аналитиков онлайн (SSE в http, константа в mock).
  streamPresence(onCount: (analystsOnline: number) => void): Unsubscribe

  // Расчёт с параметрами отбора (диапазон месяцев, продукты, клиенты).
  createCalculation(params: CalcParams): Promise<Calculation>
  getCalculation(calculationId: string): Promise<Calculation>
  // Поток прогресса (SSE в http, таймер в mock). Возвращает функцию отписки.
  streamProgress(calculationId: string, onTick: ProgressHandler): Unsubscribe
  getKpi(calculationId: string): Promise<Kpi>

  // Строки результата.
  listRows(calculationId: string, query?: RowsQuery): Promise<RowsPage>
  getRowDetails(calculationId: string, rowId: string): Promise<RowDetails>
  setManualPrice(calculationId: string, rowId: string, price: number): Promise<RowDetails>
  resetManualPrice(calculationId: string, rowId: string): Promise<RowDetails>
  selectFormula(calculationId: string, rowId: string, formulaId: string): Promise<RowDetails>

  // Выгрузка файлом (mock: CSV через Blob; http: .xlsx с сервера).
  exportCalculation(calculationId: string): Promise<void>

  // Формулы.
  parseFormula(formulaText: string): Promise<ParsedFormula>

  // Сводный документ.
  submitPart(calculationId: string): Promise<ConsolidatedPart>
  getConsolidated(period: string): Promise<ConsolidatedDocument>
  exportConsolidated(period: string): Promise<void>
}
