<script setup lang="ts">
import { ref, computed, nextTick } from 'vue'
import type { LearnWord, CommitLearningResp } from '../../types'
import * as learnService from '../../services/learn'
import { useTypingSounds } from '../../composables/useTypingSounds'
import TFIcon from '../common/TFIcon.vue'

const props = defineProps<{
  words: LearnWord[]
}>()

const emit = defineEmits<{
  done: [totalWords: number, resp: CommitLearningResp | null]
  openCaptions: [word: string]
}>()

// Local queue: words to spell (manipulated on wrong answers)
const queue = ref<LearnWord[]>([...props.words])
const currentIndex = ref(0)
const userInput = ref('')
const hiddenInput = ref<HTMLInputElement>()
const correctCount = ref(0)
const wrongCount = ref(0)
const feedbackState = ref<'idle' | 'correct' | 'wrong'>('idle')
const showDefinition = ref(false)
const showUkPhone = ref(false)
const shuffled = ref(false)
const lastCommitResp = ref<CommitLearningResp | null>(null)
const committing = ref(false)

const { playKey, playBackspace, playCorrect, playWrong } = useTypingSounds()

// Shuffle
const toggleShuffle = () => {
  if (shuffled.value) {
    queue.value = [...props.words]
  } else {
    queue.value = [...queue.value].sort(() => Math.random() - 0.5)
  }
  shuffled.value = !shuffled.value
  currentIndex.value = 0
  resetState()
}

const currentWord = computed(() => queue.value[currentIndex.value])
const targetChars = computed(() => currentWord.value ? currentWord.value.value.split('') : [])
const total = computed(() => queue.value.length)
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

const resetState = () => {
  userInput.value = ''
  feedbackState.value = 'idle'
  committing.value = false
  nextTick(() => hiddenInput.value?.focus())
}

const focusInput = () => nextTick(() => hiddenInput.value?.focus())

const normalize = (s: string) => s.trim().toLowerCase()

const handleSubmit = async () => {
  if (feedbackState.value !== 'idle' || !currentWord.value || committing.value) return
  if (userInput.value.length === 0) return
  committing.value = true

  const expected = normalize(currentWord.value.value)
  const given = normalize(userInput.value)
  const isCorrect = given === expected

  try {
    const resp = await learnService.commitLearning({
      word: currentWord.value.value,
      result: isCorrect ? 'correct' : 'wrong',
    })
    lastCommitResp.value = resp.data.data ?? null
  } catch {
    // Silent fail — still proceed with local state
  }

  if (isCorrect) {
    feedbackState.value = 'correct'
    correctCount.value++
    playCorrect()
    committing.value = false
    setTimeout(() => {
      // Remove word from queue (it was answered correctly)
      queue.value.splice(currentIndex.value, 1)
      if (queue.value.length === 0) {
        // All done
        emit('done', correctCount.value, lastCommitResp.value)
        return
      }
      // Adjust index if needed
      if (currentIndex.value >= queue.value.length) {
        currentIndex.value = queue.value.length - 1
      }
      resetState()
    }, 800)
  } else {
    feedbackState.value = 'wrong'
    wrongCount.value++
    playWrong()
    committing.value = false
    // Move wrong word to queue head for immediate retry
    const wrongWord = queue.value.splice(currentIndex.value, 1)[0]
    queue.value.unshift(wrongWord)
    currentIndex.value = 0
    // Don't reset state yet — show correct answer, user presses Enter to retry
  }
}

const handleRetryAfterWrong = () => {
  resetState()
}

const handleKeydown = (e: KeyboardEvent) => {
  if (e.key.length > 1 && e.key !== 'Backspace' && e.key !== 'Enter') return
  if (e.key === ' ' || e.key === 'Backspace') e.preventDefault()

  if (feedbackState.value === 'wrong') {
    if (e.key === 'Enter') handleRetryAfterWrong()
    return
  }

  if (feedbackState.value !== 'idle') return

  if (e.key === 'Enter') {
    handleSubmit()
    return
  }

  if (e.key === 'Backspace') {
    userInput.value = userInput.value.slice(0, -1)
    playBackspace()
    return
  }

  // Append character, but don't exceed target length
  if (userInput.value.length < targetChars.value.length) {
    userInput.value += e.key
    playKey()
  }
}
</script>

