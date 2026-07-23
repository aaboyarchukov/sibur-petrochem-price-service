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

    const after = await api.getKpi(CALC)
    expect(after.priced_rows).toBe(before.priced_rows + 1)
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
