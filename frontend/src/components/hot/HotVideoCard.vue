<script setup lang="ts">
import { useRouter } from 'vue-router'
import TFIcon from '../common/TFIcon.vue'

const router = useRouter()

defineProps<{
  item: {
    video_id: number
    cover_url: string
    title: string
    author: { username: string; avatar_url: string }
    likes_count: number
    comment_count: number
    view_count?: number
  } | null
}>()

const formatCount = (n: number) => {
  if (n >= 10000) return `${(n / 10000).toFixed(1)}w`
  if (n >= 1000) return `${(n / 1000).toFixed(1)}k`
  return String(n)
}
</script>

<template>
  <div
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
        <span><TFIcon name="visibility" :size="14" /> {{ formatCount(item?.view_count ?? 0) }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped lang="scss">
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
</style>