<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import * as learnService from '../../services/learn'
import type { HabitStatsResp } from '../../types'
import { useLearnSettings } from '../../composables/useLearnSettings'

const data = ref<HabitStatsResp | null>(null)
const loading = ref(true)
const error = ref('')

const showTodayWords = ref(false)
const todayWords = ref<{ word: string; status: number }[]>([])
const todayWordsLoading = ref(false)

const { chunkSize, setChunkSize } = useLearnSettings()

const currentYear = new Date().getFullYear()

// Build a map: dateString -> words_count
const heatmapMap = computed(() => {
  const map = new Map<string, number>()
  if (!data.value) return map
  for (const entry of data.value.heatmap) {
    map.set(entry.date, entry.words_count)
  }
  return map
})

// Generate heatmap weeks for the year
const heatmapWeeks = computed(() => {
  const year = currentYear
  const dayMap = heatmapMap.value

  // Start from the Monday before Jan 1, end at the Sunday after Dec 31
  const firstDay = new Date(year, 0, 1)
  const startDate = new Date(firstDay)
  // Go back to Monday
  const dayOfWeek = startDate.getDay()
  const diff = dayOfWeek === 0 ? 6 : dayOfWeek - 1 // Sun=0 -> Mon=6
  startDate.setDate(startDate.getDate() - diff)

  const weeks: Array<Array<{ date: string; count: number; weekDay: number }>> = []
  const current = new Date(startDate)

  // 53 weeks max
  for (let w = 0; w < 53; w++) {
    const week: Array<{ date: string; count: number; weekDay: number }> = []
    for (let d = 0; d < 7; d++) {
      const dateStr = formatDate(current)
      week.push({
        date: dateStr,
        count: dayMap.get(dateStr) ?? 0,
        weekDay: d,
      })
      current.setDate(current.getDate() + 1)
    }
    weeks.push(week)
    // Stop if we've passed Dec 31
    if (current.getFullYear() > year && current.getMonth() > 0) break
  }

  return weeks
})

// Month labels based on first week of each month
const monthLabels = computed(() => {
  const labels: Array<{ name: string; colIndex: number }> = []
  const weeks = heatmapWeeks.value
  if (weeks.length === 0) return labels

  let lastMonth = -1
  const monthNames = ['1月', '2月', '3月', '4月', '5月', '6月', '7月', '8月', '9月', '10月', '11月', '12月']

  for (let col = 0; col < weeks.length; col++) {
    const firstDayDate = weeks[col][0].date
    const dateYear = parseInt(firstDayDate.substring(0, 4), 10)
    // Skip months from the previous year
    if (dateYear < currentYear) continue
    const month = parseInt(firstDayDate.substring(5, 7), 10) - 1
    if (month !== lastMonth) {
      labels.push({ name: monthNames[month], colIndex: col })
      lastMonth = month
    }
  }
  return labels
})

// Day labels (Mon-Sun)
const dayLabels = ['一', '二', '三', '四', '五', '六', '日']

