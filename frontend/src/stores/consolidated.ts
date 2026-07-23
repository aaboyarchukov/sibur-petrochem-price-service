import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { ConsolidatedDocument } from '@/types'
import { usePricingApi } from '@/services/provide'

const PERIOD = '2026-06'

export const useConsolidatedStore = defineStore('consolidated', () => {
  const api = usePricingApi()

  const document = ref<ConsolidatedDocument | null>(null)

  async function load(): Promise<void> {
    document.value = await api.getConsolidated(PERIOD)
  }

  async function submitMyPart(): Promise<void> {
    await api.submitPart('calc-mine')
    await load()
  }

  async function exportExcel(): Promise<void> {
    await api.exportConsolidated(PERIOD)
  }

  return { document, load, submitMyPart, exportExcel }
})
