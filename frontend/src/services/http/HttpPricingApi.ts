// HTTP-реализация PricingApi поверх реального backend (контракт /api/v1).
// Заготовка: компилируется как валидная реализация интерфейса; включается заменой
// провайдера в composition root, без правки экранов.
import type { PricingApi, ProgressHandler, Unsubscribe } from '@/services/PricingApi'
import type {
  Source,
  SourcePreview,
  Calculation,
  Kpi,
  RowsPage,
  RowDetails,
  ConsolidatedDocument,
  ConsolidatedPart,
  RowsQuery,
  ParsedFormula,
} from '@/types'

const BASE = '/api/v1'

async function getJson<T>(path: string): Promise<T> {
  const res = await fetch(`${BASE}${path}`)
  if (!res.ok) throw new Error(`http ${res.status}: ${path}`)
  return (await res.json()) as T
}

async function sendJson<T>(path: string, method: string, body?: unknown): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    method,
    headers: { 'Content-Type': 'application/json' },
    body: body == null ? undefined : JSON.stringify(body),
  })
  if (!res.ok) {
    // ошибки API несут человекочитаемый message — показываем его пользователю
    const err = (await res.json().catch(() => null)) as { message?: string } | null
    throw new Error(err?.message ?? `http ${res.status}: ${path}`)
  }
  return (await res.json()) as T
}

function buildRowsQuery(query: RowsQuery = {}): string {
  const params = new URLSearchParams()
  for (const [key, value] of Object.entries(query)) {
    if (value != null) params.set(key, String(value))
  }
  const qs = params.toString()
  return qs ? `?${qs}` : ''
}

export class HttpPricingApi implements PricingApi {
  listSources(): Promise<Source[]> {
    return getJson<{ items: Source[] }>('/sources').then((r) => r.items)
  }

  loadDemoData(): Promise<Source[]> {
    // Активация демо-набора: seed-данные помечаются загруженными, расчёт разрешается.
    return sendJson<{ items: Source[] }>('/sources/demo', 'POST').then((r) => r.items)
  }

  async uploadSource(key: string, file: File): Promise<Source> {
    const form = new FormData()
    form.append('file', file, file.name)
    const res = await fetch(`${BASE}/sources/${key}/file`, { method: 'POST', body: form })
    if (!res.ok) {
      // 400 несёт человекочитаемый message со списком проблем валидации файла
      const err = (await res.json().catch(() => null)) as { message?: string } | null
      throw new Error(err?.message ?? `http ${res.status}: /sources/${key}/file`)
    }
    return (await res.json()) as Source
  }

  previewSource(key: string, limit = 5): Promise<SourcePreview> {
    return getJson<SourcePreview>(`/sources/${key}/preview?limit=${limit}`)
  }

  streamPresence(onCount: (analystsOnline: number) => void): Unsubscribe {
    const source = new EventSource(`${BASE}/presence/events`)
    source.onmessage = (event) => {
      const tick = JSON.parse(event.data) as { analysts_online: number }
      onCount(tick.analysts_online)
    }
    return () => source.close()
  }

  createCalculation(period: string): Promise<Calculation> {
    return sendJson<Calculation>('/calculations', 'POST', { period })
  }

  getCalculation(calculationId: string): Promise<Calculation> {
    return getJson<Calculation>(`/calculations/${calculationId}`)
  }

  streamProgress(calculationId: string, onTick: ProgressHandler): Unsubscribe {
    const source = new EventSource(`${BASE}/calculations/${calculationId}/events`)
    source.onmessage = (event) => {
      const tick = JSON.parse(event.data) as Parameters<ProgressHandler>[0]
      // Закрываем до onTick: иначе EventSource переподключится после завершения потока.
      if (tick.status === 'done' || tick.status === 'failed') source.close()
      onTick(tick)
    }
    return () => source.close()
  }

  getKpi(calculationId: string): Promise<Kpi> {
    return getJson<Kpi>(`/calculations/${calculationId}/kpi`)
  }

  listRows(calculationId: string, query?: RowsQuery): Promise<RowsPage> {
    return getJson<RowsPage>(`/calculations/${calculationId}/rows${buildRowsQuery(query)}`)
  }

  getRowDetails(calculationId: string, rowId: string): Promise<RowDetails> {
    return getJson<RowDetails>(`/calculations/${calculationId}/rows/${rowId}`)
  }

  setManualPrice(calculationId: string, rowId: string, price: number): Promise<RowDetails> {
    return sendJson<RowDetails>(`/calculations/${calculationId}/rows/${rowId}/manual-price`, 'PUT', {
      price,
    })
  }

  resetManualPrice(calculationId: string, rowId: string): Promise<RowDetails> {
    return sendJson<RowDetails>(`/calculations/${calculationId}/rows/${rowId}/manual-price`, 'DELETE')
  }

  selectFormula(calculationId: string, rowId: string, formulaId: string): Promise<RowDetails> {
    return sendJson<RowDetails>(`/calculations/${calculationId}/rows/${rowId}/formula`, 'PUT', {
      formula_id: formulaId,
    })
  }

  async exportCalculation(calculationId: string): Promise<void> {
    window.open(`${BASE}/calculations/${calculationId}/export`, '_blank')
  }

  parseFormula(formulaText: string): Promise<ParsedFormula> {
    return sendJson<ParsedFormula>('/formulas/parse', 'POST', { formula_text: formulaText })
  }

  submitPart(calculationId: string): Promise<ConsolidatedPart> {
    return sendJson<ConsolidatedPart>(`/calculations/${calculationId}/submission`, 'POST')
  }

  getConsolidated(period: string): Promise<ConsolidatedDocument> {
    return getJson<ConsolidatedDocument>(`/consolidated/${period}`)
  }

  async exportConsolidated(period: string): Promise<void> {
    window.open(`${BASE}/consolidated/${period}/export`, '_blank')
  }
}