function formatDate(d: Date): string {
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

function getLevel(count: number): number {
  if (count === 0) return 0
  if (count <= 5) return 1
  if (count <= 15) return 2
  if (count <= 30) return 3
  return 4
}

function getTooltip(date: string, count: number): string {
  if (count === 0) return `${date}\n无学习记录`
  return `${date}\n${count} 个单词`
}

const fetchStats = async () => {
  loading.value = true
  error.value = ''
  try {
    const resp = await learnService.getHabitStats(currentYear)
    data.value = resp.data.data ?? null
  } catch (e: any) {
    error.value = e?.message || '加载失败'
  } finally {
    loading.value = false
  }
}

function lvlLabel(status: number): string {
  const labels = ['新词', '入门', '积累', '熟练', '精通']
  return labels[Math.min(status, 4)] ?? '新词'
}

async function handleTodayWordsClick() {
  if (showTodayWords.value) {
    showTodayWords.value = false
    return
  }
  showTodayWords.value = true
  todayWordsLoading.value = true
  try {
    const resp = await learnService.getTodayWords()
    todayWords.value = resp.data.data ?? []
  } catch {
    todayWords.value = []
  } finally {
    todayWordsLoading.value = false
  }
}

onMounted(() => fetchStats())
</script>

<template>
  <div class="habit-tab">
    <!-- Loading state -->
    <div v-if="loading" class="habit-loading">
      <q-spinner color="accent" size="36px" />
    </div>

    <!-- Error state -->
    <div v-else-if="error" class="habit-error">
      <p>{{ error }}</p>
      <button class="retry-btn" @click="fetchStats">重试</button>
    </div>

    <!-- Content -->
    <template v-else-if="data">
      <!-- Today's stats -->
      <div class="today-stats">
        <div class="stat-card clickable" @click="handleTodayWordsClick">
          <span class="stat-value">{{ data.today_words }}</span>
          <span class="stat-label">今日单词</span>
        </div>
        <div class="stat-card">
          <span class="stat-value">{{ data.today_videos }}</span>
          <span class="stat-label">今日视频</span>
        </div>
        <div class="stat-card">
          <span class="stat-value">{{ data.current_streak }}</span>
          <span class="stat-label">连续天数</span>
        </div>
        <div class="stat-card">
          <span class="stat-value">{{ data.total_days }}</span>
          <span class="stat-label">累计天数</span>
        </div>
      </div>

      <!-- GitHub-style heatmap -->
      <div class="heatmap-container">
        <div class="heatmap-header">
          <span class="heatmap-title">{{ currentYear }} 学习热力图</span>
        </div>

        <div class="heatmap-grid-wrapper">
          <!-- Month labels -->
          <div class="month-row">
            <div class="day-label-col" />
            <div class="month-labels">
              <span
                v-for="(ml, i) in monthLabels"
                :key="i"
                class="month-label"
                :style="{ gridColumn: ml.colIndex + 1 }"
              >{{ ml.name }}</span>
            </div>
          </div>

          <!-- Grid with day labels -->
          <div class="grid-row">
            <div class="day-label-col">
              <span v-for="(dl, i) in dayLabels" :key="i" class="day-label">{{ dl }}</span>
            </div>
            <div class="heatmap-grid">
              <template v-for="(week, wi) in heatmapWeeks" :key="wi">
                <div
                  v-for="(cell, di) in week"
                  :key="`${wi}-${di}`"
                  class="heatmap-cell"
                  :class="`level-${getLevel(cell.count)}`"
                  :title="getTooltip(cell.date, cell.count)"
                />
              </template>
            </div>
          </div>
        </div>

        <!-- Legend -->
        <div class="heatmap-legend">
          <span class="legend-text">少</span>
          <span class="legend-cell level-0" />
          <span class="legend-cell level-1" />
          <span class="legend-cell level-2" />
          <span class="legend-cell level-3" />
          <span class="legend-cell level-4" />
          <span class="legend-text">多</span>
        </div>
      </div>
    </template>

    <!-- Learn settings -->
    <div class="habit-settings">
      <div class="settings-title">学习设置</div>
      <div class="settings-row">
        <div class="settings-label">
          <span class="label-text">每批单词数</span>
          <span class="label-hint">一次学习会话中的单词数量</span>
        </div>
        <div class="chunk-control">
          <button
            class="chunk-btn"
            :disabled="chunkSize <= 5"
            @click="setChunkSize(chunkSize - 5)"
          >-</button>
          <span class="chunk-value">{{ chunkSize }}</span>
          <button
            class="chunk-btn"
            :disabled="chunkSize >= 100"
            @click="setChunkSize(chunkSize + 5)"
          >+</button>
        </div>
      </div>
    </div>
    <!-- Today's words popup -->
    <q-dialog v-model="showTodayWords">
      <q-card class="today-words-card">
        <q-card-section class="today-words-header">
          <span class="today-words-title">今日单词</span>
          <q-space />
          <button class="today-words-close" @click="showTodayWords = false">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="#b3b3b3">
              <path d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/>
            </svg>
          </button>
        </q-card-section>
        <q-card-section class="today-words-body">
          <div v-if="todayWordsLoading" class="today-words-loading">
            <q-spinner color="accent" size="24px" />
          </div>
          <div v-else-if="todayWords.length === 0" class="today-words-empty">
            今天还没有学习记录
          </div>
          <div v-else class="today-words-list">
            <div v-for="item in todayWords" :key="item.word" class="today-word-item">
              <span class="today-word-value">{{ item.word }}</span>
              <span class="word-lvl-badge" :class="`lvl-${item.status}`">{{ lvlLabel(item.status) }}</span>
            </div>
          </div>
        </q-card-section>
      </q-card>
    </q-dialog>
  </div>
</template>

<style scoped lang="scss">
.habit-tab {
  min-height: 200px;
}

.habit-loading {
  display: flex;
  justify-content: center;
  padding: 40px;
}

.habit-error {
  text-align: center;
  padding: 40px;

  p {
    color: var(--text-muted);
    margin-bottom: var(--space-3);
  }
}

.retry-btn {
  background: var(--accent);
  color: #000;
  font-weight: 600;
  border: none;
  border-radius: 8px;
  padding: 6px 20px;
  cursor: pointer;
}

// Today stats
.today-stats {
  display: flex;
  gap: var(--space-3);
  margin-bottom: var(--space-5);
}

.stat-card {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding: var(--space-4) var(--space-2);
  background: var(--bg-card, var(--bg-base));
  border-radius: 12px;
}

.stat-value {
  font-size: 28px;
  font-weight: 700;
  color: var(--accent);
  line-height: 1;
}

.stat-label {
  font-size: 12px;
  color: var(--text-muted);
}

// Heatmap
.heatmap-container {
  background: var(--bg-card, var(--bg-base));
  border-radius: 12px;
  padding: var(--space-4);
  overflow-x: auto;
}

.heatmap-header {
  margin-bottom: var(--space-3);
}

.heatmap-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
}

