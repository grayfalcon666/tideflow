<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch, computed } from 'vue'

const props = defineProps<{
  src: string
  poster?: string
  muted?: boolean
  autoPlay?: boolean
  width?: number
  height?: number
  duration?: number
  playToken?: string
  videoId?: number
}>()

const emit = defineEmits<{
  play: []
  pause: []
  click: []
  dblclick: []
  viewReported: []
  completionReported: []
}>()

// Tracking flags - reset when src changes
const viewReported = ref(false)
const completionReported = ref(false)

// Dynamic threshold: min(5s, duration * 0.2)
const viewThreshold = computed(() => {
  const d = duration.value || props.duration || 0
  return Math.min(5, d * 0.2)
})

const videoEl = ref<HTMLVideoElement>()
const videoContainerRef = ref<HTMLDivElement>()
const isPlaying = ref(false)
const isMuted = ref(props.muted ?? true)
const showPoster = ref(true)
const showPlayIndicator = ref(false)
const showMuteState = ref(false)

// Progress / controls
const duration = ref(0)
const currentTime = ref(0)
const showControls = ref(true)
let hideControlsTimer: ReturnType<typeof setTimeout> | null = null
let clickTimer: ReturnType<typeof setTimeout> | null = null
let indicatorTimer: ReturnType<typeof setTimeout> | null = null

// Fullscreen
const isFullscreen = ref(false)

// Aspect ratio for CSS aspect-ratio property
const cssAspectRatio = computed(() => {
  if (props.width && props.height && props.width > 0 && props.height > 0) {
    return `${props.width} / ${props.height}`
  }
  return '16 / 9' // default
})

// Show skeleton until video first frame or poster loads
const showSkeleton = ref(true)

const resetHideTimer = () => {
  showControls.value = true
  if (hideControlsTimer) clearTimeout(hideControlsTimer)
  hideControlsTimer = setTimeout(() => {
    if (isPlaying.value) showControls.value = false
  }, 2500)
}

watch(() => props.muted, (muted) => {
  isMuted.value = muted
  if (videoEl.value) videoEl.value.muted = muted
  showMuteIndicator()
})

const play = () => {
  videoEl.value?.play().catch(() => {})
}

const showIndicator = () => {
  showPlayIndicator.value = true
  showMuteState.value = false
  if (indicatorTimer) clearTimeout(indicatorTimer)
  indicatorTimer = setTimeout(() => {
    showPlayIndicator.value = false
  }, 600)
}

const showMuteIndicator = () => {
  showMuteState.value = true
  if (indicatorTimer) clearTimeout(indicatorTimer)
  indicatorTimer = setTimeout(() => {
    showMuteState.value = false
  }, 600)
}

const pause = () => {
  videoEl.value?.pause()
}

const togglePlay = () => {
  if (isPlaying.value) {
    pause()
  } else {
    play()
  }
  showIndicator()
}

const handleClick = () => {
  if (clickTimer) {
    clearTimeout(clickTimer)
    clickTimer = null
    emit('dblclick')
  } else {
    clickTimer = setTimeout(() => {
      clickTimer = null
      togglePlay()
      resetHideTimer()
      emit('click')
    }, 300)
  }
}

// Progress bar: seek
const seekTo = (e: Event) => {
  const video = videoEl.value!
  video.currentTime = parseFloat((e.target as HTMLInputElement).value)
  currentTime.value = video.currentTime
  resetHideTimer()
}

// Fullscreen
const toggleFullscreen = async () => {
  const el = videoContainerRef.value!
  if (!document.fullscreenElement) {
    await el.requestFullscreen()
    isFullscreen.value = true
  } else {
    await document.exitFullscreen()
    isFullscreen.value = false
  }
  resetHideTimer()
}

// Mute toggle
const toggleMute = () => {
  isMuted.value = !isMuted.value
  if (videoEl.value) videoEl.value.muted = isMuted.value
  showMuteIndicator()
  resetHideTimer()
}

// Exposed methods
defineExpose({
  play: () => play(),
  pause: () => pause(),
  toggleMute,
  toggleFullscreen,
  seek: (seconds: number) => {
    if (videoEl.value) {
      videoEl.value.currentTime = Math.max(0, Math.min(videoEl.value.currentTime + seconds, duration.value))
      resetHideTimer()
    }
  },
})

watch(
  () => [props.src, props.autoPlay] as const,
  ([newSrc, newAutoPlay]) => {
    if (newSrc) {
      showPoster.value = true
      isPlaying.value = false
      viewReported.value = false
      completionReported.value = false
      if (newAutoPlay) {
        setTimeout(() => play(), 100)
      }
    }
  }
)

// Pause when autoPlay is withdrawn (video scrolled out of view)
watch(() => props.autoPlay, (autoPlay) => {
  if (!autoPlay && videoEl.value && !videoEl.value.paused) {
    videoEl.value.pause()
    isPlaying.value = false
  }
})

