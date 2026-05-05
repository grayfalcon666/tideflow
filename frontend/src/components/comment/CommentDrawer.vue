<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import * as commentService from '../../services/comment'
import type { Comment } from '../../types'
import TFIcon from '../common/TFIcon.vue'

const props = defineProps<{
  videoId: number
}>()

const modelValue = defineModel<boolean>({ default: false })

const comments = ref<Comment[]>([])
const cursor = ref<string | null>(null)
const hasMore = ref(true)
const loading = ref(false)
const inputText = ref('')
const replyingTo = ref<{ rootId: number; parentId: number; username: string } | null>(null)
const inputEl = ref<HTMLTextAreaElement>()

const fetchComments = async (reset = false) => {
  if (loading.value) return
  loading.value = true
  try {
    if (reset) {
      comments.value = []
      cursor.value = null
      hasMore.value = true
    }
    const resp = await commentService.getComments(props.videoId, 0, cursor.value ?? undefined, 20)
    const d = resp.data.data
    if (!d) return
    if (reset) {
      comments.value = d.items ?? []
    } else {
      comments.value.push(...(d.items ?? []))
    }
    cursor.value = d.next_cursor
    hasMore.value = d.has_more ?? false
  } finally {
    loading.value = false
  }
}

const sendComment = async () => {
  if (!inputText.value.trim()) return
  const content = inputText.value.trim()
  inputText.value = ''
  const rootId = replyingTo.value?.rootId ?? 0
  const parentId = replyingTo.value?.parentId ?? 0
  try {
    const resp = await commentService.postComment(props.videoId, content, rootId, parentId)
    const c = resp.data.data
    if (c) {
      comments.value.unshift(c)
      replyingTo.value = null
    }
  } catch {}
}

const startReply = (comment: Comment) => {
  replyingTo.value = {
    rootId: comment.root_id || comment.comment_id,
    parentId: comment.comment_id,
    username: comment.username,
  }
  inputEl.value?.focus()
}

const cancelReply = () => {
  replyingTo.value = null
}

const timeAgo = (ts: number) => {
  const diff = Date.now() - ts * 1000
  const min = Math.floor(diff / 60000)
  if (min < 1) return '刚刚'
  if (min < 60) return `${min}分钟前`
  const hr = Math.floor(min / 60)
  if (hr < 24) return `${hr}小时前`
  return `${Math.floor(hr / 24)}天前`
}

const loadMore = () => {
  if (hasMore.value && !loading.value) {
    fetchComments()
  }
}

onMounted(() => {
  if (modelValue.value) fetchComments()
})

watch(modelValue, (val) => {
  if (val) fetchComments(true)
})
</script>

<template>
  <q-drawer
    v-model="modelValue"
    side="right"
    overlay
    :width="450"
    class="comment-drawer"
  >
    <div class="drawer-inner">
      <div class="drawer-header">
        <span class="drawer-title">评论</span>
        <div class="close-btn" @click="modelValue = false">
          <TFIcon name="close" :size="20" />
        </div>
      </div>

      <div class="comment-list" @scroll="loadMore">
        <div
          v-for="c in comments"
          :key="c.comment_id"
          class="comment-item"
        >
          <q-avatar size="36px">
            <img :src="c.avatar_url || '/default-avatar.svg'" />
          </q-avatar>
          <div class="comment-body">
            <div class="comment-meta">
              <span class="comment-username">{{ c.username }}</span>
              <span class="comment-time">{{ timeAgo(c.created_at) }}</span>
            </div>
            <p class="comment-content">{{ c.content }}</p>
            <div class="comment-actions">
              <span class="action-item" @click="startReply(c)">回复</span>
            </div>
          </div>
        </div>
        <div v-if="loading && comments.length" class="loading-more">
          <q-spinner color="accent" size="24px" />
        </div>
        <div v-if="!hasMore && comments.length" class="no-more">没有更多了</div>
      </div>

      <div class="input-bar">
        <div v-if="replyingTo" class="reply-indicator">
          <span>回复 @{{ replyingTo.username }}</span>
          <div class="cancel-reply-btn" @click="cancelReply">
          <TFIcon name="close" :size="14" />
        </div>
        </div>
        <div class="input-row">
          <textarea
            ref="inputEl"
            v-model="inputText"
            placeholder="发表想法..."
            rows="1"
            class="comment-input"
            @keydown.enter.ctrl="sendComment"
          />
          <div class="send-btn" @click="sendComment" :class="{ disabled: !inputText.trim() }">
            <TFIcon name="send" :size="20" color="var(--accent)" />
          </div>
        </div>
      </div>
    </div>
  </q-drawer>
</template>

<style scoped lang="scss">
.comment-drawer {
  :deep(.q-drawer) {
    background: var(--bg-surface);
  }
}

.drawer-inner {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.drawer-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-3) var(--space-4);
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
}

.drawer-title {
  font-size: 16px;
  font-weight: 700;
  color: var(--text-base);
}

.comment-list {
  flex: 1;
  overflow-y: auto;
  padding: var(--space-3) var(--space-4);
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.comment-item {
  display: flex;
  gap: var(--space-3);
}

.comment-body {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.comment-meta {
  display: flex;
  gap: var(--space-2);
  align-items: baseline;
}

.comment-username {
  font-size: 13px;
  font-weight: 700;
  color: var(--text-base);
}

.comment-time {
  font-size: 11px;
  color: var(--text-muted);
}

.comment-content {
  font-size: 14px;
  color: var(--text-base);
  margin: 0;
  line-height: 1.4;
}

.comment-actions {
  display: flex;
  gap: var(--space-3);
}

.action-item {
  font-size: 12px;
  color: var(--text-secondary);
  cursor: pointer;
  &:hover { color: var(--text-base); }
}

.close-btn {
  cursor: pointer;
  padding: var(--space-1);
  display: flex;
  align-items: center;
  color: var(--text-secondary);
  &:hover { color: var(--text-base); }
}

.loading-more, .no-more {
  display: flex;
  justify-content: center;
  padding: var(--space-3);
  color: var(--text-secondary);
  font-size: 13px;
}

.input-bar {
  flex-shrink: 0;
  border-top: 1px solid var(--border);
  padding: var(--space-3);
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.reply-indicator {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 12px;
  color: var(--text-secondary);
  background: var(--bg-elevated);
  padding: 4px 8px;
  border-radius: var(--radius-sm);
}

.input-row {
  display: flex;
  align-items: flex-end;
  gap: var(--space-2);
}

.comment-input {
  flex: 1;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  color: var(--text-base);
  font-size: 14px;
  padding: var(--space-2) var(--space-3);
  resize: none;
  outline: none;
  font-family: var(--font-ui);

  &:focus {
    border-color: var(--accent);
  }
}

.send-btn {
  cursor: pointer;
  padding: var(--space-2);
  display: flex;
  align-items: center;
  color: var(--accent);
  transition: opacity var(--transition-fast);

  &.disabled {
    opacity: 0.4;
    cursor: default;
  }
}

.cancel-reply-btn {
  cursor: pointer;
  display: flex;
  align-items: center;
  color: var(--text-secondary);
  &:hover { color: var(--text-base); }
}
</style>
