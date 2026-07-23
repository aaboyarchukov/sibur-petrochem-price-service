<script setup lang="ts">
import { onMounted, computed } from 'vue'
import { useConsolidatedStore } from '@/stores/consolidated'
import { useToast } from '@/composables/useToast'
import { formatInt, formatNumber } from '@/lib/format'
import type { PartStatus } from '@/types'
import KpiCard from '@/components/KpiCard.vue'
import StatusChip from '@/components/StatusChip.vue'
import BlueprintPanel from '@/components/BlueprintPanel.vue'
import AppButton from '@/components/AppButton.vue'

const store = useConsolidatedStore()
const { show } = useToast()

onMounted(() => {
  void store.load()
})

const doc = computed(() => store.document)
const myPart = computed(() => doc.value?.parts.find((p) => p.part_name === 'Ваш участок') ?? null)
const isDraft = computed(() => myPart.value?.status === 'draft')

const PART_META: Record<PartStatus, { label: string; kind: string }> = {
  joined: { label: 'Присоединено', kind: 'ok' },
  review: { label: 'На проверке', kind: 'multi' },
  draft: { label: 'Черновик', kind: 'warn' },
}

const kpiCards = computed(() => {
  const k = doc.value?.kpi
  if (!k) return []
  const col = (v: number, g: number, m: number) =>
    v >= g ? 'var(--good)' : v >= m ? 'var(--mid)' : 'var(--bad)'
  return [
    { label: 'Всего строк в документе', value: String(k.total_rows), sub: 'из всех участков', color: 'var(--color-accent)', bar: '100%' },
    { label: 'Строк с ценой', value: `${k.priced_pct}%`, sub: `${k.priced_rows} строк`, color: col(k.priced_pct, 90, 75), bar: `${k.priced_pct}%` },
    { label: 'Совпало ±3%', value: `${k.matched_pct ?? 0}%`, sub: `${k.matched_rows ?? 0} строк`, color: col(k.matched_pct ?? 0, 85, 70), bar: `${k.matched_pct ?? 0}%` },
  ]
})

async function submit(): Promise<void> {
  await store.submitMyPart()
  show('Ваш участок присоединён к сводному документу')
}

async function exportExcel(): Promise<void> {
  await store.exportExcel()
  show('Файл сформирован: consolidated_2026-06.csv')
}
</script>

<template>
  <div class="scroll">
    <div class="wrap">
      <header class="head">
        <div>
          <div class="mono eyebrow">Расчётный период · Июнь 2026</div>
          <h2>Сводный документ за период</h2>
        </div>
        <AppButton variant="action" @click="exportExcel">Выгрузить весь документ</AppButton>
      </header>

      <p class="text-muted lead">
        Документ собирается из участков нескольких аналитиков. Каждый работает со своей частью и
        присоединяет её к сводной таблице за период.
      </p>

      <!-- Баннер черновика -->
      <BlueprintPanel v-if="isDraft" class="draft-banner">
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="var(--st-warn)" stroke-width="1.6">
          <path d="m10.3 3.9-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.7-3l-8-14a2 2 0 0 0-3.4 0z" />
          <path d="M12 9v4m0 4h.01" />
        </svg>
        <div class="draft-text">
          <b>Ваш участок ещё не присоединён.</b> Он показан ниже как черновик и не войдёт в
          итоговую выгрузку, пока вы не сохраните его.
        </div>
        <AppButton variant="action" @click="submit">Присоединить мой участок</AppButton>
      </BlueprintPanel>

      <!-- Карточки участков -->
      <div class="parts">
        <BlueprintPanel v-for="p in doc?.parts ?? []" :key="p.calculation_id" :data-mine="p.part_name === 'Ваш участок'">
          <div class="part">
            <div class="part-top">
              <div class="part-name">
                {{ p.analyst_name }}
                <span v-if="p.part_name === 'Ваш участок'" class="you">(вы)</span>
              </div>
              <span class="mono part-chip" :data-kind="PART_META[p.status].kind">
                {{ PART_META[p.status].label }}
              </span>
            </div>
            <div class="mono text-muted part-group">{{ p.part_name }}</div>
            <div class="part-meta">
              <span class="mono">{{ p.row_count }} строк</span>
              <span class="mono cov" :style="{ color: (p.priced_pct ?? 0) >= 90 ? 'var(--good)' : (p.priced_pct ?? 0) >= 75 ? 'var(--mid)' : 'var(--bad)' }">
                {{ p.priced_pct ?? 0 }}% с ценой
              </span>
            </div>
          </div>
        </BlueprintPanel>
      </div>

      <!-- Сводные KPI -->
      <div class="kpis">
        <KpiCard v-for="(k, i) in kpiCards" :key="i" v-bind="k" />
      </div>

      <!-- Общая таблица -->
      <div class="table-head">
        <h4>Полная таблица документа</h4>
        <span class="mono text-muted">строки со всех участков</span>
      </div>
      <BlueprintPanel class="table-panel">
        <table class="table cons-table">
          <colgroup>
            <col style="width: 150px" /><col style="width: 90px" /><col style="width: 16%" />
            <col style="width: 22%" /><col style="width: 80px" /><col style="width: 56px" />
            <col style="width: 90px" /><col style="width: 160px" /><col style="width: 140px" />
          </colgroup>
          <thead>
            <tr>
              <th>аналитик / участок</th><th>row_id</th><th>клиент</th><th>материал</th>
              <th>сделка</th><th>вал.</th><th class="num">объём, т</th><th>статус</th>
              <th class="num">итоговая цена</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(cr, i) in doc?.rows ?? []" :key="i" :data-draft="cr.is_draft">
              <td>
                <span class="mono analyst">{{ cr.analyst_name }}{{ cr.is_draft ? ' · черновик' : '' }}</span>
              </td>
              <td class="mono dim">{{ cr.row.row_id }}</td>
              <td class="ellipsis">{{ cr.row.client_name }}</td>
              <td class="ellipsis">{{ cr.row.material_name }}</td>
              <td><span class="mono deal" :data-spot="cr.row.deal_type === 'SPOT'">{{ cr.row.deal_type }}</span></td>
              <td class="mono">{{ cr.row.currency }}</td>
              <td class="mono num">{{ formatInt(cr.row.volume) }}</td>
              <td><StatusChip :status="cr.row.status" /></td>
              <td class="mono num price">
                {{ cr.row.final_price == null ? 'нет цены' : formatNumber(cr.row.final_price) }}
              </td>
            </tr>
          </tbody>
        </table>
      </BlueprintPanel>
    </div>
  </div>
