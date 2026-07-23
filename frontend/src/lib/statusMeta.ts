// Полная нормализация: statusMeta принимает ТОЛЬКО RowStatus контракта.
// Обратного маппинга чипов прототипа нет — статус контракта это данные, чип это вид.
import type { RowStatus } from '@/types'

// Вид чипа (семантический цвет из токенов). Не путать с бизнес-статусом.
type ChipKind = 'ok' | 'ext' | 'multi' | 'manual' | 'warn' | 'err'

export interface StatusMeta {
  label: string
  kind: ChipKind
}

// Тотальный словарь: TS требует запись для каждого значения RowStatus.
const META: Record<RowStatus, StatusMeta> = {
  calculated: { label: 'Посчитано', kind: 'ok' },
  calculated_expired: { label: 'Продлённая формула', kind: 'ext' },
  formula_conflict: { label: 'Выбрать формулу', kind: 'multi' },
  manual: { label: 'Правка вручную', kind: 'manual' },
  component_error: { label: 'Нет котировки', kind: 'err' },
  invalid_formula: { label: 'Ошибка формулы', kind: 'err' },
  no_formula: { label: 'Нет цены', kind: 'warn' },
  spot_not_calculated: { label: 'SPOT — без расчёта', kind: 'multi' },
}

export function statusMeta(status: RowStatus): StatusMeta {
  return META[status]
}

// Статусы, при которых цена отсутствует по смыслу.
export function isPricedStatus(status: RowStatus): boolean {
  return status !== 'no_formula' && status !== 'component_error' && status !== 'invalid_formula'
}
