<script setup lang="ts">
import { ref, computed } from 'vue'
import { useCalculationStore } from '@/stores/calculation'
import { useToast } from '@/composables/useToast'
import { formatInt, formatNumber, formatPrice } from '@/lib/format'
import StatusChip from '@/components/StatusChip.vue'
import AppButton from '@/components/AppButton.vue'
import BlueprintPanel from '@/components/BlueprintPanel.vue'
import FormulaModal from './FormulaModal.vue'

const calculation = useCalculationStore()
const { show } = useToast()

const draft = ref('')
const draftError = ref('')
const showModal = ref(false)

const details = computed(() => calculation.details)
const row = computed(() => details.value?.row ?? null)
const applied = computed(() => details.value?.applied_formula ?? null)
const conversion = computed(() => details.value?.conversion ?? null)

const priceColor = computed(() => {
  const r = row.value
  if (!r || r.final_price == null) return 'var(--st-warn)'
  return r.matched ? 'var(--st-ok)' : 'var(--color-text)'
})

const priceNote = computed(() => {
  const d = details.value
  const r = row.value
  if (!d || !r) return ''
  if (d.manual_price != null) return `Введено вручную · эталон ${formatNumber(d.reference_price)}`
  if (r.final_price == null) return applied.value ? 'Нет значения компонента' : 'Формула не подобрана'
  if (r.matched) return 'В пределах ±3% от эталона'
  return `Отклонение от эталона >3% (${formatNumber(d.reference_price)})`
})

function typeKind(type: string): string {
  if (type === 'quote') return 'ok'
  if (type === 'currency_rate') return 'ext'
  if (type === 'logistics') return 'multi'
  return 'manual'
}

async function applyManual(): Promise<void> {
  const raw = draft.value.replace(/\s/g, '').replace(',', '.')
  if (!raw) {
    draftError.value = 'Введите цену'
    return
  }
  const value = Number(raw)
  if (!Number.isFinite(value)) {
    draftError.value = 'Не число'
    return
  }
  if (value <= 0) {
    draftError.value = 'Цена должна быть > 0'
    return
  }
  if (row.value) {
    await calculation.setManualPrice(row.value.row_id, Math.round(value * 100) / 100)
    draft.value = ''
    draftError.value = ''
    show('Ручная цена применена · KPI пересчитаны')
  }
}

async function resetManual(): Promise<void> {
  if (row.value) {
    await calculation.resetManualPrice(row.value.row_id)
    show('Ручная правка сброшена')
  }
}
</script>

<template>
  <div v-if="row" class="backdrop" @click="calculation.closeDetails()">
    <BlueprintPanel class="panel" @click.stop>
      <!-- Шапка -->
      <div class="head">
        <div class="head-top">
          <div>
            <div class="mono eyebrow">РАСШИФРОВКА · {{ row.row_id }}</div>
            <div class="title">{{ row.material_name }}</div>
          </div>
          <AppButton class="close" @click="calculation.closeDetails()">✕</AppButton>
        </div>
        <div class="facts">
          <div><span class="text-muted">Клиент</span> <b>{{ row.client_name }}</b></div>
          <div><span class="text-muted">Период</span> <b class="mono">{{ row.period }}</b></div>
          <div><span class="text-muted">Сделка</span> <b class="mono">{{ row.deal_type }}</b></div>
          <div><span class="text-muted">Валюта</span> <b class="mono">{{ row.currency }}</b></div>
          <div><span class="text-muted">Объём</span> <b class="mono">{{ formatInt(row.volume) }} т</b></div>
        </div>
        <StatusChip :status="row.status" class="head-chip" />
      </div>

      <!-- Тело -->
      <div class="body">
        <!-- Нет формулы -->
        <div v-if="!applied" class="no-formula">
          <div class="no-formula-title">Формула не подобрана</div>
          <p>
            Для сочетания «клиент × материал × период» нет действующей договорной формулы. Задайте
            формулу в справочнике или введите значение вручную ниже.
          </p>
        </div>

        <!-- Применённая формула -->
        <template v-else>
          <div class="mono section-label">Применённая формула · {{ applied.formula_id }}</div>
          <BlueprintPanel class="formula-box mono">{{ applied.formula_text }}</BlueprintPanel>
          <div v-if="applied.is_extended" class="mono extended-note">
            ⏱ Формула просрочена — срок продлён до периода строки
          </div>

          <div class="mono section-label components-label">Компоненты · подстановка значений</div>
          <BlueprintPanel class="components">
            <table class="table components-table">
              <thead>
                <tr><th>перем.</th><th>тип</th><th class="num">значение</th><th>источник · дата</th></tr>
              </thead>
              <tbody>
                <tr v-for="(c, i) in details?.components ?? []" :key="i">
                  <td class="mono strong">{{ c.var_name }}</td>
                  <td><span class="mono type-chip" :data-kind="typeKind(c.type)">{{ c.type_label }}</span></td>
                  <td class="mono num" :data-error="c.status === 'error'">
                    {{ c.value == null ? '— нет' : formatNumber(c.value, 4) }}
                  </td>
                  <td class="source">
                    <div>{{ c.source }}</div>
                    <div class="mono text-muted">
                      {{ c.error ?? (c.value_date ? `${c.value_date}${c.version_type ? ' · ' + c.version_type : ''}` : '—') }}
                    </div>
                  </td>
                </tr>
              </tbody>
            </table>
          </BlueprintPanel>

          <!-- Несколько формул -->
          <div v-if="row.requires_review" class="multi">
            <div class="multi-head">
              <div class="multi-title">Подошло несколько формул ({{ details?.equal_priority_count }})</div>
              <AppButton variant="action" @click="showModal = true">Выбрать формулу</AppButton>
            </div>
            <p>По умолчанию применена самая новая. Можно переключить — цена и KPI пересчитаются.</p>
          </div>
        </template>
      </div>

      <!-- Итог + ручная правка -->
      <div class="footer">
        <div class="footer-top">
          <div>
            <div class="mono eyebrow">Итоговая цена</div>
            <div class="mono final-price" :style="{ color: priceColor }">
              {{ formatPrice(row.final_price, row.currency) }}
            </div>
            <div class="mono text-muted note">{{ priceNote }}</div>
            <div v-if="conversion" class="mono text-muted note">
              {{ conversion.from_currency }} → {{ conversion.to_currency }} по курсу
              {{ formatNumber(conversion.from_rate) }} ({{ conversion.version_type }})
            </div>
          </div>
          <AppButton v-if="details?.manual_price != null" @click="resetManual">Сбросить правку</AppButton>
        </div>
        <div class="manual">
          <div class="manual-input">
            <input
              v-model="draft"
              class="input mono"
              :placeholder="`Ручная цена, ${row.currency}`"
              :data-error="!!draftError"
              @input="draftError = ''"
            />
            <div v-if="draftError" class="mono error">{{ draftError }}</div>
          </div>
          <AppButton variant="primary" @click="applyManual">Применить</AppButton>
        </div>
      </div>
    </BlueprintPanel>

    <FormulaModal v-if="showModal" @close="showModal = false" />
  </div>
