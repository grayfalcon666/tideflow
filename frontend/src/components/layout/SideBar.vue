<script setup lang="ts">
import { useRouter } from 'vue-router'
import SideBarMenu from './SideBarMenu.vue'
import SideBarUserPanel from './SideBarUserPanel.vue'
import SideBarFooter from './SideBarFooter.vue'
import { useLayoutStore } from '../../stores/layout'
import TFIcon from '../common/TFIcon.vue'

const router = useRouter()
const layoutStore = useLayoutStore()

const onToggle = () => {
  layoutStore.toggleSidebar()
}
</script>

<template>
  <aside class="sidebar" :class="{ collapsed: layoutStore.sidebarCollapsed }">
    <div class="sidebar-header">
      <div class="logo-area">
        <TFIcon
        name="waves"
        :size="28"
        color="var(--accent)"
        class="logo-icon"
      />
        <span v-show="!layoutStore.sidebarCollapsed" class="logo-text">TideFlow</span>
      </div>
    </div>

    <SideBarMenu />

    <div class="sidebar-bottom">
      <SideBarUserPanel />
      <SideBarFooter :onToggle="onToggle" />
    </div>
  </aside>
</template>

<style scoped lang="scss">
.sidebar {
  position: fixed;
  left: 0;
  top: 0;
  height: 100svh;
  width: var(--sidebar-width);
  background: var(--bg-surface);
  display: flex;
  flex-direction: column;
  z-index: 100;
  transition: width var(--transition-normal);
  overflow: hidden;

  &.collapsed {
    width: var(--sidebar-width-collapsed);
  }
}

.sidebar-header {
  padding: var(--space-4) var(--space-4);
  display: flex;
  align-items: center;
  height: 64px;
  flex-shrink: 0;
}

.logo-area {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.logo-text {
  font-size: 18px;
  font-weight: 700;
  color: var(--text-base);
  white-space: nowrap;
  letter-spacing: 0.5px;
}

.sidebar-bottom {
  margin-top: auto;
  flex-shrink: 0;
  border-top: 1px solid var(--border);
}

.logo-icon {
  filter: drop-shadow(0 0 8px rgba(30, 215, 96, 0.4));
}
</style>
