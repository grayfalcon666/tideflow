<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'

const props = defineProps<{
  items: any[]
  activeIndex: number
}>()

const emit = defineEmits<{
  (e: 'update:activeIndex', val: number): void
  (e: 'reachEnd'): void
}>()

const scrollEl = ref<HTMLElement>()
let slideHeight = 0
let isProgrammaticScroll = false
let lastTouchY = 0
let lastTouchTime = 0
let scrollEndTimer: ReturnType<typeof setTimeout> | null = null
let resizeObserver: ResizeObserver | null = null

// Sync scroll when activeIndex changes from outside (e.g. keyboard)
watch(() => props.activeIndex, (newIdx) => {
  if (!scrollEl.value || slideHeight === 0) return
  const expectedTop = newIdx * slideHeight
  const currentTop = scrollEl.value.scrollTop
  if (Math.abs(currentTop - expectedTop) > slideHeight * 0.5) {
    isProgrammaticScroll = true
    scrollEl.value.scrollTo({ top: expectedTop, behavior: 'smooth' })
    setTimeout(() => { isProgrammaticScroll = false }, 400)
  }
})

// Track actual active index after snap settles - avoid circular updates
const syncActiveFromScroll = () => {
  if (!scrollEl.value || isProgrammaticScroll || slideHeight === 0) return
  const newIndex = Math.round(scrollEl.value.scrollTop / slideHeight)
  if (newIndex !== props.activeIndex) {
    emit('update:activeIndex', newIndex)
  }
  if (newIndex >= props.items.length - 3) {
    emit('reachEnd')
  }
}

const handleTouchStart = (e: TouchEvent) => {
  lastTouchY = e.touches[0].clientY
}

const handleTouchEnd = (e: TouchEvent) => {
  const deltaY = lastTouchY - e.changedTouches[0].clientY
  if (Math.abs(deltaY) < 50) return
  const now = Date.now()
  if (now - lastTouchTime < 300) return
  lastTouchTime = now
  if (deltaY > 0) {
    emit('update:activeIndex', Math.min(props.activeIndex + 1, props.items.length - 1))
  } else {
    emit('update:activeIndex', Math.max(props.activeIndex - 1, 0))
  }
}

// Debounce: only update after scroll animation settles (~300ms after last scroll event)
const handleScroll = () => {
  if (!scrollEl.value || isProgrammaticScroll || slideHeight === 0) return
  if (scrollEndTimer) clearTimeout(scrollEndTimer)
  scrollEndTimer = setTimeout(syncActiveFromScroll, 300)
}

onMounted(() => {
  if (scrollEl.value) {
    // Use ResizeObserver to track actual slide height
    resizeObserver = new ResizeObserver((entries) => {
      const entry = entries[0]
      if (entry) {
        slideHeight = entry.contentRect.height
      }
    })
    resizeObserver.observe(scrollEl.value)

    // Initial slide height from clientHeight (set after first paint)
    slideHeight = scrollEl.value.clientHeight

    // Initialize scroll position to activeIndex
    scrollEl.value.scrollTop = props.activeIndex * slideHeight

    scrollEl.value.addEventListener('touchstart', handleTouchStart, { passive: true })
    scrollEl.value.addEventListener('touchend', handleTouchEnd, { passive: true })
    scrollEl.value.addEventListener('scroll', handleScroll, { passive: true })
  }
})

onUnmounted(() => {
  if (scrollEl.value) {
    scrollEl.value.removeEventListener('touchstart', handleTouchStart)
    scrollEl.value.removeEventListener('touchend', handleTouchEnd)
    scrollEl.value.removeEventListener('scroll', handleScroll)
  }
  if (scrollEndTimer) clearTimeout(scrollEndTimer)
  if (resizeObserver) resizeObserver.disconnect()
})

defineExpose({ scrollToIndex: (idx: number) => {
  if (!scrollEl.value) return
  const target = Math.max(0, Math.min(idx, props.items.length - 1))
  isProgrammaticScroll = true
  scrollEl.value.scrollTo({ top: target * slideHeight, behavior: 'smooth' })
  setTimeout(() => { isProgrammaticScroll = false }, 400)
}, getActiveVideoPlayer: () => {
  const slides = scrollEl.value?.querySelectorAll('.feed-slide')
  const slide = slides?.[props.activeIndex]
  return slide?.querySelector('.video-player')?.__vueParentComponent?.exposed as any
} })
</script>

<template>
  <div ref="scrollEl" class="feed-swiper">
    <div
      v-for="(item, index) in items"
      :key="item?.video_id ?? index"
      class="feed-slide"
    >
      <slot :item="item ?? {}" :active="index === activeIndex" />
    </div>
  </div>
</template>

<style scoped lang="scss">
.feed-swiper {
  overflow-y: scroll;
  scroll-snap-type: y mandatory;
  scroll-behavior: smooth;
  height: 100%;
  flex: 1;

  &::-webkit-scrollbar {
    display: none;
  }
}

.feed-slide {
  scroll-snap-align: start;
  scroll-snap-stop: always;
  position: relative;
  height: 100%;
}
</style>
