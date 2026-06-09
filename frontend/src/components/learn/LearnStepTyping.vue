<script setup lang="ts">
import { ref, computed, nextTick, onMounted } from 'vue'
import type { LearnWord } from '../../types'
import { useTypingSounds } from '../../composables/useTypingSounds'
import TFIcon from '../common/TFIcon.vue'

const props = defineProps<{
  words: LearnWord[]
}>()

const emit = defineEmits<{
  done: [totalWords: number]
  openCaptions: [word: string]
}>()

const queue = ref<LearnWord[]>([...props.words])
const currentIndex = ref(0)
const correctCount = ref(0)
const wrongCount = ref(0)
const showDefinition = ref(false)
const showUkPhone = ref(false)
const hiddenInput = ref<HTMLInputElement>()

// Per-character tracing state
const charStates = ref<Array<'pending' | 'correct' | 'wrong'>>([])
const cursorPos = ref(0)
const wordDone = ref(false)
const wordHasError = ref(false)

const { playKey, playBackspace, playCorrect, playWrong } = useTypingSounds()

const currentWord = computed(() => queue.value[currentIndex.value])
const targetChars = computed(() => currentWord.value ? currentWord.value.value.split('') : [])

const progress = computed(() => {
  const original = props.words.length
  return original > 0 ? (correctCount.value / original) * 100 : 0
})

const phone = computed(() => {
  if (!currentWord.value) return ''
  return showUkPhone.value ? currentWord.value.ukphone : currentWord.value.usphone
})

const hint = computed(() => {
  if (!currentWord.value) return ''
  return showDefinition.value ? currentWord.value.definition : currentWord.value.translation
})

const initWord = () => {
  charStates.value = targetChars.value.map(() => 'pending')
  cursorPos.value = 0
  wordDone.value = false
  wordHasError.value = false
  nextTick(() => hiddenInput.value?.focus())
}

const goNextWord = () => {
  queue.value.splice(currentIndex.value, 1)
  if (queue.value.length === 0) {
    emit('done', correctCount.value)
    return
  }
  if (currentIndex.value >= queue.value.length) {
    currentIndex.value = queue.value.length - 1
  }
  initWord()
}

const handleKeydown = (e: KeyboardEvent) => {
  if (e.key.length > 1 && e.key !== 'Backspace' && e.key !== 'Enter') return
  if (e.key === 'Backspace' || e.key === ' ') e.preventDefault()

  if (wordDone.value) {
    if (e.key === 'Enter') {
      e.preventDefault()
      if (wordHasError.value) {
        handleRetry()
      } else {
        goNextWord()
      }
    }
    return
  }

  if (e.key === 'Backspace') {
    if (cursorPos.value > 0) {
      cursorPos.value--
      charStates.value[cursorPos.value] = 'pending'
      wordHasError.value = charStates.value.some(s => s === 'wrong')
      playBackspace()
    }
    return
  }

  if (e.key === 'Enter') return

  if (cursorPos.value >= targetChars.value.length) return

  const expected = targetChars.value[cursorPos.value].toLowerCase()
  const given = e.key.toLowerCase()

  if (given === expected) {
    charStates.value[cursorPos.value] = 'correct'
    playKey()
  } else {
    charStates.value[cursorPos.value] = 'wrong'
    wordHasError.value = true
    playWrong()
  }

  cursorPos.value++

  if (cursorPos.value >= targetChars.value.length) {
    wordDone.value = true
    if (!wordHasError.value) {
      correctCount.value++
      playCorrect()
      setTimeout(() => {
        if (wordDone.value) goNextWord()
      }, 600)
    } else {
      wrongCount.value++
    }
  }

  // Clear the hidden input to keep it empty
  if (hiddenInput.value) hiddenInput.value.value = ''
}

const handleRetry = () => {
  initWord()
}

// Keep input focused when clicking anywhere in the card
const focusInput = () => nextTick(() => hiddenInput.value?.focus())

onMounted(() => {
  initWord()
  // Delayed fallback focus in case panel animation blocks initial focus
  setTimeout(() => hiddenInput.value?.focus(), 100)
})
</script>

