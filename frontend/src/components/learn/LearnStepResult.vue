<script setup lang="ts">
import { computed } from 'vue'
import type { LearnWord } from '../../types'
import TFIcon from '../common/TFIcon.vue'

const props = defineProps<{
  total: number
  correctCount: number
  wrongWords: LearnWord[]
}>()

const emit = defineEmits<{
  retryWrong: [words: LearnWord[]]
  close: []
  openCaptions: [word: string]
}>()

const accuracy = computed(() => {
  if (props.total === 0) return 0
  return Math.round((props.correctCount / props.total) * 100)
})

const circumference = 2 * Math.PI * 40
const dashOffset = computed(() => circumference * (1 - accuracy.value / 100))
</script>

<template>
  <div class="result">
    <!-- 标题 -->
    <div class="result-header">
      <TFIcon name="school" :size="20" color="#1ed760" />
      <span>学习完成</span>
    </div>

    <!-- 正确率圆环 -->
    <div class="accuracy-ring">
      <svg width="100" height="100" viewBox="0 0 100 100">
        <circle
          cx="50" cy="50" r="40"
          fill="none"
          stroke="#2a2a2a"
          stroke-width="8"
        />
        <circle
          cx="50" cy="50" r="40"
          fill="none"
          stroke="#1ed760"
          stroke-width="8"
          stroke-linecap="round"
          :stroke-dasharray="circumference"
          :stroke-dashoffset="dashOffset"
          transform="rotate(-90 50 50)"
          style="transition: stroke-dashoffset 1s ease"
        />
      </svg>
      <div class="ring-label">
        <span class="ring-pct">{{ accuracy }}%</span>
        <span class="ring-sub">正确率</span>
      </div>
    </div>

    <!-- 统计数据 -->
    <div class="stats-row">
      <div class="stat-item">
        <span class="stat-value correct">{{ correctCount }}</span>
        <span class="stat-label">正确</span>
      </div>
      <div class="stat-divider" />
      <div class="stat-item">
        <span class="stat-value wrong">{{ wrongWords.length }}</span>
        <span class="stat-label">错误</span>
      </div>
      <div class="stat-divider" />
      <div class="stat-item">
        <span class="stat-value">{{ total }}</span>
        <span class="stat-label">总计</span>
      </div>
    </div>

    <!-- 错词列表 -->
    <div v-if="wrongWords.length > 0" class="wrong-section">
      <div class="section-label">需要复习</div>
      <div class="wrong-list">
        <div
          v-for="word in wrongWords"
          :key="word.value"
          class="wrong-item"
        >
          <div class="wrong-info">
            <span class="wrong-word">{{ word.value }}</span>
            <span class="wrong-trans">{{ word.translation }}</span>
          </div>
          <button class="context-btn" @click="emit('openCaptions', word.value)">
            <TFIcon name="lightbulb" :size="16" color="#1ed760" />
          </button>
        </div>
      </div>
    </div>

    <!-- 操作按钮 -->
    <div class="actions">
      <button
        v-if="wrongWords.length > 0"
        class="btn-retry"
        @click="emit('retryWrong', wrongWords)"
      >
        <TFIcon name="shuffle" :size="16" />
        重练错词
      </button>
      <button class="btn-close" @click="emit('close')">
        关闭面板
      </button>
    </div>
  </div>
</template>

<style scoped lang="scss">
.result {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.result-header {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 18px;
  font-weight: 700;
  color: #fff;
}

.accuracy-ring {
  position: relative;
  width: 100px;
  height: 100px;
  margin: 0 auto;
}

.ring-label {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}

.ring-pct {
  font-size: 22px;
  font-weight: 700;
  color: #1ed760;
}

.ring-sub {
  font-size: 11px;
  color: #b3b3b3;
}

.stats-row {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 24px;
  background: #181818;
  border-radius: 12px;
  padding: 16px;
}

.stat-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
}

.stat-value {
  font-size: 22px;
  font-weight: 700;
  color: #fff;

  &.correct { color: #1ed760; }
  &.wrong { color: #f3727f; }
}

.stat-label {
  font-size: 12px;
  color: #b3b3b3;
}

.stat-divider {
  width: 1px;
  height: 40px;
  background: #2a2a2a;
}

.wrong-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.section-label {
  font-size: 13px;
  font-weight: 600;
  color: #f3727f;
  text-transform: uppercase;
  letter-spacing: 1px;
}

.wrong-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-height: 200px;
  overflow-y: auto;
}

.wrong-item {
  display: flex;
  align-items: center;
  gap: 12px;
  background: #181818;
  border-radius: 8px;
  padding: 10px 14px;
}

.wrong-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.wrong-word {
  font-size: 15px;
  font-weight: 700;
  color: #fff;
}

.wrong-trans {
  font-size: 12px;
  color: #b3b3b3;
}

.context-btn {
  background: transparent;
  border: none;
  cursor: pointer;
  padding: 6px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  transition: background 0.15s;

  &:hover { background: rgba(30, 215, 96, 0.15); }
}

.actions {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.btn-retry {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  background: rgba(30, 215, 96, 0.15);
  color: #1ed760;
  border: 1px solid rgba(30, 215, 96, 0.4);
  border-radius: 9999px;
  padding: 12px 24px;
  font-size: 14px;
  font-weight: 700;
  letter-spacing: 1.4px;
  text-transform: uppercase;
  cursor: pointer;
  transition: background 0.15s;

  &:hover { background: rgba(30, 215, 96, 0.25); }
}

.btn-close {
  width: 100%;
  background: #1f1f1f;
  color: #b3b3b3;
  border: none;
  border-radius: 9999px;
  padding: 12px 24px;
  font-size: 14px;
  font-weight: 600;
  letter-spacing: 1px;
  text-transform: uppercase;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;

  &:hover { background: #2a2a2a; color: #fff; }
}
</style>