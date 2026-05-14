<script setup lang="ts">
import { ref, onMounted } from 'vue'
import * as feedService from '../services/feed'
import HotVideoCard from '../components/hot/HotVideoCard.vue'

type WindowOption = '1m' | '5m' | '15m' | '1h' | '6h'

type RawVideoItem = {
  id: number
  author_id: number
  username: string
  avatar_url?: string
  title: string
  description?: string
  play_url: string
  cover_url: string
  create_time: string
  likes_count: number
  comment_count: number
  view_count?: number
  popularity: number
  is_liked: boolean
  is_big_v?: boolean
  tags?: string[]
  wordbank_status?: string
  subtitle_status?: string
}

const normalizeVideos = (items: RawVideoItem[]): any[] =>
  (items ?? []).filter(Boolean).map(v => ({
    video_id: v.id,
    author_id: v.author_id,
    username: v.username,
    title: v.title,
    description: v.description,
    play_url: v.play_url,
    cover_url: v.cover_url,
    create_time: new Date(v.create_time).getTime() / 1000,
    likes_count: v.likes_count,
    comment_count: v.comment_count,
    view_count: v.view_count,
    popularity: v.popularity,
    is_liked: v.is_liked,
    author: {
      id: v.author_id,
      username: v.username,
      avatar_url: v.avatar_url ?? '',
      follower_count: 0,
      is_big_v: v.is_big_v ?? false,
    },
    tags: v.tags,
    wordbank_status: v.wordbank_status,
    subtitle_status: v.subtitle_status,
  }))

const activeWindow = ref<WindowOption>('1h')
const items = ref<any[]>([])
const cursor = ref('0')
const hasMore = ref(true)
const loading = ref(false)
const windows: { label: string; value: WindowOption }[] = [
  { label: '1分钟', value: '1m' },
  { label: '5分钟', value: '5m' },
  { label: '15分钟', value: '15m' },
  { label: '1小时', value: '1h' },
  { label: '6小时', value: '6h' },
]

const load = async (reset = false) => {
  if (loading.value) return
  loading.value = true
  try {
    if (reset) {
      items.value = []
      cursor.value = '0'
      hasMore.value = true
    }
    const resp = await feedService.getPopular(activeWindow.value, cursor.value, 20)
    const d = resp.data.data
    if (!d) return
    if (reset) {
      items.value = normalizeVideos(d.items as any)
    } else {
      items.value.push(...normalizeVideos(d.items as any))
    }
    cursor.value = d.next_cursor ?? '0'
    hasMore.value = d.has_more ?? false
  } finally {
    loading.value = false
  }
}

const onWindowChange = (w: WindowOption) => {
  activeWindow.value = w
  load(true)
}

onMounted(() => load())
</script>

<template>
  <div class="hot-page">
    <div class="hot-header">
      <h2 class="page-title">热门榜单</h2>
      <div class="window-selector">
        <button
          v-for="w in windows"
          :key="w.value"
          class="window-btn"
          :class="{ active: activeWindow === w.value }"
          @click="onWindowChange(w.value)"
        >
          {{ w.label }}
        </button>
      </div>
    </div>

    <div class="video-grid">
      <HotVideoCard
        v-for="item in items"
        :key="item?.video_id ?? Math.random()"
        :item="item"
      />
    </div>

    <div v-if="loading" class="loading-more">
      <q-spinner color="accent" size="32px" />
    </div>
    <div v-if="!hasMore && items.length" class="no-more">没有更多了</div>
  </div>
</template>

<style scoped lang="scss">
.hot-page {
  padding: var(--space-4);
  padding-bottom: 80px;
  width: 100%;
}

.video-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: var(--space-4);
}

.hot-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: var(--space-3);
  margin-bottom: var(--space-4);
}

.page-title {
  font-size: 20px;
  font-weight: 700;
  color: var(--text-base);
}

.window-selector {
  display: flex;
  gap: var(--space-2);
  flex-wrap: wrap;
}

.window-btn {
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  color: var(--text-secondary);
  font-size: 13px;
  font-weight: 600;
  padding: 4px 12px;
  border-radius: var(--radius-full);
  cursor: pointer;
  transition: all var(--transition-fast);

  &.active {
    background: var(--accent);
    border-color: var(--accent);
    color: #000;
  }

  &:hover:not(.active) {
    color: var(--text-base);
  }
}

.loading-more {
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
