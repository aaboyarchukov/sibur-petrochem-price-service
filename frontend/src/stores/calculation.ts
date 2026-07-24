import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { CalculationRow, RowDetails, Kpi, RowStatus, CalculationProgressEvent } from '@/types'
import type { RowsQuery, CalcParams } from '@/types'
import { usePricingApi } from '@/services/provide'

export const useCalculationStore = defineStore('calculation', () => {
  const api = usePricingApi()

  const calculationId = ref<string | null>(null)
  const rows = ref<CalculationRow[]>([])
  const statusCounts = ref<Record<string, number>>({})
  const kpi = ref<Kpi | null>(null)
  const total = ref(0)

  const query = ref('')
  const statusFilter = ref<RowStatus | null>(null)
  // Фильтр «ошибка формулы» — взаимоисключающий со statusFilter.
  const onlyFormulaErrors = ref(false)
  const sort = ref<NonNullable<RowsQuery['sort']>>('row_id')
  const order = ref<'asc' | 'desc'>('asc')

  const selectedRowId = ref<string | null>(null)
  const details = ref<RowDetails | null>(null)

  const progress = ref<CalculationProgressEvent | null>(null)
  const submitted = ref(false)

  const hasSelection = computed(() => details.value != null)

  async function start(params: CalcParams): Promise<void> {
    const calc = await api.createCalculation(params)
    calculationId.value = calc.id
  }

  // Подписка на SSE-поток прогресса; onDone вызывается по завершении.
  function streamProgress(onDone: () => void): () => void {
    if (!calculationId.value) return () => {}
    return api.streamProgress(calculationId.value, (event) => {
      progress.value = event
      if (event.status === 'done') onDone()
    })
  }

  async function refresh(): Promise<void> {
    if (!calculationId.value) return
    const page = await api.listRows(calculationId.value, {
      query: query.value || undefined,
      status: statusFilter.value ?? undefined,
      onlyFormulaErrors: onlyFormulaErrors.value || undefined,
      sort: sort.value,
      order: order.value,
    })
    rows.value = page.items
    total.value = page.total
    statusCounts.value = page.status_counts
    kpi.value = await api.getKpi(calculationId.value)
  }

  async function select(rowId: string): Promise<void> {
    if (!calculationId.value) return
    if (selectedRowId.value === rowId) {
      selectedRowId.value = null
      details.value = null
      return
    }
    selectedRowId.value = rowId
    details.value = await api.getRowDetails(calculationId.value, rowId)
  }

  function closeDetails(): void {
    selectedRowId.value = null
    details.value = null
  }

  function setSort(key: NonNullable<RowsQuery['sort']>): void {
    if (sort.value === key) order.value = order.value === 'asc' ? 'desc' : 'asc'
    else {
      sort.value = key
      order.value = 'asc'
    }
    void refresh()
  }

  async function applyMutation(next: Promise<RowDetails>): Promise<void> {
    details.value = await next
    await refresh()
  }

  async function setManualPrice(rowId: string, price: number): Promise<void> {
    if (!calculationId.value) return
    await applyMutation(api.setManualPrice(calculationId.value, rowId, price))
  }

  async function resetManualPrice(rowId: string): Promise<void> {
    if (!calculationId.value) return
    await applyMutation(api.resetManualPrice(calculationId.value, rowId))
  }

  async function selectFormula(rowId: string, formulaId: string): Promise<void> {
    if (!calculationId.value) return
    await applyMutation(api.selectFormula(calculationId.value, rowId, formulaId))
  }

  async function exportExcel(): Promise<void> {
    if (!calculationId.value) return
    await api.exportCalculation(calculationId.value)
  }

  return {
    calculationId,
    rows,
    statusCounts,
    kpi,
    total,
    query,
    statusFilter,
    onlyFormulaErrors,
    sort,
    order,
    selectedRowId,
    details,
    progress,
    submitted,
    hasSelection,
    start,
    streamProgress,
    refresh,
    select,
    closeDetails,
    setSort,
    setManualPrice,
    resetManualPrice,
    selectFormula,
    exportExcel,
  }
})
