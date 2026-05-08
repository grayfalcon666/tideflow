<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import FeedTabBar from '../components/feed/FeedTabBar.vue'
import FeedSwiper from '../components/feed/FeedSwiper.vue'
import FeedSlideContent from '../components/feed/FeedSlideContent.vue'
import CommentDrawer from '../components/comment/CommentDrawer.vue'
import BottomNav from '../components/layout/BottomNav.vue'
import { useVideoControls } from '../composables/useVideoControls'
import { useFeedStore, type FeedTab } from '../stores/feed'
import { useAuthStore } from '../stores/auth'
import { useNotificationStore } from '../stores/notification'
import type { VideoItem } from '../types'

const router = useRouter()
const feedStore = useFeedStore()
const authStore = useAuthStore()
const notifStore = useNotificationStore()

const isMobile = ref(window.innerWidth < 1024)
const onResize = () => { isMobile.value = window.innerWidth < 1024 }

const activeTab = ref<FeedTab>('latest')
const activeIndex = ref(0)
const commentDrawerOpen = ref(false)
const commentVideoId = ref(0)
const playToken = ref<string | null>(null)

// Play token from current video item for tracking
const currentPlayToken = computed(() => {
  const item = items.value[activeIndex.value]
  return item?.play_token ?? playToken.value
})

const items = computed(() =>
  activeTab.value === 'latest' ? feedStore.latestItems : feedStore.followingItems
)
const hasMore = computed(() =>
  activeTab.value === 'latest' ? feedStore.latestHasMore : feedStore.followingHasMore
)
const isLoading = computed(() =>
  activeTab.value === 'latest' ? feedStore.latestLoading : feedStore.followingLoading
)

const loadData = (reset = false) => {
  if (activeTab.value === 'latest') {
    feedStore.loadLatest(reset)
  } else {
    feedStore.loadFollowing(reset)
  }
}

const onTabChange = (tab: FeedTab) => {
  activeTab.value = tab
  activeIndex.value = 0
  loadData(true)
}

const swiperRef = ref()
const { isMuted } = useVideoControls((key) => {
  if (key === 'c' || key === 'C') {
    const id = items.value[activeIndex.value]?.video_id
    if (id) {
      commentVideoId.value = id
      commentDrawerOpen.value = true
    }
  }
})

watch(activeIndex, (newIdx, oldIdx) => {
  if (newIdx >= items.value.length - 3 && hasMore.value && !isLoading.value) {
    feedStore.loadMore(activeTab.value)
  }
  // Pause previous video, play current video
  if (oldIdx !== undefined && oldIdx !== newIdx) {
    const prevPlayer = getPlayerAtIndex(oldIdx)
    prevPlayer?.pause()
  }
  const currPlayer = getPlayerAtIndex(newIdx)
  currPlayer?.play()
})

onMounted(() => {
  loadData()
  window.addEventListener('keydown', onKeyDown)
  if (authStore.isLoggedIn) {
    notifStore.connectSSE()
  }

  window.addEventListener('resize', onResize)

  const tabBar = document.querySelector('.feed-tab-bar') as HTMLElement || document.querySelector('[class*="feed-tab-bar"]') as HTMLElement
  const bottomNav = document.querySelector('.bottom-nav') as HTMLElement
  if (tabBar) {
    document.documentElement.style.setProperty('--tab-bar-height', tabBar.offsetHeight + 'px')
  }
  if (bottomNav) {
    document.documentElement.style.setProperty('--bottom-nav-height', bottomNav.offsetHeight + 'px')
  }
})

onUnmounted(() => {
  window.removeEventListener('keydown', onKeyDown)
  window.removeEventListener('resize', onResize)
})

const getActivePlayer = () => {
  return getPlayerAtIndex(activeIndex.value)
}

const getPlayerAtIndex = (idx: number) => {
  const slides = swiperRef.value?.$el?.querySelectorAll('.feed-slide')
  const slide = slides?.[idx]
  return (slide?.querySelector('.video-player') as any)?.__vueParentComponent?.exposed
}

// Keyboard shortcuts
const onKeyDown = (e: KeyboardEvent) => {
  const tag = (e.target as HTMLElement).tagName
  if (tag === 'INPUT' || tag === 'TEXTAREA') return
  const player = getActivePlayer()
  switch (e.key) {
    case 'ArrowUp':
    case 'w':
    case 'W':
      activeIndex.value = Math.max(0, activeIndex.value - 1)
      break
    case 'ArrowDown':
    case 's':
    case 'S':
      activeIndex.value = Math.min(items.value.length - 1, activeIndex.value + 1)
      break
    case ' ':
      e.preventDefault()
      break
    case 'c':
    case 'C':
      const id = items.value[activeIndex.value]?.video_id
      if (id) {
        commentVideoId.value = id
        commentDrawerOpen.value = true
      }
      break
    case 'f':
    case 'F':
      player?.toggleFullscreen?.()
      break
    case 'ArrowLeft':
      player?.seek?.(-5)
      break
    case 'ArrowRight':
      player?.seek?.(5)
      break
  }
}

</script>

<template>
  <div class="feed-page">
    <FeedTabBar :activeTab="activeTab" @update:tab="onTabChange" />

    <div class="feed-swiper-wrap" :class="{ 'compressed': commentDrawerOpen }">
      <FeedSwiper
        ref="swiperRef"
        :items="items"
        :activeIndex="activeIndex"
        @update:activeIndex="activeIndex = $event"
        @reachEnd="feedStore.loadMore(activeTab)"
      >
        <template #default="{ item, active }">
          <FeedSlideContent
            :item="item"
            :active="active"
            :muted="isMuted"
            @openComments="(videoId) => { commentVideoId = videoId; commentDrawerOpen = true }"
          />
        </template>
      </FeedSwiper>

      <BottomNav v-if="isMobile" />
    </div>

    <CommentDrawer
      v-model="commentDrawerOpen"
      :videoId="commentVideoId"
      seamless
    />
  </div>
</template>

<style scoped lang="scss">
.feed-page {
  height: 100svh;
  display: flex;
  flex-direction: column;
  background: var(--bg-base);
  overflow: hidden;
}

.feed-tab-bar {
  flex-shrink: 0;
}

.feed-swiper-wrap {
  flex: 1;
  overflow: hidden;
  position: relative;
  // Reserve space for bottom nav on mobile so swiper content doesn't hide behind it
  @media (max-width: 1023px) {
    padding-bottom: var(--bottom-nav-height, 60px);
  }

  &.compressed {
    width: 70%;
    transition: width 0.3s ease;
  }
}
</style>