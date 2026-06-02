<script setup lang="ts">
import { ref } from 'vue'
import type { Caption } from '../../types'
import * as learnService from '../../services/learn'
import TFIcon from '../common/TFIcon.vue'

const props = defineProps<{
  videoId: number
  word: string
  videoSrc?: string
  videoPoster?: string
}>()

const emit = defineEmits<{
  seekTo: [seconds: number]
}>()

// 会话内缓存
const cache = new Map<string, Caption[]>()
const captions = ref<Caption[]>([])
const loading = ref(false)
const error = ref('')

// 小窗视频
const miniVideoRef = ref<HTMLVideoElement>()

const handleCaptionClick = (cap: Caption) => {
  const seconds = parseSeconds(cap.start)
  emit('seekTo', seconds)
  if (miniVideoRef.value) {
    miniVideoRef.value.currentTime = seconds
    miniVideoRef.value.play().catch(() => {})
  }
}

// 加载字幕
const load = async () => {
  if (cache.has(props.word)) {
    captions.value = cache.get(props.word)!
    return
  }
  loading.value = true
  error.value = ''
  try {
    const resp = await learnService.getWordCaptions(props.videoId, props.word)
    captions.value = resp.data.data?.captions ?? []
    cache.set(props.word, captions.value)
  } catch (e: any) {
    error.value = '语境加载失败'
  } finally {
    loading.value = false
  }
}

load()

// 时间字符串 -> 秒数
const parseSeconds = (t: string) => {
  const parts = t.split(':')
  if (parts.length === 3) {
    return parseInt(parts[0]) * 3600 + parseInt(parts[1]) * 60 + parseFloat(parts[2])
  }
  if (parts.length === 2) {
    return parseInt(parts[0]) * 60 + parseFloat(parts[1])
  }
  return parseFloat(t)
}

// 高亮单词
const highlightWord = (content: string) => {
  const re = new RegExp(`(${props.word})`, 'gi')
  return content.replace(re, '<mark>$1</mark>')
}
</script>

<template>
  <div class="caption-drawer">
    <div class="drawer-header">
      <span class="word-label">{{ word }}</span>
    </div>

    <div v-if="loading" class="state-loading">
      <q-spinner color="accent" />
    </div>
    <div v-else-if="error" class="state-error">
      <TFIcon name="error_outline" :size="32" color="#f3727f" />
      <span>{{ error }}</span>
    </div>
    <div v-else-if="captions.length === 0" class="state-empty">
      <span>暂无语境</span>
    </div>
    <div v-else class="caption-list" :class="{ 'has-mini-player': videoSrc }">
      <div
        v-for="(cap, i) in captions"
        :key="i"
        class="caption-item"
        @click="handleCaptionClick(cap)"
      >
        <span class="caption-time">{{ cap.start }}</span>
        <span class="caption-content" v-html="highlightWord(cap.content)" />
      </div>
    </div>

    <div v-if="videoSrc" class="mini-player-wrap">
      <video
        ref="miniVideoRef"
        :src="videoSrc"
        :poster="videoPoster"
        class="mini-video"
        playsinline
        preload="metadata"
      />
    </div>
  </div>
</template>

<style scoped lang="scss">
.caption-drawer {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--bg-card, #181818);
  color: var(--text-base, #fff);
}

.drawer-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px;
  border-bottom: 1px solid rgba(255,255,255,0.1);
}

.word-label {
  font-size: 18px;
  font-weight: 700;
  color: #1ed760;
}

.state-loading,
.state-error,
.state-empty {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  color: #b3b3b3;
  font-size: 14px;
}

.caption-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px 0;

  &.has-mini-player {
    padding-bottom: 8px;
  }
}

.mini-player-wrap {
  flex-shrink: 0;
  padding: 8px 16px 12px;
  border-top: 1px solid rgba(255,255,255,0.1);
}

.mini-video {
  width: 100%;
  max-height: 180px;
  border-radius: 8px;
  background: #000;
  object-fit: contain;
}

.caption-item {
  display: flex;
  gap: 12px;
  padding: 10px 16px;
  cursor: pointer;
  transition: background 0.15s;

  &:hover {
    background: rgba(255,255,255,0.05);
  }
}

.caption-time {
  font-size: 12px;
  color: #1ed760;
  font-family: ui-monospace, Consolas, monospace;
  flex-shrink: 0;
  padding-top: 2px;
}

.caption-content {
  font-size: 14px;
  line-height: 1.5;
  color: #e0e0e0;

  :deep(mark) {
    background: rgba(30, 215, 96, 0.25);
    color: #1ed760;
    border-radius: 2px;
    padding: 0 2px;
  }
}
</style>