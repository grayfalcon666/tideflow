<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import VideoPlayer from '../components/video/VideoPlayer.vue'
import VideoMetaSection from '../components/video/VideoMetaSection.vue'
import VideoCommentPreview from '../components/video/VideoCommentPreview.vue'
import CommentDrawer from '../components/comment/CommentDrawer.vue'
import * as videoService from '../services/video'
import type { VideoDetail } from '../types'
import TFIcon from '../components/common/TFIcon.vue'

const route = useRoute()
const router = useRouter()
const video = ref<VideoDetail | null>(null)
const loading = ref(true)
const error = ref('')
const showDrawer = ref(false)

const videoId = Number(route.params.id)

const fetchVideo = async () => {
  try {
    const resp = await videoService.getVideo(videoId)
    const d = resp.data.data
    if (d) {
      video.value = d
    } else {
      error.value = '视频不存在或已删除'
    }
  } catch {
    error.value = '视频不存在或已删除'
  } finally {
    loading.value = false
  }
}

const onKeyDown = (e: KeyboardEvent) => {
  if ((e.target as HTMLElement).tagName === 'INPUT') return
  if (e.key === 'c' || e.key === 'C') {
    showDrawer.value = true
  }
}

onMounted(async () => {
  await fetchVideo()
  window.addEventListener('keydown', onKeyDown)
})

onUnmounted(() => {
  window.removeEventListener('keydown', onKeyDown)
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

      <div class="video-player-wrap">
        <VideoPlayer
          :src="video.play_url"
          :poster="video.cover_url"
          :muted="true"
          :autoPlay="true"
        />
      </div>

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

<script lang="ts">
// Sub-component defined inline
import { defineComponent } from 'vue'
const VideoDetailHeader = defineComponent({
  props: { title: String },
  emits: ['back'],
  template: `
    <div class="detail-header">
      <div class="back-btn" @click="$emit('back')">
        <TFIcon name="arrow_back" :size="24" />
      </div>
      <h2 class="detail-title">{{ title }}</h2>
    </div>
  `,
})
</script>

<style scoped lang="scss">
.video-detail-page {
  min-height: 100svh;
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

.video-player-wrap {
  width: 100%;
  max-width: 900px;
  margin: 0 auto;
  background: #000;
  aspect-ratio: 16 / 9;
}

.detail-header {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-4);
  background: var(--bg-surface);
  border-bottom: 1px solid var(--border);
  position: sticky;
  top: 0;
  z-index: 10;
}

.detail-title {
  flex: 1;
  font-size: 16px;
  font-weight: 700;
  color: var(--text-base);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  margin: 0;
}
</style>
