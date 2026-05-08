<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import type { VideoItem } from '../../types'
import * as videoService from '../../services/video'
import { useInteractionStore } from '../../stores/interaction'
import TFIcon from '../common/TFIcon.vue'

const props = defineProps<{
  item: VideoItem
}>()

const router = useRouter()
const interactionStore = useInteractionStore()
const liked = computed(() => interactionStore.isLiked(props.item.video_id))
const likesCount = ref(props.item.likes_count)

const toggleLike = async (e: Event) => {
  e.stopPropagation()
  const wasLiked = liked.value
  liked.value // computed ref access
  interactionStore.toggleLike(props.item.video_id)
  likesCount.value += wasLiked ? -1 : 1
  try {
    if (wasLiked) {
      await videoService.unlikeVideo(props.item.video_id)
    } else {
      await videoService.likeVideo(props.item.video_id)
    }
  } catch {
    interactionStore.toggleLike(props.item.video_id)
    likesCount.value += wasLiked ? 1 : -1
  }
}

const openComments = (e: Event) => {
  e.stopPropagation()
  // emit up to parent
  emit('openComments', props.item.video_id)
}

const emit = defineEmits<{
  openComments: [videoId: number]
}>()

const formatCount = (n: number) => {
  if (n >= 10000) return `${(n / 10000).toFixed(1)}w`
  if (n >= 1000) return `${(n / 1000).toFixed(1)}k`
  return String(n)
}
</script>

<template>
  <div class="slide-action-bar">
    <div class="action-item" @click="toggleLike">
      <TFIcon :name="liked ? 'favorite' : 'favorite_border'" :size="28" :color="liked ? 'var(--accent)' : 'currentColor'" />
      <span class="action-count">{{ formatCount(likesCount) }}</span>
    </div>
    <div class="action-item" @click="openComments">
      <TFIcon name="chat_bubble_outline" :size="28" />
      <span class="action-count">{{ formatCount(item.comment_count ?? 0) }}</span>
    </div>
    <div class="action-item">
      <TFIcon name="visibility" :size="28" />
      <span class="action-count">{{ formatCount(item.view_count ?? 0) }}</span>
    </div>
    <div class="action-item">
      <TFIcon name="share" :size="28" />
    </div>
  </div>
</template>

<style scoped lang="scss">
.slide-action-bar {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-4);
}

.action-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  cursor: pointer;
  opacity: 0.9;
  transition: opacity var(--transition-fast), transform var(--transition-fast);

  &:hover {
    opacity: 1;
    transform: scale(1.1);
  }
}

.action-count {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-base);
}
</style>
