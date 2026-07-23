<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useSourcesStore } from '@/stores/sources'
import { useCalculationStore } from '@/stores/calculation'
import { formatInt } from '@/lib/format'
import BlueprintPanel from '@/components/BlueprintPanel.vue'
import AppButton from '@/components/AppButton.vue'

type UploadKey = 'ssp' | 'formulas'

const router = useRouter()
const sources = useSourcesStore()
const calculation = useCalculationStore()

const uploadZones: { key: UploadKey; name: string; file: string }[] = [
  { key: 'ssp', name: 'Прогноз спроса', file: 'ssp.xlsx' },
  { key: 'formulas', name: 'Каталог формул', file: 'formulas.xlsx' },
]

const fileInput = ref<HTMLInputElement | null>(null)
const pendingKey = ref<UploadKey>('ssp')
const dragKey = ref<UploadKey | null>(null)
const computeError = ref('')

// Кнопка демо-набора активируется флагом сборки (VITE_DEMO_ENABLED=true).
const demoEnabled = import.meta.env.VITE_DEMO_ENABLED === 'true'

function pick(key: UploadKey): void {
  pendingKey.value = key
  fileInput.value?.click()
}

function onFileChange(event: Event): void {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (file) void handleFile(pendingKey.value, file)
  input.value = ''
}

function onDrop(key: UploadKey, event: DragEvent): void {
  dragKey.value = null
  const file = event.dataTransfer?.files?.[0]
  if (file) void handleFile(key, file)
}

async function handleFile(key: UploadKey, file: File): Promise<void> {
  if (!file.name.toLowerCase().endsWith('.xlsx')) {
    sources.uploadError = `«${file.name}» — нужен файл .xlsx`
    return
  }
  await sources.upload(key, file)
}

// Файл, брошенный мимо зоны, не должен открываться браузером вместо страницы.
function preventWindowDrop(event: DragEvent): void {
  event.preventDefault()
}
onMounted(() => {
  window.addEventListener('dragover', preventWindowDrop)
  window.addEventListener('drop', preventWindowDrop)
  // актуальное состояние источников (uploaded_at живёт на backend)
  void sources.refresh().catch(() => undefined)
})
onBeforeUnmount(() => {
  window.removeEventListener('dragover', preventWindowDrop)
  window.removeEventListener('drop', preventWindowDrop)
})

async function loadDemo(): Promise<void> {
  await sources.loadDemo()
}

async function compute(): Promise<void> {
  computeError.value = ''
  try {
    await calculation.start('2026-06')
    await router.push({ name: 'computing' })
  } catch (e) {
    // 409 sources_not_loaded и прочие ошибки запуска
    computeError.value = e instanceof Error ? e.message : String(e)
  }
}
</script>

<template>
  <div class="scroll">
    <div class="wrap">
      <header class="head">
        <div>
          <div class="mono eyebrow">Исходные данные</div>
          <h2>Загрузка источников</h2>
        </div>
        <AppButton v-if="sources.loaded" variant="action" @click="compute">
          Рассчитать {{ formatInt(sources.preview?.total_rows ?? 0) }} строк →
        </AppButton>
      </header>

      <p class="text-muted lead">
        Загрузите прогноз спроса <span class="mono">ssp.xlsx</span> и каталог формул
        <span class="mono">formulas.xlsx</span> — файл замещает данные источника.
        Остальные справочники подтягиваются сервисом.
      </p>

      <!-- Зоны загрузки пользовательских .xlsx -->
      <input ref="fileInput" type="file" accept=".xlsx" class="file-input" @change="onFileChange" />
      <div class="upload-grid">
        <div
          v-for="zone in uploadZones"
          :key="zone.key"
          class="drop-zone"
          :class="{ drag: dragKey === zone.key, busy: sources.uploadingKey === zone.key }"
          @click="pick(zone.key)"
          @dragover.prevent="dragKey = zone.key"
          @dragleave="dragKey = null"
          @drop.prevent="onDrop(zone.key, $event)"
        >
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M12 15V3m0 0-4 4m4-4 4 4M4 17v2a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-2" />
          </svg>
          <div class="drop-info">
            <div class="drop-name">{{ zone.name }}</div>
            <div class="mono text-muted drop-hint">
              {{ sources.uploadingKey === zone.key ? 'загрузка…' : zone.file + ' — перетащите или кликните' }}
            </div>
          </div>
        </div>
      </div>
      <p v-if="sources.uploadError" class="mono upload-error">{{ sources.uploadError }}</p>
      <p v-if="computeError" class="mono upload-error">{{ computeError }}</p>

      <!-- Статус пользовательских источников -->
      <div v-if="sources.fetched" class="src-grid">
        <BlueprintPanel v-for="s in sources.uploaded" :key="s.key">
          <div class="src-card">
            <div class="src-icon">▦</div>
            <div class="src-info">
              <div class="src-name">{{ s.name }}</div>
              <div class="mono text-muted src-meta">
                {{ s.file_name }} · {{ formatInt(s.row_count) }} строк
              </div>
            </div>
            <span v-if="s.uploaded_at" class="mono chip-ok">загружен</span>
            <span v-else class="mono chip-pending">не загружен</span>
          </div>
        </BlueprintPanel>
      </div>

      <!-- Пустое состояние: расчёт запрещён, пока не загружены оба файла -->
      <BlueprintPanel v-if="!sources.loaded" class="empty-panel">
        <div class="empty">
          <div class="upload-icon">
            <svg width="26" height="26" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <path d="M12 15V3m0 0-4 4m4-4 4 4M4 17v2a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-2" />
            </svg>
          </div>
          <div class="empty-title">Данные не загружены</div>
          <p class="text-muted empty-sub">
            Загрузите ssp.xlsx и formulas.xlsx выше — без них расчёт недоступен.
          </p>
          <AppButton v-if="demoEnabled" variant="action" @click="loadDemo">
            Загрузить демо-набор
          </AppButton>
        </div>
      </BlueprintPanel>

      <!-- Загруженное состояние -->
      <template v-else>
        <div class="mono eyebrow ref-title">Справочники сервиса · подтягиваются автоматически</div>
        <BlueprintPanel class="ref-panel">
          <div class="ref-grid">
            <div v-for="r in sources.references" :key="r.key" class="ref-item">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="var(--st-ok)" stroke-width="1.8">
                <path d="M20 6 9 17l-5-5" />
              </svg>
              <span class="ref-name">{{ r.name }}</span>
              <span class="mono text-muted">{{ formatInt(r.row_count) }}</span>
            </div>
          </div>
        </BlueprintPanel>

        <div class="preview-head">
          <h4>Превью — ssp.csv</h4>
          <span class="mono text-muted">первые {{ sources.preview?.rows.length }} строк</span>
        </div>
        <BlueprintPanel class="preview-panel">
          <table class="table mono preview-table">
            <thead>
              <tr>
                <th>row_id</th><th>период</th><th>клиент</th><th>материал</th>
                <th class="num">объём, т</th><th>сделка</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(row, i) in sources.preview?.rows ?? []" :key="i">
                <td>{{ row[0] }}</td>
                <td>{{ row[1] }}</td>
                <td>{{ row[2] }}</td>
                <td>{{ row[3] }}</td>
                <td class="num">{{ row[4] }}</td>
                <td>{{ row[5] }}</td>
              </tr>
            </tbody>
          </table>
        </BlueprintPanel>
      </template>
    </div>
  </div>
