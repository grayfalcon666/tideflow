<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch, computed } from 'vue'

const MUTED_STORAGE_KEY = 'tideflow-video-muted'

const loadMutedPref = (): boolean => {
  try {
    const v = localStorage.getItem(MUTED_STORAGE_KEY)
    if (v === null) return false
    return v === 'true'
  } catch {
    return false
  }
}

const props = withDefaults(defineProps<{
  src: string
  poster?: string
  muted?: boolean
  autoPlay?: boolean
  width?: number
  height?: number
  duration?: number
  playToken?: string
  videoId?: number
  noteTimestamps?: number[]
  visible?: boolean
}>(), {
  visible: true,
})

const emit = defineEmits<{
  play: []
  pause: []
  click: []
  dblclick: []
  viewReported: []
  completionReported: []
  historyReported: [videoId: number]
  timeupdate: []
}>()

// Tracking flags - reset when src changes
const viewReported = ref(false)
const completionReported = ref(false)
const historyReported = ref(false)
const hasPlayed = ref(false)

// Dynamic threshold: min(5s, duration * 0.2)
const viewThreshold = computed(() => {
  const d = duration.value || props.duration || 0
  return Math.min(5, d * 0.2)
})

const videoEl = ref<HTMLVideoElement>()
const videoContainerRef = ref<HTMLDivElement>()
const isPlaying = ref(false)
const isMuted = ref(props.muted ?? loadMutedPref())
const showPoster = ref(true)
const showPlayIndicator = ref(false)
const showMuteState = ref(false)
const showVolumeSlider = ref(false)
const volume = ref(1)

// Seek preview (swipe)
const showSeekIndicator = ref(false)
const seekDelta = ref(0)
const seekIndicatorText = ref('')

// Tap seek (double-tap left/right)
const showTapSeekIndicator = ref(false)
const tapSeekDirection = ref<'left' | 'right'>('left')
const tapSeekDelta = ref(0)
let leftTapTimer: ReturnType<typeof setTimeout> | null = null
let rightTapTimer: ReturnType<typeof setTimeout> | null = null

// Progress / controls
const duration = ref(0)
const currentTime = ref(0)
const buffered = ref(0)
const showControls = ref(true)
let hideControlsTimer: ReturnType<typeof setTimeout> | null = null
let clickTimer: ReturnType<typeof setTimeout> | null = null
let indicatorTimer: ReturnType<typeof setTimeout> | null = null

// Fullscreen
const isFullscreen = ref(false)

// Buffered percentage
const bufferedPercent = computed(() => {
  if (!duration.value) return 0
  return Math.min((buffered.value / duration.value) * 100, 100)
})

// Played progress percentage
const progressPercent = computed(() => {
  if (!duration.value) return 0
  return (currentTime.value / duration.value) * 100
})

const formatTime = (seconds: number) => {
  if (!isFinite(seconds) || seconds < 0) return '0:00'
  const m = Math.floor(seconds / 60)
  const s = Math.floor(seconds % 60)
  return `${m}:${s.toString().padStart(2, '0')}`
}

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

