<script setup lang="ts">
import { ref, computed } from 'vue'
import type { LearnWord, VocabList, LearnWordsResp, LearnMode, CommitLearningResp } from '../../types'
import { abortBatch } from '../../services/learn'
import { useLearnSettings } from '../../composables/useLearnSettings'
import LearnStepSelectList from './LearnStepSelectList.vue'
import LearnStepWordList from './LearnStepWordList.vue'
import LearnStepSpelling from './LearnStepSpelling.vue'
import LearnStepTyping from './LearnStepTyping.vue'
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

// Step state machine
type Step = 'select' | 'preview' | 'spelling' | 'result'
const step = ref<Step>('select')
const selectedList = ref<VocabList | null>(null)
const spellingWords = ref<LearnWord[]>([])
const totalWordsCompleted = ref(0)
const lastCommitResp = ref<CommitLearningResp | null>(null)
const batchRemaining = ref(0)
const learnMode = ref<LearnMode>('spell')

const { chunkSize } = useLearnSettings()

// Caption drawer
const showCaptionDrawer = ref(false)
const captionWord = ref('')

// Exit confirm
const showExitConfirm = ref(false)

const closePanel = () => {
  emit('update:modelValue', false)
}

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

const confirmExit = async () => {
  showExitConfirm.value = false
  // Abort batch on exit during spell mode (type mode has no batch)
  if (step.value === 'spelling' && learnMode.value === 'spell') {
    await abortBatch().catch(() => {})
  }
  step.value = 'select'
  selectedList.value = null
  spellingWords.value = []
  closePanel()
}

const cancelExit = () => {
  showExitConfirm.value = false
}

const handleOpenCaptions = (word: string) => {
  captionWord.value = word
  showCaptionDrawer.value = true
}

const handleSeekTo = (seconds: number) => {
  props.videoPlayerRef?.seekTo(seconds)
}

// List selection
const handleListSelect = (list: VocabList) => {
  selectedList.value = list
  step.value = 'preview'
}

// Start learning
const handleStartLearning = (words: LearnWord[], batchResp: LearnWordsResp, mode: LearnMode) => {
  spellingWords.value = words
  totalWordsCompleted.value = 0
  lastCommitResp.value = null
  batchRemaining.value = batchResp.batch_remaining
  learnMode.value = mode
  step.value = 'spelling'
}

// Whether there are more batches to continue
const hasMoreBatches = computed(() => {
  return learnMode.value === 'spell' && batchRemaining.value > 0
})

// Spelling done (spell mode: all committed; type mode: all typed correctly)
const handleSpellingDone = (totalWords: number, resp?: CommitLearningResp | null) => {
  totalWordsCompleted.value = totalWords
  lastCommitResp.value = resp ?? null
  if (resp) {
    batchRemaining.value = resp.batch_remaining
  }
  step.value = 'result'
}

// Continue next batch
const handleContinueNextBatch = () => {
  if (!selectedList.value) return
  step.value = 'preview'
}

// Panel width
const panelWidth = computed(() => {
  if (typeof window !== 'undefined' && window.innerWidth < 768) return '100vw'
  return '520px'
})

// Title
const panelTitle = computed(() => {
  switch (step.value) {
    case 'select': return '语境学习'
    case 'preview': return selectedList.value?.name ?? '选择词表'
    case 'spelling': return learnMode.value === 'spell' ? '默写练习' : '跟打练习'
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
      <!-- Header -->
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

      <!-- Body -->
      <div class="panel-body">
        <LearnStepSelectList
          v-if="step === 'select'"
          @select="handleListSelect"
        />

        <LearnStepWordList
          v-else-if="step === 'preview' && selectedList"
          :videoId="videoId"
          :list="selectedList"
          :chunkSize="chunkSize"
          @start="handleStartLearning"
          @openCaptions="handleOpenCaptions"
        />

        <!-- Spell mode -->
        <LearnStepSpelling
          v-else-if="step === 'spelling' && learnMode === 'spell'"
          :words="spellingWords"
          @done="(total, resp) => handleSpellingDone(total, resp)"
          @openCaptions="handleOpenCaptions"
        />

        <!-- Type mode -->
        <LearnStepTyping
          v-else-if="step === 'spelling' && learnMode === 'type'"
          :words="spellingWords"
          @done="(total) => handleSpellingDone(total)"
          @openCaptions="handleOpenCaptions"
        />

        <LearnStepResult
          v-else-if="step === 'result'"
          :total="totalWordsCompleted"
          :dailyWordsToday="lastCommitResp?.daily_words_today ?? 0"
          :hasMoreBatches="hasMoreBatches"
          :mode="learnMode"
          @continueNextBatch="handleContinueNextBatch"
          @close="closePanel"
        />
      </div>

      <!-- Exit confirm dialog -->
      <q-dialog v-model="showExitConfirm" persistent>
        <q-card class="exit-confirm-card">
          <q-card-section class="confirm-body">
            <p>{{ learnMode === 'spell' ? '学习进度不会保存，确定退出吗？' : '确定退出跟打练习吗？' }}</p>
          </q-card-section>
          <q-card-actions align="right">
            <q-btn flat no-caps label="取消" @click="cancelExit" />
            <q-btn flat no-caps label="确定退出" color="negative" @click="confirmExit" />
          </q-card-actions>
        </q-card>
      </q-dialog>

      <!-- Caption drawer -->
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
