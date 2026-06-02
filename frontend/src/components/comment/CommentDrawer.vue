<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { useRouter } from 'vue-router'
import * as commentService from '../../services/comment'
import type { Comment } from '../../types'
import { normalizeComment } from '../../types'
import TFIcon from '../common/TFIcon.vue'

const props = withDefaults(defineProps<{
  videoId: number
  open: boolean
  mode?: 'inline' | 'dialog'
}>(), {
  mode: 'inline',
})

const emit = defineEmits<{
  close: []
}>()

const dialogModel = computed({
  get: () => props.open,
  set: (val: boolean) => { if (!val) emit('close') },
})

const router = useRouter()

const comments = ref<Comment[]>([])
const cursor = ref<string | null>(null)
const hasMore = ref(true)
const loading = ref(false)
const sending = ref(false)
const inputText = ref('')
const replyingTo = ref<{ rootId: number; parentId: number; username: string } | null>(null)
const inputEl = ref<HTMLTextAreaElement>()

// username lookup map built from loaded replies
const usernameMap = ref<Record<number, string>>({})

const fetchComments = async (reset = false) => {
  if (loading.value) return
  loading.value = true
  try {
    if (reset) {
      comments.value = []
      cursor.value = null
      hasMore.value = true
      usernameMap.value = {}
    }
    const resp = await commentService.getComments(props.videoId, 0, cursor.value ?? undefined, 20)
    const d = resp.data.data
    if (!d) return

    const items = (d.items ?? []).map(normalizeComment)
    items.forEach((c) => {
      if (c.replies) {
        c.replies.forEach((r) => {
          usernameMap.value[r.comment_id] = r.username
        })
      }
    })

    if (reset) {
      comments.value = items
    } else {
      comments.value.push(...items)
    }
    cursor.value = d.next_cursor
    hasMore.value = d.has_more ?? false
  } finally {
    loading.value = false
  }
}

const toggleReplies = (c: Comment) => {
  c.showReplies = !c.showReplies
}

const sendComment = async () => {
  if (!inputText.value.trim() || sending.value) return
  sending.value = true
  const content = inputText.value.trim()
  inputText.value = ''
  const rootId = replyingTo.value?.rootId ?? 0
  const parentId = replyingTo.value?.parentId ?? 0
  try {
    const resp = await commentService.postComment(props.videoId, content, rootId, parentId)
    const c = resp.data.data
    if (!c) return
    const nc = normalizeComment(c)
    usernameMap.value[nc.comment_id] = nc.username
    if (rootId === 0) {
      // new root comment
      comments.value.unshift(nc)
    } else {
      // reply to existing root
      const root = comments.value.find((r) => r.comment_id === rootId)
      if (root) {
        if (!root.replies) root.replies = []
        root.replies.push(nc)
        root.reply_count++
        root.showReplies = true
      }
    }
    replyingTo.value = null
  } catch {} finally {
    sending.value = false
  }
}

const startReply = (c: Comment) => {
  const rootId = c.root_id === 0 ? c.comment_id : c.root_id
  replyingTo.value = {
    rootId,
    parentId: c.comment_id,
    username: c.username,
  }
  inputEl.value?.focus()
}

const cancelReply = () => {
  replyingTo.value = null
}

const goToUser = (authorId: number) => {
  router.push(`/u/${authorId}`)
}