.heatmap-grid-wrapper {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.month-row {
  display: flex;
  font-size: 11px;
  color: var(--text-muted);
}

.month-labels {
  display: grid;
  grid-template-columns: repeat(53, 14px);
  gap: 3px;
}

.month-label {
  font-size: 11px;
  white-space: nowrap;
}

.grid-row {
  display: flex;
}

.day-label-col {
  display: flex;
  flex-direction: column;
  gap: 3px;
  margin-right: 4px;
}

.day-label {
  display: flex;
  align-items: center;
  height: 14px;
  font-size: 10px;
  color: var(--text-muted);
  width: 20px;
}

.heatmap-grid {
  display: grid;
  grid-auto-flow: column;
  grid-template-rows: repeat(7, 14px);
  gap: 3px;
}

.heatmap-cell {
  width: 14px;
  height: 14px;
  border-radius: 2px;
  cursor: default;
  transition: outline 0.1s;

  &:hover {
    outline: 1px solid rgba(255, 255, 255, 0.5);
  }

  &.level-0 {
    background: #1d1f1d;
  }

  &.level-1 {
    background: #0e4429;
  }

  &.level-2 {
    background: #006d32;
  }

  &.level-3 {
    background: #26a641;
  }

  &.level-4 {
    background: #39d353;
  }
}

// Legend
.heatmap-legend {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 2px;
  margin-top: var(--space-3);
  padding-top: var(--space-2);
}

.legend-text {
  font-size: 10px;
  color: var(--text-muted);
  margin: 0 4px;
}

.legend-cell {
  width: 12px;
  height: 12px;
  border-radius: 2px;

  &.level-0 {
    background: #1d1f1d;
  }

  &.level-1 {
    background: #0e4429;
  }

  &.level-2 {
    background: #006d32;
  }

  &.level-3 {
    background: #26a641;
  }

  &.level-4 {
    background: #39d353;
  }
}

// Settings
.habit-settings {
  margin-top: var(--space-5);
  background: var(--bg-card, var(--bg-base));
  border-radius: 12px;
  padding: var(--space-4);
}

.settings-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: var(--space-3);
}

.settings-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.settings-label {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.label-text {
  font-size: 14px;
  color: var(--text-primary);
}

.label-hint {
  font-size: 12px;
  color: var(--text-muted);
}

.chunk-control {
  display: flex;
  align-items: center;
  gap: 12px;
}

.chunk-btn {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-hover, #1f1f1f);
  border: 1px solid var(--border, #333);
  border-radius: 8px;
  color: var(--text-primary);
  font-size: 18px;
  font-weight: 700;
  cursor: pointer;
  transition: background 0.15s, border-color 0.15s;

  &:hover:not(:disabled) {
    background: var(--accent);
    border-color: var(--accent);
    color: #000;
  }

  &:disabled {
    opacity: 0.3;
    cursor: not-allowed;
  }
}

.chunk-value {
  font-size: 20px;
  font-weight: 700;
  color: var(--accent);
  min-width: 36px;
  text-align: center;
}

// Today words popup
.today-words-card {
  background: #181818;
  color: #fff;
  border-radius: 12px;
  min-width: 300px;
  max-width: 380px;
  max-height: 480px;
  display: flex;
  flex-direction: column;
}

.today-words-header {
  display: flex;
  align-items: center;
  padding: 16px 20px 12px;
}

.today-words-title {
  font-size: 16px;
  font-weight: 700;
}

.today-words-close {
  background: transparent;
  border: none;
  cursor: pointer;
  padding: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 6px;

  &:hover { background: rgba(255, 255, 255, 0.08); }
}

.today-words-body {
  padding: 0 20px 20px;
  overflow-y: auto;
  min-height: 60px;
}

.today-words-loading {
  display: flex;
  justify-content: center;
  padding: 32px;
}

.today-words-empty {
  text-align: center;
  padding: 32px;
  color: #b3b3b3;
  font-size: 14px;
}

.today-words-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.today-word-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 12px;
  background: #1f1f1f;
  border-radius: 8px;
}

.today-word-value {
  font-size: 15px;
  font-weight: 600;
  color: #fff;
}

.word-lvl-badge {
  font-size: 11px;
  font-weight: 600;
  padding: 2px 10px;
  border-radius: 10px;
  flex-shrink: 0;
  line-height: 1.6;

  &.lvl-0 { background: #222; color: #666; }
  &.lvl-1 { background: rgba(243, 114, 127, 0.15); color: #f3727f; }
  &.lvl-2 { background: rgba(249, 168,  37, 0.15); color: #f9a825; }
  &.lvl-3 { background: rgba(102, 187, 106, 0.15); color: #66bb6a; }
  &.lvl-4 { background: rgba(30, 215,  96, 0.15); color: #1ed760; }
}

.stat-card.clickable {
  cursor: pointer;
  transition: background 0.15s;

  &:hover { background: rgba(255, 255, 255, 0.05); }
}
</style>
