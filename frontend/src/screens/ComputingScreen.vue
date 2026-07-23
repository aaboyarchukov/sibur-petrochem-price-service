<script setup lang="ts">
import { onMounted, onBeforeUnmount, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useCalculationStore } from '@/stores/calculation'
import { formatInt } from '@/lib/format'

const router = useRouter()
const calculation = useCalculationStore()

let unsubscribe: (() => void) | null = null

const percent = computed(() => calculation.progress?.percent ?? 0)
const processed = computed(() => calculation.progress?.processed_rows ?? 0)

onMounted(() => {
  // Подписка на SSE-поток прогресса; по завершению — переход к результатам.
  unsubscribe = calculation.streamProgress(async () => {
    await calculation.refresh()
    await router.push({ name: 'results' })
  })
})

onBeforeUnmount(() => {
  unsubscribe?.()
  unsubscribe = null
})
</script>

<template>
  <div class="center">
    <div class="box">
      <div class="spinner" />
      <h3>Идёт расчёт</h3>
      <p class="text-muted mono note">Подбор формул · подстановка котировок · курсы валют</p>
      <div class="track">
        <div class="fill" :style="{ width: percent + '%' }" />
      </div>
      <div class="mono meta">
        <span>{{ formatInt(processed) }} строк</span>
        <span>{{ percent }}%</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.center {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}
.box {
  width: min(460px, 86vw);
  text-align: center;
}
.spinner {
  width: 60px;
  height: 60px;
  border: 2px solid var(--color-divider);
  border-top-color: var(--color-accent);
  border-radius: 50%;
  margin: 0 auto 22px;
  animation: spin 0.8s linear infinite;
}
h3 {
  font-size: 22px;
  margin-bottom: 4px;
}
.note {
  font-size: 13px;
  margin-bottom: 20px;
}
.track {
  height: 8px;
  background: var(--color-surface);
  border: 1px solid var(--color-divider);
  overflow: hidden;
}
.fill {
  height: 100%;
  background: var(--action);
  transition: width 0.18s linear;
}
.meta {
  display: flex;
  justify-content: space-between;
  font-size: 12px;
  margin-top: 8px;
  opacity: 0.75;
}
</style>
