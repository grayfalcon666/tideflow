<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import SideBar from './components/layout/SideBar.vue'
import BottomNav from './components/layout/BottomNav.vue'
import { useLayoutStore } from './stores/layout'

const router = useRouter()
const layoutStore = useLayoutStore()

const isMobile = ref(window.innerWidth < 1024)
const currentRoute = computed(() => router.currentRoute.value.path)

// Bottom nav visible on these routes
const bottomNavRoutes = ['/', '/hot', '/messages', '/notifications']
const showBottomNav = computed(() => {
  if (!isMobile.value) return false
  // FeedPage renders its own BottomNav, skip here
  if (currentRoute.value === '/') return false
  // HotPage and UserPage use the App-level one
  if (['/hot', '/messages', '/notifications'].some(r => currentRoute.value === r)) return true
  if (currentRoute.value.startsWith('/u/')) return true
  return false
})

// Video detail / upload / account / register / 404 — hide nav & sidebar
const hideLayout = computed(() =>
  ['/upload', '/account', '/register'].some(p => currentRoute.value.startsWith(p)) ||
  currentRoute.value === '/:catchAll(.*)' ||
  currentRoute.value.includes('messages/')
)

onMounted(() => {
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
      <router-view :key="$route.fullPath" class="router-view" />
    </main>

    <!-- BottomNav: FeedPage renders its own, HotPage/UserPage use this one -->
    <BottomNav v-if="showBottomNav" />
  </div>
</template>

<style scoped lang="scss">
.tideflow-app {
  display: flex;
  min-height: 100svh;
  width: 100%;
  background: var(--bg-base);
}

.main-content {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  align-items: stretch;
  min-height: 100svh;
  transition: margin-left var(--transition-normal);

  &.with-sidebar {
    margin-left: var(--sidebar-width);
  }

  &.sidebar-collapsed {
    margin-left: var(--sidebar-width-collapsed);
  }
}

.router-view {
  width: 100%;
}
</style>
