<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import type { VideoItem } from '../../types'
import * as videoService from '../../services/video'
import { useAuthStore } from '../../stores/auth'
import { useInteractionStore } from '../../stores/interaction'
import TFIcon from '../common/TFIcon.vue'

const props = defineProps<{
  item: VideoItem
  noteCount?: number
  wordbankStatus?: import('../../types').WordbankStatus
}>()

const router = useRouter()
const authStore = useAuthStore()
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
    // 回滚乐观更新
    interactionStore.toggleLike(props.item.video_id)
    likesCount.value += wasLiked ? 1 : -1
    // Pinia 可能过期，从服务端拉取最新点赞列表重新同步
    try {
      const resp = await videoService.getMyLiked(undefined, 200)
      const items = resp.data.data?.items ?? []
      interactionStore.replaceLikes(items.map((v: any) => v.id))
    } catch {}
  }
}

const openComments = (e: Event) => {
  e.stopPropagation()
  emit('openComments', props.item.video_id)
}

const openNotes = (e: Event) => {
  e.stopPropagation()
  emit('openNotes', props.item.video_id)
}

const handleShare = (e: Event) => {
  e.stopPropagation()
  if (!authStore.isLoggedIn) {
    router.push('/account')
    return
  }
  emit('share', props.item.video_id)
}

const emit = defineEmits<{
  openComments: [videoId: number]
  openNotes: [videoId: number]
  share: [videoId: number]
  openLearn: [videoId: number]
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
    <div class="action-item" @click="openNotes">
      <TFIcon name="sticky_note_2" :size="28" />
      <span v-if="(noteCount ?? 0) > 0" class="action-count">{{ formatCount(noteCount ?? 0) }}</span>
    </div>
    <div class="action-item">
      <TFIcon name="visibility" :size="28" />
      <span class="action-count">{{ formatCount(item.view_count ?? 0) }}</span>
    </div>
    <div class="action-item" @click="handleShare">
      <TFIcon name="share" :size="28" />
    </div>
    <div v-if="props.wordbankStatus === 'ready'" class="action-item" @click="(e) => { e.stopPropagation(); emit('openLearn', props.item.video_id) }">
      <TFIcon name="school" :size="28" color="#1ed760" />
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
