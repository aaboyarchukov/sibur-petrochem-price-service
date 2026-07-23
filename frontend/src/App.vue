<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useTheme } from '@/composables/useTheme'
import { useToast } from '@/composables/useToast'
import { useSourcesStore } from '@/stores/sources'
import { usePricingApi } from '@/services/provide'
import AppToast from '@/components/AppToast.vue'

const route = useRoute()
const router = useRouter()
const { theme, toggle } = useTheme()
const { message } = useToast()
const sources = useSourcesStore()
const api = usePricingApi()

// Присутствие аналитиков в реальном времени (SSE).
const analystsOnline = ref(1)
let unsubscribePresence: (() => void) | null = null
onMounted(() => {
  unsubscribePresence = api.streamPresence((count) => {
    analystsOnline.value = count
  })
})
onBeforeUnmount(() => unsubscribePresence?.())

const analystsLabel = computed(() => {
  const n = analystsOnline.value
  const mod10 = n % 10
  const mod100 = n % 100
  if (mod10 === 1 && mod100 !== 11) return `${n} аналитик онлайн`
  if (mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14)) return `${n} аналитика онлайн`
  return `${n} аналитиков онлайн`
})

const steps = computed(() => [
  { name: 'upload', num: '1', label: 'Данные', enabled: true },
  { name: 'results', num: '2', label: 'Результаты', enabled: sources.loaded },
])

function go(name: string, enabled: boolean): void {
  if (!enabled) return
  void router.push({ name })
}
</script>

<template>
  <div class="shell">
    <header class="topbar">
      <div class="brand">
        <div class="logo">P</div>
        <div class="brand-text">
          PETRO&#8202;PRICE<span class="accent">.</span>
          <br /><span class="brand-sub">Расчёт цен</span>
        </div>
      </div>

      <nav class="steps">
        <button
          v-for="s in steps"
          :key="s.name"
          class="mono step"
          :data-active="route.name === s.name || (s.name === 'results' && route.name === 'computing')"
          :disabled="!s.enabled"
          @click="go(s.name, s.enabled)"
        >
          <span class="num">{{ s.num }}</span>{{ s.label }}
        </button>
        <div class="sep" />
        <button
          class="mono step"
          :data-active="route.name === 'consolidated'"
          :disabled="!sources.loaded"
          @click="go('consolidated', sources.loaded)"
        >
          Сводный документ
        </button>
      </nav>

      <div class="right">
        <div class="mono online"><span class="online-dot" />{{ analystsLabel }}</div>
        <button class="mono theme-btn" @click="toggle">
          {{ theme === 'light' ? '◑ Тёмная' : '◐ Светлая' }}
        </button>
      </div>
    </header>

    <main class="content">
      <RouterView />
    </main>

    <AppToast :message="message" />
  </div>
</template>

<style scoped>
.shell {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}
.topbar {
  display: flex;
  align-items: center;
  gap: 20px;
  padding: 0 20px;
  height: 56px;
  border-bottom: 1px solid var(--color-divider);
  background: var(--panel);
  flex: none;
}
.brand {
  display: flex;
  align-items: center;
  gap: 10px;
}
.logo {
  width: 26px;
  height: 26px;
  background: var(--color-accent);
  color: var(--color-bg);
  display: grid;
  place-items: center;
  font-family: var(--font-heading);
  font-weight: 600;
  font-size: 15px;
}
.brand-text {
  font-family: var(--font-heading);
  font-weight: 600;
  font-size: 17px;
  line-height: 1;
}
.accent {
  color: var(--color-accent);
}
.brand-sub {
  font-family: var(--font-body);
  font-weight: 400;
  font-size: 10px;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  opacity: 0.6;
}
.steps {
  display: flex;
  align-items: center;
  gap: 4px;
}
.step {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  height: 34px;
  padding: 0 13px;
  border: 1px solid transparent;
  background: transparent;
  color: var(--color-text);
  font-family: var(--font-heading);
  font-weight: 600;
  font-size: 13px;
  cursor: pointer;
}
.step[data-active='true'] {
  border-color: var(--color-accent);
  background: var(--color-accent-100);
}
.step:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}
.num {
  width: 19px;
  height: 19px;
  display: grid;
  place-items: center;
  font-size: 11px;
  background: var(--color-divider);
}
.step[data-active='true'] .num {
  background: var(--color-accent);
  color: var(--color-bg);
}
.sep {
  width: 1px;
  height: 22px;
  background: var(--color-divider);
  margin: 0 4px;
}
.right {
  margin-left: auto;
  display: flex;
  align-items: center;
  gap: 10px;
}
.online {
  font-size: 12px;
  opacity: 0.7;
  display: flex;
  align-items: center;
  gap: 6px;
}
.online-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--good);
}
.theme-btn {
  height: 34px;
  padding: 0 12px;
  border: 1px solid var(--color-divider);
  background: transparent;
  color: var(--color-text);
  font-size: 12px;
  cursor: pointer;
}
.content {
  flex: 1;
  min-height: 0;
  position: relative;
  overflow: hidden;
}
</style>
