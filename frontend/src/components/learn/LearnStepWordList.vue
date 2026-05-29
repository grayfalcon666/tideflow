<script setup lang="ts">
import { ref, computed } from 'vue'
import type { LearnWord, VocabList, LearnWordsResp, LearnMode } from '../../types'
import * as learnService from '../../services/learn'
import TFIcon from '../common/TFIcon.vue'

const props = defineProps<{
  videoId: number
  list: VocabList
  chunkSize?: number
}>()

const emit = defineEmits<{
  start: [words: LearnWord[], batchResp: LearnWordsResp, mode: LearnMode]
  openCaptions: [word: string]
}>()

const words = ref<LearnWord[]>([])
const batchTotal = ref(0)
const batchRemaining = ref(0)
const dailyRemaining = ref<number | null>(null)
const loading = ref(true)
const error = ref('')
const hasStarted = ref(false)
const batchConflict = ref(false)
const mode = ref<LearnMode>('spell')

const load = async () => {
  loading.value = true
  error.value = ''
  batchConflict.value = false
  try {
    const resp = await learnService.getVideoLearnWords(props.videoId, props.list.id, props.chunkSize ?? 15, mode.value)
    const data = resp.data.data
    words.value = data?.words ?? []
    batchTotal.value = data?.batch_total ?? 0
    batchRemaining.value = data?.batch_remaining ?? 0
    dailyRemaining.value = data?.daily_remaining ?? null
  } catch (e: any) {
    if (e?.response?.status === 412) {
      error.value = '词库尚未生成'
    } else if (e?.response?.status === 409) {
      batchConflict.value = true
    } else {
      error.value = '加载单词失败'
    }
  } finally {
    loading.value = false
  }
}

const handleAbortBatch = async () => {
  try {
    await learnService.abortBatch()
    batchConflict.value = false
    words.value = []
    await load()
  } catch { /* ignore */ }
}

const switchMode = async (m: LearnMode) => {
  if (m === mode.value) return
  mode.value = m
  await load()
}

load()

const handleStart = () => {
  hasStarted.value = true
  emit('start', words.value, {
    words: words.value,
    batch_total: batchTotal.value,
    batch_remaining: batchRemaining.value,
    daily_remaining: dailyRemaining.value,
  }, mode.value)
}

const formatTime = (t: string) => {
  if (!t) return ''
  const parts = t.split(':')
  if (parts.length === 3) return `${parts[0]}:${parts[1]}`
  if (parts.length === 2) return `00:${parts[0]}`
  return t
}
</script>

<template>
  <div class="word-list">
    <div class="step-header">
      <TFIcon name="school" :size="20" color="#1ed760" />
      <span class="list-name">{{ list.name }}</span>
    </div>

    <!-- Mode tabs -->
    <div class="mode-tabs">
      <button
        class="mode-tab"
        :class="{ active: mode === 'spell' }"
        @click="switchMode('spell')"
      >
        <TFIcon name="edit" :size="14" />
        默写模式
      </button>
      <button
        class="mode-tab"
        :class="{ active: mode === 'type' }"
        @click="switchMode('type')"
      >
        <TFIcon name="keyboard" :size="14" />
        跟打模式
      </button>
    </div>

    <div v-if="loading" class="state-loading">
      <q-spinner color="accent" />
    </div>

    <!-- Batch conflict (409) -->
    <div v-else-if="batchConflict" class="state-error">
      <TFIcon name="warning" :size="32" color="#f3727f" />
      <span>有未完成的学习批次</span>
      <span class="error-hint">请先完成或放弃当前批次</span>
      <button class="btn-abort" @click="handleAbortBatch">
        <TFIcon name="close" :size="16" />
        放弃当前批次
      </button>
    </div>

    <div v-else-if="error" class="state-error">
      <TFIcon name="error_outline" :size="32" color="#f3727f" />
      <span>{{ error }}</span>
    </div>

    <template v-else>
      <!-- Summary -->
      <div class="word-summary">
        <span class="total-count">{{ words.length }}</span>
        <span class="total-label">个待学单词</span>
        <span v-if="dailyRemaining !== null" class="daily-info">
          今日剩余 {{ dailyRemaining }}
        </span>
        <span v-else-if="mode === 'type'" class="daily-info type-mode-hint">
          跟打模式不计入学习进度
        </span>
      </div>

      <div v-if="words.length === 0" class="state-empty">
        <TFIcon name="check_circle" :size="32" color="#1ed760" />
        <span v-if="dailyRemaining === 0">今日学习目标已完成！</span>
        <span v-else>该视频中暂未出现此词表内的单词</span>
      </div>

      <div v-else class="word-scroll">
        <div
          v-for="word in words"
          :key="word.value"
          class="word-item"
        >
          <div class="word-main">
            <div class="word-text">
              <span class="word-value">{{ word.value }}</span>
              <span class="word-pos">{{ word.pos }}</span>
            </div>
            <div class="word-translation">{{ word.translation }}</div>
            <div v-if="word.first_caption_start" class="word-time">
              <TFIcon name="access_time" :size="12" color="#b3b3b3" />
              <span>{{ formatTime(word.first_caption_start) }}</span>
            </div>
          </div>
          <button class="caption-btn" @click="emit('openCaptions', word.value)">
            <TFIcon name="lightbulb" :size="18" color="#1ed760" />
          </button>
        </div>
      </div>

      <div v-if="words.length > 0" class="action-area">
        <button class="start-btn" @click="handleStart">
          {{ mode === 'spell' ? '开始默写' : '开始跟打' }}
        </button>
      </div>
    </template>
  </div>
