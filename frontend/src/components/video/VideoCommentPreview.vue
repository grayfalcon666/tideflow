<script setup lang="ts">
import { ref, onMounted } from 'vue'
import * as commentService from '../../services/comment'
import type { Comment } from '../../types'
import TFIcon from '../common/TFIcon.vue'

const props = defineProps<{
  videoId: number
  commentCount: number
}>()

const emit = defineEmits<{ click: [] }>()

const comments = ref<Comment[]>([])
const loading = ref(true)

const fetchPreview = async () => {
  try {
    const resp = await commentService.getComments(props.videoId, 0, undefined, 3)
    const d = resp.data.data
    comments.value = d?.items ?? []
  } finally {
    loading.value = false
  }
}

onMounted(() => fetchPreview())

const timeAgo = (ts: number) => {
  const diff = Date.now() - ts * 1000
  const min = Math.floor(diff / 60000)
  if (min < 1) return '刚刚'
  if (min < 60) return `${min}分钟前`
  return `${Math.floor(min / 60)}小时前`
}
</script>

<template>
  <div class="comment-preview" @click="emit('click')">
    <div class="preview-header">
      <span class="preview-label">评论 {{ commentCount }}</span>
      <TFIcon name="chevron_right" :size="20" color="var(--text-secondary)" />
    </div>
    <div v-if="loading" class="preview-loading">
      <q-spinner color="accent" size="20px" />
    </div>
    <div v-else-if="comments.length" class="preview-list">
      <div v-for="c in comments" :key="c.comment_id" class="preview-item">
        <q-avatar size="28px">
          <img :src="c.avatar_url || '/default-avatar.svg'" />
        </q-avatar>
        <div class="item-content">
          <span class="item-username">{{ c.username }}</span>
          <span class="item-text">{{ c.content }}</span>
        </div>
        <span class="item-time">{{ timeAgo(c.created_at) }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped lang="scss">
.comment-preview {
  padding: var(--space-4);
  cursor: pointer;
  transition: background var(--transition-fast);

  &:hover { background: var(--bg-card); }
}

.preview-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--space-3);
}

.preview-label {
  font-size: 16px;
  font-weight: 700;
  color: var(--text-base);
}

.preview-loading {
  display: flex;
  justify-content: center;
  padding: var(--space-3);
}

.preview-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.preview-item {
  display: flex;
  align-items: flex-start;
  gap: var(--space-2);
}

.item-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.item-username {
  font-size: 13px;
  font-weight: 700;
  color: var(--text-base);
}

.item-text {
  font-size: 13px;
  color: var(--text-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.item-time {
  font-size: 11px;
  color: var(--text-muted);
  flex-shrink: 0;
}
</style>
