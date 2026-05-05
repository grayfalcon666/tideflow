<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import * as feedService from '../services/feed'
import TFIcon from '../components/common/TFIcon.vue'

type WindowOption = '1m' | '5m' | '15m' | '1h' | '6h'

const router = useRouter()
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
      items.value = d.items ?? []
    } else {
      items.value.push(...(d.items ?? []))
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

const formatCount = (n: number) => {
  if (n >= 10000) return `${(n / 10000).toFixed(1)}w`
  if (n >= 1000) return `${(n / 1000).toFixed(1)}k`
  return String(n)
}
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
      <div
        v-for="item in items"
        :key="item?.video_id ?? Math.random()"
        class="video-card"
        @click="item && router.push(`/video/${item.video_id}`)"
      >
        <div class="card-cover">
          <img :src="item?.cover_url ?? ''" loading="lazy" />
          <div class="card-play-icon">
            <TFIcon name="play_arrow" :size="20" />
          </div>
        </div>
        <div class="card-meta">
          <h4 class="card-title">{{ item?.title ?? '' }}</h4>
          <div class="card-author">
            <q-avatar size="20px">
              <img :src="item?.author?.avatar_url || '/default-avatar.svg'" />
            </q-avatar>
            <span class="author-name">{{ item?.author?.username ?? '' }}</span>
          </div>
          <div class="card-stats">
            <span><TFIcon name="favorite" :size="14" /> {{ formatCount(item?.likes_count ?? 0) }}</span>
            <span><TFIcon name="chat_bubble_outline" :size="14" /> {{ formatCount(item?.comment_count ?? 0) }}</span>
          </div>
        </div>
      </div>
    </div>

    <div v-if="loading" class="loading-more">
      <q-spinner color="accent" size="32px" />
    </div>
    <div v-if="!hasMore && items.length" class="no-more">没有更多了</div>
  </div>
</template>

<style scoped lang="scss">
.hot-page {
  max-width: 900px;
  margin: 0 auto;
  padding: var(--space-4);
  padding-bottom: 80px;
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

.video-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: var(--space-4);

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

  &:hover {
    transform: translateY(-2px);
  }
}

.card-cover {
  position: relative;
  aspect-ratio: 16 / 9;
  background: var(--bg-elevated);

  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
}

.card-play-icon {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.3);
  opacity: 0;
  transition: opacity var(--transition-fast);
  color: #fff;

  .video-card:hover & {
    opacity: 1;
  }
}

.card-meta {
  padding: var(--space-3);
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
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
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.author-name {
  font-size: 12px;
  color: var(--text-secondary);
}

.card-stats {
  display: flex;
  gap: var(--space-3);
  font-size: 12px;
  color: var(--text-secondary);

  span {
    display: flex;
    align-items: center;
    gap: 2px;
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
