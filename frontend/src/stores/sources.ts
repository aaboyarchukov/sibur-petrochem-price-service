import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Source, SourcePreview } from '@/types'
import { usePricingApi } from '@/services/provide'

export const useSourcesStore = defineStore('sources', () => {
  const api = usePricingApi()

  const sources = ref<Source[]>([])
  const preview = ref<SourcePreview | null>(null)
  const loaded = ref(false)

  const uploaded = computed(() => sources.value.filter((s) => s.kind === 'uploaded'))
  const references = computed(() => sources.value.filter((s) => s.kind === 'reference'))

  async function loadDemo(): Promise<void> {
    sources.value = await api.loadDemoData()
    preview.value = await api.previewSource('ssp', 5)
    loaded.value = true
  }

  return { sources, preview, loaded, uploaded, references, loadDemo }
})
