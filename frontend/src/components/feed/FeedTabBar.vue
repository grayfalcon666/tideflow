<script setup lang="ts">
import type { FeedTab } from '../../stores/feed'

defineProps<{
  activeTab: FeedTab
}>()

const emit = defineEmits<{
  (e: 'update:tab', tab: FeedTab): void
}>()
</script>

<template>
  <div class="feed-tab-bar">
    <button
      class="tab-btn"
      :class="{ active: activeTab === 'latest' }"
      @click="emit('update:tab', 'latest')"
    >
      推荐
    </button>
    <button
      class="tab-btn"
      :class="{ active: activeTab === 'following' }"
      @click="emit('update:tab', 'following')"
    >
      关注
    </button>
  </div>
</template>

<style scoped lang="scss">
.feed-tab-bar {
  display: flex;
  gap: var(--space-2);
  padding: var(--space-3) var(--space-4);
  background: var(--bg-surface);
  border-bottom: 1px solid var(--border);
}

.tab-btn {
  background: transparent;
  border: none;
  color: var(--text-secondary);
  font-size: 16px;
  font-weight: 600;
  padding: var(--space-2) var(--space-4);
  border-radius: var(--radius-full);
  cursor: pointer;
  transition: all var(--transition-fast);

  &.active {
    background: var(--bg-elevated);
    color: var(--text-base);
  }

  &:hover:not(.active) {
    color: var(--text-base);
  }
}
</style>