<template>
  <div class="typing" @click="focusInput">
    <!-- Progress bar -->
    <div class="progress-bar-wrap">
      <div class="progress-bar">
        <div class="progress-fill" :style="{ width: progress + '%' }" />
      </div>
      <div class="progress-meta">
        <span class="progress-count">{{ correctCount }}/{{ props.words.length }}</span>
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

    <!-- Word card -->
    <div class="typing-card">
      <!-- Target word (reference) -->
      <div class="target-word">
        {{ currentWord?.value }}
      </div>

      <!-- Char boxes (tracing area) -->
      <div class="char-boxes">
        <div
          v-for="(ch, i) in targetChars"
          :key="i"
          class="char-box"
          :class="{
            'state-correct': charStates[i] === 'correct',
            'state-wrong': charStates[i] === 'wrong',
            'state-cursor': i === cursorPos && !wordDone,
            'state-pending': charStates[i] === 'pending' && i !== cursorPos,
          }"
        >
          <span v-if="charStates[i] === 'correct'" class="char-display">{{ ch }}</span>
          <span v-else-if="charStates[i] === 'wrong'" class="char-display wrong-char">✗</span>
          <span v-else-if="i === cursorPos" class="cursor-line">|</span>
          <span v-else class="char-placeholder">_</span>
        </div>
      </div>

      <!-- Phonetic -->
      <div class="phonetic-row">
        <span class="phoneme" @click="showUkPhone = !showUkPhone">
          {{ phone || '—' }}
          <span class="phone-toggle">{{ showUkPhone ? '美' : '英' }}</span>
        </span>
        <span class="pos-tag">{{ currentWord?.pos }}</span>
      </div>

      <!-- Hint -->
      <div class="hint-row" @click="showDefinition = !showDefinition">
        <span class="hint-text">{{ hint || '—' }}</span>
        <span class="hint-toggle">{{ showDefinition ? '译' : '义' }}</span>
      </div>

      <!-- Word completion feedback -->
      <div v-if="wordDone && !wordHasError" class="feedback-overlay correct">
        <TFIcon name="check_circle" :size="48" color="#1ed760" />
      </div>

      <!-- Wrong: show retry -->
      <div v-if="wordDone && wordHasError" class="wrong-actions">
        <div class="wrong-hint">
          <TFIcon name="close" :size="16" color="#f3727f" />
          <span>有错误，按 Enter 或点击重试</span>
        </div>
        <button class="retry-btn" @click="handleRetry">
          <TFIcon name="replay" :size="16" />
          重新输入
        </button>
      </div>

      <!-- Hidden input for keyboard capture -->
      <input
        ref="hiddenInput"
        class="hidden-input"
        type="text"
        autocomplete="off"
        autocorrect="off"
        autocapitalize="off"
        spellcheck="false"
        @keydown="handleKeydown"
      />

      <!-- Context button -->
      <button class="context-btn" @click="emit('openCaptions', currentWord?.value ?? '')">
        <TFIcon name="lightbulb" :size="16" color="#1ed760" />
        <span>查看语境</span>
      </button>
    </div>
  </div>
</template>

<style scoped lang="scss">
.typing {
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

.typing-card {
  position: relative;
  background: #181818;
  border-radius: 12px;
  padding: 24px 20px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.target-word {
  text-align: center;
  font-size: 32px;
  font-weight: 700;
  color: #fff;
  letter-spacing: 4px;
  padding: 8px 0;
}

.char-boxes {
  display: flex;
  justify-content: center;
  gap: 6px;
  flex-wrap: wrap;
  padding: 4px 0;
}

.char-box {
  width: 36px;
  height: 44px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
  border: 2px solid #333;
  background: #121212;
  font-size: 20px;
  font-weight: 700;
  transition: all 0.15s;
  user-select: none;

  &.state-cursor {
    border-color: #1ed760;
    background: rgba(30, 215, 96, 0.08);
  }

  &.state-correct {
    border-color: #1ed760;
    background: rgba(30, 215, 96, 0.15);
  }

  &.state-wrong {
    border-color: #f3727f;
    background: rgba(243, 114, 127, 0.15);
    animation: shake 0.3s ease;
  }

  &.state-pending {
    border-color: #2a2a2a;
  }
}

@keyframes shake {
  0%, 100% { transform: translateX(0); }
  25% { transform: translateX(-4px); }
  75% { transform: translateX(4px); }
}

.char-display {
  color: #1ed760;
  font-size: 20px;
  font-weight: 700;

  &.wrong-char {
    color: #f3727f;
    font-size: 16px;
  }
}

.cursor-line {
  color: #1ed760;
  font-weight: 300;
  font-size: 24px;
  animation: blink 1s step-end infinite;
}

@keyframes blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0; }
}

.char-placeholder {
  color: #333;
  font-size: 20px;
}

.phonetic-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.phoneme {
  font-size: 18px;
  font-weight: 600;
  color: #e0e0e0;
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

.hidden-input {
  position: fixed;
  opacity: 0;
  width: 1px;
  height: 1px;
  left: -9999px;
  top: 0;
}

.feedback-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.4);
  border-radius: 12px;
  animation: popIn 0.3s ease;
  pointer-events: none;
}

@keyframes popIn {
  0% { transform: scale(0.8); opacity: 0; }
  100% { transform: scale(1); opacity: 1; }
}

.wrong-actions {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
}

.wrong-hint {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: #f3727f;
}

.retry-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  background: rgba(243, 114, 127, 0.15);
  border: 1px solid rgba(243, 114, 127, 0.4);
  border-radius: 9999px;
  padding: 8px 20px;
  font-size: 13px;
  font-weight: 600;
  color: #f3727f;
  cursor: pointer;
  transition: background 0.15s;

  &:hover { background: rgba(243, 114, 127, 0.25); }
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
