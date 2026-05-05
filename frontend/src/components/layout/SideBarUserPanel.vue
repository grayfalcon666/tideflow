<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../../stores/auth'
import { useLayoutStore } from '../../stores/layout'
import TFIcon from '../common/TFIcon.vue'

const router = useRouter()
const authStore = useAuthStore()
const layoutStore = useLayoutStore()
const showDropdown = ref(false)

const myId = computed(() => authStore.accountId)

const menuItems = computed(() => [
  { label: '我的作品', route: myId.value ? `/u/${myId.value}` : '/' },
  { label: '我的喜欢', route: myId.value ? `/u/${myId.value}?tab=liked` : '/' },
  { label: '观看历史', route: '/' },
  { label: '退出登录', action: 'logout' },
])

const navigate = (item: { route?: string; action?: string }) => {
  showDropdown.value = false
  if (item.action === 'logout') {
    authStore.logout()
    router.push('/account')
  } else if (item.route) {
    router.push(item.route)
  }
}
</script>

<template>
  <div class="user-panel" v-show="!layoutStore.sidebarCollapsed">
    <q-btn flat no-caps class="user-btn" @click="showDropdown = !showDropdown">
      <q-avatar size="32px">
        <img :src="authStore.avatarUrl || '/default-avatar.svg'" />
      </q-avatar>
      <span v-show="!layoutStore.sidebarCollapsed" class="username">{{ authStore.username || '未登录' }}</span>
      <TFIcon v-if="!layoutStore.sidebarCollapsed" name="arrow_drop_down" :size="20" />
    </q-btn>

    <q-menu
      v-model="showDropdown"
      anchor="top middle"
      self="top right"
      class="user-dropdown"
    >
      <div class="dropdown-user-info">
        <q-avatar size="48px">
          <img :src="authStore.avatarUrl || '/default-avatar.svg'" />
        </q-avatar>
        <div class="dropdown-username">{{ authStore.username }}</div>
        <div class="dropdown-stats">
          <span>{{ authStore.followerCount }} 粉丝</span>
        </div>
      </div>
      <q-separator />
      <q-list>
        <q-item
          v-for="item in menuItems"
          :key="item.label"
          clickable
          @click="navigate(item)"
        >
          <q-item-section>{{ item.label }}</q-item-section>
        </q-item>
      </q-list>
    </q-menu>
  </div>
</template>

<style scoped lang="scss">
.user-panel {
  padding: var(--space-3) var(--space-3);
}

.user-btn {
  width: 100%;
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-2);
  border-radius: var(--radius-md);
  color: var(--text-base);
  justify-content: flex-start;

  .sidebar.collapsed & {
    justify-content: center;
    padding: var(--space-2);
  }
}

.username {
  flex: 1;
  text-align: left;
  font-size: 14px;
  font-weight: 400;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dropdown-user-info {
  padding: var(--space-4);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-2);
  min-width: 180px;
}

.dropdown-username {
  font-size: 16px;
  font-weight: 700;
  color: var(--text-base);
}

.dropdown-stats {
  font-size: 12px;
  color: var(--text-secondary);
}
</style>
