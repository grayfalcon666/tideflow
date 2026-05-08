<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import VideoPlayer from '../components/video/VideoPlayer.vue'
import VideoPlayerContainer from '../components/video/VideoPlayerContainer.vue'
import VideoDetailHeader from '../components/video/VideoDetailHeader.vue'
import VideoMetaSection from '../components/video/VideoMetaSection.vue'
import VideoCommentPreview from '../components/video/VideoCommentPreview.vue'
import CommentDrawer from '../components/comment/CommentDrawer.vue'
import { useVideoControls } from '../composables/useVideoControls'
import * as videoService from '../services/video'
import { useInteractionStore } from '../stores/interaction'
import type { VideoDetail } from '../types'
import TFIcon from '../components/common/TFIcon.vue'

const route = useRoute()
const router = useRouter()
const video = ref<VideoDetail | null>(null)
const loading = ref(true)
const error = ref('')
const showDrawer = ref(false)

const { isMuted } = useVideoControls((key) => {
  if (key === 'c' || key === 'C') {
    showDrawer.value = true
  }
})

const handleViewReported = async (token?: string) => {
  if (!token) return
  try {
    await videoService.recordView(token)
  } catch {}
}

const handleCompletionReported = async (token?: string) => {
  if (!token) return
  try {
    await videoService.recordView(token)
  } catch {}
}

const videoId = Number(route.params.id)

const fetchVideo = async () => {
  try {
    const resp = await videoService.getVideo(videoId)
    const d = resp.data.data
    if (d) {
      // Normalize: backend returns "id" but components expect "video_id"
      video.value = { ...d, video_id: d.id } as VideoDetail
      // sync with persisted interaction store
      const interactionStore = useInteractionStore()
      if (d.is_liked && d.id) interactionStore.syncLike(d.id)
      if (d.is_following_author && d.author?.id) interactionStore.syncFollow(d.author.id)
    } else {
      error.value = '视频不存在或已删除'
    }
  } catch {
    error.value = '视频不存在或已删除'
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  await fetchVideo()
})
</script>

<template>
  <div class="video-detail-page">
    <div v-if="loading" class="loading-state">
      <q-spinner color="accent" size="48px" />
    </div>

    <div v-else-if="error" class="error-state">
      <TFIcon name="error_outline" :size="48" color="var(--text-negative)" />
      <p>{{ error }}</p>
      <q-btn flat no-caps label="返回首页" @click="router.push('/')" />
    </div>

    <template v-else-if="video">
      <VideoDetailHeader :title="video.title" @back="router.back()" />

      <VideoPlayerContainer :fixedRatio="true">
        <VideoPlayer
          :src="video.play_url"
          :poster="video.cover_url"
          :muted="isMuted"
          :autoPlay="true"
          :width="video.width"
          :height="video.height"
          :duration="video.duration"
          :playToken="video.play_token"
          :videoId="videoId"
          @viewReported="handleViewReported(video.play_token)"
          @completionReported="handleCompletionReported(video.play_token)"
        />
      </VideoPlayerContainer>

      <VideoMetaSection :video="video" />

      <VideoCommentPreview
        :videoId="videoId"
        :commentCount="video.comment_count"
        @click="showDrawer = true"
      />

      <CommentDrawer
        v-model="showDrawer"
        :videoId="videoId"
      />
    </template>
  </div>
</template>

<style scoped lang="scss">
.video-detail-page {
  min-height: 100svh;
  width: 100%;
  display: flex;
  flex-direction: column;
  background: var(--bg-base);
}

.loading-state,
.error-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 60svh;
  gap: var(--space-4);
  color: var(--text-secondary);
}
</style>
