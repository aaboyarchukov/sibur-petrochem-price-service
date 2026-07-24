<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useParamsStore } from '@/stores/params'
import { useCalculationStore } from '@/stores/calculation'
import AppButton from '@/components/AppButton.vue'

const router = useRouter()
const params = useParamsStore()
const calculation = useCalculationStore()

const productQuery = ref('')
const clientQuery = ref('')
const startError = ref('')

const filteredProducts = computed(() => {
  const q = productQuery.value.trim().toLowerCase()
  if (!q) return params.products
  return params.products.filter((p) => `${p.name} ${p.id}`.toLowerCase().includes(q))
})

const filteredClients = computed(() => {
  const q = clientQuery.value.trim().toLowerCase()
  if (!q) return params.clients
  return params.clients.filter((c) => `${c.name} ${c.id}`.toLowerCase().includes(q))
})

onMounted(() => {
  if (!params.loaded) void params.loadFacets()
})

async function compute(): Promise<void> {
  startError.value = ''
  try {
    await calculation.start(params.params)
    await router.push({ name: 'computing' })
  } catch (e) {
    startError.value = e instanceof Error ? e.message : String(e)
  }
}
</script>

<template>
  <div class="scroll">
    <div class="wrap">
      <header class="head">
        <div>
          <div class="mono eyebrow">Параметры расчёта</div>
          <h2>Выбор параметров</h2>
        </div>
        <AppButton variant="primary" @click="compute">Рассчитать →</AppButton>
      </header>

      <p v-if="startError" class="error mono">{{ startError }}</p>

      <!-- Дата -->
      <section class="block">
        <h3>Период</h3>
        <label class="radio">
          <input type="radio" :value="false" v-model="params.rangeMode" />
          <span
          >Весь период<span v-if="params.periodMin" class="hint mono">
            {{ params.periodMin }} — {{ params.periodMax }}</span
          ></span
          >
        </label>
        <label class="radio">
          <input type="radio" :value="true" v-model="params.rangeMode" />
          <span>Диапазон месяцев</span>
        </label>
        <div v-if="params.rangeMode" class="range">
          <label class="mono"
          >с
            <input
              type="month"
              v-model="params.periodFrom"
              :min="params.periodMin ?? undefined"
              :max="params.periodMax ?? undefined"
            />
          </label>
          <label class="mono"
          >по
            <input
              type="month"
              v-model="params.periodTo"
              :min="params.periodMin ?? undefined"
              :max="params.periodMax ?? undefined"
            />
          </label>
        </div>
      </section>

      <div class="cols">
        <!-- Продукт -->
        <section class="block">
          <h3>
            Продукт <span class="hint mono">{{ params.selectedProducts.size || 'все' }}</span>
          </h3>
          <input class="search mono" v-model="productQuery" placeholder="поиск продукта…" />
          <div class="list">
            <label v-for="p in filteredProducts" :key="p.id" class="check">
              <input
                type="checkbox"
                :checked="params.selectedProducts.has(p.id)"
                @change="params.toggleProduct(p.id)"
              />
              <span
              >{{ p.name }} <span class="hint mono">{{ p.id }}</span></span
              >
            </label>
          </div>
        </section>

        <!-- Клиент -->
        <section class="block">
          <h3>
            Клиент <span class="hint mono">{{ params.selectedClients.size || 'все' }}</span>
          </h3>
          <input class="search mono" v-model="clientQuery" placeholder="поиск клиента…" />
          <div class="list">
            <label v-for="c in filteredClients" :key="c.id" class="check">
              <input
                type="checkbox"
                :checked="params.selectedClients.has(c.id)"
                @change="params.toggleClient(c.id)"
              />
              <span
              >{{ c.name }} <span class="hint mono">{{ c.id }}</span></span
              >
            </label>
          </div>
        </section>
      </div>
    </div>
  </div>
</template>

<style scoped>
.scroll {
  height: 100%;
  overflow-y: auto;
}
.wrap {
  max-width: 1100px;
  margin: 0 auto;
  padding: 28px 32px 48px;
}
.head {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  margin-bottom: 24px;
}
.eyebrow {
  font-size: 11px;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: var(--color-text-muted);
}
h2 {
  margin: 4px 0 0;
}
h3 {
  margin: 0 0 12px;
  font-size: 14px;
}
.error {
  color: var(--bad);
  margin-bottom: 16px;
}
.block {
  border: 1px solid var(--color-divider);
  padding: 18px 20px;
  margin-bottom: 20px;
}
.cols {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
}
.radio {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
  cursor: pointer;
}
.hint {
  color: var(--color-text-muted);
  font-size: 12px;
  margin-left: 6px;
}
.range {
  display: flex;
  gap: 20px;
  margin-top: 12px;
}
.range label {
  display: flex;
  align-items: center;
  gap: 8px;
}
.range input {
  height: 34px;
  padding: 0 8px;
  border: 1px solid var(--color-divider);
  background: transparent;
  color: var(--color-text);
}
.search {
  width: 100%;
  height: 34px;
  padding: 0 10px;
  margin-bottom: 12px;
  border: 1px solid var(--color-divider);
  background: transparent;
  color: var(--color-text);
}
.list {
  max-height: 360px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.check {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  cursor: pointer;
  font-size: 13px;
  line-height: 1.35;
}
@media (max-width: 820px) {
  .cols {
    grid-template-columns: 1fr;
  }
}
</style>
