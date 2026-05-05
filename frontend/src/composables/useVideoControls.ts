import { ref, onMounted, onUnmounted } from 'vue'

export function useVideoControls(onKeyChange?: (key: string) => void) {
  const isMuted = ref(true)

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
