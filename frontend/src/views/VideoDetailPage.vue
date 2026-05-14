<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import VideoPlayer from '../components/video/VideoPlayer.vue'
import VideoPlayerContainer from '../components/video/VideoPlayerContainer.vue'
import VideoDetailHeader from '../components/video/VideoDetailHeader.vue'
import VideoMetaSection from '../components/video/VideoMetaSection.vue'
import VideoCommentPreview from '../components/video/VideoCommentPreview.vue'
import CommentDrawer from '../components/comment/CommentDrawer.vue'
import NotePanel from '../components/note/NotePanel.vue'
import LearnPanel from '../components/learn/LearnPanel.vue'
import { useVideoControls } from '../composables/useVideoControls'
import * as videoService from '../services/video'
import * as noteService from '../services/note'
import { useInteractionStore } from '../stores/interaction'
import type { VideoDetail, RawNote } from '../types'
import TFIcon from '../components/common/TFIcon.vue'

const route = useRoute()
const router = useRouter()
const video = ref<VideoDetail | null>(null)
const isDeleted = ref(false)
const loading = ref(true)
const error = ref('')
const showDrawer = ref(false)
const showNotePanel = ref(false)
const showLearnPanel = ref(false)
const videoPlayerRef = ref<InstanceType<typeof VideoPlayer>>()
const playerCurrentTime = ref(0)
const playerNoteTimestamps = ref<number[]>([])

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

const handleTimeUpdate = () => {
  playerCurrentTime.value = videoPlayerRef.value?.getCurrentTime() ?? 0
}

const handleNoteSeek = (timestamp: number) => {
  videoPlayerRef.value?.seekTo(timestamp)
}

const fetchNoteTimestamps = async () => {
  try {
    const resp = await noteService.getNotes(videoId, '0', 200)
    const items = (resp.data.data?.items ?? []) as RawNote[]
    playerNoteTimestamps.value = items.map((n: RawNote) => n.timestamp)
  } catch {}
}

const videoId = Number(route.params.id)

const fetchVideo = async () => {
  try {
    const resp = await videoService.getVideo(videoId)
    const d = resp.data.data
    if (d) {
      if ((d as any).is_deleted) {
        isDeleted.value = true
        loading.value = false
        return
      }
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
  fetchNoteTimestamps()
})
</script>

<template>
  <div class="video-detail-page">
    <div v-if="loading" class="loading-state">
      <q-spinner color="accent" size="48px" />
    </div>

    <div v-else-if="isDeleted" class="error-state">
      <TFIcon name="delete" :size="48" color="var(--text-secondary)" />
      <p>视频已删除</p>
      <q-btn flat no-caps label="返回首页" @click="router.push('/')" />
    </div>

    <div v-else-if="error" class="error-state">
      <TFIcon name="error_outline" :size="48" color="#ef4444" />
      <p>{{ error }}</p>
      <q-btn flat no-caps label="返回首页" @click="router.push('/')" />
    </div>

    <template v-else-if="video">
      <VideoDetailHeader :title="video.title" @back="router.back()" />

      <VideoPlayerContainer :fixedRatio="true">
        <VideoPlayer
          ref="videoPlayerRef"
          :src="video.play_url"
          :poster="video.cover_url"
          :muted="isMuted"
          :autoPlay="true"
          :width="video.width"
          :height="video.height"
          :duration="video.duration"
          :playToken="video.play_token"
          :videoId="videoId"
          :noteTimestamps="playerNoteTimestamps"
          @viewReported="handleViewReported(video.play_token)"
          @completionReported="handleCompletionReported(video.play_token)"
          @timeupdate="handleTimeUpdate"
        />
      </VideoPlayerContainer>

      <VideoMetaSection :video="video" />

      <!-- Note entry (above comments) -->
      <div class="note-entry" @click="showNotePanel = true">
        <TFIcon name="sticky_note_2" :size="20" color="var(--text-secondary)" />
        <span class="note-entry-label">时间轴笔记</span>
        <TFIcon name="chevron_right" :size="20" color="var(--text-muted)" />
      </div>

      <div
        v-if="video.wordbank_status === 'ready'"
        class="note-entry"
        @click="showLearnPanel = true"
      >
        <TFIcon name="school" :size="20" color="#1ed760" />
        <span class="note-entry-label">学习模式</span>
        <TFIcon name="chevron_right" :size="20" color="var(--text-muted)" />
      </div>

      <VideoCommentPreview
        :videoId="videoId"
        :commentCount="video.comment_count"
        @click="showDrawer = true"
      />

      <CommentDrawer
        v-model="showDrawer"
        :videoId="videoId"
      />

      <NotePanel
        v-model="showNotePanel"
        :videoId="videoId"
        :currentTime="playerCurrentTime"
        @seek="handleNoteSeek"
      />

      <LearnPanel
        v-model="showLearnPanel"
        :videoId="videoId"
        :videoPlayerRef="videoPlayerRef"
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

.note-entry {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  cursor: pointer;
  border-bottom: 1px solid var(--border-color-light, rgba(0,0,0,0.04));
  transition: background 0.15s;

  &:hover {
    background: var(--bg-hover);
  }
}

.note-entry-label {
  flex: 1;
  font-size: 14px;
  color: var(--text-secondary);
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
