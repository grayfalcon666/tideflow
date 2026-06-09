<script setup lang="ts">
import { useRouter } from 'vue-router'
import type { FeedTab } from '../../stores/feed'
import TFIcon from '../common/TFIcon.vue'

defineProps<{
  activeTab: FeedTab
}>()

const emit = defineEmits<{
  (e: 'update:tab', tab: FeedTab): void
}>()

const router = useRouter()
</script>

<template>
  <div class="feed-tab-bar">
    <div class="tab-group">
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
    <div class="tab-actions">
      <div class="search-trigger" @click="router.push('/search')">
        <TFIcon name="search" :size="22" />
      </div>
    </div>
  </div>
</template>

<style scoped lang="scss">
.feed-tab-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-2);
  padding: var(--space-3) var(--space-4);
  background: var(--bg-surface);
  border-bottom: 1px solid var(--border);
}

.tab-group {
  display: flex;
  gap: var(--space-2);
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

.tab-actions {
  display: flex;
  align-items: center;
}

.search-trigger {
  cursor: pointer;
  padding: var(--space-2);
  border-radius: var(--radius-full);
  color: var(--text-secondary);
  display: flex;
  align-items: center;
  transition: all var(--transition-fast);

  &:hover {
    color: var(--text-base);
    background: var(--bg-elevated);
  }
}
</style>
