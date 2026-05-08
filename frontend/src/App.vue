<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useQuasar } from 'quasar'
import { useRouter } from 'vue-router'
import SideBar from './components/layout/SideBar.vue'
import BottomNav from './components/layout/BottomNav.vue'
import { useAuthStore } from './stores/auth'
import { useLayoutStore } from './stores/layout'

const $q = useQuasar()
const router = useRouter()
const authStore = useAuthStore()
const layoutStore = useLayoutStore()

const isMobile = ref(window.innerWidth < 1024)
const currentRoute = computed(() => router.currentRoute.value.path)

// Bottom nav visible on these routes
const bottomNavRoutes = ['/', '/hot', '/messages']
const showBottomNav = computed(() => {
  if (isMobile.value && bottomNavRoutes.some(r => currentRoute.value === r)) return true
  if (isMobile.value && currentRoute.value.startsWith('/u/')) return true
  return false
})

// Video detail / upload / account / register / 404 — hide nav & sidebar
const hideLayout = computed(() =>
  ['/video/', '/upload', '/account', '/register'].some(p => currentRoute.value.startsWith(p)) ||
  currentRoute.value === '/:catchAll(.*)' ||
  currentRoute.value.includes('messages/')
)

onMounted(() => {
  authStore.initFromStorage()

  const onResize = () => {
    isMobile.value = window.innerWidth < 1024
  }
  window.addEventListener('resize', onResize)
})
</script>

<template>
  <div class="tideflow-app" :class="{ 'app-mobile': isMobile }">
    <SideBar v-if="!isMobile && !hideLayout" />

    <main
      class="main-content"
      :class="{
        'with-sidebar': !isMobile && !hideLayout && !layoutStore.sidebarCollapsed,
        'sidebar-collapsed': !isMobile && !hideLayout && layoutStore.sidebarCollapsed,
      }"
    >
      <router-view :key="$route.fullPath" />
    </main>

    <BottomNav v-if="showBottomNav && !hideLayout" />
  </div>
</template>

<style scoped lang="scss">
.tideflow-app {
  display: flex;
  min-height: 100svh;
  background: var(--bg-base);
}

.main-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  min-height: 100svh;
  width: 100%;
  transition: margin-left var(--transition-normal);

  &.with-sidebar {
    margin-left: var(--sidebar-width);
  }

  &.sidebar-collapsed {
    margin-left: var(--sidebar-width-collapsed);
  }
}
</style>
