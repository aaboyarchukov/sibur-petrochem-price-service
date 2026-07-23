<script setup lang="ts">
import BlueprintPanel from './BlueprintPanel.vue'

defineProps<{ width?: string }>()
const emit = defineEmits<{ close: [] }>()
</script>

<template>
  <div class="dialog-backdrop" @click="emit('close')">
    <BlueprintPanel :style="{ width: width ?? 'min(600px, 94vw)', maxHeight: '88vh' }" @click.stop>
      <div class="dialog-body">
        <slot />
      </div>
    </BlueprintPanel>
  </div>
</template>

<style scoped>
.dialog-body {
  display: flex;
  flex-direction: column;
  max-height: 88vh;
  box-shadow: var(--shadow-lg);
}
</style>
