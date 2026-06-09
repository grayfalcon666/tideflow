<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import VideoPlayerContainer from '../video/VideoPlayerContainer.vue'
import VideoPlayer from '../video/VideoPlayer.vue'
import SlideAuthorBar from './SlideAuthorBar.vue'
import SlideInfoBar from './SlideInfoBar.vue'
import SlideActionBar from './SlideActionBar.vue'
import CommentDrawer from '../comment/CommentDrawer.vue'
import NotePanel from '../note/NotePanel.vue'
import LearnPanel from '../learn/LearnPanel.vue'

defineProps<{
  item: any
  active: boolean
  muted?: boolean
  noteTimestamps?: number[]
  noteCount?: number
  wordbankStatus?: import('../../types').WordbankStatus
  commentOpen?: boolean
  commentVideoId?: number
  noteOpen?: boolean
  noteVideoId?: number
  currentTime?: number
  learnOpen?: boolean
  learnVideoId?: number
}>()

const emit = defineEmits<{
  (e: 'openComments', videoId: number): void
  (e: 'openNotes', videoId: number): void
  (e: 'share', videoId: number): void
  (e: 'timeupdate'): void
  (e: 'openLearn', videoId: number): void
  (e: 'closeComments'): void
  (e: 'closeNotes'): void
  (e: 'closeLearn'): void
  (e: 'noteSeek', timestamp: number): void
  (e: 'learnSeek', seconds: number): void
  (e: 'learnPause'): void
  (e: 'viewReported'): void
  (e: 'historyReported', videoId: number): void
}>()

// Visibility tracking for Feed page: ≥50% visible for ≥200ms
const playerVisible = ref(false)
let visibleTimer: ReturnType<typeof setTimeout> | null = null
let observer: IntersectionObserver | null = null
const slideRoot = ref<HTMLDivElement>()

onMounted(() => {
  if (!slideRoot.value) return
  observer = new IntersectionObserver(
    (entries) => {
      const entry = entries[0]
      if (entry && entry.intersectionRatio >= 0.5) {
        if (!visibleTimer) {
          visibleTimer = setTimeout(() => { playerVisible.value = true }, 200)
        }
      } else {
        if (visibleTimer) { clearTimeout(visibleTimer); visibleTimer = null }
        playerVisible.value = false
      }
    },
    { threshold: [0, 0.5] }
  )
  observer.observe(slideRoot.value)
})

onUnmounted(() => {
  if (visibleTimer) { clearTimeout(visibleTimer); visibleTimer = null }
  observer?.disconnect()
})
</script>

<template>
  <div ref="slideRoot" class="slide-content" :class="{ 'has-comments': commentOpen }">
    <div class="slide-left">
      <VideoPlayerContainer class="slide-player-container">
        <VideoPlayer
          :src="item?.play_url ?? ''"
          :poster="item?.cover_url ?? ''"
          :muted="muted"
          :autoPlay="active"
          :width="item?.width"
          :height="item?.height"
          :duration="item?.duration"
          :playToken="item?.play_token"
          :videoId="item?.video_id"
          :noteTimestamps="noteTimestamps"
          :visible="playerVisible"
          @timeupdate="emit('timeupdate')"
          @viewReported="emit('viewReported')"
          @historyReported="(vid) => emit('historyReported', vid)"
        />
      </VideoPlayerContainer>

      <div class="slide-overlay-author">
        <SlideAuthorBar :item="item" />
      </div>
      <div class="slide-overlay-info">
        <SlideInfoBar :item="item" />
      </div>
      <div class="slide-overlay-actions">
        <SlideActionBar
          :item="item"
          :noteCount="noteCount ?? 0"
          :wordbankStatus="wordbankStatus"
          @openComments="emit('openComments', item?.video_id)"
          @openNotes="emit('openNotes', item?.video_id)"
          @share="emit('share', item?.video_id)"
          @openLearn="emit('openLearn', item?.video_id)"
        />
      </div>
    </div>

    <div class="slide-right" :class="{ open: commentOpen || noteOpen || learnOpen }">
      <CommentDrawer
        v-if="commentOpen"
        :videoId="commentVideoId ?? item?.video_id ?? 0"
        :open="commentOpen"
        @close="emit('closeComments')"
      />
      <NotePanel
        v-if="noteOpen"
        :videoId="noteVideoId ?? item?.video_id ?? 0"
        :currentTime="currentTime ?? 0"
        :open="noteOpen"
        @seek="(ts) => emit('noteSeek', ts)"
        @close="emit('closeNotes')"
      />
      <LearnPanel
        v-if="learnOpen"
        :open="learnOpen"
        :videoId="learnVideoId ?? item?.video_id ?? 0"
        :videoSrc="item?.play_url"
        :videoPoster="item?.cover_url"
        @close="emit('closeLearn')"
        @seekTo="(ts) => emit('learnSeek', ts)"
        @pause="emit('learnPause')"
      />
    </div>
  </div>
</template>

<style scoped lang="scss">
.slide-content {
  position: relative;
  width: 100%;
  height: 100%;
  display: flex;
}

.slide-left {
  flex: 1;
  position: relative;
  min-width: 0;
  height: 100%;
}

.slide-right {
  width: 0;
  flex-shrink: 0;
  height: 100%;
  overflow: hidden;
  transition: width 0.35s cubic-bezier(0.4, 0, 0.2, 1);

  &.open {
    width: 400px;
    box-shadow: inset 1px 0 0 var(--border);
  }

  // Mobile: overlay on top of video instead of pushing aside
  @media (max-width: 1023px) {
    position: absolute;
    top: 0;
    right: 0;
    z-index: 100;

    &.open {
      width: 100%;
    }
  }
}

.slide-player-container {
  width: 100%;
  height: 100%;
}

.slide-overlay-author {
  position: absolute;
  top: var(--space-4);
  right: var(--space-4);
  z-index: 20;
}

.slide-overlay-info {
  position: absolute;
  bottom: 80px;
  left: 0;
  right: 80px;
  z-index: 20;
}

.slide-overlay-actions {
  position: absolute;
  right: var(--space-4);
  bottom: 100px;
  z-index: 20;
}
</style>
