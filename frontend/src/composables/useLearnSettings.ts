import { ref, watch } from 'vue'

const CHUNK_SIZE_KEY = 'tideflow_learn_chunk_size'
const DAILY_GOAL_KEY = 'tideflow_learn_daily_goal'
const DEFAULT_CHUNK = 15
const DEFAULT_GOAL = 20

const chunkSize = ref<number>(loadNumber(CHUNK_SIZE_KEY, DEFAULT_CHUNK, 1, 100))
const dailyGoal = ref<number>(loadNumber(DAILY_GOAL_KEY, DEFAULT_GOAL, 0, 200))

function loadNumber(key: string, fallback: number, min: number, max: number): number {
  try {
    const raw = localStorage.getItem(key)
    if (raw) {
      const n = parseInt(raw, 10)
      if (n >= min && n <= max) return n
    }
  } catch { /* ignore */ }
  return fallback
}

export function useLearnSettings() {
  watch(chunkSize, (val) => {
    try { localStorage.setItem(CHUNK_SIZE_KEY, String(val)) } catch { /* ignore */ }
  })

  watch(dailyGoal, (val) => {
    try { localStorage.setItem(DAILY_GOAL_KEY, String(val)) } catch { /* ignore */ }
  })

  function setChunkSize(n: number) {
    if (n >= 1 && n <= 100) chunkSize.value = n
  }

  function setDailyGoal(n: number) {
    if (n >= 0 && n <= 200) dailyGoal.value = n
  }

  return { chunkSize, setChunkSize, dailyGoal, setDailyGoal }
}