</template>

<style scoped>
.scroll {
  height: 100%;
  overflow: auto;
}
.wrap {
  max-width: 1240px;
  margin: 0 auto;
  padding: 22px 24px 64px;
}
.head {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;
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
  font-size: 14px;
  max-width: 760px;
  margin-bottom: 18px;
}
.draft-banner {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  margin-bottom: 16px;
  background: var(--st-warn-bg);
}
.draft-text {
  flex: 1;
  font-size: 13.5px;
}
.parts {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 10px;
  margin-bottom: 16px;
}
.part {
  padding: 15px 16px;
}
.part-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.part-name {
  font-family: var(--font-heading);
  font-weight: 600;
  font-size: 15px;
}
.you {
  color: var(--color-accent);
  font-size: 12px;
}
.part-chip {
  font-size: 11px;
  padding: 3px 10px;
  color: var(--chip-fg);
  background: var(--chip-bg);
}
.part-chip[data-kind='ok'] {
  --chip-fg: var(--st-ok);
  --chip-bg: var(--st-ok-bg);
}
.part-chip[data-kind='multi'] {
  --chip-fg: var(--st-multi);
  --chip-bg: var(--st-multi-bg);
}
.part-chip[data-kind='warn'] {
  --chip-fg: var(--st-warn);
  --chip-bg: var(--st-warn-bg);
}
.part-group {
  font-size: 12px;
  margin: 4px 0 10px;
}
.part-meta {
  display: flex;
  justify-content: space-between;
  font-size: 12px;
}
.kpis {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
  margin-bottom: 24px;
}
.table-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}
.table-head h4 {
  font-size: 17px;
}
.table-panel {
  max-height: 52vh;
  overflow: auto;
}
.cons-table {
  table-layout: fixed;
  min-width: 1000px;
}
.cons-table thead th {
  position: sticky;
  top: 0;
  z-index: 2;
  background: var(--header-bg);
}
.num {
  text-align: right;
}
tr[data-draft='true'] {
  opacity: 0.72;
}
.analyst {
  font-size: 11px;
  padding: 2px 8px;
  color: var(--st-ok);
  background: var(--st-ok-bg);
}
tr[data-draft='true'] .analyst {
  color: var(--st-warn);
  background: var(--st-warn-bg);
}
.dim {
  opacity: 0.7;
  font-size: 12.5px;
}
.ellipsis {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.deal {
  font-size: 11px;
  padding: 2px 7px;
  border: 1px solid var(--color-divider);
  color: var(--color-accent);
}
.deal[data-spot='true'] {
  color: var(--action);
}
.price {
  font-size: 13.5px;
  font-weight: 600;
}
</style>
