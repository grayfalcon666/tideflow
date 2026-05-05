<script setup lang="ts">
import { useRouter } from 'vue-router'
import type { VideoItem } from '../../types'

const props = defineProps<{
  item: VideoItem
}>()

const router = useRouter()

const goToTag = (tag: string) => {
  router.push(`/tag/${encodeURIComponent(tag)}`)
}

const goToVideo = () => {
  router.push(`/video/${props.item.video_id}`)
}
</script>

<template>
  <div class="slide-info-bar">
    <h3 class="video-title" @click="goToVideo">{{ item.title }}</h3>
    <p v-if="item.description" class="video-desc">{{ item.description }}</p>
    <div v-if="item.tags && item.tags.length" class="video-tags">
      <span
        v-for="tag in item.tags"
        :key="tag"
        class="tag-chip"
        @click.stop="goToTag(tag)"
      ># {{ tag }}</span>
    </div>
  </div>
</template>

<style scoped lang="scss">
.slide-info-bar {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding: 0 var(--space-4);
}

.video-title {
  font-size: 16px;
  font-weight: 700;
  color: var(--text-base);
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  line-height: 1.3;
}

.video-desc {
  font-size: 14px;
  color: var(--text-secondary);
  display: -webkit-box;
  -webkit-line-clamp: 1;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.video-tags {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}

.tag-chip {
  font-size: 13px;
  color: var(--text-base);
  background: rgba(255, 255, 255, 0.1);
  padding: 2px 8px;
  border-radius: var(--radius-full);
  cursor: pointer;
  transition: background var(--transition-fast);

  &:hover {
    background: rgba(255, 255, 255, 0.2);
  }
}
</style>