</template>

<style scoped lang="scss">
.word-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.step-header {
  display: flex;
  align-items: center;
  gap: 8px;
}

.list-name {
  font-size: 16px;
  font-weight: 600;
  color: #fff;
}

.mode-tabs {
  display: flex;
  gap: 8px;
}

.mode-tab {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  background: #1f1f1f;
  border: 1px solid #333;
  border-radius: 9999px;
  padding: 8px 16px;
  font-size: 13px;
  font-weight: 600;
  color: #b3b3b3;
  cursor: pointer;
  transition: all 0.15s;

  &.active {
    background: rgba(30, 215, 96, 0.15);
    border-color: #1ed760;
    color: #1ed760;
  }
  &:hover:not(.active) {
    border-color: #555;
    color: #fff;
  }
}

.state-loading {
  display: flex;
  justify-content: center;
  padding: 48px;
}

.state-error,
.state-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  padding: 48px 16px;
  color: #b3b3b3;
  font-size: 14px;
  text-align: center;
}

.error-hint {
  font-size: 12px;
  color: #888;
}

.btn-abort {
  display: flex;
  align-items: center;
  gap: 6px;
  background: transparent;
  color: #f3727f;
  border: 1px solid #f3727f;
  border-radius: 9999px;
  padding: 8px 20px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.15s;

  &:hover { background: rgba(243, 114, 127, 0.15); }
}

.word-summary {
  display: flex;
  align-items: baseline;
  gap: 6px;
  padding: 0 4px;
}

.total-count {
  font-size: 28px;
  font-weight: 700;
  color: #1ed760;
}

.total-label {
  font-size: 14px;
  color: #b3b3b3;
}

.daily-info {
  font-size: 12px;
  color: #888;
  margin-left: 8px;
}

.type-mode-hint {
  color: #666;
  font-style: italic;
}

.word-scroll {
  max-height: 360px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 6px;

  &::-webkit-scrollbar { width: 4px; }
  &::-webkit-scrollbar-track { background: transparent; }
  &::-webkit-scrollbar-thumb { background: #333; border-radius: 2px; }
}

.word-item {
  display: flex;
  align-items: center;
  gap: 12px;
  background: #181818;
  border-radius: 8px;
  padding: 12px 14px;
}

.word-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 3px;
  min-width: 0;
}

.word-text {
  display: flex;
  align-items: baseline;
  gap: 6px;
}

.word-value {
  font-size: 15px;
  font-weight: 700;
  color: #fff;
}

.word-pos {
  font-size: 11px;
  color: #b3b3b3;
  font-style: italic;
}

.word-translation {
  font-size: 13px;
  color: #b3b3b3;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.word-time {
  display: flex;
  align-items: center;
  gap: 3px;
  font-size: 11px;
  color: #b3b3b3;
  margin-top: 2px;
}

.caption-btn {
  background: transparent;
  border: none;
  cursor: pointer;
  padding: 6px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  flex-shrink: 0;
  transition: background 0.15s;

  &:hover {
    background: rgba(30, 215, 96, 0.15);
  }
}

.action-area {
  padding-top: 8px;
}

.start-btn {
  width: 100%;
  background: #1ed760;
  color: #000;
  border: none;
  border-radius: 9999px;
  padding: 12px 24px;
  font-size: 14px;
  font-weight: 700;
  letter-spacing: 1.4px;
  text-transform: uppercase;
  cursor: pointer;
  transition: background 0.15s, transform 0.1s;

  &:hover { background: #1fd665; }
  &:active { transform: scale(0.98); }
}
</style>
