<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'

const props = defineProps<{
  src: string
  poster?: string
  muted?: boolean
  autoPlay?: boolean
}>()

const emit = defineEmits<{
  play: []
  pause: []
  click: []
  dblclick: []
}>()

const videoEl = ref<HTMLVideoElement>()
const isPlaying = ref(false)
const showPoster = ref(true)
let clickTimer: ReturnType<typeof setTimeout> | null = null


const play = () => {
  videoEl.value?.play().catch(() => {})
}

const pause = () => {
  videoEl.value?.pause()
}

const togglePlay = () => {
  if (isPlaying.value) pause()
  else play()
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
      emit('click')
    }, 300)
  }
}

watch(
  () => props.src,
  () => {
    showPoster.value = true
    isPlaying.value = false
  }
)

onMounted(() => {
  if (props.autoPlay && props.src) {
    setTimeout(() => play(), 100)
  }
  if (videoEl.value) {
    videoEl.value.addEventListener('playing', () => {
      isPlaying.value = true
      showPoster.value = false
      emit('play')
    })
    videoEl.value.addEventListener('pause', () => {
      isPlaying.value = false
      emit('pause')
    })
    videoEl.value.addEventListener('ended', () => {
      isPlaying.value = false
      showPoster.value = true
    })
  }
})
</script>

<template>
  <div class="video-player" @click="handleClick">
    <video
      v-if="src"
      ref="videoEl"
      :src="src"
      :poster="poster"
      :muted="muted ?? true"
      :playsinline="true"
      webkit-playsinline="true"
      x5-video-player-type="h5"
      class="video-el"
    />
    <img v-if="showPoster && poster" :src="poster" class="poster-img" alt="cover" />
    <div v-if="showPoster && !poster" class="poster-placeholder">
      <span>NO POSTER</span>
    </div>
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

.video-el {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.poster-img {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.poster-placeholder {
  position: absolute;
  inset: 0;
  background: #333;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #888;
  font-size: 14px;
}
</style>
