<script setup lang="ts">
import { ref, computed, nextTick } from 'vue'
import type { LearnWord } from '../../types'
import TFIcon from '../common/TFIcon.vue'

const props = defineProps<{
  words: LearnWord[]
}>()

const emit = defineEmits<{
  done: [correctIds: string[], wrongWords: LearnWord[]]
  openCaptions: [word: string]
}>()

// 状态
const currentIndex = ref(0)
const userInput = ref('')
const inputRef = ref<HTMLInputElement>()
const correctCount = ref(0)
const wrongCount = ref(0)
const wrongWords = ref<LearnWord[]>([])
const feedbackState = ref<'idle' | 'correct' | 'wrong'>('idle')
const showDefinition = ref(false)  // 切换显示释义/翻译
const showUkPhone = ref(false)     // 切换英式/美式音标
const shuffled = ref(false)
const shuffledWords = ref<LearnWord[]>([])

// 随机排序
const toggleShuffle = () => {
  if (shuffled.value) {
    shuffledWords.value = [...props.words]
  } else {
    shuffledWords.value = [...props.words].sort(() => Math.random() - 0.5)
  }
  shuffled.value = !shuffled.value
  currentIndex.value = 0
  resetState()
}

// 初始化
const displayWords = computed(() => shuffled.value ? shuffledWords.value : props.words)
const currentWord = computed(() => displayWords.value[currentIndex.value])
const total = computed(() => displayWords.value.length)
const progress = computed(() => total.value > 0 ? (currentIndex.value / total.value) * 100 : 0)

const phone = computed(() => {
  if (!currentWord.value) return ''
  return showUkPhone.value ? currentWord.value.ukphone : currentWord.value.usphone
})

const hint = computed(() => {
  if (!currentWord.value) return ''
  return showDefinition.value ? currentWord.value.definition : currentWord.value.translation
})

// 重置状态
const resetState = () => {
  userInput.value = ''
  feedbackState.value = 'idle'
  nextTick(() => inputRef.value?.focus())
}

// 自动聚焦
const focusInput = () => nextTick(() => inputRef.value?.focus())

// 判定
const normalize = (s: string) => s.trim().toLowerCase()

const handleSubmit = () => {
  if (feedbackState.value !== 'idle' || !currentWord.value) return

  const expected = normalize(currentWord.value.value)
  const given = normalize(userInput.value)

  if (given === expected) {
    feedbackState.value = 'correct'
    correctCount.value++
    setTimeout(() => {
      goNext()
    }, 800)
  } else {
    feedbackState.value = 'wrong'
    wrongCount.value++
    if (!wrongWords.value.find(w => w.value === currentWord.value!.value)) {
      wrongWords.value.push(currentWord.value)
    }
  }
}

const goNext = () => {
  if (currentIndex.value < total.value - 1) {
    currentIndex.value++
    resetState()
  } else {
    // 完成
    const correct = displayWords.value
      .filter(w => !wrongWords.value.find(ww => ww.value === w.value))
      .map(w => w.value)
    emit('done', correct, wrongWords.value)
  }
}

const handleKeydown = (e: KeyboardEvent) => {
  if (e.key === 'Enter') {
    if (feedbackState.value === 'idle') {
      handleSubmit()
    } else if (feedbackState.value === 'wrong') {
      goNext()
    }
  }
  if (e.key === 'Escape') {
    // 忽略
  }
}

// 初始化
resetState()
focusInput()
</script>

<template>
  <div class="spelling">
    <!-- 顶部进度 -->
    <div class="progress-bar-wrap">
      <div class="progress-bar">
        <div class="progress-fill" :style="{ width: progress + '%' }" />
      </div>
      <div class="progress-meta">
        <span class="progress-count">{{ currentIndex + 1 }}/{{ total }}</span>
        <div class="score-badges">
          <span class="score-correct">
            <TFIcon name="check" :size="12" color="#1ed760" />
            {{ correctCount }}
          </span>
          <span class="score-wrong">
            <TFIcon name="close" :size="12" color="#f3727f" />
            {{ wrongCount }}
          </span>
        </div>
      </div>
    </div>

    <!-- 随机按钮 -->
    <div class="toolbar">
      <button class="tool-btn" :class="{ active: shuffled }" @click="toggleShuffle">
        <TFIcon name="shuffle" :size="16" />
        <span>随机</span>
      </button>
    </div>

    <!-- 单词卡片 -->
    <div class="spelling-card" :class="`feedback-${feedbackState}`">
      <!-- 音标 -->
      <div class="phonetic-row">
        <span class="phoneme" @click="showUkPhone = !showUkPhone">
          {{ phone || '—' }}
          <span class="phone-toggle">{{ showUkPhone ? '美' : '英' }}</span>
        </span>
        <span class="pos-tag">{{ currentWord?.pos }}</span>
      </div>

      <!-- 提示（释义/翻译） -->
      <div class="hint-row" @click="showDefinition = !showDefinition">
        <span class="hint-text">{{ hint || '—' }}</span>
        <span class="hint-toggle">{{ showDefinition ? '译' : '义' }}</span>
      </div>

      <!-- 输入区 -->
      <div class="input-area">
        <input
          ref="inputRef"
          v-model="userInput"
          class="spell-input"
          :class="{ 'input-correct': feedbackState === 'correct', 'input-wrong': feedbackState === 'wrong' }"
          type="text"
          autocomplete="off"
          autocorrect="off"
          autocapitalize="off"
          spellcheck="false"
          :disabled="feedbackState !== 'idle'"
          @keydown="handleKeydown"
        />
        <button
          class="submit-btn"
          :class="{ 'submit-correct': feedbackState === 'correct', 'submit-wrong': feedbackState === 'wrong' }"
          @click="feedbackState === 'idle' ? handleSubmit() : (feedbackState === 'wrong' ? goNext() : null)"
        >
          <template v-if="feedbackState === 'idle'">确认</template>
          <template v-else-if="feedbackState === 'correct'">
            <TFIcon name="check" :size="18" color="#000" />
          </template>
          <template v-else>下一个</template>
        </button>
      </div>

      <!-- 正确时的动画反馈 -->
      <div v-if="feedbackState === 'correct'" class="feedback-overlay correct">
        <TFIcon name="check_circle" :size="48" color="#1ed760" />
      </div>

      <!-- 错误时的正确答案 -->
      <div v-if="feedbackState === 'wrong'" class="wrong-answer">
        <span class="correct-label">正确答案：</span>
        <span class="correct-word">{{ currentWord?.value }}</span>
      </div>

      <!-- 语境按钮 -->
      <button class="context-btn" @click="emit('openCaptions', currentWord?.value ?? '')">
        <TFIcon name="lightbulb" :size="16" color="#1ed760" />
        <span>查看语境</span>
      </button>
    </div>
  </div>
