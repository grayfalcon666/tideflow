<script setup lang="ts">
defineProps<{
  videoId?: number
}>()
</script>

<template>
  <div class="player-container">
    <slot :videoId="videoId" />
  </div>
</template>

<style scoped lang="scss">
.player-container {
  width: 100%;
  background: #000;
  display: flex;
  justify-content: center;
  align-items: center;
}
</style>