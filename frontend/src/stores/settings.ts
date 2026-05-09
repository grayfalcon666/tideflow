import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useSettingsStore = defineStore('settings', () => {
  const rememberPosition = ref(true)
  const likesPublic = ref(true)
  const lastActiveIndex = ref(0)
  const lastActiveTab = ref<'latest' | 'following'>('latest')

  const savePosition = (index: number, tab: string) => {
    if (rememberPosition.value) {
      lastActiveIndex.value = index
      lastActiveTab.value = tab as 'latest' | 'following'
    }
  }

  const reset = () => {
    rememberPosition.value = true
    likesPublic.value = true
    lastActiveIndex.value = 0
    lastActiveTab.value = 'latest'
  }

  return {
    rememberPosition,
    likesPublic,
    lastActiveIndex,
    lastActiveTab,
    savePosition,
    reset,
  }
}, {
  persist: {
    key: 'tideflow-settings',
    storage: localStorage,
  },
})