// 持久化静音偏好
watch(isMuted, (val) => {
  try { localStorage.setItem(MUTED_STORAGE_KEY, String(val)) } catch {}
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
  if (isDragging) return
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

// Double-tap left/right to seek ±5s (YouTube-style)
const handleSeekTap = (direction: 'left' | 'right') => {
  if (isDragging || !duration.value) return
  const timer = direction === 'left' ? leftTapTimer : rightTapTimer
  if (timer) {
    // Double-tap confirmed
    clearTimeout(timer)
    if (direction === 'left') leftTapTimer = null
    else rightTapTimer = null
    const delta = direction === 'left' ? -5 : 5
    if (videoEl.value) {
      videoEl.value.currentTime = Math.max(0, Math.min(videoEl.value.currentTime + delta, duration.value))
      currentTime.value = videoEl.value.currentTime
      resetHideTimer()
    }
    tapSeekDirection.value = direction
    tapSeekDelta.value = delta
    showTapSeekIndicator.value = true
    setTimeout(() => { showTapSeekIndicator.value = false }, 600)
  } else {
    if (direction === 'left') {
      leftTapTimer = setTimeout(() => { leftTapTimer = null }, 300)
    } else {
      rightTapTimer = setTimeout(() => { rightTapTimer = null }, 300)
    }
  }
}

// Progress bar: seek (only handle user-initiated input events)
const seekTo = (e: Event) => {
  if (!e.isTrusted) return // Ignore programmatic value changes on mobile
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
  showVolumeSlider.value = !showVolumeSlider.value
  resetHideTimer()
}

const handleVolumeChange = (e: Event) => {
  const val = parseFloat((e.target as HTMLInputElement).value)
  volume.value = val
  if (videoEl.value) {
    videoEl.value.volume = val
    videoEl.value.muted = val === 0
    isMuted.value = val === 0
  }
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
  getCurrentTime: () => currentTime.value,
  getDuration: () => duration.value,
  seekTo: (timestamp: number) => {
    if (videoEl.value) {
      videoEl.value.currentTime = Math.max(0, Math.min(timestamp, duration.value))
      currentTime.value = videoEl.value.currentTime
      resetHideTimer()
    }
  },
})

// Mobile seek: horizontal swipe to fast-forward/rewind
let seekStartX = 0
let seekStartTime = 0
let isDragging = false
let seekTouchStartY = 0

const handleTouchStart = (e: TouchEvent) => {
  // Don't interfere with native range input (progress bar) touch handling
  if ((e.target as HTMLElement)?.closest('.progress-bar-wrap')) return
  seekStartX = e.touches[0].clientX
  seekStartTime = videoEl.value?.currentTime ?? 0
  isDragging = false
  seekTouchStartY = e.touches[0].clientY
}

const handleTouchMove = (e: TouchEvent) => {
  // Don't interfere with native range input (progress bar) touch handling
  if ((e.target as HTMLElement)?.closest('.progress-bar-wrap')) return
  if (!duration.value) return
  const dx = e.touches[0].clientX - seekStartX
  const dy = e.touches[0].clientY - seekTouchStartY
  // If it's more vertical than horizontal, skip seek
  if (Math.abs(dy) > Math.abs(dx) && Math.abs(dy) > 10) return
  if (Math.abs(dx) > 10) {
    isDragging = true
    // Stop the event from reaching the progress bar input
    e.preventDefault()
  }
  // 1px = 0.5s of seek
  const delta = Math.round(dx * 0.5)
  seekDelta.value = delta
  const newTime = Math.max(0, Math.min(seekStartTime + delta, duration.value))
  const pct = ((newTime / duration.value) * 100).toFixed(1)
  seekIndicatorText.value = `${delta >= 0 ? '+' : ''}${delta}s · ${pct}%`
  if (delta !== 0) {
    showSeekIndicator.value = true
    showPlayIndicator.value = false
  }
}

const handleTouchEnd = () => {
  if (seekDelta.value !== 0 && videoEl.value) {
    const newTime = Math.max(0, Math.min(seekStartTime + seekDelta.value, duration.value))
    videoEl.value.currentTime = newTime
    currentTime.value = newTime
    resetHideTimer()
  }
  seekDelta.value = 0
  showSeekIndicator.value = false
  // Reset isDragging after a tick so click handler can't accidentally fire
  setTimeout(() => { isDragging = false }, 0)
}

watch(
  () => [props.src, props.autoPlay] as const,
  ([newSrc, newAutoPlay]) => {
    if (newSrc) {
      showPoster.value = true
      isPlaying.value = false
      viewReported.value = false
      completionReported.value = false
      historyReported.value = false
      hasPlayed.value = false
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
      volume.value = video.volume
    })
    // Fallback: durationchange fires even if loadedmetadata was missed
    video.addEventListener('durationchange', () => {
      if (video.duration && !duration.value) {
        duration.value = video.duration
      }
    })
    video.addEventListener('timeupdate', () => {
      currentTime.value = video.currentTime
      emit('timeupdate')
      // Valid play tracking: report once when threshold reached
      if (!viewReported.value && props.playToken && currentTime.value >= viewThreshold.value) {
        viewReported.value = true
        emit('viewReported')
      }
      // Watch history: report once when played >= 1s, has actually played, and visible
      // visible defaults to true; Feed page passes playerVisible (false when <50% visible)
      if (!historyReported.value && props.videoId && hasPlayed.value
          && currentTime.value >= 1 && props.visible) {
        historyReported.value = true
        emit('historyReported', props.videoId)
      }
    })
    video.addEventListener('playing', () => {
      isPlaying.value = true
      hasPlayed.value = true
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
    video.addEventListener('progress', () => {
      if (video.buffered.length > 0) {
        buffered.value = video.buffered.end(video.buffered.length - 1)
      }
    })
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
  if (leftTapTimer) clearTimeout(leftTapTimer)
  if (rightTapTimer) clearTimeout(rightTapTimer)
})
</script>

<template>
  <div
    ref="videoContainerRef"
    class="video-player"
    :style="{ aspectRatio: '16 / 9' }"
    @touchstart="handleTouchStart"
    @touchmove="handleTouchMove"
    @touchend="handleTouchEnd"
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

    <!-- Tap zones: left/center/right -->
    <div class="tap-zones">
      <div class="tap-zone-left" @click.stop="handleSeekTap('left')" />
      <div class="tap-zone-center">
        <div class="tap-zone-center-inner" @click.stop="handleClick" />
      </div>
      <div class="tap-zone-right" @click.stop="handleSeekTap('right')" />
    </div>

    <!-- Swipe seek indicator (center overlay) -->
    <transition name="fade">
      <div v-if="showSeekIndicator" class="play-indicator">
        <div class="play-icon-inner seek-indicator-text">
          {{ seekIndicatorText }}
        </div>
      </div>
    </transition>

    <!-- Double-tap seek indicator (left/right side) -->
    <transition name="tap-seek">
      <div v-if="showTapSeekIndicator" class="tap-seek-indicator" :class="tapSeekDirection">
        <div class="tap-seek-circle">
          <svg width="28" height="28" viewBox="0 0 24 24" fill="currentColor">
            <path v-if="tapSeekDirection === 'left'" d="M11 18V6l-8.5 6 8.5 6zm.5-6l8.5 6V6l-8.5 6z"/>
            <path v-else d="M4 18l8.5-6L4 6v12zm9-12v12l8.5-6L13 6z"/>
          </svg>
        </div>
        <div class="tap-seek-label">{{ tapSeekDelta > 0 ? '+' : '' }}{{ tapSeekDelta }}s</div>
      </div>
    </transition>

    <transition name="fade">
      <div v-if="showPlayIndicator && !showSeekIndicator" class="play-indicator">
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

          <!-- 进度条区域（含时间标签） -->
          <div class="progress-area">
            <span class="time-label time-current">{{ formatTime(currentTime) }}</span>
            <div class="progress-bar-wrap">
              <div class="progress-track" />
              <div class="progress-buffered" :style="{ width: bufferedPercent + '%' }" />
              <div class="progress-fill" :style="{ width: progressPercent + '%' }" />
              <div class="progress-thumb" :style="{ left: progressPercent + '%' }" />
              <div v-if="noteTimestamps?.length && duration" class="note-markers">
                <div
                  v-for="(ts, i) in noteTimestamps"
                  :key="i"
                  class="note-marker"
                  :style="{ left: (ts / duration * 100) + '%' }"
                />
              </div>
              <input
                type="range"
                :min="0"
                :max="duration || 0"
                :value="currentTime"
                step="0.1"
                class="progress-input"
                @click.stop
                @input="seekTo"
              />
            </div>
            <span class="time-label time-duration">{{ formatTime(duration) }}</span>
          </div>

          <div class="volume-control" @click.stop>
            <button class="ctrl-btn" @click.stop="toggleMute">
              <svg v-if="isMuted || volume === 0" width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
                <path d="M16.5 12c0-1.77-1.02-3.29-2.5-4.03v2.21l2.45 2.45c.03-.2.05-.41.05-.63zm2.5 0c0 .94-.2 1.82-.54 2.64l1.51 1.51C20.63 14.91 21 13.5 21 12c0-4.28-2.99-7.86-7-8.77v2.06c2.89.86 5 3.54 5 6.71zM4.27 3L3 4.27 7.73 9H3v6h4l5 5v-6.73l4.25 4.25c-.67.52-1.42.93-2.25 1.18v2.06c1.38-.31 2.63-.95 3.69-1.81L19.73 21 21 19.73l-9-9L4.27 3zM12 4L9.91 6.09 12 8.18V4z"/>
              </svg>
              <svg v-else width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
                <path d="M3 9v6h4l5 5V4L7 9H3zm13.5 3c0-1.77-1.02-3.29-2.5-4.03v8.05c1.48-.73 2.5-2.25 2.5-4.02zM14 3.23v2.06c2.89.86 5 3.54 5 6.71s-2.11 5.85-5 6.71v2.06c4.01-.91 7-4.49 7-8.77s-2.99-7.86-7-8.77z"/>
              </svg>
            </button>
            <div class="volume-slider-wrap">
              <input
                type="range"
                min="0"
                max="1"
                step="0.02"
                :value="isMuted ? 0 : volume"
                class="volume-slider"
                @input="handleVolumeChange"
              />
            </div>
          </div>

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

// Tap zones: left / center / right
.tap-zones {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 48px; // leave room for controls bar below
  display: flex;
  z-index: 5;
}

.tap-zone-left,
.tap-zone-right,
.tap-zone-center {
  height: 100%;
  cursor: pointer;
}

.tap-zone-left {
  flex: 0 0 35%;
}

.tap-zone-center {
  flex: 0 0 30%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.tap-zone-center-inner {
  width: 100%;
  height: 55%;
}

.tap-zone-right {
  flex: 0 0 35%;
}

// Double-tap seek indicator (YouTube-style)
.tap-seek-indicator {
  position: absolute;
  top: 50%;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  transform: translateY(-50%);
  pointer-events: none;
  z-index: 6;

  &.left {
    left: 12.5%;
  }

  &.right {
    right: 12.5%;
  }
}

.tap-seek-circle {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
}

.tap-seek-label {
  color: #fff;
  font-size: 13px;
  font-weight: 600;
  text-shadow: 0 1px 3px rgba(0, 0, 0, 0.5);
}

// Play/pause indicator overlay
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

.seek-indicator-text {
  width: auto;
  height: auto;
  padding: 12px 20px;
  border-radius: 8px;
  font-size: 16px;
  font-weight: 700;
  white-space: nowrap;
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

// ── Progress area (time labels + bar) ──
.progress-area {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 10px;
}

.time-label {
  font-size: 12px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  color: rgba(255, 255, 255, 0.7);
  min-width: 34px;
  text-align: center;
  user-select: none;
  pointer-events: none;
  letter-spacing: 0.3px;
}

// Progress bar
.progress-bar-wrap {
  flex: 1;
  height: 16px;
  position: relative;
  cursor: pointer;
  display: flex;
  align-items: center;

  // Invisible extended hit area for mobile
  &::after {
    content: '';
    position: absolute;
    left: 0;
    right: 0;
    top: -12px;
    bottom: -12px;
    z-index: 0;
  }

  &:hover .progress-thumb {
    transform: translate(-50%, -50%) scale(1);
    opacity: 1;
  }

  &:hover .progress-track,
  &:hover .progress-buffered,
  &:hover .progress-fill {
    height: 4px;
    border-radius: 2px;
  }
}

.progress-track {
  position: absolute;
  left: 0;
  right: 0;
  top: 50%;
  transform: translateY(-50%);
  height: 2px;
  border-radius: 1px;
  background: rgba(255, 255, 255, 0.15);
  pointer-events: none;
  z-index: 1;
  transition: height 0.15s ease;
}

.progress-buffered {
  position: absolute;
  left: 0;
  top: 50%;
  transform: translateY(-50%);
  height: 2px;
  border-radius: 1px;
  background: rgba(255, 255, 255, 0.25);
  pointer-events: none;
  z-index: 2;
  transition: height 0.15s ease, width 0.3s ease;
}

.progress-fill {
  position: absolute;
  left: 0;
  top: 50%;
  transform: translateY(-50%);
  height: 2px;
  border-radius: 1px;
  background: linear-gradient(90deg, var(--accent, #1db954), #1ed760);
  pointer-events: none;
  z-index: 3;
  transition: height 0.15s ease, width 0.1s linear;
  box-shadow: 0 0 6px rgba(29, 185, 84, 0.4);
}

.progress-thumb {
  position: absolute;
  top: 50%;
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: #fff;
  transform: translate(-50%, -50%) scale(0);
  opacity: 0;
  pointer-events: none;
  z-index: 4;
  transition: transform 0.12s ease, opacity 0.12s ease, left 0.1s linear;
  box-shadow: 0 0 8px rgba(0, 0, 0, 0.5), 0 0 4px rgba(255, 255, 255, 0.3);
}

.note-markers {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  pointer-events: none;
  z-index: 1;
}

.note-marker {
  position: absolute;
  top: 50%;
  transform: translate(-50%, -50%);
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: rgba(255, 215, 0, 0.9);
  box-shadow: 0 0 3px rgba(255, 215, 0, 0.5);
  transition: transform 0.15s ease;
}

// Native range input — invisible, sits on top for interaction
.progress-input {
  position: absolute;
  left: 0;
  right: 0;
  top: 50%;
  transform: translateY(-50%);
  width: 100%;
  height: 16px;
  -webkit-appearance: none;
  appearance: none;
  background: transparent;
  margin: 0;
  cursor: pointer;
  outline: none;
  z-index: 5;

  &::-webkit-slider-thumb {
    -webkit-appearance: none;
    width: 14px;
    height: 14px;
    border-radius: 50%;
    background: transparent;
    cursor: pointer;
    opacity: 0;
  }

  &::-moz-range-thumb {
    width: 14px;
    height: 14px;
    border-radius: 50%;
    background: transparent;
    cursor: pointer;
    border: none;
    opacity: 0;
  }

  &::-webkit-slider-runnable-track {
    background: transparent;
    height: 16px;
  }

  &::-moz-range-track {
    background: transparent;
    height: 16px;
  }
}

// Volume slider
.volume-control {
  display: flex;
  align-items: center;
  gap: 4px;
  position: relative;
}

.volume-slider-wrap {
  width: 0;
  overflow: hidden;
  transition: width 0.2s;

  .volume-control:hover &,
  .volume-control:focus-within & {
    width: 70px;
  }
}

.volume-slider {
  width: 70px;
  height: 4px;
  -webkit-appearance: none;
  appearance: none;
  background: rgba(255, 255, 255, 0.3);
  border-radius: 2px;
  outline: none;
  cursor: pointer;

  &::-webkit-slider-thumb {
    -webkit-appearance: none;
    width: 10px;
    height: 10px;
    border-radius: 50%;
    background: #fff;
    cursor: pointer;
  }

  &::-moz-range-thumb {
    width: 10px;
    height: 10px;
    border-radius: 50%;
    background: #fff;
    cursor: pointer;
    border: none;
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

// Tap seek indicator animation (scale + fade)
.tap-seek-enter-active {
  transition: opacity 0.15s, transform 0.15s;
}
.tap-seek-leave-active {
  transition: opacity 0.4s, transform 0.4s;
}
.tap-seek-enter-from {
  opacity: 0;
  transform: translateY(-50%) scale(0.7);
}
.tap-seek-leave-to {
  opacity: 0;
  transform: translateY(-50%) scale(1.15);
}
</style>
