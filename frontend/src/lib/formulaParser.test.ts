import { describe, it, expect } from 'vitest'
import { parseFormulaExpression } from './formulaParser'

describe('parseFormulaExpression', () => {
  it('извлекает переменные и функцию IF', () => {
    const result = parseFormulaExpression(
      'IF ( ( ( CPT_MOSCOW_109 - L ) / H1 ) * D < SPOT , ( ( CPT_MOSCOW_109 - L ) / H1 ) * D , SPOT )',
    )
    expect(result.valid).toBe(true)
    expect(result.functions).toContain('IF')
    expect(result.variables).toEqual(['CPT_MOSCOW_109', 'L', 'H1', 'D', 'SPOT'])
  })

  it('поддерживает переменные с валютными символами и цифрой в начале', () => {
    const result = parseFormulaExpression('( SCI / 1_13 / 1_02 - 1_1 ) * CUR_$_¥_PBC')
    expect(result.valid).toBe(true)
    expect(result.variables).toContain('1_13')
    expect(result.variables).toContain('CUR_$_¥_PBC')
  })

  it('ловит несбалансированные скобки', () => {
    const result = parseFormulaExpression('( A + B ) )')
    expect(result.valid).toBe(false)
    expect(result.errors?.length ?? 0).toBeGreaterThan(0)
  })

  it('не считает числа переменными', () => {
    const result = parseFormulaExpression('A * 30 + 25')
    expect(result.variables).toEqual(['A'])
  })
})