</template>

<style scoped>
.scroll {
  height: 100%;
  overflow: auto;
}
.wrap {
  max-width: 1080px;
  margin: 0 auto;
  padding: 28px 24px 64px;
}
.head {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 10px;
}
.eyebrow {
  font-size: 11px;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: var(--color-accent);
  margin-bottom: 4px;
}
h2 {
  font-size: 30px;
}
.lead {
  max-width: 640px;
  font-size: 14px;
  margin-bottom: 22px;
}
.file-input {
  display: none;
}
.upload-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 10px;
  margin-bottom: 12px;
}
.drop-zone {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 16px;
  border: 2px dashed var(--color-divider);
  color: var(--color-accent);
  cursor: pointer;
  transition: border-color 0.15s, background 0.15s;
}
.drop-zone:hover,
.drop-zone.drag {
  border-color: var(--color-accent);
}
.drop-zone.busy {
  opacity: 0.6;
  pointer-events: none;
}
.drop-info {
  min-width: 0;
}
.drop-name {
  font-family: var(--font-heading);
  font-weight: 600;
  font-size: 14px;
  color: var(--color-text);
}
.drop-hint {
  font-size: 11px;
}
.upload-error {
  font-size: 12px;
  color: var(--st-err, #c0392b);
  margin-bottom: 16px;
  white-space: pre-wrap;
}
.empty {
  display: grid;
  place-items: center;
  text-align: center;
  padding: 52px 24px;
  border: 2px dashed var(--color-divider);
  margin: 14px;
}
.upload-icon {
  width: 52px;
  height: 52px;
  border: 1.5px solid var(--color-accent);
  color: var(--color-accent);
  display: grid;
  place-items: center;
  margin-bottom: 14px;
}
.empty-title {
  font-family: var(--font-heading);
  font-weight: 600;
  font-size: 20px;
  margin-bottom: 4px;
}
.empty-sub {
  font-size: 13px;
  margin-bottom: 18px;
  max-width: 440px;
}
.src-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 10px;
  margin-bottom: 24px;
}
.src-card {
  padding: 13px 15px;
  display: flex;
  align-items: center;
  gap: 13px;
}
.src-icon {
  width: 34px;
  height: 34px;
  flex: none;
  display: grid;
  place-items: center;
  color: var(--color-accent);
  border: 1px solid var(--color-divider);
}
.src-info {
  flex: 1;
  min-width: 0;
}
.src-name {
  font-family: var(--font-heading);
  font-weight: 600;
  font-size: 15px;
}
.src-meta {
  font-size: 11px;
}
.chip-ok {
  font-size: 11px;
  padding: 3px 9px;
  color: var(--st-ok);
  background: var(--st-ok-bg);
}
.chip-pending {
  font-size: 11px;
  padding: 3px 9px;
  color: var(--st-warn, #b8860b);
  border: 1px dashed var(--color-divider);
}
.ref-title {
  margin: 2px 0 8px;
}
.ref-panel {
  padding: 14px 16px;
  margin-bottom: 24px;
}
.ref-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 6px 22px;
}
.ref-item {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
}
.ref-name {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.preview-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}
.preview-head h4 {
  font-size: 17px;
}
.preview-panel {
  overflow: hidden;
}
.preview-table {
  font-size: 13px;
}
.num {
  text-align: right;
}
</style>
