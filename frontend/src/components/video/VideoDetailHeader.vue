<script setup lang="ts">
import TFIcon from '../common/TFIcon.vue'

defineProps<{ title?: string }>()
defineEmits<{ back: [] }>()
</script>

<template>
  <div class="detail-header">
    <div class="back-btn" @click="$emit('back')">
      <TFIcon name="arrow_back" :size="24" />
    </div>
    <h2 class="detail-title">{{ title }}</h2>
  </div>
</template>

<style scoped lang="scss">
.detail-header {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: 0.1px var(--space-4);
  background: var(--bg-surface);
  border-bottom: 1px solid var(--border);
  position: sticky;
  top: 0;
  z-index: 10;
}

.back-btn {
  cursor: pointer;
  display: flex;
  align-items: center;
  color: var(--text-base);
  padding: var(--space-1);
}

.detail-title {
  flex: 1;
  font-size: 16px;
  font-weight: 700;
  color: var(--text-base);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  margin: 0;
}
</style>
