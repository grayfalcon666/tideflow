<script setup lang="ts">
import VideoPlayerContainer from '../video/VideoPlayerContainer.vue'
import VideoPlayer from '../video/VideoPlayer.vue'
import SlideAuthorBar from './SlideAuthorBar.vue'
import SlideInfoBar from './SlideInfoBar.vue'
import SlideActionBar from './SlideActionBar.vue'

defineProps<{
  item: any
  active: boolean
  muted?: boolean
  noteTimestamps?: number[]
  noteCount?: number
}>()

const emit = defineEmits<{
  (e: 'openComments', videoId: number): void
  (e: 'openNotes', videoId: number): void
  (e: 'share', videoId: number): void
  (e: 'timeupdate'): void
}>()
</script>

<template>
  <div class="slide-content">
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
          @timeupdate="emit('timeupdate')"
        />
      </VideoPlayerContainer>

    <div class="slide-overlay-author">
      <SlideAuthorBar :item="item" />
    </div>
    <div class="slide-overlay-info">
      <SlideInfoBar :item="item" />
    </div>
    <div class="slide-overlay-actions">
      <SlideActionBar :item="item" :noteCount="noteCount ?? 0" @openComments="emit('openComments', item?.video_id)" @openNotes="emit('openNotes', item?.video_id)" @share="emit('share', item?.video_id)" />
    </div>
  </div>
</template>

<style scoped lang="scss">
.slide-content {
  position: relative;
  width: 100%;
  height: 100%;
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