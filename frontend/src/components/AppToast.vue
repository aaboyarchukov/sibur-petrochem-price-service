<script setup lang="ts">
defineProps<{ message: string }>()
</script>

<template>
  <Transition name="toast">
    <div v-if="message" class="mono toast">
      <svg
        width="16"
        height="16"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="1.8"
      >
        <path d="M20 6 9 17l-5-5" />
      </svg>
      {{ message }}
    </div>
  </Transition>
</template>

<style scoped>
.toast {
  position: fixed;
  bottom: 22px;
  left: 50%;
  transform: translateX(-50%);
  background: var(--color-accent);
  color: var(--color-bg);
  padding: 11px 18px;
  font-size: 13px;
  box-shadow: var(--shadow-lg);
  z-index: 60;
  display: flex;
  align-items: center;
  gap: 9px;
}
.toast-enter-active,
.toast-leave-active {
  transition: opacity 0.2s;
}
.toast-enter-from,
.toast-leave-to {
  opacity: 0;
}
</style>
