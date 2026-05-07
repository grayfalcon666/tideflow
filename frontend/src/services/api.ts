import axios, { AxiosInstance, AxiosError } from 'axios'
import { Notify } from 'quasar'
import type { Router } from 'vue-router'

const API_BASE = '/api/v1'

const api: AxiosInstance = axios.create({
  baseURL: API_BASE,
  timeout: 30000,
})

// Injected dependencies (set via initApi)
let _router: Router | null = null
let _getAuthStore: (() => any) | null = null

export function initApi(router: Router, getAuthStore: () => any) {
  _router = router
  _getAuthStore = getAuthStore
}

// Request interceptor: attach JWT
api.interceptors.request.use((config) => {
  const store = _getAuthStore?.()
  const token = store?.accessToken
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// Response interceptor: handle errors
api.interceptors.response.use(
  (res) => {
    const data = res.data
    if (data.code !== undefined && data.code !== 0) {
      return Promise.reject(new Error(data.message || `API error ${data.code}`))
    }
    return res
  },
  async (error: AxiosError) => {
    const status = error.response?.status
    const data = error.response?.data as any

    if (status === 401) {
      const store = _getAuthStore?.()
      if (store) {
        const ok = await store.refreshAccessToken()
        if (ok && error.config) {
          error.config.headers.Authorization = `Bearer ${store.accessToken}`
          return api(error.config)
        }
        // Refresh failed -> logout and soft redirect
        store.logout(_router ?? undefined)
      }
    } else if (status === 429) {
      Notify.create({ message: '操作太频繁，请稍后再试', type: 'warning', icon: 'warning' })
    } else if (status && status >= 500) {
      Notify.create({ message: '服务器繁忙，请稍后重试', type: 'negative', icon: 'error' })
    } else if (data?.message) {
      Notify.create({ message: data.message, type: 'negative', icon: 'error' })
    }
    return Promise.reject(error)
  }
)

export default api

// ============ Service modules ============
export * as authService from './auth'
export * as userService from './user'
export * as videoService from './video'
export * as feedService from './feed'
export * as commentService from './comment'
export * as notificationService from './notification'
export * as messageService from './message'
export * as tagService from './tag'
