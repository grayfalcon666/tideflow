<script setup lang="ts">
import type { LearnMode } from '../../types'
import TFIcon from '../common/TFIcon.vue'

const props = defineProps<{
  total: number
  dailyWordsToday: number
  hasMoreBatches: boolean
  mode?: LearnMode
}>()

const emit = defineEmits<{
  continueNextBatch: []
  close: []
}>()
</script>

<template>
  <div class="result">
    <!-- Title -->
    <div class="result-header">
      <TFIcon name="school" :size="20" color="#1ed760" />
      <span>{{ mode === 'type' ? '跟打练习完成' : '本批学习完成' }}</span>
    </div>

    <!-- Stats -->
    <div class="stats-row">
      <div class="stat-item">
        <span class="stat-value">{{ total }}</span>
        <span class="stat-label">{{ mode === 'type' ? '练习单词' : '本批单词' }}</span>
      </div>
      <template v-if="mode !== 'type'">
        <div class="stat-divider" />
        <div class="stat-item">
          <span class="stat-value accent">{{ dailyWordsToday }}</span>
          <span class="stat-label">今日累计</span>
        </div>
      </template>
    </div>

    <!-- Actions -->
    <div class="actions">
      <button
        v-if="hasMoreBatches"
        class="btn-next-batch"
        @click="emit('continueNextBatch')"
      >
        <TFIcon name="arrow_forward" :size="16" />
        继续学习下一批
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

  &.accent { color: #1ed760; }
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

.actions {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.btn-next-batch {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
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
  transition: background 0.15s;

  &:hover { background: #1fd665; }
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
