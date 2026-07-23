<script setup lang="ts">
import { ref, computed } from 'vue'
import { useCalculationStore } from '@/stores/calculation'
import { useToast } from '@/composables/useToast'
import { formatPrice } from '@/lib/format'
import AppDialog from '@/components/AppDialog.vue'
import AppButton from '@/components/AppButton.vue'

const emit = defineEmits<{ close: [] }>()
const calculation = useCalculationStore()
const { show } = useToast()

const details = computed(() => calculation.details)
const row = computed(() => details.value?.row ?? null)
const alternatives = computed(() => details.value?.alternatives ?? [])

const currentSelected = computed(
  () => alternatives.value.find((a) => a.is_selected)?.formula_id ?? null,
)
const pick = ref<string | null>(currentSelected.value)

async function confirm(): Promise<void> {
  if (row.value && pick.value) {
    await calculation.selectFormula(row.value.row_id, pick.value)
    show('Формула переключена · KPI пересчитаны')
  }
  emit('close')
}
</script>

<template>
  <AppDialog v-if="row" width="min(620px, 94vw)" @close="emit('close')">
    <div class="modal">
      <div class="modal-head">
        <div>
          <div class="mono eyebrow">{{ row.row_id }} · {{ row.material_name }}</div>
          <div class="modal-title">Выбор договорной формулы</div>
        </div>
        <AppButton class="close" @click="emit('close')">✕</AppButton>
      </div>
      <p class="text-muted intro">
        Найдено {{ alternatives.length }} подходящих формул. Выберите применяемую — цена строки и KPI
        пересчитаются.
      </p>

      <div class="cards">
        <div
          v-for="a in alternatives"
          :key="a.formula_id"
          class="card"
          :data-active="pick === a.formula_id"
          @click="pick = a.formula_id"
        >
          <div class="card-main">
            <div class="card-head">
              <span class="radio" :data-on="pick === a.formula_id" />
              <span class="mono code">{{ a.formula_id }}</span>
              <span class="mono tag">{{ a.is_actual ? 'актуальная' : 'просроченная' }}</span>
            </div>
            <div class="mono validity">Действует: {{ a.valid_from }} — {{ a.valid_to }}</div>
            <div class="mono expr">{{ a.formula_text }}</div>
          </div>
          <div class="card-price">
            <div class="mono price" :data-matched="a.matched">
              {{ a.price == null ? 'нет цены' : formatPrice(a.price, row.currency) }}
            </div>
            <div class="mono match" :data-matched="a.matched">
              {{ a.price == null ? '—' : a.matched ? '±3% ✓' : '>3%' }}
            </div>
          </div>
        </div>
      </div>

      <div class="actions">
        <AppButton @click="emit('close')">Отмена</AppButton>
        <AppButton variant="action" @click="confirm">Применить выбранную</AppButton>
      </div>
    </div>
  </AppDialog>
</template>

<style scoped>
.modal {
  padding: 18px;
  overflow: auto;
}
.modal-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 10px;
}
.eyebrow {
  font-size: 11px;
  color: var(--color-accent);
  letter-spacing: 0.06em;
}
.modal-title {
  font-family: var(--font-heading);
  font-weight: 600;
  font-size: 22px;
}
.close {
  width: 32px;
  padding: 0;
  justify-content: center;
  flex: none;
}
.intro {
  font-size: 13px;
  margin: 6px 0;
}
.cards {
  display: grid;
  gap: 10px;
  margin-top: 4px;
}
.card {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 14px;
  cursor: pointer;
  border: 1px solid var(--color-divider);
  background: var(--panel);
}
.card[data-active='true'] {
  border-color: var(--color-accent);
  background: var(--color-accent-100);
}
.card-head {
  display: flex;
  align-items: center;
  gap: 8px;
}
.radio {
  width: 14px;
  height: 14px;
  border-radius: 50%;
  border: 1.5px solid var(--color-divider);
  flex: none;
}
.radio[data-on='true'] {
  border-color: var(--color-accent);
  background: var(--color-accent);
  box-shadow: inset 0 0 0 3px var(--panel);
}
.code {
  font-weight: 600;
  font-size: 13px;
}
.tag {
  font-size: 10px;
  padding: 1px 7px;
  color: var(--st-ext);
  background: var(--st-ext-bg);
}
.validity {
  font-size: 11.5px;
  margin: 5px 0 0 22px;
  opacity: 0.75;
}
.expr {
  font-size: 11.5px;
  margin: 6px 0 0 22px;
  opacity: 0.85;
  word-break: break-word;
}
.card-price {
  text-align: right;
  flex: none;
}
.price {
  font-size: 18px;
  font-weight: 600;
}
.price[data-matched='true'] {
  color: var(--st-ok);
}
.match {
  font-size: 10.5px;
  color: var(--st-multi);
}
.match[data-matched='true'] {
  color: var(--st-ok);
}
.actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 16px;
}
</style>
