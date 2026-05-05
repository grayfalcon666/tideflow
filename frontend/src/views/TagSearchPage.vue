<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import * as videoService from '../services/video'
import type { VideoItem } from '../types'
import TFIcon from '../components/common/TFIcon.vue'

const route = useRoute()
const router = useRouter()
const tagName = String(route.params.tagName)
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
    const resp = await videoService.getVideosByTag(
      tagName,
      cursor.value ?? undefined,
      10
    )
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
  <div class="tag-search-page">
    <div class="tag-header">
      <div class="back-btn" @click="router.back()">
        <TFIcon name="arrow_back" :size="24" />
      </div>
      <h2>#{{ tagName }}</h2>
    </div>

    <div v-if="!items.length && !loading" class="empty-state">
      <TFIcon name="search_off" :size="64" color="var(--text-secondary)" />
      <p>未找到相关视频</p>
    </div>

    <div class="tag-video-list">
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
          <div class="card-author">{{ item.author.username }}</div>
          <div class="card-stats">
            <TFIcon name="favorite" :size="14" />
            {{ formatCount(item.likes_count) }}
          </div>
        </div>
      </div>
    </div>

    <div v-if="loading" class="loading-state">
      <q-spinner color="accent" size="32px" />
    </div>
    <div v-if="!hasMore && items.length" class="no-more">没有更多了</div>
  </div>
</template>

<style scoped lang="scss">
.tag-search-page {
  max-width: 900px;
  margin: 0 auto;
  padding: var(--space-4);
  padding-bottom: 80px;
}

.tag-header {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  margin-bottom: var(--space-4);

  h2 { font-size: 20px; }
}

.back-btn {
  cursor: pointer;
  padding: var(--space-1);
  display: flex;
  align-items: center;
  color: var(--text-secondary);

  &:hover { color: var(--text-base); }
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 60px 0;
  gap: var(--space-4);
  color: var(--text-secondary);
}

.tag-video-list {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: var(--space-3);

  @media (max-width: 600px) {
    grid-template-columns: 1fr;
  }
}

.video-card {
  background: var(--bg-card);
  border-radius: var(--radius-md);
  overflow: hidden;
  cursor: pointer;
  transition: transform var(--transition-fast);

  &:hover { transform: translateY(-2px); }
}

.card-cover {
  aspect-ratio: 16 / 9;
  background: var(--bg-elevated);

  img { width: 100%; height: 100%; object-fit: cover; }
}

.card-info {
  padding: var(--space-3);
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.card-title {
  font-size: 14px;
  font-weight: 700;
  color: var(--text-base);
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  line-height: 1.3;
  margin: 0;
}

.card-author {
  font-size: 12px;
  color: var(--text-secondary);
}

.card-stats {
  font-size: 12px;
  color: var(--text-secondary);
  display: flex;
  align-items: center;
  gap: 2px;
}

.loading-state {
  display: flex;
  justify-content: center;
  padding: var(--space-6);
}

.no-more {
  text-align: center;
  padding: var(--space-4);
  color: var(--text-secondary);
  font-size: 14px;
}
</style>
