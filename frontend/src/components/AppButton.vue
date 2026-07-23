<script setup lang="ts">
withDefaults(defineProps<{ variant?: 'primary' | 'secondary' | 'action' }>(), {
  variant: 'secondary',
})
</script>

<template>
  <button class="btn" :data-variant="variant">
    <slot />
  </button>
</template>

<style scoped>
.btn {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  height: 36px;
  padding: 0 15px;
  font-family: var(--font-heading);
  font-weight: 600;
  font-size: 13px;
  cursor: pointer;
  border: 1px solid var(--color-divider);
  background: transparent;
  color: var(--color-text);
  white-space: nowrap;
}
.btn:hover {
  border-color: var(--color-accent);
}
.btn[data-variant='primary'] {
  background: var(--color-accent);
  color: var(--color-bg);
  border-color: var(--color-accent);
}
.btn[data-variant='action'] {
  background: var(--action);
  color: #fff;
  border-color: var(--action);
}
.btn[data-variant='action']:hover {
  background: var(--action-600);
  border-color: var(--action-600);
}
.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
