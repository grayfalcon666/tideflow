import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'
import { useAuthStore } from '../stores/auth'

declare module 'vue-router' {
  interface RouteMeta {
    requiresAuth?: boolean
  }
}

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    component: () => import('../views/FeedPage.vue'),
    meta: {},
  },
  {
    path: '/hot',
    component: () => import('../views/HotPage.vue'),
    meta: {},
  },
  {
    path: '/upload',
    component: () => import('../views/PublishPage.vue'),
    meta: { requiresAuth: true },
  },
  {
    path: '/video/:id',
    component: () => import('../views/VideoDetailPage.vue'),
    meta: {},
  },
  {
    path: '/video/:id/edit',
    component: () => import('../views/VideoEditPage.vue'),
    meta: { requiresAuth: true },
  },
  {
    path: '/u/:id',
    component: () => import('../views/UserProfilePage.vue'),
    meta: {},
  },
  {
    path: '/me',
    redirect: () => {
      const authStore = useAuthStore()
      return authStore.accountId ? `/u/${authStore.accountId}` : '/account'
    },
    meta: { requiresAuth: true },
  },
  {
    path: '/account',
    component: () => import('../views/LoginPage.vue'),
    meta: {},
  },
  {
    path: '/register',
    component: () => import('../views/RegisterPage.vue'),
    meta: {},
  },
  {
    path: '/notifications',
    component: () => import('../views/NotificationPage.vue'),
    meta: { requiresAuth: true },
  },
  {
    path: '/messages',
    component: () => import('../views/ConversationListPage.vue'),
    meta: { requiresAuth: true },
  },
  {
    path: '/messages/:peerId',
    component: () => import('../views/DirectMessagePage.vue'),
    meta: { requiresAuth: true },
  },
  {
    path: '/history',
    component: () => import('../views/HistoryPage.vue'),
    meta: { requiresAuth: true },
  },
  {
    path: '/search',
    component: () => import('../views/SearchPage.vue'),
    meta: {},
  },
  {
    path: '/tag/:tagName',
    component: () => import('../views/TagSearchPage.vue'),
    meta: {},
  },
  {
    path: '/:catchAll(.*)',
    component: () => import('../views/NotFoundPage.vue'),
    meta: {},
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

// Navigation guard (synchronous)
router.beforeEach((to) => {
  const authStore = useAuthStore()
  if (to.meta.requiresAuth && !authStore.isLoggedIn) {
    return `/account?redirect=${to.fullPath}`
  }
})

// Chunk load failure retry: Vite dev server偶尔丢包导致懒加载失败，自动重试一次
router.onError((err, to) => {
  if (err.message?.includes('dynamically imported module')) {
    const key = '__chunk_retry_' + to.path
    if (!sessionStorage.getItem(key)) {
      sessionStorage.setItem(key, '1')
      window.location.reload()
    } else {
      sessionStorage.removeItem(key)
    }
  }
})

export default router