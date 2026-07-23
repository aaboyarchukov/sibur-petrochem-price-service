// Клиентская генерация CSV для выгрузки на моках (fallback вместо серверного .xlsx).
import type { CalculationRow } from '@/types'

const HEADERS = [
  'row_id',
  'period',
  'client_id',
  'client_name',
  'material_id',
  'material_name',
  'deal_type',
  'currency',
  'volume',
  'status',
  'final_price',
]

function escapeCell(value: string | number | null | undefined): string {
  const text = value == null ? '' : String(value)
  if (/[",\n;]/.test(text)) return `"${text.replace(/"/g, '""')}"`
  return text
}

export function rowsToCsv(rows: CalculationRow[]): string {
  const lines = [HEADERS.join(';')]
  for (const r of rows) {
    lines.push(
      [
        r.row_id,
        r.period,
        r.client_id,
        r.client_name,
        r.material_id,
        r.material_name,
        r.deal_type,
        r.currency,
        r.volume,
        r.status,
        r.final_price,
      ]
        .map(escapeCell)
        .join(';'),
    )
  }
  return lines.join('\n')
}

export function downloadCsv(fileName: string, content: string): void {
  // BOM, чтобы Excel корректно открыл кириллицу в UTF-8.
  const blob = new Blob(['﻿', content], { type: 'text/csv;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = fileName
  link.click()
  URL.revokeObjectURL(url)
}
