// Mock-реализация PricingApi: детерминированный датасет + мутационное состояние в памяти.
import type { PricingApi, ProgressHandler, Unsubscribe } from '@/services/PricingApi'
import type {
  Source,
  SourcePreview,
  Calculation,
  Kpi,
  RowsPage,
  RowDetails,
  CalculationRow,
  ConsolidatedDocument,
  ConsolidatedPart,
  ConsolidatedRow,
  RowsQuery,
  RowStatus,
  ParsedFormula,
  AlternativeFormula,
} from '@/types'
import { downloadCsv, rowsToCsv } from '@/lib/csv'
import { parseFormulaExpression } from '@/lib/formulaParser'
import {
  buildRows,
  buildSources,
  OTHER_PARTS,
  CALCULATION_PERIOD,
  MY_ANALYST,
  type RowRecord,
  type FormulaCandidate,
} from './dataset'

const CALC_ID = 'calc-mine'
const TOLERANCE = 0.03
const PROGRESS_STEP_MS = 130

export class MockPricingApi implements PricingApi {
  private rows: RowRecord[] = buildRows()
  private sources: Source[] = buildSources()
  private manual = new Map<string, number>()
  private selected = new Map<string, string>()
  private submitted = false

  // ── источники ──
  async listSources(): Promise<Source[]> {
    return structuredClone(this.sources)
  }

  async loadDemoData(): Promise<Source[]> {
    return structuredClone(this.sources)
  }

  async previewSource(_key: string, limit = 5): Promise<SourcePreview> {
    const columns = ['row_id', 'period', 'client_name', 'material_name', 'forecast', 'deal_type']
    const rows = this.rows.slice(0, limit).map((r) => [
      r.row_id,
      r.period,
      r.client_name,
      r.material_name,
      r.volume == null ? null : String(r.volume),
      r.deal_type,
    ])
    return { columns, rows, total_rows: this.rows.length }
  }

  // ── расчёт ──
  async createCalculation(): Promise<Calculation> {
    return this.calculation('running', 0)
  }

  async getCalculation(): Promise<Calculation> {
    return this.calculation('done', 100)
  }

  streamProgress(_calculationId: string, onTick: ProgressHandler): Unsubscribe {
    const total = this.rows.length
    let percent = 0
    const timer = setInterval(() => {
      percent = Math.min(100, percent + 8 + Math.round(Math.random() * 10))
      const processed = Math.round((total * percent) / 100)
      const done = percent >= 100
      onTick({ status: done ? 'done' : 'running', processed_rows: processed, total_rows: total, percent })
      if (done) clearInterval(timer)
    }, PROGRESS_STEP_MS)
    return () => clearInterval(timer)
  }

  async getKpi(_calculationId?: string): Promise<Kpi> {
    return this.computeKpi(this.rows)
  }

  // ── строки ──
  async listRows(_calculationId: string, query: RowsQuery = {}): Promise<RowsPage> {
    let list = this.rows.map((r) => this.toRow(r))

    const statusCounts: Record<string, number> = {}
    for (const row of list) statusCounts[row.status] = (statusCounts[row.status] ?? 0) + 1

    if (query.query) {
      const q = query.query.toLowerCase()
      list = list.filter((r) =>
        `${r.client_name} ${r.material_name} ${r.row_id}`.toLowerCase().includes(q),
      )
    }
    if (query.status) list = list.filter((r) => r.status === query.status)

    if (query.sort) {
      const dir = query.order === 'desc' ? -1 : 1
      list.sort((a, b) => this.compareRows(a, b, query.sort!) * dir)
    }

    const total = list.length
    const offset = query.offset ?? 0
    const limit = query.limit ?? total
    return { items: list.slice(offset, offset + limit), total, status_counts: statusCounts }
  }

  async getRowDetails(_calculationId: string, rowId: string): Promise<RowDetails> {
    return this.toDetails(this.find(rowId))
  }