</template>

<style scoped lang="scss">
.spelling {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.progress-bar-wrap {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.progress-bar {
  height: 4px;
  background: #2a2a2a;
  border-radius: 2px;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: #1ed760;
  border-radius: 2px;
  transition: width 0.3s ease;
}

.progress-meta {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.progress-count {
  font-size: 13px;
  color: #b3b3b3;
}

.score-badges {
  display: flex;
  gap: 12px;
}

.score-correct,
.score-wrong {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
  font-weight: 600;
}

.toolbar {
  display: flex;
  justify-content: flex-end;
}

.tool-btn {
  display: flex;
  align-items: center;
  gap: 5px;
  background: #1f1f1f;
  border: none;
  border-radius: 9999px;
  padding: 5px 14px;
  font-size: 12px;
  color: #b3b3b3;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;

  &.active {
    color: #1ed760;
    background: rgba(30, 215, 96, 0.12);
  }
  &:hover { color: #fff; }
}

.spelling-card {
  position: relative;
  background: #181818;
  border-radius: 12px;
  padding: 24px 20px;
  display: flex;
  flex-direction: column;
  gap: 16px;
  border: 2px solid transparent;
  transition: border-color 0.2s;

  &.feedback-correct { border-color: #1ed760; }
  &.feedback-wrong { border-color: #f3727f; }
}

.phonetic-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.phoneme {
  font-size: 22px;
  font-weight: 700;
  color: #fff;
  cursor: pointer;
  display: flex;
  align-items: baseline;
  gap: 6px;
}

.phone-toggle {
  font-size: 10px;
  font-weight: 400;
  color: #666;
  background: #252525;
  padding: 2px 6px;
  border-radius: 4px;
}

.pos-tag {
  font-size: 12px;
  color: #b3b3b3;
  font-style: italic;
}

.hint-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  cursor: pointer;
  padding: 8px 12px;
  background: #1f1f1f;
  border-radius: 8px;
}

.hint-text {
  font-size: 15px;
  color: #e0e0e0;
}

.hint-toggle {
  font-size: 11px;
  color: #666;
  background: #252525;
  padding: 2px 6px;
  border-radius: 4px;
}

.input-area {
  display: flex;
  gap: 10px;
}

.spell-input {
  flex: 1;
  background: #121212;
  border: 1px solid #333;
  border-radius: 8px;
  padding: 12px 16px;
  font-size: 18px;
  font-weight: 700;
  color: #fff;
  outline: none;
  transition: border-color 0.2s;

  &::placeholder { color: #444; font-weight: 400; }
  &:focus { border-color: #555; }
  &.input-correct { border-color: #1ed760; }
  &.input-wrong { border-color: #f3727f; }
}

.submit-btn {
  background: #1ed760;
  color: #000;
  border: none;
  border-radius: 8px;
  padding: 0 20px;
  font-size: 14px;
  font-weight: 700;
  cursor: pointer;
  transition: background 0.15s;

  &.submit-correct { background: #1ed760; }
  &.submit-wrong { background: #f3727f; color: #fff; }
  &:hover { filter: brightness(1.1); }
}

.feedback-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.5);
  border-radius: 12px;
  animation: fadeIn 0.2s ease;

  &.correct { animation: popIn 0.3s ease; }
}

@keyframes fadeIn { from { opacity: 0; } to { opacity: 1; } }
@keyframes popIn {
  0% { transform: scale(0.8); opacity: 0; }
  100% { transform: scale(1); opacity: 1; }
}

.wrong-answer {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 12px;
  background: rgba(243, 114, 127, 0.1);
  border-radius: 8px;
}

.correct-label {
  font-size: 13px;
  color: #f3727f;
}

.correct-word {
  font-size: 16px;
  font-weight: 700;
  color: #fff;
}

.context-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  background: transparent;
  border: 1px solid #333;
  border-radius: 9999px;
  padding: 7px 16px;
  font-size: 12px;
  color: #b3b3b3;
  cursor: pointer;
  transition: border-color 0.15s, color 0.15s;

  &:hover { border-color: #1ed760; color: #1ed760; }
}
</style>