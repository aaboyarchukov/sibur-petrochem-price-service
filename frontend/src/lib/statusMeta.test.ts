import { describe, it, expect } from 'vitest'
import { statusMeta, isPricedStatus } from './statusMeta'
import type { RowStatus } from '@/types'

const ALL_STATUSES: RowStatus[] = [
  'calculated',
  'calculated_expired',
  'formula_conflict',
  'manual',
  'component_error',
  'invalid_formula',
  'no_formula',
  'spot_not_calculated',
]

describe('statusMeta', () => {
  it('возвращает подпись и вид для каждого статуса контракта', () => {
    for (const status of ALL_STATUSES) {
      const meta = statusMeta(status)
      expect(meta.label).toBeTruthy()
      expect(['ok', 'ext', 'multi', 'manual', 'warn', 'err']).toContain(meta.kind)
    }
  })

  it('маппит конфликт формул на вид multi с подписью выбора', () => {
    expect(statusMeta('formula_conflict')).toEqual({ label: 'Выбрать формулу', kind: 'multi' })
  })

  it('статусы без цены помечены как не-priced', () => {
    expect(isPricedStatus('no_formula')).toBe(false)
    expect(isPricedStatus('component_error')).toBe(false)
    expect(isPricedStatus('invalid_formula')).toBe(false)
    expect(isPricedStatus('calculated')).toBe(true)
    expect(isPricedStatus('manual')).toBe(true)
  })
})
