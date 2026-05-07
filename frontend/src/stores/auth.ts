import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Router } from 'vue-router'
import type { UserInfo } from '../types'
import * as authService from '../services/auth'
import * as userService from '../services/user'

export const useAuthStore = defineStore('auth', () => {
  const accountId = ref<number | null>(null)
  const username = ref<string>('')
  const avatarUrl = ref<string>('')
  const accessToken = ref<string>('')
  const isLoggedIn = ref(false)
  const followerCount = ref(0)
  const initialized = ref(false)

  const isBigV = computed(() => followerCount.value >= 10000)

  const login = async (username: string, password: string) => {
    const resp = await authService.login({ username, password })
    const data = resp.data.data
    if (!data) return
    accessToken.value = data.access_token
    localStorage.setItem('tideflow_refresh_token', data.refresh_token)
    await fetchMe()
  }

  const register = async (username: string, password: string) => {
    const resp = await authService.register({ username, password })
    const data = resp.data.data
    if (!data) return
    accessToken.value = data.access_token
    localStorage.setItem('tideflow_refresh_token', data.refresh_token)
    await fetchMe()
  }

  const logout = (router?: Router) => {
    accountId.value = null
    username.value = ''
    avatarUrl.value = ''
    accessToken.value = ''
    isLoggedIn.value = false
    followerCount.value = 0
    initialized.value = false
    localStorage.removeItem('tideflow_refresh_token')
    if (router) router.push('/account')
  }

  const fetchMe = async () => {
    try {
      const resp = await userService.getMe()
      const u: UserInfo | null = resp.data.data ?? null
      if (!u) return
      accountId.value = u.id
      username.value = u.username
      avatarUrl.value = u.avatar_url
      followerCount.value = u.follower_count
      isLoggedIn.value = true
    } catch {
      // 网络错误等临时问题，不清理 token，等下次刷新重试
      // 只有明确需要登出时才调用 logout（如 token 被后端拉黑）
    }
  }

  const refreshAccessToken = async () => {
    const rt = localStorage.getItem('tideflow_refresh_token')
    if (!rt) return false
    try {
      const resp = await authService.refreshToken(rt)
      const data = resp.data.data
      if (!data?.access_token) return false
      accessToken.value = data.access_token
      return true
    } catch {
      console.error('Refresh token failed')
      return false
    }
  }

  const initFromStorage = async () => {
    const rt = localStorage.getItem('tideflow_refresh_token')
    if (rt) {
      const ok = await refreshAccessToken()
      if (ok) {
        await fetchMe()
      }
    }
    initialized.value = true
  }

  return {
    accountId, username, avatarUrl, accessToken,
    isLoggedIn, isBigV, followerCount, initialized,
    login, register, logout, fetchMe, initFromStorage, refreshAccessToken,
  }
})
