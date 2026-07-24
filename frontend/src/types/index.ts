// Доменные алиасы, выведенные из контракта api/openapi.yaml (через сгенерированный schema.d.ts).
// Ручные типы не дублируют контракт — только переименовывают схемы в удобные имена.
import type { components } from '@/api/schema'

type Schemas = components['schemas']

export type RowStatus = Schemas['RowStatus']
export type DealType = Schemas['DealType']
export type MatchScope = Schemas['MatchScope']
export type SelectionReason = Schemas['SelectionReason']
export type ComponentType = Schemas['ComponentType']

export type Source = Schemas['Source']
export type SourceKey = Schemas['SourceKey']
export type SourcePreview = Schemas['SourcePreview']
export type SourceFacets = Schemas['SourceFacets']
export type ProductFacet = Schemas['ProductFacet']
export type ClientFacet = Schemas['ClientFacet']
export type CreateCalculationRequest = Schemas['CreateCalculationRequest']

export type Calculation = Schemas['Calculation']
export type CalculationStatus = Schemas['CalculationStatus']
export type CalculationProgressEvent = Schemas['CalculationProgressEvent']
export type Kpi = Schemas['Kpi']

export type CalculationRow = Schemas['CalculationRow']
export type RowsPage = Schemas['RowsPage']
export type RowDetails = Schemas['RowDetails']
export type FormulaComponent = Schemas['FormulaComponent']
export type AppliedFormula = Schemas['AppliedFormula']
export type AlternativeFormula = Schemas['AlternativeFormula']
export type CurrencyConversion = Schemas['CurrencyConversion']

export type ParsedFormula = Schemas['ParsedFormula']

export type ConsolidatedDocument = Schemas['ConsolidatedDocument']
export type ConsolidatedPart = Schemas['ConsolidatedPart']
export type ConsolidatedRow = Schemas['ConsolidatedRow']
export type PartStatus = Schemas['PartStatus']

// Параметры запуска расчёта: диапазон месяцев (обе границы опциональны — весь
// горизонт) и фильтры продукта/клиента (пустые = все).
export interface CalcParams {
  periodFrom?: string
  periodTo?: string
  productIds?: number[]
  clientIds?: string[]
}

// Параметры запроса строк результата.
export interface RowsQuery {
  status?: RowStatus
  query?: string
  sort?: 'row_id' | 'client' | 'material' | 'volume' | 'price'
  order?: 'asc' | 'desc'
  limit?: number
  offset?: number
  onlyFormulaErrors?: boolean
}