onMounted(() => {
  if (props.autoPlay && props.src) {
    setTimeout(() => play(), 100)
  }

  const video = videoEl.value
  if (video) {
    video.addEventListener('loadedmetadata', () => {
      duration.value = video.duration
    })
    video.addEventListener('timeupdate', () => {
      currentTime.value = video.currentTime
      // Valid play tracking: report once when threshold reached
      if (!viewReported.value && props.playToken && currentTime.value >= viewThreshold.value) {
        viewReported.value = true
        emit('viewReported')
      }
    })
    video.addEventListener('playing', () => {
      isPlaying.value = true
      showPoster.value = false
      showSkeleton.value = false
      emit('play')
      resetHideTimer()
    })
    video.addEventListener('pause', () => {
      isPlaying.value = false
      showControls.value = true
      emit('pause')
    })
    video.addEventListener('ended', () => {
      isPlaying.value = false
      showPoster.value = true
      showControls.value = true
      if (!completionReported.value && props.playToken) {
        completionReported.value = true
        emit('completionReported')
      }
    })
    // Also hide skeleton when poster loads (covers video first frame)
    video.addEventListener('loadeddata', () => {
      showSkeleton.value = false
    })
  }

  document.addEventListener('fullscreenchange', () => {
    isFullscreen.value = !!document.fullscreenElement
  })
})

onUnmounted(() => {
  if (hideControlsTimer) clearTimeout(hideControlsTimer)
  if (indicatorTimer) clearTimeout(indicatorTimer)
  if (clickTimer) clearTimeout(clickTimer)
})
</script>

<template>
  <div
    ref="videoContainerRef"
    class="video-player"
    :style="{ aspectRatio: cssAspectRatio }"
    @click="handleClick"
    @mouseenter="showControls = true"
    @mouseleave="isPlaying && (showControls = false)"
  >
    <!-- Aspect-ratio skeleton shown before video loads -->
    <div v-if="showSkeleton" class="video-skeleton">
      <div class="skeleton-shimmer" />
    </div>

    <video
      ref="videoEl"
      :src="src"
      :poster="poster"
      :muted="isMuted"
      :playsinline="true"
      class="video-el"
    />
    <img v-if="showPoster && poster" :src="poster" class="poster-img" alt="cover" />
    <div v-if="showPoster && !poster" class="poster-placeholder"></div>

    <transition name="fade">
      <div v-if="showPlayIndicator" class="play-indicator">
        <div class="play-icon-inner">
          <svg v-if="showMuteState && isMuted" width="40" height="40" viewBox="0 0 24 24" fill="currentColor">
            <path d="M16.5 12c0-1.77-1.02-3.29-2.5-4.03v2.21l2.45 2.45c.03-.2.05-.41.05-.63zm2.5 0c0 .94-.2 1.82-.54 2.64l1.51 1.51C20.63 14.91 21 13.5 21 12c0-4.28-2.99-7.86-7-8.77v2.06c2.89.86 5 3.54 5 6.71zM4.27 3L3 4.27 7.73 9H3v6h4l5 5v-6.73l4.25 4.25c-.67.52-1.42.93-2.25 1.18v2.06c1.38-.31 2.63-.95 3.69-1.81L19.73 21 21 19.73l-9-9L4.27 3zM12 4L9.91 6.09 12 8.18V4z"/>
          </svg>
          <svg v-else-if="showMuteState && !isMuted" width="40" height="40" viewBox="0 0 24 24" fill="currentColor">
            <path d="M3 9v6h4l5 5V4L7 9H3zm13.5 3c0-1.77-1.02-3.29-2.5-4.03v8.05c1.48-.73 2.5-2.25 2.5-4.02zM14 3.23v2.06c2.89.86 5 3.54 5 6.71s-2.11 5.85-5 6.71v2.06c4.01-.91 7-4.49 7-8.77s-2.99-7.86-7-8.77z"/>
          </svg>
          <svg v-else-if="!isPlaying" width="40" height="40" viewBox="0 0 24 24" fill="currentColor">
            <path d="M8 5v14l11-7z"/>
          </svg>
          <svg v-else width="40" height="40" viewBox="0 0 24 24" fill="currentColor">
            <path d="M6 19h4V5H6v14zm8-14v14h4V5h-4z"/>
          </svg>
        </div>
      </div>
    </transition>

    <!-- Controls overlay -->
    <transition name="controls-fade">
      <div v-show="showControls || !isPlaying" class="video-controls">
        <div class="controls-inner">
          <button class="ctrl-btn" @click.stop="togglePlay">
            <svg v-if="!isPlaying" width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
              <path d="M8 5v14l11-7z"/>
            </svg>
            <svg v-else width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
              <path d="M6 19h4V5H6v14zm8-14v14h4V5h-4z"/>
            </svg>
          </button>

          <div class="progress-bar-wrap">
            <div
              class="progress-bar-fill"
              :style="{ width: duration ? (currentTime / duration * 100) + '%' : '0%' }"
            />
            <input
              type="range"
              :min="0"
              :max="duration || 0"
              :value="currentTime"
              step="0.1"
              class="progress-bar"
              @click.stop
              @input="seekTo"
            />
          </div>

          <button class="ctrl-btn" @click.stop="toggleMute">
            <svg v-if="isMuted" width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
              <path d="M16.5 12c0-1.77-1.02-3.29-2.5-4.03v2.21l2.45 2.45c.03-.2.05-.41.05-.63zm2.5 0c0 .94-.2 1.82-.54 2.64l1.51 1.51C20.63 14.91 21 13.5 21 12c0-4.28-2.99-7.86-7-8.77v2.06c2.89.86 5 3.54 5 6.71zM4.27 3L3 4.27 7.73 9H3v6h4l5 5v-6.73l4.25 4.25c-.67.52-1.42.93-2.25 1.18v2.06c1.38-.31 2.63-.95 3.69-1.81L19.73 21 21 19.73l-9-9L4.27 3zM12 4L9.91 6.09 12 8.18V4z"/>
            </svg>
            <svg v-else width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
              <path d="M3 9v6h4l5 5V4L7 9H3zm13.5 3c0-1.77-1.02-3.29-2.5-4.03v8.05c1.48-.73 2.5-2.25 2.5-4.02zM14 3.23v2.06c2.89.86 5 3.54 5 6.71s-2.11 5.85-5 6.71v2.06c4.01-.91 7-4.49 7-8.77s-2.99-7.86-7-8.77z"/>
            </svg>
          </button>

          <button class="ctrl-btn" @click.stop="toggleFullscreen">
            <svg v-if="!isFullscreen" width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
              <path d="M7 14H5v5h5v-2H7v-3zm-2-4h2V7h3V5H5v5zm12 7h-3v2h5v-5h-2v3zM14 5v2h3v3h2V5h-5z"/>
            </svg>
            <svg v-else width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
              <path d="M5 16h3v3h2v-5H5v2zm3-8H5v2h5V5H8v3zm6 11h2v-3h3v-2h-5v5zm2-12v2h3v3h2V5h-5z"/>
            </svg>
          </button>
        </div>
      </div>
    </transition>
  </div>