<template>
  <div class="spelling">
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

    <!-- Shuffle button -->
    <div class="toolbar">
      <button class="tool-btn" :class="{ active: shuffled }" @click="toggleShuffle">
        <TFIcon name="shuffle" :size="16" />
        <span>随机</span>
      </button>
    </div>

    <!-- Word card -->
    <div class="spelling-card" @click="focusInput">
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

      <!-- Char boxes -->
      <div class="char-boxes">
        <div
          v-for="(ch, i) in targetChars"
          :key="i"
          class="char-box"
          :class="{
            'state-filled': feedbackState === 'idle' && i < userInput.length,
            'state-cursor': feedbackState === 'idle' && i === userInput.length,
            'state-correct': feedbackState === 'correct',
            'state-match': feedbackState === 'wrong' && i < userInput.length && userInput[i].toLowerCase() === ch.toLowerCase(),
            'state-mismatch': feedbackState === 'wrong' && i < userInput.length && userInput[i].toLowerCase() !== ch.toLowerCase(),
            'state-missed': feedbackState === 'wrong' && i >= userInput.length,
            'state-pending': feedbackState === 'idle' && i > userInput.length,
          }"
        >
          <span v-if="feedbackState === 'correct'" class="char-display correct-char">{{ ch }}</span>
          <span v-else-if="feedbackState === 'wrong' && i < userInput.length" class="char-display" :class="userInput[i].toLowerCase() === ch.toLowerCase() ? 'match-char' : 'mismatch-char'">{{ userInput[i] }}</span>
          <span v-else-if="feedbackState === 'wrong'" class="char-display missed-char">{{ ch }}</span>
          <span v-else-if="i < userInput.length" class="char-display">{{ userInput[i] }}</span>
          <span v-else-if="i === userInput.length" class="cursor-line">|</span>
          <span v-else class="char-placeholder">_</span>
        </div>
      </div>

      <!-- Submit button -->
      <div class="submit-row">
        <button
          v-if="feedbackState === 'idle'"
          class="submit-btn"
          :disabled="committing || userInput.length === 0"
          @click="handleSubmit"
        >
          <q-spinner v-if="committing" :size="16" color="#000" />
          <span v-else>确认</span>
        </button>
        <button
          v-else-if="feedbackState === 'wrong'"
          class="retry-btn"
          @click="handleRetryAfterWrong"
        >
          <TFIcon name="replay" :size="16" />
          <span>重新输入</span>
        </button>
      </div>

      <!-- Correct feedback overlay -->
      <div v-if="feedbackState === 'correct'" class="feedback-overlay correct">
        <TFIcon name="check_circle" :size="48" color="#1ed760" />
      </div>

      <!-- Wrong answer display -->
      <div v-if="feedbackState === 'wrong'" class="wrong-answer">
        <TFIcon name="close" :size="16" color="#f3727f" />
        <span class="correct-label">正确答案：</span>
        <span class="correct-word">{{ currentWord?.value }}</span>
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

// Char boxes
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

  &.state-filled {
    border-color: #555;
    background: #1a1a1a;
  }

  &.state-correct {
    border-color: #1ed760;
    background: rgba(30, 215, 96, 0.15);
  }

  &.state-match {
    border-color: #1ed760;
    background: rgba(30, 215, 96, 0.1);
  }

  &.state-mismatch {
    border-color: #f3727f;
    background: rgba(243, 114, 127, 0.15);
    animation: shake 0.3s ease;
  }

  &.state-missed {
    border-color: #f3727f;
    background: rgba(243, 114, 127, 0.08);
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
  color: #e0e0e0;
  font-size: 20px;
  font-weight: 700;

  &.correct-char {
    color: #1ed760;
  }

  &.match-char {
    color: #1ed760;
  }

  &.mismatch-char {
    color: #f3727f;
  }

  &.missed-char {
    color: rgba(243, 114, 127, 0.5);
    font-size: 14px;
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

.hidden-input {
  position: absolute;
  opacity: 0;
  width: 0;
  height: 0;
  pointer-events: none;
}

.submit-row {
  display: flex;
  justify-content: center;
}

.submit-btn {
  background: #1ed760;
  color: #000;
  border: none;
  border-radius: 9999px;
  padding: 10px 40px;
  font-size: 14px;
  font-weight: 700;
  cursor: pointer;
  transition: background 0.15s, opacity 0.15s;

  &:hover { background: #1fd665; }
  &:disabled { opacity: 0.4; cursor: not-allowed; }
}

.retry-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  background: rgba(243, 114, 127, 0.15);
  border: 1px solid rgba(243, 114, 127, 0.4);
  border-radius: 9999px;
  padding: 10px 24px;
  font-size: 14px;
  font-weight: 600;
  color: #f3727f;
  cursor: pointer;
  transition: background 0.15s;

  &:hover { background: rgba(243, 114, 127, 0.25); }
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
  pointer-events: none;

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
