import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { ProductFacet, ClientFacet, CalcParams } from '@/types'
import { usePricingApi } from '@/services/provide'

// Стор экрана параметров расчёта: режим даты (весь период | диапазон месяцев),
// выбор продуктов и клиентов, загрузка фасетов для пикеров.
export const useParamsStore = defineStore('params', () => {
  const api = usePricingApi()

  const products = ref<ProductFacet[]>([])
  const clients = ref<ClientFacet[]>([])
  const periodMin = ref<string | null>(null)
  const periodMax = ref<string | null>(null)
  const loaded = ref(false)

  // Режим даты: false — весь период, true — диапазон месяцев.
  const rangeMode = ref(false)
  const periodFrom = ref('')
  const periodTo = ref('')

  const selectedProducts = ref<Set<number>>(new Set())
  const selectedClients = ref<Set<string>>(new Set())

  async function loadFacets(): Promise<void> {
    const facets = await api.getSourceFacets()
    products.value = facets.products
    clients.value = facets.clients
    periodMin.value = facets.period_min ?? null
    periodMax.value = facets.period_max ?? null
    periodFrom.value = facets.period_min ?? ''
    periodTo.value = facets.period_max ?? ''
    loaded.value = true
  }

  function toggleProduct(id: number): void {
    const next = new Set(selectedProducts.value)
    if (next.has(id)) next.delete(id)
    else next.add(id)
    selectedProducts.value = next
  }

  function toggleClient(id: string): void {
    const next = new Set(selectedClients.value)
    if (next.has(id)) next.delete(id)
    else next.add(id)
    selectedClients.value = next
  }

  // params — выбранные значения в форму запроса расчёта (пустое = не сужаем).
  const params = computed<CalcParams>(() => ({
    periodFrom: rangeMode.value && periodFrom.value ? periodFrom.value : undefined,
    periodTo: rangeMode.value && periodTo.value ? periodTo.value : undefined,
    productIds: selectedProducts.value.size ? [...selectedProducts.value] : undefined,
    clientIds: selectedClients.value.size ? [...selectedClients.value] : undefined,
  }))

  return {
    products,
    clients,
    periodMin,
    periodMax,
    loaded,
    rangeMode,
    periodFrom,
    periodTo,
    selectedProducts,
    selectedClients,
    loadFacets,
    toggleProduct,
    toggleClient,
    params,
  }
})
