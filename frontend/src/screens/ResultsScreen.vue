<script setup lang="ts">
import { onMounted, computed, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useCalculationStore } from '@/stores/calculation'
import { useToast } from '@/composables/useToast'
import { formatInt, formatNumber } from '@/lib/format'
import type { RowStatus } from '@/types'
import KpiCard from '@/components/KpiCard.vue'
import StatusChip from '@/components/StatusChip.vue'
import AppButton from '@/components/AppButton.vue'
import DecodePanel from './DecodePanel.vue'

const router = useRouter()
const calculation = useCalculationStore()
const { show } = useToast()

const searchInput = ref('')

onMounted(() => {
  if (!calculation.calculationId) {
    void router.push({ name: 'upload' })
    return
  }
  void calculation.refresh()
})

// Дебаунс поиска через простую задержку.
let searchTimer: ReturnType<typeof setTimeout> | undefined
watch(searchInput, (value) => {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    calculation.query = value
    void calculation.refresh()
  }, 200)
})

// Пять канонических KPI (documents/кпэ_для_отображения.md).
const kpiCards = computed(() => {
  const k = calculation.kpi
  if (!k) return []
  const col = (v: number, good: number, mid: number) =>
    v >= good ? 'var(--good)' : v >= mid ? 'var(--mid)' : 'var(--bad)'
  // Для «плохих» процентов шкала обратная: чем меньше, тем лучше.
  const colInverse = (v: number, good: number, mid: number) =>
    v <= good ? 'var(--good)' : v <= mid ? 'var(--mid)' : 'var(--bad)'
  return [
    {
      label: 'Покрытие формулами',
      value: `${k.formula_coverage_pct}%`,
      sub: 'строк Formula',
      color: col(k.formula_coverage_pct, 90, 75),
      bar: `${k.formula_coverage_pct}%`,
    },
    {
      label: 'Формул без ошибок',
      value: `${k.formulas_ok_pct}%`,
      sub: 'все кандидаты посчитаны',
      color: col(k.formulas_ok_pct, 90, 75),
      bar: `${k.formulas_ok_pct}%`,
    },
    {
      label: 'Строк с ошибкой расчёта',
      value: String(k.calc_error_rows),
      sub: 'формула есть, цены нет',
      color: k.calc_error_rows === 0 ? 'var(--good)' : k.calc_error_rows <= 4 ? 'var(--mid)' : 'var(--bad)',
      bar: `${Math.min(100, k.calc_error_rows * 12)}%`,
    },
    {
      label: 'Контрольная сумма',
      value: `${formatNumber(k.control_sum_mln, 1)} млн ₽`,
      sub: 'Σ цена × объём',
      color: 'var(--color-accent)',
      bar: '100%',
    },
    {
      label: 'Непонятных ошибок',
      value: `${k.unclassified_error_pct}%`,
      sub: 'вне классификатора',
      color: colInverse(k.unclassified_error_pct, 5, 15),
      bar: `${k.unclassified_error_pct}%`,
    },
  ]
})

const FILTERS: { key: RowStatus | 'all'; label: string }[] = [
  { key: 'all', label: 'Все' },
  { key: 'calculated', label: 'Посчитано' },
  { key: 'formula_conflict', label: 'Выбрать' },
  { key: 'no_formula', label: 'Нет цены' },
  { key: 'component_error', label: 'Нет котир.' },
  { key: 'manual', label: 'Правка' },
  { key: 'calculated_expired', label: 'Продление' },
  { key: 'spot_not_calculated', label: 'SPOT' },
]

const filters = computed(() =>
  FILTERS.map((f) => ({
    ...f,
    count: f.key === 'all' ? calculation.total : (calculation.statusCounts[f.key] ?? 0),
    active: (calculation.statusFilter ?? 'all') === f.key,
  })),
)

function applyFilter(key: RowStatus | 'all'): void {
  calculation.statusFilter = key === 'all' ? null : key
  void calculation.refresh()
}

const sortIndicator = (key: string) =>
  calculation.sort === key ? (calculation.order === 'asc' ? ' ▲' : ' ▼') : ''

async function exportExcel(): Promise<void> {
  await calculation.exportExcel()
  show('Файл сформирован: prices_2026-06.csv')
}

function saveMyPart(): void {
  calculation.submitted = true
  void router.push({ name: 'consolidated' })
  show('Ваш участок сохранён и присоединён к сводному документу')
}
</script>

