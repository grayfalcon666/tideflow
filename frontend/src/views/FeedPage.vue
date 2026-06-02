<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import FeedTabBar from '../components/feed/FeedTabBar.vue'
import FeedSwiper from '../components/feed/FeedSwiper.vue'
import FeedSlideContent from '../components/feed/FeedSlideContent.vue'
import ShareMenu from '../components/video/ShareMenu.vue'
import BottomNav from '../components/layout/BottomNav.vue'
import { useVideoControls } from '../composables/useVideoControls'
import { useFeedStore, type FeedTab } from '../stores/feed'
import { useAuthStore } from '../stores/auth'
import { useNotificationStore } from '../stores/notification'
import { useSettingsStore } from '../stores/settings'
import * as noteService from '../services/note'
import type { VideoItem, RawNote } from '../types'

const router = useRouter()
const feedStore = useFeedStore()
const authStore = useAuthStore()
const notifStore = useNotificationStore()
const settingsStore = useSettingsStore()

const isMobile = ref(window.innerWidth < 1024)
const onResize = () => { isMobile.value = window.innerWidth < 1024 }

const activeTab = ref<FeedTab>('latest')
const activeIndex = ref(0)
const commentDrawerOpen = ref(false)
const commentVideoId = ref(0)
const shareMenuOpen = ref(false)
const shareVideoId = ref(0)
const shareVideoTitle = ref('')
const shareVideoCover = ref('')
const shareAuthorName = ref('')
const notePanelOpen = ref(false)
const noteVideoId = ref(0)
const learnPanelOpen = ref(false)
const learnVideoId = ref(0)
const feedPlayerTime = ref(0)
const noteTimestampsMap = ref<Record<number, number[]>>({})
let timePollTimer: ReturnType<typeof setInterval> | null = null
const playToken = ref<string | null>(null)
let pendingRestoreIndex = 0

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

// Note panel helpers
const noteTimestamps = computed(() => {
  const id = items.value[activeIndex.value]?.video_id
  return id ? (noteTimestampsMap.value[id] ?? []) : []
})

const noteCount = computed(() => {
  const id = items.value[activeIndex.value]?.video_id
  return id ? (noteTimestampsMap.value[id]?.length ?? 0) : 0
})

const fetchFeedNoteTimestamps = async (videoId: number) => {
  if (noteTimestampsMap.value[videoId]) return
  try {
    const resp = await noteService.getNotes(videoId, '0', 200)
    const rawItems = (resp.data.data?.items ?? []) as RawNote[]
    noteTimestampsMap.value[videoId] = rawItems.map((n: RawNote) => n.timestamp)
  } catch {}
}

const startTimePoll = () => {
  if (timePollTimer) return
  timePollTimer = setInterval(() => {
    if (!notePanelOpen.value) return
    const player = getActivePlayer()
    if (player) {
      feedPlayerTime.value = player.getCurrentTime?.() ?? 0
    }
  }, 500)
}

const stopTimePoll = () => {
  if (timePollTimer) {
    clearInterval(timePollTimer)
    timePollTimer = null
  }
}

const handleShare = (item: any) => {
  shareVideoId.value = item.video_id ?? item.id
  shareVideoTitle.value = item.title ?? ''
  shareVideoCover.value = item.cover_url ?? ''
  shareAuthorName.value = item.author?.username ?? item.username ?? ''
  shareMenuOpen.value = true
}

const handleOpenNotes = (videoId: number) => {
  if (notePanelOpen.value && noteVideoId.value === videoId) {
    notePanelOpen.value = false
    stopTimePoll()
    return
  }
  commentDrawerOpen.value = false
  learnPanelOpen.value = false
  noteVideoId.value = videoId
  notePanelOpen.value = true
  fetchFeedNoteTimestamps(videoId)
  startTimePoll()
}

const handleOpenComments = (videoId: number) => {
  if (commentDrawerOpen.value && commentVideoId.value === videoId) {
    commentDrawerOpen.value = false
    return
  }
  notePanelOpen.value = false
  learnPanelOpen.value = false
  commentVideoId.value = videoId
  commentDrawerOpen.value = true
}

const handleCloseNotes = () => {
  notePanelOpen.value = false
  stopTimePoll()
}

const handleCloseLearn = () => {
  learnPanelOpen.value = false
}

const handleOpenLearn = (videoId: number) => {
  if (learnPanelOpen.value && learnVideoId.value === videoId) {
    learnPanelOpen.value = false
    return
  }
  commentDrawerOpen.value = false
  notePanelOpen.value = false
  learnVideoId.value = videoId
  learnPanelOpen.value = true
}

const handleFeedNoteSeek = (timestamp: number) => {
  const player = getActivePlayer()
  player?.seekTo?.(timestamp)
}

// Fetch note timestamps + poll for current video
watch(notePanelOpen, (val) => {
  if (val && noteVideoId.value) {
    fetchFeedNoteTimestamps(noteVideoId.value)
    startTimePoll()
  } else {
    stopTimePoll()
  }
})