  async setManualPrice(_c: string, rowId: string, price: number): Promise<RowDetails> {
    this.manual.set(rowId, price)
    return this.toDetails(this.find(rowId))
  }

  async resetManualPrice(_c: string, rowId: string): Promise<RowDetails> {
    this.manual.delete(rowId)
    return this.toDetails(this.find(rowId))
  }

  async selectFormula(_c: string, rowId: string, formulaId: string): Promise<RowDetails> {
    this.selected.set(rowId, formulaId)
    return this.toDetails(this.find(rowId))
  }

  async exportCalculation(): Promise<void> {
    const rows = this.rows.map((r) => this.toRow(r))
    downloadCsv(`prices_${CALCULATION_PERIOD}.csv`, rowsToCsv(rows))
  }

  async parseFormula(formulaText: string): Promise<ParsedFormula> {
    return parseFormulaExpression(formulaText)
  }

  // ── сводный документ ──
  async submitPart(): Promise<ConsolidatedPart> {
    this.submitted = true
    return this.myPart()
  }

  async getConsolidated(period: string): Promise<ConsolidatedDocument> {
    const parts: ConsolidatedPart[] = [this.myPart(), ...OTHER_PARTS]
    const mineRows: ConsolidatedRow[] = this.rows.map((r) => ({
      analyst_name: MY_ANALYST,
      is_draft: !this.submitted,
      row: this.toRow(r),
    }))
    const kpi = this.computeKpi(this.rows)
    return { period, parts, kpi, rows: mineRows, total_rows: mineRows.length }
  }

  async exportConsolidated(period: string): Promise<void> {
    const rows = this.rows.map((r) => this.toRow(r))
    downloadCsv(`consolidated_${period}.csv`, rowsToCsv(rows))
  }

  // ── внутреннее ──
  private find(rowId: string): RowRecord {
    const record = this.rows.find((r) => r.row_id === rowId)
    if (!record) throw new Error(`row not found: ${rowId}`)
    return record
  }

  private activeCandidate(r: RowRecord): FormulaCandidate | null {
    if (!r.candidates.length) return null
    const selectedId = this.selected.get(r.row_id) ?? r.default_formula_id
    return r.candidates.find((c) => c.formula_id === selectedId) ?? r.candidates[0]!
  }

  private effStatus(r: RowRecord): RowStatus {
    if (this.manual.has(r.row_id)) return 'manual'
    if (r.base_status === 'formula_conflict' && this.selected.has(r.row_id)) return 'calculated'
    return r.base_status
  }

  private effPrice(r: RowRecord): number | null {
    const manual = this.manual.get(r.row_id)
    if (manual != null) return manual
    const active = this.activeCandidate(r)
    return active ? active.price : null
  }

  private isMatched(r: RowRecord): boolean | null {
    const price = this.effPrice(r)
    if (price == null || r.reference_price == null) return null
    return Math.abs(price - r.reference_price) / r.reference_price <= TOLERANCE
  }

  private toRow(r: RowRecord): CalculationRow {
    const status = this.effStatus(r)
    return {
      row_id: r.row_id,
      period: r.period,
      client_id: r.client_id,
      client_name: r.client_name,
      material_id: r.material_id,
      material_name: r.material_name,
      material_group_m: r.material_group_m,
      deal_type: r.deal_type,
      currency: r.currency,
      volume: r.volume,
      status,
      final_price: this.effPrice(r),
      candidate_count: r.candidates.length,
      requires_review: status === 'formula_conflict',
      warning: r.base_status === 'calculated_expired' ? 'использована просроченная формула, срок продлён' : null,
      error: this.rowError(r),
      matched: this.isMatched(r),
    }
  }

  private rowError(r: RowRecord): string | null {
    const active = this.activeCandidate(r)
    if (active?.calc_error) return active.calc_error
    if (this.effStatus(r) === 'component_error') return 'нет значения компонента на дату периода'
    return null
  }

