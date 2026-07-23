<script setup lang="ts">
import { computed } from 'vue'
import type { RowStatus } from '@/types'
import { statusMeta } from '@/lib/statusMeta'

const props = defineProps<{ status: RowStatus }>()
const meta = computed(() => statusMeta(props.status))
</script>

<template>
  <span class="mono chip" :data-kind="meta.kind">
    <span class="dot" />
    {{ meta.label }}
  </span>
</template>

<style scoped>
.chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  padding: 3px 9px;
  white-space: nowrap;
  color: var(--chip-fg);
  background: var(--chip-bg);
}
.dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--chip-fg);
  flex: none;
}
.chip[data-kind='ok'] {
  --chip-fg: var(--st-ok);
  --chip-bg: var(--st-ok-bg);
}
.chip[data-kind='ext'] {
  --chip-fg: var(--st-ext);
  --chip-bg: var(--st-ext-bg);
}
.chip[data-kind='multi'] {
  --chip-fg: var(--st-multi);
  --chip-bg: var(--st-multi-bg);
}
.chip[data-kind='manual'] {
  --chip-fg: var(--st-manual);
  --chip-bg: var(--st-manual-bg);
}
.chip[data-kind='warn'] {
  --chip-fg: var(--st-warn);
  --chip-bg: var(--st-warn-bg);
}
.chip[data-kind='err'] {
  --chip-fg: var(--st-err);
  --chip-bg: var(--st-err-bg);
}
</style>
