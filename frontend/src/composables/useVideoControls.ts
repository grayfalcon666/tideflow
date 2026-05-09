import { ref, watch, onMounted, onUnmounted } from 'vue'

const MUTED_STORAGE_KEY = 'tideflow-video-muted'

function loadMutedPref(): boolean {
  try {
    const v = localStorage.getItem(MUTED_STORAGE_KEY)
    if (v === null) return false // 默认不静音
    return v === 'true'
  } catch {
    return false
  }
}

export function useVideoControls(onKeyChange?: (key: string) => void) {
  const isMuted = ref(loadMutedPref())

  // 持久化用户偏好
  watch(isMuted, (val) => {
    try { localStorage.setItem(MUTED_STORAGE_KEY, String(val)) } catch {}
  })

  const handleKeyDown = (e: KeyboardEvent) => {
    // Skip if focus is in an input
    if ((e.target as HTMLElement).tagName === 'INPUT') return
    if ((e.target as HTMLElement).tagName === 'TEXTAREA') return

    if (e.key === 'm' || e.key === 'M') {
      e.preventDefault()
      isMuted.value = !isMuted.value
    }

    onKeyChange?.(e.key)
  }

  onMounted(() => window.addEventListener('keydown', handleKeyDown))
  onUnmounted(() => window.removeEventListener('keydown', handleKeyDown))

  return { isMuted }
}
