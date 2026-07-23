// Форматирование чисел и дат в ru-локали.

export function formatNumber(value: number | null | undefined, digits = 2): string {
  if (value == null) return '—'
  return value.toLocaleString('ru-RU', {
    minimumFractionDigits: digits,
    maximumFractionDigits: digits,
  })
}

export function formatInt(value: number | null | undefined): string {
  if (value == null) return '—'
  return Math.round(value).toLocaleString('ru-RU')
}

export function formatPrice(value: number | null | undefined, currency: string): string {
  if (value == null) return 'нет цены'
  return `${formatNumber(value)} ${currency}`
}

// YYYY-MM-DD | YYYY-MM -> MM.YYYY
export function formatPeriod(period: string): string {
  const [year, month] = period.split('-')
  if (!year || !month) return period
  return `${month}.${year}`
}
