import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
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

  const isBigV = computed(() => followerCount.value >= 10000)

  const login = async (username: string, password: string) => {
    const resp = await authService.login({ username, password })
    const data = resp.data.data
    if (!data) return
    accessToken.value = data.access_token
    window.__tideflow_access_token__ = data.access_token
    localStorage.setItem('tideflow_refresh_token', data.refresh_token)
    await fetchMe()
  }

  const register = async (username: string, password: string) => {
    const resp = await authService.register({ username, password })
    const data = resp.data.data
    if (!data) return
    accessToken.value = data.access_token
    window.__tideflow_access_token__ = data.access_token
    localStorage.setItem('tideflow_refresh_token', data.refresh_token)
    await fetchMe()
  }

  const logout = () => {
    accountId.value = null
    username.value = ''
    avatarUrl.value = ''
    accessToken.value = ''
    isLoggedIn.value = false
    followerCount.value = 0
    delete window.__tideflow_access_token__
    localStorage.removeItem('tideflow_refresh_token')
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
      logout()
    }
  }

  const initFromStorage = () => {
    const rt = localStorage.getItem('tideflow_refresh_token')
    if (rt) {
      fetchMe()
    }
  }

  return {
    accountId, username, avatarUrl, accessToken,
    isLoggedIn, isBigV, followerCount,
    login, register, logout, fetchMe, initFromStorage,
  }
})