</template>

<style scoped lang="scss">
.video-player {
  position: relative;
  width: 100%;
  height: 100%;
  background: #000;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
}

.video-skeleton {
  position: absolute;
  inset: 0;
  background: #222;
  overflow: hidden;
  z-index: 1;
}

.skeleton-shimmer {
  position: absolute;
  inset: 0;
  background: linear-gradient(
    90deg,
    #222 0%,
    #333 40%,
    #444 60%,
    #222 100%
  );
  background-size: 200% 100%;
  animation: shimmer 1.4s ease-in-out infinite;
}

@keyframes shimmer {
  0% { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}

.video-el {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.poster-img {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: contain;
  background: #000;
}

.poster-placeholder {
  position: absolute;
  inset: 0;
  background: #1a1a1a;
}

.play-indicator {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  pointer-events: none;
}

.play-icon-inner {
  width: 72px;
  height: 72px;
  border-radius: 50%;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
}

// Controls bar
.video-controls {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  padding: 8px 12px 12px;
  background: linear-gradient(transparent, rgba(0,0,0,0.6));
  z-index: 10;
}

.controls-inner {
  display: flex;
  align-items: center;
  gap: 8px;
}

.ctrl-btn {
  background: none;
  border: none;
  color: #fff;
  cursor: pointer;
  padding: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  opacity: 0.9;
  transition: opacity 0.2s;

  &:hover {
    opacity: 1;
  }
}

// Progress bar
.progress-bar-wrap {
  flex: 1;
  height: 4px;
  position: relative;
  cursor: pointer;
}

.progress-bar-fill {
  position: absolute;
  left: 0;
  top: 0;
  height: 100%;
  background: rgba(255, 255, 255, 0.9);
  border-radius: 2px;
  pointer-events: none;
  z-index: 1;
}

.progress-bar {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  -webkit-appearance: none;
  appearance: none;
  background: transparent;
  margin: 0;
  cursor: pointer;
  outline: none;

  &::-webkit-slider-thumb {
    -webkit-appearance: none;
    width: 12px;
    height: 12px;
    border-radius: 50%;
    background: #fff;
    cursor: pointer;
    position: relative;
    z-index: 2;
  }

  &::-webkit-slider-runnable-track {
    background: transparent;
    height: 4px;
  }
}

// Transitions
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.3s;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

.controls-fade-enter-active,
.controls-fade-leave-active {
  transition: opacity 0.3s;
}
.controls-fade-enter-from,
.controls-fade-leave-to {
  opacity: 0;
}
</style>
