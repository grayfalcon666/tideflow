<script setup lang="ts">
import { ref, provide, computed } from 'vue'
import type { LearnWord, VocabList } from '../../types'
import LearnStepSelectList from './LearnStepSelectList.vue'
import LearnStepWordList from './LearnStepWordList.vue'
import LearnStepSpelling from './LearnStepSpelling.vue'
import LearnStepResult from './LearnStepResult.vue'
import LearnCaptionDrawer from './LearnCaptionDrawer.vue'
import TFIcon from '../common/TFIcon.vue'

const props = defineProps<{
  modelValue: boolean
  videoId: number
  videoPlayerRef?: any
}>()

const emit = defineEmits<{
  'update:modelValue': [val: boolean]
}>()

// 步骤状态机
type Step = 'select' | 'preview' | 'spelling' | 'result'
const step = ref<Step>('select')
const selectedList = ref<VocabList | null>(null)
const spellingWords = ref<LearnWord[]>([])
const resultCorrectCount = ref(0)
const resultWrongWords = ref<LearnWord[]>([])

// 语境抽屉
const showCaptionDrawer = ref(false)
const captionWord = ref('')

// 退出确认
const showExitConfirm = ref(false)

// 关闭面板
const closePanel = () => {
  emit('update:modelValue', false)
}

// 返回按钮
const goBack = () => {
  if (step.value === 'preview') {
    step.value = 'select'
    selectedList.value = null
  } else if (step.value === 'spelling') {
    showExitConfirm.value = true
  } else if (step.value === 'result') {
    closePanel()
  }
}

const confirmExit = () => {
  showExitConfirm.value = false
  closePanel()
}

const cancelExit = () => {
  showExitConfirm.value = false
}

// 打开语境抽屉
const handleOpenCaptions = (word: string) => {
  captionWord.value = word
  showCaptionDrawer.value = true
}

// 语境跳转视频
const handleSeekTo = (seconds: number) => {
  props.videoPlayerRef?.seekTo(seconds)
}

// 词表选择
const handleListSelect = (list: VocabList) => {
  selectedList.value = list
  step.value = 'preview'
}

// 开始学习
const handleStartLearning = (words: LearnWord[]) => {
  spellingWords.value = words
  resultCorrectCount.value = 0
  resultWrongWords.value = []
  step.value = 'spelling'
}

// 完成拼写
const handleSpellingDone = (correctIds: string[], wrongWords: LearnWord[]) => {
  resultCorrectCount.value = correctIds.length
  resultWrongWords.value = wrongWords
  step.value = 'result'
}

// 重练错词
const handleRetryWrong = (words: LearnWord[]) => {
  spellingWords.value = words
  resultCorrectCount.value = 0
  resultWrongWords.value = []
  step.value = 'spelling'
}

// 面板宽度
const panelWidth = computed(() => {
  if (typeof window !== 'undefined' && window.innerWidth < 768) return '100vw'
  return '520px'
})

// 标题
const panelTitle = computed(() => {
  switch (step.value) {
    case 'select': return '语境学习'
    case 'preview': return selectedList.value?.name ?? '选择词表'
    case 'spelling': return '拼写练习'
    case 'result': return '学习结果'
  }
})
</script>

<template>
  <q-dialog
    :model-value="modelValue"
    @update:model-value="emit('update:modelValue', $event)"
    position="right"
    :max-width="panelWidth"
    :max-height="'100vh'"
    seamless
  >
    <div class="learn-panel">
      <!-- 头部 -->
      <div class="panel-header">
        <button v-if="step !== 'select'" class="back-btn" @click="goBack">
          <TFIcon name="arrow_back" :size="20" />
        </button>
        <div class="header-title">
          <TFIcon name="school" :size="20" color="#1ed760" />
          <span>{{ panelTitle }}</span>
        </div>
        <button class="close-btn" @click="step === 'spelling' ? (showExitConfirm = true) : closePanel()">
          <TFIcon name="close" :size="20" />
        </button>
      </div>

      <!-- 内容区 -->
      <div class="panel-body">
        <LearnStepSelectList
          v-if="step === 'select'"
          @select="handleListSelect"
        />

        <LearnStepWordList
          v-else-if="step === 'preview' && selectedList"
          :videoId="videoId"
          :list="selectedList"
          @start="handleStartLearning"
          @openCaptions="handleOpenCaptions"
        />

        <LearnStepSpelling
          v-else-if="step === 'spelling'"
          :words="spellingWords"
          @done="handleSpellingDone"
          @openCaptions="handleOpenCaptions"
        />

        <LearnStepResult
          v-else-if="step === 'result'"
          :total="spellingWords.length"
          :correctCount="resultCorrectCount"
          :wrongWords="resultWrongWords"
          @retryWrong="handleRetryWrong"
          @close="closePanel"
          @openCaptions="handleOpenCaptions"
        />
      </div>

      <!-- 退出确认对话框 -->
      <q-dialog v-model="showExitConfirm" persistent>
        <q-card class="exit-confirm-card">
          <q-card-section class="confirm-body">
            <p>学习进度不会保存，确定退出吗？</p>
          </q-card-section>
          <q-card-actions align="right">
            <q-btn flat no-caps label="取消" @click="cancelExit" />
            <q-btn flat no-caps label="确定退出" color="negative" @click="confirmExit" />
          </q-card-actions>
        </q-card>
      </q-dialog>

      <!-- 语境抽屉（子抽屉） -->
      <q-dialog
        v-model="showCaptionDrawer"
        position="right"
        :max-width="420"
        seamless
      >
        <LearnCaptionDrawer
          v-if="captionWord"
          :videoId="videoId"
          :word="captionWord"
          @seekTo="handleSeekTo"
          @close="showCaptionDrawer = false"
        />
      </q-dialog>
    </div>
  </q-dialog>
</template>

<style scoped lang="scss">
.learn-panel {
  width: 520px;
  max-width: 100vw;
  height: 100vh;
  background: #121212;
  display: flex;
  flex-direction: column;
  box-shadow: rgba(0,0,0,0.5) 0px 8px 24px;

  @media (max-width: 768px) {
    width: 100vw;
    height: 100svh;
  }
}

.panel-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px 20px;
  border-bottom: 1px solid rgba(255,255,255,0.08);
  flex-shrink: 0;
}

.back-btn {
  background: transparent;
  border: none;
  cursor: pointer;
  color: #b3b3b3;
  padding: 6px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  &:hover { color: #fff; background: rgba(255,255,255,0.06); }
}

.header-title {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 16px;
  font-weight: 700;
  color: #fff;
}

.close-btn {
  background: transparent;
  border: none;
  cursor: pointer;
  color: #b3b3b3;
  padding: 6px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  &:hover { color: #fff; background: rgba(255,255,255,0.06); }
}

.panel-body {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
  &::-webkit-scrollbar { width: 4px; }
  &::-webkit-scrollbar-track { background: transparent; }
  &::-webkit-scrollbar-thumb { background: #333; border-radius: 2px; }
}

.exit-confirm-card {
  background: #181818;
  color: #fff;
  border-radius: 12px;
  min-width: 280px;
}

.confirm-body {
  padding: 24px;

  p {
    margin: 0;
    font-size: 15px;
    color: #e0e0e0;
  }
}

:deep(.q-dialog__inner--right) {
  top: 0;
  right: 0;
  bottom: 0;

  @media (max-width: 768px) {
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
  }
}
</style>