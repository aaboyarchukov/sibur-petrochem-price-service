import { describe, it, expect, beforeEach } from 'vitest'
import { MockPricingApi } from './MockPricingApi'

const CALC = 'calc-mine'

describe('MockPricingApi', () => {
  let api: MockPricingApi

  beforeEach(() => {
    api = new MockPricingApi()
  })

  it('датасет содержит строки каждого статуса', async () => {
    const page = await api.listRows(CALC)
    const counts = page.status_counts
    for (const status of [
      'calculated',
      'formula_conflict',
      'component_error',
      'no_formula',
      'spot_not_calculated',
      'calculated_expired',
    ]) {
      expect(counts[status] ?? 0).toBeGreaterThan(0)
    }
  })

  it('детерминирован между инстансами', async () => {
    const a = await api.listRows(CALC)
    const b = await new MockPricingApi().listRows(CALC)
    expect(a.items.map((r) => r.row_id)).toEqual(b.items.map((r) => r.row_id))
    expect(a.items.map((r) => r.final_price)).toEqual(b.items.map((r) => r.final_price))
  })

  it('ручная цена меняет статус на manual и пересчитывает KPI', async () => {
    const before = await api.getKpi(CALC)
    const target = (await api.listRows(CALC)).items.find((r) => r.status === 'no_formula')!
    expect(target.final_price).toBeNull()

    const details = await api.setManualPrice(CALC, target.row_id, 100000)
    expect(details.row.status).toBe('manual')
    expect(details.row.final_price).toBe(100000)

    // Новая цена входит в контрольную сумму.
    const after = await api.getKpi(CALC)
    expect(after.control_sum_mln).toBeGreaterThan(before.control_sum_mln)
  })

  it('KPI — пять канонических показателей без метрики ±3%', async () => {
    const kpi = await api.getKpi(CALC)
    expect(Object.keys(kpi).sort()).toEqual([
      'calc_error_rows',
      'control_sum_mln',
      'formula_coverage_pct',
      'formulas_ok_pct',
      'unclassified_error_pct',
    ])
    expect('matched_pct' in kpi).toBe(false)
  })

  it('покрытие формулами считается только по строкам Formula (SPOT исключён)', async () => {
    const rows = (await api.listRows(CALC)).items
    const formulaRows = rows.filter((r) => r.deal_type === 'Formula')
    const covered = formulaRows.filter((r) => (r.candidate_count ?? 0) > 0)
    const expected = Math.round((covered.length / formulaRows.length) * 100)

    const kpi = await api.getKpi(CALC)
    expect(kpi.formula_coverage_pct).toBe(expected)
  })

  it('контрольная сумма = Σ price × forecast × курс_к_RUB / 1e6', async () => {
    const rubRates: Record<string, number> = { RUB: 1, USD: 89.42 }
    const rows = (await api.listRows(CALC)).items
    const expected =
      rows.reduce((sum, r) => {
        if (r.final_price == null || r.volume == null) return sum
        return sum + r.final_price * r.volume * (rubRates[r.currency] ?? 1)
      }, 0) / 1_000_000

    const kpi = await api.getKpi(CALC)
    expect(kpi.control_sum_mln).toBeCloseTo(expected, 6)
    expect(kpi.control_sum_mln).toBeGreaterThan(0)
  })

  it('строки с ошибкой расчёта: формула найдена, но цены нет', async () => {
    const rows = (await api.listRows(CALC)).items
    const expected = rows.filter(
      (r) =>
        r.deal_type === 'Formula' &&
        (r.candidate_count ?? 0) > 0 &&
        r.final_price == null &&
        (r.status === 'component_error' || r.status === 'invalid_formula'),
    ).length

    const kpi = await api.getKpi(CALC)
    expect(kpi.calc_error_rows).toBe(expected)
    expect(kpi.calc_error_rows).toBeGreaterThan(0)
    // Все ошибки mock-датасета классифицированы.
    expect(kpi.unclassified_error_pct).toBe(0)
  })

  it('кандидаты формул содержат канонические поля экрана 2', async () => {
    const conflict = (await api.listRows(CALC)).items.find((r) => r.status === 'formula_conflict')!
    const details = await api.getRowDetails(CALC, conflict.row_id)

    for (const a of details.alternatives) {
      expect(a.formula_currency).toBeTruthy()
      expect(a.price_formula_currency).not.toBeUndefined()
      expect(a.status).toBeTruthy()
      expect(a.equal_priority_count).toBe(details.alternatives.length)
      expect('warning' in a).toBe(true)
    }
    const selected = details.alternatives.find((a) => a.is_selected)!
    expect(selected.selection_reason).toBeTruthy()
  })

  it('компонент-котировка содержит quote_name рядом с quote_code', async () => {
    const calc = (await api.listRows(CALC)).items.find((r) => r.status === 'calculated')!
    const details = await api.getRowDetails(CALC, calc.row_id)
    const quote = details.components.find((c) => c.type === 'quote')!
    expect(quote.quote_name).toBeTruthy()
    expect(quote.quote_code).toBeTruthy()
  })

  it('сброс ручной правки возвращает исходный статус', async () => {
    const target = (await api.listRows(CALC)).items.find((r) => r.status === 'no_formula')!
    await api.setManualPrice(CALC, target.row_id, 50000)
    const reset = await api.resetManualPrice(CALC, target.row_id)
    expect(reset.row.status).toBe('no_formula')
    expect(reset.row.final_price).toBeNull()
  })

  it('выбор формулы у конфликта пересчитывает цену и снимает требование выбора', async () => {
    const conflict = (await api.listRows(CALC)).items.find((r) => r.status === 'formula_conflict')!
    const details = await api.getRowDetails(CALC, conflict.row_id)
    const other = details.alternatives.find((a) => !a.is_selected)!

    const updated = await api.selectFormula(CALC, conflict.row_id, other.formula_id)
    expect(updated.row.status).toBe('calculated')
    expect(updated.row.final_price).toBe(other.price)
    expect(updated.applied_formula?.formula_id).toBe(other.formula_id)
  })

  it('фильтр по статусу и поиск сужают выборку', async () => {
    const onlyErrors = await api.listRows(CALC, { status: 'component_error' })
    expect(onlyErrors.items.every((r) => r.status === 'component_error')).toBe(true)

    const all = await api.listRows(CALC)
    const sample = all.items[0]!
    const search = await api.listRows(CALC, { query: sample.client_name })
    expect(search.items.length).toBeGreaterThan(0)
    expect(search.items.every((r) => r.client_name.includes(sample.client_name))).toBe(true)
  })
})
