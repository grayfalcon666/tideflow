import { ref, watch } from 'vue'

const STORAGE_KEY = 'tideflow_learn_chunk_size'
const DEFAULT_SIZE = 15

const chunkSize = ref<number>(loadChunkSize())

function loadChunkSize(): number {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (raw) {
      const n = parseInt(raw, 10)
      if (n >= 1 && n <= 100) return n
    }
  } catch { /* ignore */ }
  return DEFAULT_SIZE
}

export function useLearnSettings() {
  watch(chunkSize, (val) => {
    try {
      localStorage.setItem(STORAGE_KEY, String(val))
    } catch { /* ignore */ }
  })

  function setChunkSize(n: number) {
    if (n >= 1 && n <= 100) {
      chunkSize.value = n
    }
  }

  return { chunkSize, setChunkSize }
}
