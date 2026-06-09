<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useQuasar } from 'quasar'
import TFIcon from '../components/common/TFIcon.vue'
import * as historyApi from '../services/history'
import type { VideoItem } from '../types'

const router = useRouter()
const $q = useQuasar()

const videos = ref<VideoItem[]>([])
const loading = ref(false)
const hasMore = ref(false)
const nextCursor = ref<string | null>(null)
const clearing = ref(false)

const timeAgo = (ts: string | number) => {
  const timestamp = typeof ts === 'string' ? new Date(ts).getTime() : ts * 1000
  const diff = Date.now() - timestamp
  const min = Math.floor(diff / 60000)
  if (min < 1) return '刚刚'
  if (min < 60) return `${min}分钟前`
  const hr = Math.floor(min / 60)
  if (hr < 24) return `${hr}小时前`
  const day = Math.floor(hr / 24)
  return `${day}天前`
}

const formatDuration = (seconds?: number) => {
  if (!seconds) return ''
  const m = Math.floor(seconds / 60)
  const s = Math.floor(seconds % 60)
  return `${m}:${String(s).padStart(2, '0')}`
}

const fetchHistory = async (cursor?: string) => {
  if (loading.value) return
  loading.value = true
  try {
    const resp = await historyApi.getHistory(cursor)
    const data = resp.data.data
    const items = (data.items || []).map((v: any) => ({ ...v, video_id: v.video_id ?? v.id }))
    if (cursor) {
      videos.value.push(...items)
    } else {
      videos.value = items
    }
    nextCursor.value = data.next_cursor
    hasMore.value = data.has_more
  } catch (e) {
    console.error('获取观看历史失败', e)
  } finally {
    loading.value = false
  }
}

const loadMore = () => {
  if (hasMore.value && nextCursor.value) {
    fetchHistory(nextCursor.value)
  }
}

const deleteItem = async (videoId: number) => {
  try {
    await historyApi.deleteHistoryItem(videoId)
    videos.value = videos.value.filter(v => v.video_id !== videoId)
    $q.notify({ message: '已删除', type: 'positive', position: 'top' })
  } catch (e) {
    console.error('删除失败', e)
  }
}

const clearAll = async () => {
  $q.dialog({
    title: '清空观看历史',
    message: '确定要清空所有观看历史吗？此操作不可撤销。',
    cancel: { label: '取消', flat: true, color: 'grey' },
    ok: { label: '清空', color: 'negative' },
  }).onOk(async () => {
    clearing.value = true
    try {
      await historyApi.clearHistory()
      videos.value = []
      $q.notify({ message: '已清空观看历史', type: 'positive', position: 'top' })
    } catch (e) {
      console.error('清空失败', e)
    } finally {
      clearing.value = false
    }
  })
}

const goToVideo = (videoId: number) => {
  router.push(`/video/${videoId}`)
}

onMounted(() => fetchHistory())
</script>

<template>
  <div class="history-page">
    <div class="history-header">
      <div class="header-left">
        <button class="back-btn" @click="router.back()">
          <TFIcon name="arrow_back" :size="22" />
        </button>
        <h2 class="page-title">观看历史</h2>
        <span v-if="videos.length" class="video-count">{{ videos.length }} 个视频</span>
      </div>
      <q-btn
        v-if="videos.length"
        flat
        no-caps
        dense
        label="清空历史"
        class="clear-btn"
        :loading="clearing"
        @click="clearAll"
      />
    </div>

    <div v-if="!loading && !videos.length" class="empty-state">
      <div class="empty-icon">
        <svg width="64" height="64" viewBox="0 0 24 24" fill="none">
          <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z" fill="currentColor" opacity="0.3"/>
          <path d="M13 7h-2v6h6v-2h-4V7z" fill="currentColor" opacity="0.6"/>
        </svg>
      </div>
      <p class="empty-text">还没有观看记录</p>
      <p class="empty-hint">去看一些视频吧</p>
    </div>

    <div v-else class="video-list">
      <div
        v-for="video in videos"
        :key="video.video_id"
        class="video-card"
        @click="goToVideo(video.video_id)"
      >
        <div class="card-cover">
          <img :src="video.cover_url" :alt="video.title" />
          <div v-if="video.duration" class="duration-badge">
            {{ formatDuration(video.duration) }}
          </div>
          <div class="play-overlay">
            <svg width="32" height="32" viewBox="0 0 24 24" fill="white">
              <path d="M8 5v14l11-7z"/>
            </svg>
          </div>
        </div>
        <div class="card-info">
          <div class="card-title">{{ video.title }}</div>
          <div class="card-meta">
            <span class="author-name">{{ video.author?.username || video.username }}</span>
            <span class="meta-dot">·</span>
            <span class="view-count">{{ video.view_count || 0 }} 次观看</span>
          </div>
          <div class="card-time">{{ timeAgo(video.create_time) }}</div>
        </div>
        <button
          class="delete-btn"
          @click.stop="deleteItem(video.video_id)"
        >
          <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor">
            <path d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/>
          </svg>
        </button>
      </div>
    </div>

    <div v-if="loading" class="loading-state">
      <q-spinner color="accent" size="36px" />
    </div>

    <div v-if="hasMore && !loading" class="load-more">
      <q-btn flat no-caps label="加载更多" class="load-more-btn" @click="loadMore" />
    </div>
  </div>
