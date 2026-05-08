<script setup lang="ts">
defineProps<{
  videoId?: number
  fixedRatio?: boolean
}>()
</script>

<template>
  <div class="player-container" :class="{ 'fixed-ratio': fixedRatio }">
    <slot />
  </div>
</template>

<style scoped lang="scss">
.player-container {
  width: 100%;
  background: #000;
  display: flex;
  justify-content: center;
  align-items: center;

  &.fixed-ratio {
    aspect-ratio: 16 / 9;
  }
}
</style>