  private toDetails(r: RowRecord): RowDetails {
    const active = this.activeCandidate(r)
    const manual = this.manual.get(r.row_id) ?? null
    const alternatives: AlternativeFormula[] = r.candidates.map((c) => ({
      formula_id: c.formula_id,
      formula_text: c.formula_text,
      match_scope: c.match_scope,
      valid_from: c.valid_from,
      valid_to: c.valid_to,
      created_on: c.created_on,
      is_actual: c.is_actual,
      price: c.price,
      calc_error: c.calc_error,
      matched:
        c.price != null && r.reference_price != null
          ? Math.abs(c.price - r.reference_price) / r.reference_price <= TOLERANCE
          : null,
      is_selected: active != null && c.formula_id === active.formula_id,
    }))

    return {
      row: this.toRow(r),
      alternatives,
      applied_formula: active
        ? {
            formula_id: active.formula_id,
            formula_text: active.formula_text,
            variables: active.variables,
            formula_currency: active.formula_currency,
            match_scope: active.match_scope,
            valid_from: active.valid_from,
            valid_to: active.valid_to,
            created_on: active.created_on,
            is_actual: active.is_actual,
            is_extended: active.is_extended,
            selection_reason: this.selected.has(r.row_id)
              ? 'user_selected'
              : active.is_extended
                ? 'latest_expired_successful'
                : 'actual_successful',
          }
        : undefined,
      components: active ? active.components : [],
      price_formula_currency: active?.price ?? null,
      conversion:
        active && r.currency !== active.formula_currency
          ? {
              from_currency: active.formula_currency,
              to_currency: r.currency,
              from_rate: 89.42,
              to_rate: 1,
              rate_date: '2026-06-15',
              version_type: 'Факт',
            }
          : undefined,
      equal_priority_count: r.base_status === 'formula_conflict' ? r.candidates.length : 0,
      manual_price: manual,
      reference_price: r.reference_price,
    }
  }

  private computeKpi(records: RowRecord[]): Kpi {
    const priced = records.filter((r) => this.effPrice(r) != null)
    const matched = records.filter((r) => this.isMatched(r) === true)
    const errors = records.filter((r) => {
      const s = this.effStatus(r)
      return s === 'component_error' || s === 'no_formula' || s === 'invalid_formula'
    })
    return {
      total_rows: records.length,
      priced_rows: priced.length,
      priced_pct: records.length ? Math.round((priced.length / records.length) * 100) : 0,
      matched_rows: matched.length,
      matched_pct: priced.length ? Math.round((matched.length / priced.length) * 100) : 0,
      error_rows: errors.length,
    }
  }

  private calculation(status: Calculation['status'], percent: number): Calculation {
    const total = this.rows.length
    return {
      id: CALC_ID,
      period: CALCULATION_PERIOD,
      status,
      progress: {
        processed_rows: Math.round((total * percent) / 100),
        total_rows: total,
        percent,
      },
      created_at: '2026-06-20T09:00:00Z',
      finished_at: status === 'done' ? '2026-06-20T09:00:05Z' : null,
    }
  }

  private myPart(): ConsolidatedPart {
    const priced = this.rows.filter((r) => this.effPrice(r) != null).length
    return {
      calculation_id: CALC_ID,
      analyst_name: MY_ANALYST,
      part_name: 'Ваш участок',
      status: this.submitted ? 'joined' : 'draft',
      row_count: this.rows.length,
      priced_pct: Math.round((priced / this.rows.length) * 100),
      submitted_at: this.submitted ? '2026-06-20T09:05:00Z' : null,
    }
  }

  private compareRows(a: CalculationRow, b: CalculationRow, key: NonNullable<RowsQuery['sort']>): number {
    switch (key) {
      case 'price':
        return (a.final_price ?? -1) - (b.final_price ?? -1)
      case 'volume':
        return (a.volume ?? -1) - (b.volume ?? -1)
      case 'client':
        return a.client_name.localeCompare(b.client_name)
      case 'material':
        return a.material_name.localeCompare(b.material_name)
      default:
        return a.row_id.localeCompare(b.row_id)
    }
  }
}
