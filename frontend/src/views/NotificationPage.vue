<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useNotificationStore } from '../stores/notification'
import type { Notification } from '../types'
import TFIcon from '../components/common/TFIcon.vue'

const route = useRoute()
const router = useRouter()
const notifStore = useNotificationStore()
const authStore = useAuthStore()

const timeAgo = (ts: number) => {
  const diff = Date.now() - ts * 1000
  const min = Math.floor(diff / 60000)
  if (min < 1) return '刚刚'
  if (min < 60) return `${min}分钟前`
  const hr = Math.floor(min / 60)
  if (hr < 24) return `${hr}小时前`
  const day = Math.floor(hr / 24)
  return `${day}天前`
}

const notifIcon = (type: string) => {
  if (type === 'like') return 'favorite'
  if (type === 'comment') return 'chat_bubble'
  return 'person_add'
}

const notifColor = (type: string) => {
  if (type === 'like') return 'negative'
  if (type === 'comment') return 'info'
  return 'accent'
}

const handleNotifClick = (n: Notification) => {
  if (n.type === 'follow') {
    router.push(`/u/${n.sender_id}`)
  } else {
    router.push(`/video/${n.target_id}`)
  }
  if (!n.is_read) {
    notifStore.markRead([n.id])
  }
}

onMounted(() => {
  notifStore.fetchNotifications()
})
</script>

<template>
  <div class="notification-page">
    <div class="notif-header">
      <h2>通知</h2>
      <q-btn
        v-if="notifStore.unreadCount > 0"
        flat
        no-caps
        dense
        label="全部已读"
        color="accent"
        @click="notifStore.markAllRead()"
      />
    </div>

    <div v-if="!notifStore.notifications.length" class="empty-state">
      <TFIcon name="notifications_none" :size="64" color="var(--text-secondary)" />
      <p>你还没有任何通知</p>
    </div>

    <q-list class="notif-list" separator>
      <q-item
        v-for="n in notifStore.notifications"
        :key="n.id"
        class="notif-item"
        :class="{ unread: !n.is_read }"
        clickable
        @click="handleNotifClick(n)"
      >
        <q-item-section avatar>
          <q-avatar size="44px">
            <img :src="n.sender_avatar || '/default-avatar.svg'" />
          </q-avatar>
        </q-item-section>
        <q-item-section>
          <q-item-label class="notif-text">
            <strong>{{ n.sender_username }}</strong>
            {{ n.content }}
          </q-item-label>
          <q-item-label caption class="notif-time">{{ timeAgo(n.occurred_at) }}</q-item-label>
        </q-item-section>
        <q-item-section side>
          <TFIcon :name="notifIcon(n.type)" :color="notifColor(n.type)" :size="20" />
        </q-item-section>
        <div v-if="!n.is_read" class="unread-dot" />
      </q-item>
    </q-list>
  </div>
</template>

<style scoped lang="scss">
.notification-page {
  max-width: 700px;
  margin: 0 auto;
  padding: var(--space-4);
  padding-bottom: 80px;
}

.notif-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--space-4);

  h2 {
    font-size: 20px;
  }
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 60px 0;
  gap: var(--space-4);
  color: var(--text-secondary);
}

.notif-list {
  background: transparent;
}

.notif-item {
  background: var(--bg-card);
  border-radius: var(--radius-md);
  margin-bottom: var(--space-2);
  padding: var(--space-3) var(--space-4);
  position: relative;

  &.unread {
    background: var(--bg-elevated);
  }
}

.notif-text {
  color: var(--text-base);
  font-size: 14px;

  strong {
    font-weight: 700;
  }
}

.notif-time {
  color: var(--text-secondary);
  font-size: 12px;
  margin-top: 2px;
}

.unread-dot {
  width: 8px;
  height: 8px;
  background: var(--accent);
  border-radius: 50%;
  position: absolute;
  right: 12px;
  top: 12px;
}
</style>
