<script setup lang="ts">
import { computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '../../stores/auth'
import { useMessageStore } from '../../stores/message'
import { useNotificationStore } from '../../stores/notification'
import TFIcon from '../common/TFIcon.vue'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const msgStore = useMessageStore()
const notifStore = useNotificationStore()

interface Tab {
  label: string
  icon: string
  route: string
  badge?: () => number
}

const tabs = computed<Tab[]>(() => [
  { label: '首页', icon: 'compass', route: '/' },
  { label: '热门', icon: 'local_fire_department', route: '/hot' },
  { label: '发布', icon: 'add_circle_outline', route: '/upload' },
  {
    label: '消息',
    icon: 'chat_bubble',
    route: '/messages',
    badge: () => msgStore.totalUnread + notifStore.unreadCount,
  },
  {
    label: '我',
    icon: 'person',
    route: authStore.accountId ? `/u/${authStore.accountId}` : '/account',
  },
])

const isActive = (tab: Tab) => {
  if (tab.route === '/') return route.path === '/'
  return route.path.startsWith(tab.route)
}

const navigate = (tab: Tab) => {
  if (tab.route === '/upload' && !authStore.isLoggedIn) {
    router.push('/account?redirect=/upload')
    return
  }
  router.push(tab.route)
}
</script>

<template>
  <nav class="bottom-nav">
    <div
      v-for="tab in tabs"
      :key="tab.label"
      class="nav-tab"
      :class="{ active: isActive(tab) }"
      @click="navigate(tab)"
    >
      <div class="tab-icon-wrap">
        <TFIcon :name="tab.icon" :size="26" />
        <q-badge
          v-if="tab.badge && tab.badge() > 0"
          class="tab-badge"
          color="negative"
          :label="tab.badge() > 99 ? '99+' : tab.badge()"
        />
      </div>
      <span class="tab-label">{{ tab.label }}</span>
    </div>
  </nav>
</template>

<style scoped lang="scss">
.bottom-nav {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  height: var(--bottom-nav-height);
  background: var(--bg-surface);
  border-top: 1px solid var(--border);
  display: flex;
  align-items: center;
  justify-content: space-around;
  z-index: 100;
  padding-bottom: env(safe-area-inset-bottom);
}

.nav-tab {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  cursor: pointer;
  color: var(--text-secondary);
  transition: color var(--transition-fast);
  padding: var(--space-1) var(--space-3);
  min-width: 56px;
  flex: 1;

  &:hover,
  &.active {
    color: var(--text-base);
  }

  &.active .tab-icon-wrap :deep(.q-icon) {
    color: var(--accent);
  }
}

.tab-icon-wrap {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
}

.tab-badge {
  position: absolute;
  top: -4px;
  right: -8px;
  font-size: 9px;
  min-width: 14px;
  height: 14px;
  border-radius: var(--radius-full);
}

.tab-label {
  font-size: 10px;
  font-weight: 400;
}
</style>