</template>

<style scoped lang="scss">
.history-page {
  max-width: 800px;
  margin: 0 auto;
  padding: var(--space-4);
  padding-bottom: 80px;
  min-height: 100vh;

  @media (max-width: 520px) {
    padding: var(--space-3);
  }
}

.history-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--space-5);
  padding-bottom: var(--space-3);
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
}

.header-left {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.back-btn {
  background: none;
  border: none;
  cursor: pointer;
  color: var(--text-base);
  padding: 4px;
  display: flex;
  align-items: center;
  justify-content: center;

  &:hover {
    opacity: 0.7;
  }
}

.page-title {
  font-size: 24px;
  font-weight: 700;
  color: var(--text-base);
  margin: 0;
}

.video-count {
  font-size: 13px;
  color: var(--text-secondary);
}

.clear-btn {
  color: var(--text-secondary);
  font-size: 13px;
  transition: color 0.2s;

  &:hover {
    color: #ff4444;
  }
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 80px 0;
  gap: var(--space-3);
}

.empty-icon {
  color: var(--text-secondary);
  opacity: 0.4;
}

.empty-text {
  font-size: 16px;
  color: var(--text-secondary);
  margin: 0;
}

.empty-hint {
  font-size: 13px;
  color: var(--text-secondary);
  opacity: 0.6;
  margin: 0;
}

.video-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.video-card {
  display: flex;
  align-items: center;
  gap: var(--space-4);
  padding: var(--space-3);
  border-radius: var(--radius-md);
  background: var(--bg-card);
  cursor: pointer;
  transition: background 0.2s;

  &:hover {
    background: var(--bg-elevated);

    .play-overlay {
      opacity: 1;
    }

    .delete-btn {
      opacity: 1;
    }
  }
}

.card-cover {
  position: relative;
  width: 160px;
  min-width: 160px;
  aspect-ratio: 16 / 9;
  border-radius: var(--radius-sm);
  overflow: hidden;
  background: #1a1a1a;

  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  @media (max-width: 520px) {
    width: 120px;
    min-width: 120px;
  }
}

.duration-badge {
  position: absolute;
  bottom: 4px;
  right: 4px;
  background: rgba(0, 0, 0, 0.8);
  color: #fff;
  font-size: 11px;
  padding: 1px 4px;
  border-radius: 2px;
  font-variant-numeric: tabular-nums;
}

.play-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.4);
  opacity: 0;
  transition: opacity 0.2s;
}

.card-info {
  flex: 1;
  min-width: 0;
}

.card-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--text-base);
  margin-bottom: 4px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.card-meta {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
  color: var(--text-secondary);
  margin-bottom: 2px;
}

.meta-dot {
  opacity: 0.5;
}

.card-time {
  font-size: 12px;
  color: var(--text-secondary);
  opacity: 0.7;
}

.delete-btn {
  opacity: 0;
  color: var(--text-secondary);
  background: none;
  border: none;
  cursor: pointer;
  padding: 6px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;

  &:hover {
    color: #ff4444;
    background: rgba(255, 68, 68, 0.1);
  }
}

.loading-state {
  display: flex;
  justify-content: center;
  padding: var(--space-6);
}

.load-more {
  display: flex;
  justify-content: center;
  padding: var(--space-4);
}

.load-more-btn {
  color: var(--accent);
  font-size: 14px;

  &:hover {
    background: rgba(var(--accent-rgb, 29 185 84), 0.1);
  }
}
</style>
