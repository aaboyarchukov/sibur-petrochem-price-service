<script setup lang="ts">
// Панель-обёртка с угловыми маркерами (blueprint) из прототипа.
</script>

<template>
  <div class="blueprint">
    <i class="corner tl" /><i class="corner tr" /><i class="corner bl" /><i class="corner br" />
    <slot />
  </div>
</template>

<style scoped>
.blueprint {
  position: relative;
  border: 1px solid var(--color-divider);
  background: var(--panel);
}
.corner {
  position: absolute;
  width: 7px;
  height: 7px;
  border-color: var(--color-accent);
  border-style: solid;
  border-width: 0;
}
.corner.tl {
  top: -1px;
  left: -1px;
  border-top-width: 2px;
  border-left-width: 2px;
}
.corner.tr {
  top: -1px;
  right: -1px;
  border-top-width: 2px;
  border-right-width: 2px;
}
.corner.bl {
  bottom: -1px;
  left: -1px;
  border-bottom-width: 2px;
  border-left-width: 2px;
}
.corner.br {
  bottom: -1px;
  right: -1px;
  border-bottom-width: 2px;
  border-right-width: 2px;
}
</style>