</template>

<style scoped>
.backdrop {
  position: absolute;
  inset: 0;
  background: rgba(0, 20, 25, 0.42);
  display: flex;
  justify-content: flex-end;
  z-index: 35;
}
.panel {
  width: min(600px, 94vw);
  display: flex;
  flex-direction: column;
  box-shadow: var(--shadow-lg);
}
.head {
  padding: 14px 18px;
  border-bottom: 1px solid var(--color-divider);
  flex: none;
}
.head-top {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 10px;
}
.eyebrow {
  font-size: 11px;
  color: var(--color-accent);
  letter-spacing: 0.06em;
  text-transform: uppercase;
}
.title {
  font-family: var(--font-heading);
  font-weight: 600;
  font-size: 19px;
  line-height: 1.15;
  margin-top: 2px;
}
.close {
  width: 32px;
  padding: 0;
  justify-content: center;
  flex: none;
}
.facts {
  display: flex;
  flex-wrap: wrap;
  gap: 5px 14px;
  margin-top: 10px;
  font-size: 12.5px;
}
.facts b {
  font-weight: 500;
}
.head-chip {
  margin-top: 10px;
}
.body {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 16px 18px;
}
.section-label {
  font-size: 10.5px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  opacity: 0.65;
  margin-bottom: 6px;
}
.components-label {
  margin-top: 18px;
}
.formula-box {
  padding: 14px;
  background: var(--color-accent-100);
  font-size: 13px;
  line-height: 1.7;
  word-break: break-word;
}
.extended-note {
  font-size: 11.5px;
  margin-top: 8px;
  color: var(--st-ext);
}
.components {
  overflow: hidden;
}
.components-table {
  font-size: 12.5px;
}
.num {
  text-align: right;
}
.strong {
  font-weight: 500;
}
.type-chip {
  font-size: 10.5px;
  padding: 2px 7px;
  color: var(--chip-fg);
  background: var(--chip-bg);
}
.type-chip[data-kind='ok'] {
  --chip-fg: var(--st-ok);
  --chip-bg: var(--st-ok-bg);
}
.type-chip[data-kind='ext'] {
  --chip-fg: var(--st-ext);
  --chip-bg: var(--st-ext-bg);
}
.type-chip[data-kind='multi'] {
  --chip-fg: var(--st-multi);
  --chip-bg: var(--st-multi-bg);
}
.type-chip[data-kind='manual'] {
  --chip-fg: var(--st-manual);
  --chip-bg: var(--st-manual-bg);
}
[data-error='true'] {
  color: var(--bad);
  font-weight: 600;
}
.source {
  font-size: 11px;
  line-height: 1.35;
}
.no-formula {
  padding: 16px;
  background: var(--st-warn-bg);
}
.no-formula-title {
  font-family: var(--font-heading);
  font-weight: 600;
  font-size: 16px;
  color: var(--st-warn);
  margin-bottom: 4px;
}
.no-formula p {
  font-size: 13px;
  margin: 0;
  opacity: 0.85;
}
.multi {
  margin-top: 18px;
  padding: 14px;
  background: var(--st-multi-bg);
}
.multi-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}
.multi-title {
  font-family: var(--font-heading);
  font-weight: 600;
  font-size: 15px;
  color: var(--st-multi);
}
.multi p {
  font-size: 12.5px;
  margin: 6px 0 0;
  opacity: 0.85;
}
.footer {
  border-top: 1px solid var(--color-divider);
  padding: 14px 18px;
  flex: none;
}
.footer-top {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 12px;
}
.final-price {
  font-size: 30px;
  font-weight: 600;
  line-height: 1.05;
}
.note {
  font-size: 11px;
}
.manual {
  display: flex;
  gap: 8px;
  margin-top: 12px;
  align-items: flex-start;
}
.manual-input {
  flex: 1;
}
.manual-input .input[data-error='true'] {
  border-color: var(--bad);
}
.error {
  font-size: 11px;
  color: var(--bad);
  margin-top: 4px;
}
</style>