<template>
  <div class="results">
    <div class="main">
      <!-- KPI -->
      <div class="kpis">
        <KpiCard v-for="(k, i) in kpiCards" :key="i" v-bind="k" />
      </div>

      <!-- Тулбар -->
      <div class="toolbar">
        <div class="search">
          <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6">
            <circle cx="11" cy="11" r="7" /><path d="m21 21-4.3-4.3" />
          </svg>
          <input
            v-model="searchInput"
            class="input mono search-input"
            placeholder="Поиск: клиент, материал, row_id…"
          />
        </div>
        <div class="chips">
          <button
            v-for="f in filters"
            :key="f.key"
            class="mono chip"
            :data-active="f.active"
            @click="applyFilter(f.key)"
          >
            {{ f.label }} <span class="chip-count">{{ f.count }}</span>
          </button>
        </div>
        <AppButton @click="exportExcel">
          <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6">
            <path d="M12 3v12m0 0 4-4m-4 4-4-4M4 17v2a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-2" />
          </svg>
          Excel
        </AppButton>
        <AppButton v-if="!calculation.submitted" variant="action" @click="saveMyPart">
          Сохранить часть
        </AppButton>
      </div>

      <!-- Таблица -->
      <div class="table-scroll">
        <table class="table results-table">
          <colgroup>
            <col style="width: 88px" /><col style="width: 78px" /><col style="width: 16%" />
            <col style="width: 20%" /><col style="width: 90px" /><col style="width: 60px" />
            <col style="width: 96px" /><col style="width: 168px" /><col style="width: 62px" />
            <col style="width: 44px" /><col style="width: 144px" />
          </colgroup>
          <thead>
            <tr>
              <th class="sortable" @click="calculation.setSort('row_id')">row_id{{ sortIndicator('row_id') }}</th>
              <th>период</th>
              <th class="sortable" @click="calculation.setSort('client')">клиент{{ sortIndicator('client') }}</th>
              <th class="sortable" @click="calculation.setSort('material')">материал{{ sortIndicator('material') }}</th>
              <th>сделка</th>
              <th>вал.</th>
              <th class="num sortable" @click="calculation.setSort('volume')">объём, т{{ sortIndicator('volume') }}</th>
              <th>статус подбора</th>
              <th class="num">канд.</th>
              <th title="предупреждение">⚠</th>
              <th class="num sortable" @click="calculation.setSort('price')">итоговая цена{{ sortIndicator('price') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="row in calculation.rows"
              :key="row.row_id"
              class="row"
              :data-selected="calculation.selectedRowId === row.row_id"
              @click="calculation.select(row.row_id)"
            >
              <td class="mono dim">{{ row.row_id }}</td>
              <td class="mono">{{ row.period }}</td>
              <td class="ellipsis">{{ row.client_name }}</td>
              <td class="ellipsis">{{ row.material_name }}</td>
              <td>
                <span class="mono deal" :data-spot="row.deal_type === 'SPOT'">{{ row.deal_type }}</span>
              </td>
              <td class="mono">{{ row.currency }}</td>
              <td class="mono num">{{ formatInt(row.volume) }}</td>
              <td><StatusChip :status="row.status" /></td>
              <td class="mono num">{{ row.candidate_count ?? 0 }}</td>
              <td>
                <span v-if="row.warning" class="warn-icon" :title="row.warning">⚠</span>
              </td>
              <td class="mono num price">
                {{ row.final_price == null ? 'нет цены' : formatNumber(row.final_price) }}
              </td>
            </tr>
          </tbody>
        </table>
        <div v-if="calculation.rows.length === 0" class="empty-rows mono text-muted">
          Нет строк под текущий фильтр.
        </div>
      </div>
    </div>

    <DecodePanel v-if="calculation.hasSelection" />
  </div>
</template>

<style scoped>
.results {
  height: 100%;
  display: flex;
  min-height: 0;
}
.main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
}
.kpis {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 10px;
  padding: 14px 16px 4px;
  flex: none;
}
.toolbar {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 16px;
  flex: none;
  flex-wrap: wrap;
}
.search {
  position: relative;
  flex: 1;
  max-width: 280px;
}
.search svg {
  position: absolute;
  left: 10px;
  top: 50%;
  transform: translateY(-50%);
  opacity: 0.5;
}
.search-input {
  padding-left: 31px;
  font-size: 13px;
}
.chips {
  display: flex;
  align-items: center;
  gap: 5px;
  flex-wrap: wrap;
  flex: 1;
}
.chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 30px;
  padding: 0 10px;
  font-size: 12px;
  cursor: pointer;
  border: 1px solid var(--color-divider);
  background: transparent;
  color: var(--color-text);
}
.chip[data-active='true'] {
  border-color: var(--color-accent);
  background: var(--color-accent-100);
}
.chip-count {
  font-size: 10.5px;
  padding: 0 5px;
  background: var(--color-divider);
}
.chip[data-active='true'] .chip-count {
  background: var(--color-accent);
  color: var(--color-bg);
}
.table-scroll {
  flex: 1;
  min-height: 0;
  overflow: auto;
  border-top: 1px solid var(--color-divider);
  margin: 0 16px 16px;
}
.results-table {
  table-layout: fixed;
  min-width: 1140px;
}
.results-table thead th {
  position: sticky;
  top: 0;
  z-index: 2;
  background: var(--header-bg);
}
.sortable {
  cursor: pointer;
}
.num {
  text-align: right;
}
.row {
  cursor: pointer;
}
.row[data-selected='true'] {
  background: var(--color-accent-100);
  box-shadow: inset 3px 0 0 var(--color-accent);
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
.warn-icon {
  color: var(--st-warn);
  cursor: help;
}
.empty-rows {
  text-align: center;
  padding: 48px;
  font-size: 13px;
}
</style>