const deleteComment = async (c: Comment) => {
  try {
    await commentService.deleteComment(props.videoId, c.comment_id)
    if (c.root_id === 0) {
      const idx = comments.value.findIndex((r) => r.comment_id === c.comment_id)
      if (idx !== -1) comments.value.splice(idx, 1)
    } else {
      const root = comments.value.find((r) => r.comment_id === c.root_id)
      if (root?.replies) {
        const idx = root.replies.findIndex((r) => r.comment_id === c.comment_id)
        if (idx !== -1) {
          root.replies.splice(idx, 1)
          root.reply_count--
        }
      }
    }
  } catch {}
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

// Load comments when opened; clear when closed
watch(() => props.open, (val) => {
  if (val) {
    fetchComments(true)
  } else {
    comments.value = []
    cursor.value = null
    hasMore.value = true
    replyingTo.value = null
    inputText.value = ''
  }
})
</script>

<template>
  <q-dialog v-if="mode === 'dialog'" v-model="dialogModel" position="right">
    <div class="drawer-inner drawer-dialog-inner">
      <div class="drawer-header">
        <span class="drawer-title">评论</span>
        <div class="close-btn" @click="emit('close')">
          <TFIcon name="close" :size="20" />
        </div>
      </div>
      <div class="comment-list" @scroll="loadMore">
        <div v-for="c in comments" :key="c.comment_id" class="comment-item">
          <div class="comment-row">
            <q-avatar size="36px" class="clickable-avatar" @click="goToUser(c.author_id)">
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
                <span v-if="c.is_mine" class="action-item delete" @click="deleteComment(c)">删除</span>
              </div>
            </div>
          </div>
          <div v-if="c.reply_count > 0" class="replies-section">
            <div v-if="c.showReplies && c.replies" class="replies-list">
              <div v-for="r in c.replies" :key="r.comment_id" class="comment-row reply-row">
                <q-avatar size="28px" class="clickable-avatar" @click="goToUser(r.author_id)">
                  <img :src="r.avatar_url || '/default-avatar.svg'" />
                </q-avatar>
                <div class="comment-body">
                  <div class="comment-meta">
                    <span class="comment-username">{{ r.username }}</span>
                    <span v-if="r.parent_id !== r.root_id" class="reply-target">回复 @{{ usernameMap[r.parent_id] || '' }}</span>
                    <span class="comment-time">{{ timeAgo(r.created_at) }}</span>
                  </div>
                  <p class="comment-content">{{ r.content }}</p>
                  <div class="comment-actions">
                    <span class="action-item" @click="startReply(r)">回复</span>
                    <span v-if="r.is_mine" class="action-item delete" @click="deleteComment(r)">删除</span>
                  </div>
                </div>
              </div>
            </div>
            <div class="toggle-replies" @click="toggleReplies(c)">
              <span>{{ c.showReplies ? '收起' : `展开${c.reply_count}条回复` }}</span>
            </div>
          </div>
        </div>
        <div v-if="loading" class="loading-more">
          <q-spinner color="accent" size="24px" />
        </div>
        <div v-if="!hasMore && comments.length" class="no-more">没有更多了</div>
        <div v-if="!loading && !comments.length" class="empty-state">
          <TFIcon name="chat_bubble_outline" :size="32" color="var(--text-secondary)" />
          <span>还没有评论</span>
        </div>
      </div>
      <div class="input-bar">
        <div v-if="replyingTo" class="reply-indicator">
          <span>回复 @{{ replyingTo.username }}</span>
          <div class="cancel-reply-btn" @click="cancelReply">
            <TFIcon name="close" :size="14" />
          </div>
        </div>
        <div class="input-row">
          <textarea ref="inputEl" v-model="inputText" placeholder="发表想法..." rows="1" class="comment-input" @keydown.enter.ctrl="sendComment" />
          <div class="send-btn" :class="{ disabled: !inputText.trim() || sending }" @click="sendComment">
            <TFIcon name="send" :size="20" color="var(--accent)" />
          </div>
        </div>
      </div>
    </div>
  </q-dialog>

  <div v-else class="drawer-inner">
    <div class="drawer-header">
      <span class="drawer-title">评论</span>
      <div class="close-btn" @click="emit('close')">
        <TFIcon name="close" :size="20" />
      </div>
    </div>
    <div class="comment-list" @scroll="loadMore">
      <div v-for="c in comments" :key="c.comment_id" class="comment-item">
        <div class="comment-row">
          <q-avatar size="36px" class="clickable-avatar" @click="goToUser(c.author_id)">
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
              <span v-if="c.is_mine" class="action-item delete" @click="deleteComment(c)">删除</span>
            </div>
          </div>
        </div>
        <div v-if="c.reply_count > 0" class="replies-section">
          <div v-if="c.showReplies && c.replies" class="replies-list">
            <div v-for="r in c.replies" :key="r.comment_id" class="comment-row reply-row">
              <q-avatar size="28px" class="clickable-avatar" @click="goToUser(r.author_id)">
                <img :src="r.avatar_url || '/default-avatar.svg'" />
              </q-avatar>
              <div class="comment-body">
                <div class="comment-meta">
                  <span class="comment-username">{{ r.username }}</span>
                  <span v-if="r.parent_id !== r.root_id" class="reply-target">回复 @{{ usernameMap[r.parent_id] || '' }}</span>
                  <span class="comment-time">{{ timeAgo(r.created_at) }}</span>
                </div>
                <p class="comment-content">{{ r.content }}</p>
                <div class="comment-actions">
                  <span class="action-item" @click="startReply(r)">回复</span>
                  <span v-if="r.is_mine" class="action-item delete" @click="deleteComment(r)">删除</span>
                </div>
              </div>
            </div>
          </div>
          <div class="toggle-replies" @click="toggleReplies(c)">
            <span>{{ c.showReplies ? '收起' : `展开${c.reply_count}条回复` }}</span>
          </div>
        </div>
      </div>
      <div v-if="loading" class="loading-more">
        <q-spinner color="accent" size="24px" />
      </div>
      <div v-if="!hasMore && comments.length" class="no-more">没有更多了</div>
      <div v-if="!loading && !comments.length" class="empty-state">
        <TFIcon name="chat_bubble_outline" :size="32" color="var(--text-secondary)" />
        <span>还没有评论</span>
      </div>
    </div>
    <div class="input-bar">
      <div v-if="replyingTo" class="reply-indicator">
        <span>回复 @{{ replyingTo.username }}</span>
        <div class="cancel-reply-btn" @click="cancelReply">
          <TFIcon name="close" :size="14" />
        </div>
      </div>
      <div class="input-row">
        <textarea ref="inputEl" v-model="inputText" placeholder="发表想法..." rows="1" class="comment-input" @keydown.enter.ctrl="sendComment" />
        <div class="send-btn" :class="{ disabled: !inputText.trim() || sending }" @click="sendComment">
          <TFIcon name="send" :size="20" color="var(--accent)" />
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped lang="scss">
.drawer-inner {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: var(--bg-surface);
}

.drawer-dialog-inner {
  width: 450px;
  height: 100svh;
  max-width: 80vw;
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
  flex-direction: column;
  gap: var(--space-2);
}

.comment-row {
  display: flex;
  gap: var(--space-3);
}

.clickable-avatar {
  cursor: pointer;
}

.reply-row {
  padding-left: var(--space-2);
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
  flex-wrap: wrap;
}

.comment-username {
  font-size: 13px;
  font-weight: 700;
  color: var(--text-base);
}

.reply-target {
  font-size: 11px;
  color: var(--accent);
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
  &.delete:hover { color: var(--text-negative); }
}

.replies-section {
  padding-left: calc(36px + var(--space-3));
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.replies-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.toggle-replies {
  font-size: 12px;
  color: var(--accent);
  cursor: pointer;
  padding: 2px 0;
  &:hover { opacity: 0.8; }
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-6);
  color: var(--text-secondary);
  font-size: 14px;
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
