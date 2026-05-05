<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useMessageStore } from '../stores/message'
import type { Conversation } from '../types'
import TFIcon from '../components/common/TFIcon.vue'

const router = useRouter()
const msgStore = useMessageStore()

const timeAgo = (ts: number) => {
  const diff = Date.now() - ts * 1000
  const min = Math.floor(diff / 60000)
  if (min < 1) return '刚刚'
  if (min < 60) return `${min}分钟前`
  const hr = Math.floor(min / 60)
  if (hr < 24) return `${hr}小时前`
  return `${Math.floor(hr / 24)}天前`
}

const goToChat = (conv: Conversation) => {
  router.push(`/messages/${conv.peer_id}`)
}

onMounted(() => {
  msgStore.fetchConversations()
})
</script>

<template>
  <div class="conv-list-page">
    <div class="page-header">
      <h2>私信</h2>
    </div>

    <div v-if="!msgStore.conversations.length" class="empty-state">
      <TFIcon name="chat_bubble_outline" :size="64" color="var(--text-secondary)" />
      <p>你还没有任何消息</p>
    </div>

    <div
      v-for="conv in msgStore.conversations"
      :key="conv.peer_id"
      class="conv-item"
      :class="{ unread: conv.unread_count > 0 }"
      @click="goToChat(conv)"
    >
      <q-avatar size="48px">
        <img :src="conv.peer_avatar || '/default-avatar.svg'" />
      </q-avatar>
      <div class="conv-info">
        <div class="conv-top">
          <span class="conv-username">{{ conv.peer_username }}</span>
          <span class="conv-time">{{ timeAgo(conv.last_message.created_at) }}</span>
        </div>
        <div class="conv-preview">
          {{ conv.last_message.content }}
        </div>
      </div>
      <div v-if="conv.unread_count > 0" class="unread-badge">
        {{ conv.unread_count > 99 ? '99+' : conv.unread_count }}
      </div>
    </div>
  </div>
</template>

<style scoped lang="scss">
.conv-list-page {
  max-width: 700px;
  margin: 0 auto;
  padding: var(--space-4);
  padding-bottom: 80px;
}

.page-header {
  h2 { font-size: 20px; margin-bottom: var(--space-4); }
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 60px 0;
  gap: var(--space-4);
  color: var(--text-secondary);
}

.conv-item {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-4);
  cursor: pointer;
  border-radius: var(--radius-md);
  transition: background var(--transition-fast);
  margin-bottom: var(--space-2);

  &:hover { background: var(--bg-card); }
  &.unread { background: var(--bg-elevated); }
}

.conv-info {
  flex: 1;
  overflow: hidden;
}

.conv-top {
  display: flex;
  justify-content: space-between;
  margin-bottom: 2px;
}

.conv-username {
  font-size: 15px;
  font-weight: 700;
  color: var(--text-base);
}

.conv-time {
  font-size: 12px;
  color: var(--text-secondary);
}

.conv-preview {
  font-size: 13px;
  color: var(--text-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.unread-badge {
  background: var(--accent);
  color: #000;
  font-size: 11px;
  font-weight: 700;
  padding: 2px 6px;
  border-radius: var(--radius-full);
  min-width: 18px;
  text-align: center;
}
</style>
