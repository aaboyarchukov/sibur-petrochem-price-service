import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Source, SourcePreview } from '@/types'
import { usePricingApi } from '@/services/provide'

// Пользовательские источники: без них расчёт запрещён.
const REQUIRED_KEYS = ['ssp', 'formulas'] as const

export const useSourcesStore = defineStore('sources', () => {
  const api = usePricingApi()

  const sources = ref<Source[]>([])
  const preview = ref<SourcePreview | null>(null)
  const fetched = ref(false)
  const uploadError = ref<string | null>(null)
  const uploadingKey = ref<string | null>(null)

  const uploaded = computed(() => sources.value.filter((s) => s.kind === 'uploaded'))
  const references = computed(() => sources.value.filter((s) => s.kind === 'reference'))

  // Готовность к расчёту: ssp и formulas загружены (файлами либо демо-набором).
  const loaded = computed(() =>
    REQUIRED_KEYS.every((key) => sources.value.some((s) => s.key === key && s.uploaded_at)),
  )

  async function refresh(): Promise<void> {
    sources.value = await api.listSources()
    preview.value = await api.previewSource('ssp', 5)
    fetched.value = true
  }

  async function loadDemo(): Promise<void> {
    sources.value = await api.loadDemoData()
    preview.value = await api.previewSource('ssp', 5)
    fetched.value = true
  }

  // Загрузка пользовательского .xlsx; ошибка валидации файла остаётся в uploadError.
  async function upload(key: string, file: File): Promise<void> {
    uploadError.value = null
    uploadingKey.value = key
    try {
      await api.uploadSource(key, file)
      await refresh()
    } catch (e) {
      uploadError.value = e instanceof Error ? e.message : String(e)
    } finally {
      uploadingKey.value = null
    }
  }

  return {
    sources,
    preview,
    fetched,
    loaded,
    uploadError,
    uploadingKey,
    uploaded,
    references,
    refresh,
    loadDemo,
    upload,
  }
})
