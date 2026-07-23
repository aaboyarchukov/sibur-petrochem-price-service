<script setup lang="ts">
import BlueprintPanel from './BlueprintPanel.vue'

defineProps<{
  label: string
  value: string
  sub?: string
  color?: string
  bar?: string
}>()
</script>

<template>
  <BlueprintPanel>
    <div class="kpi">
      <div class="mono label text-muted">{{ label }}</div>
      <div class="row">
        <div class="mono value" :style="{ color: color ?? 'var(--color-text)' }">{{ value }}</div>
        <div v-if="sub" class="sub">{{ sub }}</div>
      </div>
      <div class="track">
        <div class="fill" :style="{ width: bar ?? '0%', background: color ?? 'var(--color-accent)' }" />
      </div>
    </div>
  </BlueprintPanel>
</template>

<style scoped>
.kpi {
  padding: 12px 14px;
}
.label {
  font-size: 10.5px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  margin-bottom: 6px;
}
.row {
  display: flex;
  align-items: baseline;
  gap: 8px;
}
.value {
  font-size: 28px;
  font-weight: 600;
  line-height: 1;
}
.sub {
  font-size: 12px;
  opacity: 0.65;
}
.track {
  height: 4px;
  background: var(--color-surface);
  margin-top: 9px;
}
.fill {
  height: 100%;
}
</style>
