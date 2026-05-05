<script setup lang="ts">
import { computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '../../stores/auth'
import { useNotificationStore } from '../../stores/notification'
import { useLayoutStore } from '../../stores/layout'
import TFIcon from '../common/TFIcon.vue'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const notifStore = useNotificationStore()
const layoutStore = useLayoutStore()

interface MenuItem {
  label: string
  icon: string
  route: string
  requiresAuth?: boolean
  badge?: () => number
}

const menuItems = computed<MenuItem[]>(() => [
  {
    label: '推荐',
    icon: 'compass',
    route: '/',
  },
  {
    label: '关注',
    icon: 'people',
    route: '/',
    requiresAuth: true,
  },
  {
    label: '热门',
    icon: 'local_fire_department',
    route: '/hot',
  },
  {
    label: '通知',
    icon: 'notifications',
    route: '/notifications',
    requiresAuth: true,
    badge: () => notifStore.unreadCount,
  },
  {
    label: '私信',
    icon: 'chat_bubble',
    route: '/messages',
    requiresAuth: true,
  },
  {
    label: '发布',
    icon: 'add_circle_outline',
    route: '/upload',
    requiresAuth: true,
  },
])

const isActive = (item: MenuItem) => {
  if (item.route === '/') return route.path === '/'
  return route.path.startsWith(item.route)
}

const navigate = (item: MenuItem) => {
  if (item.requiresAuth && !authStore.isLoggedIn) {
    router.push(`/account?redirect=${item.route}`)
    return
  }
  router.push(item.route)
}
</script>

<template>
  <nav class="sidebar-menu">
    <div
      v-for="item in menuItems"
      :key="item.label"
      class="menu-item"
      :class="{ active: isActive(item), 'has-badge': item.badge && item.badge() > 0 }"
      @click="navigate(item)"
    >
      <TFIcon :name="item.icon" :size="24" />
      <span v-show="!layoutStore.sidebarCollapsed" class="menu-label">{{ item.label }}</span>
      <q-badge
        v-if="item.badge && item.badge() > 0"
        class="menu-badge"
        color="negative"
        :label="item.badge() > 99 ? '99+' : item.badge()"
      />
    </div>
  </nav>
</template>

<style scoped lang="scss">
.sidebar-menu {
  flex: 1;
  overflow-y: auto;
  padding: var(--space-2) 0;
}

.menu-item {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-4);
  cursor: pointer;
  color: var(--text-secondary);
  transition: color var(--transition-fast), padding var(--transition-normal);
  position: relative;
  user-select: none;

  .sidebar.collapsed & {
    justify-content: center;
    padding: var(--space-3) 0;
  }

  &:hover {
    color: var(--text-base);
  }

  &.active {
    color: var(--text-base);
    font-weight: 700;
  }

  &.has-badge {
    .menu-label {
      padding-right: 28px;
    }
  }
}

.menu-label {
  font-size: 14px;
  font-weight: 400;
  white-space: nowrap;
}

.menu-badge {
  position: absolute;
  right: 16px;
  top: 8px;
  font-size: 10px;
  min-width: 16px;
  height: 16px;
  border-radius: var(--radius-full);

  .sidebar.collapsed & {
    right: 4px;
    top: 4px;
  }
}
</style>
