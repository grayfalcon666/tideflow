<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import * as messageService from '../services/message'
import type { Message } from '../types'
import TFIcon from '../components/common/TFIcon.vue'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const peerId = Number(route.params.peerId)

const messages = ref<Message[]>([])
const nextCursor = ref<string | null>(null)
const hasMore = ref(true)
const loading = ref(false)
const sending = ref(false)
const inputText = ref('')
const listEl = ref<HTMLElement>()

const fetchMessages = async (cursor?: string) => {
  loading.value = true
  try {
    const resp = await messageService.getMessages(peerId, cursor)
    const d = resp.data.data
    if (!d) return
    if (cursor) {
      messages.value.unshift(...(d.items ?? []))
    } else {
      messages.value = d.items ?? []
    }
    nextCursor.value = d.next_cursor
    hasMore.value = d.has_more ?? false
  } finally {
    loading.value = false
  }
}

const send = async () => {
  if (!inputText.value.trim() || sending.value) return
  sending.value = true
  const content = inputText.value.trim()
  inputText.value = ''
  try {
    const resp = await messageService.sendMessage(peerId, content)
    const m = resp.data.data
    if (m) {
      messages.value.push(m)
      scrollToBottom()
    } else {
      inputText.value = content
    }
  } catch {
    inputText.value = content
  } finally {
    sending.value = false
  }
}

const scrollToBottom = () => {
  setTimeout(() => {
    listEl.value?.scrollTo({ top: listEl.value.scrollHeight, behavior: 'smooth' })
  }, 50)
}

onMounted(async () => {
  await fetchMessages()
  await messageService.markRead(peerId)
  scrollToBottom()
  // Poll every 5s
  setInterval(async () => {
    await fetchMessages()
  }, 5000)
})

const isMine = (m: Message) => m.from_id === authStore.accountId
</script>

<template>
  <div class="dm-page">
    <div class="dm-header">
      <div class="back-btn" @click="router.back()">
        <TFIcon name="arrow_back" :size="24" />
      </div>
      <h3>{{ $route.params.peerId }}</h3>
    </div>

    <div ref="listEl" class="message-list">
      <div v-if="loading && !messages.length" class="loading-state">
        <q-spinner color="accent" size="32px" />
      </div>
      <div v-if="hasMore && messages.length" class="load-more-trigger" @click="fetchMessages(nextCursor ?? undefined)">
        <q-spinner v-if="loading" color="accent" size="24px" />
        <span v-else class="load-more-text">加载更多</span>
      </div>

      <div
        v-for="m in messages"
        :key="m.id"
        class="message-bubble"
        :class="{ mine: isMine(m), theirs: !isMine(m) }"
      >
        <div class="bubble-content">{{ m.content }}</div>
        <div class="bubble-time">{{ new Date(m.created_at * 1000).toLocaleTimeString() }}</div>
      </div>
    </div>

    <div class="input-bar">
      <q-input
        v-model="inputText"
        outlined
        dense
        dark
        placeholder="发送消息..."
        class="input-field"
        @keyup.enter="send"
      />
      <div class="send-btn" @click="send" :class="{ disabled: !inputText.trim() || sending }">
        <TFIcon name="send" :size="20" color="var(--accent)" />
      </div>
    </div>
  </div>
</template>

<style scoped lang="scss">
.dm-page {
  height: 100svh;
  display: flex;
  flex-direction: column;
  background: var(--bg-base);
}

.dm-header {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-3);
  background: var(--bg-surface);
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;

  h3 { flex: 1; font-size: 16px; }
}

.back-btn {
  cursor: pointer;
  padding: var(--space-1);
  display: flex;
  align-items: center;
  color: var(--text-secondary);

  &:hover { color: var(--text-base); }
}

.message-list {
  flex: 1;
  overflow-y: auto;
  padding: var(--space-4);
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.load-more-trigger {
  text-align: center;
  padding: var(--space-2);
  cursor: pointer;
  color: var(--text-secondary);
}

.load-more-text {
  font-size: 13px;
}

.message-bubble {
  max-width: 70%;

  &.mine {
    align-self: flex-end;

    .bubble-content {
      background: var(--accent);
      color: #000;
      border-radius: var(--radius-lg) var(--radius-lg) 4px var(--radius-lg);
    }
  }

  &.theirs {
    align-self: flex-start;

    .bubble-content {
      background: var(--bg-elevated);
      color: var(--text-base);
      border-radius: var(--radius-lg) var(--radius-lg) var(--radius-lg) 4px;
    }
  }
}

.bubble-content {
  padding: var(--space-2) var(--space-3);
  font-size: 14px;
  word-break: break-word;
}

.bubble-time {
  font-size: 11px;
  color: var(--text-secondary);
  margin-top: 2px;
  text-align: right;
}

.input-bar {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-3) var(--space-4);
  background: var(--bg-surface);
  border-top: 1px solid var(--border);
  flex-shrink: 0;
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

.input-field {
  flex: 1;
  :deep(.q-field__control) {
    background: var(--bg-elevated);
    border-radius: var(--radius-pill);
  }
}
</style>
