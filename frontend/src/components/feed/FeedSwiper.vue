<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'

const props = defineProps<{
  items: any[]
  activeIndex: number
}>()

const emit = defineEmits<{
  (e: 'update:activeIndex', val: number): void
  (e: 'reachEnd'): void
}>()

const scrollEl = ref<HTMLElement>()
const slideHeight = ref(window.innerHeight)
let lastWheelTime = 0
let lastTouchY = 0

const scrollToIndex = (index: number) => {
  if (!scrollEl.value) return
  const target = Math.max(0, Math.min(index, props.items.length - 1))
  scrollEl.value.scrollTo({ top: target * slideHeight.value, behavior: 'smooth' })
}

const handleWheel = (e: WheelEvent) => {
  e.preventDefault()
  const now = Date.now()
  if (now - lastWheelTime < 800) return
  if (e.deltaY > 0) {
    lastWheelTime = now
    emit('update:activeIndex', Math.min(props.activeIndex + 1, props.items.length - 1))
  } else if (e.deltaY < 0) {
    lastWheelTime = now
    emit('update:activeIndex', Math.max(props.activeIndex - 1, 0))
  }
}

const handleTouchStart = (e: TouchEvent) => {
  lastTouchY = e.touches[0].clientY
}

const handleTouchEnd = (e: TouchEvent) => {
  const deltaY = lastTouchY - e.changedTouches[0].clientY
  if (Math.abs(deltaY) > 50) {
    if (deltaY > 0) {
      emit('update:activeIndex', Math.min(props.activeIndex + 1, props.items.length - 1))
    } else {
      emit('update:activeIndex', Math.max(props.activeIndex - 1, 0))
    }
  }
}

const handleScroll = () => {
  if (!scrollEl.value) return
  const newIndex = Math.round(scrollEl.value.scrollTop / slideHeight.value)
  if (newIndex !== props.activeIndex) {
    emit('update:activeIndex', newIndex)
  }
  if (props.activeIndex >= props.items.length - 3) {
    emit('reachEnd')
  }
}

onMounted(() => {
  if (scrollEl.value) {
    slideHeight.value = scrollEl.value.clientHeight
    scrollEl.value.addEventListener('wheel', handleWheel, { passive: false })
    scrollEl.value.addEventListener('touchstart', handleTouchStart, { passive: true })
    scrollEl.value.addEventListener('touchend', handleTouchEnd, { passive: true })
    scrollEl.value.addEventListener('scroll', handleScroll, { passive: true })
  }
})

onUnmounted(() => {
  if (scrollEl.value) {
    scrollEl.value.removeEventListener('wheel', handleWheel)
    scrollEl.value.removeEventListener('touchstart', handleTouchStart)
    scrollEl.value.removeEventListener('touchend', handleTouchEnd)
    scrollEl.value.removeEventListener('scroll', handleScroll)
  }
})

defineExpose({ scrollToIndex })
</script>

<template>
  <div ref="scrollEl" class="feed-swiper">
    <div
      v-for="(item, index) in items"
      :key="item?.video_id ?? index"
      class="feed-slide"
    >
      <!-- Use default slot, pass active flag for each item -->
      <slot :item="item ?? {}" :active="index === activeIndex" />
    </div>
  </div>
</template>

<style scoped lang="scss">
.feed-swiper {
  height: calc(100svh - 48px);
  overflow-y: scroll;
  scroll-snap-type: y mandatory;
  scroll-behavior: smooth;
  -webkit-overflow-scrolling: touch;

  &::-webkit-scrollbar {
    display: none;
  }
}

.feed-slide {
  height: calc(100svh - 48px);
  scroll-snap-align: start;
  scroll-snap-stop: always;
  position: relative;
}
</style>
