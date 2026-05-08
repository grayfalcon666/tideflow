<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import * as videoService from '../../services/video'
import type { VideoItem } from '../../types'
import TFIcon from '../common/TFIcon.vue'

const router = useRouter()
const items = ref<VideoItem[]>([])
const cursor = ref<string | null>(null)
const hasMore = ref(true)
const loading = ref(false)

const load = async (reset = false) => {
  if (loading.value) return
  loading.value = true
  try {
    if (reset) {
      items.value = []
      cursor.value = null
      hasMore.value = true
    }
    const resp = await videoService.getMyLiked(cursor.value ?? undefined, 10)
    const d = resp.data.data
    if (!d) return
    if (reset) {
      items.value = d.items ?? []
    } else {
      items.value.push(...(d.items ?? []))
    }
    cursor.value = d.next_cursor
    hasMore.value = d.has_more ?? false
  } finally {
    loading.value = false
  }
}

onMounted(() => load())

const formatCount = (n: number) => {
  if (n >= 10000) return `${(n / 10000).toFixed(1)}w`
  if (n >= 1000) return `${(n / 1000).toFixed(1)}k`
  return String(n)
}
</script>

<template>
  <div class="user-liked-tab">
    <div v-if="!items.length && !loading" class="empty-state">
      <TFIcon name="favorite_border" :size="48" color="var(--text-secondary)" />
      <p>还没有喜欢的视频</p>
    </div>
    <div class="video-grid">
      <div
        v-for="item in items"
        :key="item.video_id"
        class="video-card"
        @click="router.push(`/video/${item.video_id}`)"
      >
        <div class="card-cover">
          <img :src="item.cover_url" loading="lazy" />
        </div>
        <div class="card-info">
          <h4 class="card-title">{{ item.title }}</h4>
          <div class="card-stats">
            <TFIcon name="favorite" :size="12" />
            {{ formatCount(item.likes_count) }}
            <TFIcon name="chat_bubble_outline" :size="12" />
            {{ formatCount(item.comment_count) }}
            <TFIcon name="visibility" :size="12" />
            {{ formatCount(item.view_count ?? 0) }}
          </div>
        </div>
      </div>
    </div>
    <div v-if="loading" class="loading-state">
      <q-spinner color="accent" size="28px" />
    </div>
    <div
      v-if="hasMore && items.length"
      class="load-more"
      @click="load()"
    >
      <span v-if="!loading">加载更多</span>
      <q-spinner v-else color="accent" size="20px" />
    </div>
    <div v-if="!hasMore && items.length" class="no-more">没有更多了</div>
  </div>
</template>

<style scoped lang="scss">
.user-liked-tab {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 40px 0;
  gap: var(--space-3);
  color: var(--text-secondary);
}

.video-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: var(--space-2);

  @media (max-width: 600px) {
    grid-template-columns: repeat(2, 1fr);
  }
}

.video-card {
  cursor: pointer;
  overflow: hidden;
  border-radius: var(--radius-sm);
  background: var(--bg-card);
}

.card-cover {
  aspect-ratio: 3 / 4;
  background: var(--bg-elevated);

  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
}

.card-info {
  padding: var(--space-2);
}

.card-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-base);
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  margin: 0 0 4px;
}

.card-stats {
  font-size: 12px;
  color: var(--text-secondary);
  display: flex;
  align-items: center;
  gap: 2px;
}

.loading-state, .load-more, .no-more {
  display: flex;
  justify-content: center;
  padding: var(--space-3);
  color: var(--text-secondary);
  font-size: 13px;
  cursor: pointer;
}
</style>