watch(activeIndex, () => {
  const id = items.value[activeIndex.value]?.video_id
  if (id) {
    fetchFeedNoteTimestamps(id)
  }
  if (notePanelOpen.value && noteVideoId.value) {
    if (id && id !== noteVideoId.value) {
      noteVideoId.value = id
    }
  }
})

// 数据加载完成后恢复上次位置
watch(() => items.value.length, (len) => {
  if (len > 0 && pendingRestoreIndex > 0) {
    const idx = Math.min(pendingRestoreIndex, len - 1)
    activeIndex.value = idx
    pendingRestoreIndex = 0
    // 确保 swiper 滚动到正确位置，同时暂停非活跃播放器
    nextTick(() => {
      swiperRef.value?.scrollToIndex?.(idx)
      setTimeout(() => pauseAllExcept(idx), 200)
    })
  }
})

watch([activeIndex, activeTab], () => {
  settingsStore.savePosition(activeIndex.value, activeTab.value)
})

const { isMuted } = useVideoControls((key) => {
  if (learnPanelOpen.value) return
  if (key === 'c' || key === 'C') {
    const id = items.value[activeIndex.value]?.video_id
    if (id) {
      handleOpenComments(id)
    }
  }
})

watch(activeIndex, (newIdx, oldIdx) => {
  if (newIdx !== oldIdx) {
    commentDrawerOpen.value = false
    notePanelOpen.value = false
    learnPanelOpen.value = false
  }
  if (newIdx >= items.value.length - 3 && hasMore.value && !isLoading.value) {
    feedStore.loadMore(activeTab.value)
  }
  // Pause all except current
  pauseAllExcept(newIdx)
  const currPlayer = getPlayerAtIndex(newIdx)
  currPlayer?.play()
})

onMounted(() => {
  // 恢复上次刷到的位置 —— 先记录目标位置，等数据加载完再跳转
  if (settingsStore.rememberPosition && settingsStore.lastActiveIndex > 0) {
    activeTab.value = settingsStore.lastActiveTab
    pendingRestoreIndex = settingsStore.lastActiveIndex
  }
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
  pauseAllPlayers()
  window.removeEventListener('keydown', onKeyDown)
  window.removeEventListener('resize', onResize)
  stopTimePoll()
})

const getActivePlayer = () => {
  return getPlayerAtIndex(activeIndex.value)
}

const getPlayerAtIndex = (idx: number) => {
  const slides = swiperRef.value?.$el?.querySelectorAll('.feed-slide')
  const slide = slides?.[idx]
  return (slide?.querySelector('.video-player') as any)?.__vueParentComponent?.exposed
}

const pauseAllExcept = (exceptIdx: number) => {
  const slides = swiperRef.value?.$el?.querySelectorAll('.feed-slide')
  if (!slides) return
  slides.forEach((slide: Element, i: number) => {
    if (i === exceptIdx) return
    const player = (slide.querySelector('.video-player') as any)?.__vueParentComponent?.exposed
    player?.pause()
  })
}

const pauseAllPlayers = () => {
  const slides = swiperRef.value?.$el?.querySelectorAll('.feed-slide')
  if (!slides) return
  slides.forEach((slide: Element) => {
    const player = (slide.querySelector('.video-player') as any)?.__vueParentComponent?.exposed
    player?.pause()
  })
}

// Keyboard shortcuts
const onKeyDown = (e: KeyboardEvent) => {
  // Skip all shortcuts when learn panel or other overlays are open
  if (learnPanelOpen.value) return
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
        handleOpenComments(id)
      }
      break
    case 'n':
    case 'N':
      const nid = items.value[activeIndex.value]?.video_id
      if (nid) {
        handleOpenNotes(nid)
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

    <div class="feed-swiper-wrap">
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
            :noteTimestamps="noteTimestamps"
            :noteCount="noteCount"
            :commentOpen="commentDrawerOpen && commentVideoId === item?.video_id"
            :commentVideoId="commentVideoId"
            :noteOpen="notePanelOpen && noteVideoId === item?.video_id"
            :noteVideoId="noteVideoId"
            :currentTime="feedPlayerTime"
            :learnOpen="learnPanelOpen && learnVideoId === item?.video_id"
            :learnVideoId="learnVideoId"
            @openComments="(videoId) => handleOpenComments(videoId)"
            @closeComments="commentDrawerOpen = false"
            @openNotes="(videoId) => handleOpenNotes(videoId)"
            @closeNotes="handleCloseNotes"
            @closeLearn="handleCloseLearn"
            @noteSeek="(ts) => handleFeedNoteSeek(ts)"
            @share="(videoId: number) => handleShare(item)"
            @openLearn="(videoId: number) => handleOpenLearn(videoId)"
            :wordbankStatus="item.wordbank_status"
            @timeupdate="notePanelOpen && (feedPlayerTime = getActivePlayer()?.getCurrentTime?.() ?? 0)"
          />
        </template>
      </FeedSwiper>

      <BottomNav v-if="isMobile" />
    </div>

    <ShareMenu
      v-model="shareMenuOpen"
      :videoId="shareVideoId"
      :videoTitle="shareVideoTitle"
      :videoCover="shareVideoCover"
      :authorName="shareAuthorName"
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

}
</style>