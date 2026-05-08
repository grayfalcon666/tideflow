<script setup lang="ts">
import { useLayoutStore } from '../../stores/layout'
import TFIcon from '../common/TFIcon.vue'

defineProps<{ onToggle: () => void }>()

const layoutStore = useLayoutStore()
</script>

<template>
  <div class="sidebar-footer">
    <button
      class="collapse-btn"
      :class="{ collapsed: layoutStore.sidebarCollapsed }"
      @click="$props.onToggle()"
      :title="layoutStore.sidebarCollapsed ? '展开侧边栏' : '收起侧边栏'"
    >
      <TFIcon
        :name="layoutStore.sidebarCollapsed ? 'chevron_right' : 'chevron_left'"
        :size="20"
      />
    </button>
  </div>
</template>

<style scoped lang="scss">
.sidebar-footer {
  padding: var(--space-2) var(--space-3);
  display: flex;
  justify-content: flex-end;
}

.collapse-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: var(--radius-md);
  border: 1px solid var(--border);
  background: var(--bg-elevated);
  color: var(--text-secondary);
  cursor: pointer;
  transition: all var(--transition-fast);

  &:hover {
    background: var(--bg-card);
    color: var(--text-base);
    border-color: var(--accent);
  }

  &.collapsed {
    background: var(--accent);
    border-color: var(--accent);
    color: #000;
  }
}
</style>
