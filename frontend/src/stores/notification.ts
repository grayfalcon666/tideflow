import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Notification, SSENotification } from '../types'
import * as notificationService from '../services/notification'
import { useAuthStore } from './auth'

export const useNotificationStore = defineStore('notification', () => {
  const notifications = ref<Notification[]>([])
  const unreadCount = ref(0)
  const connected = ref(false)
  let eventSource: EventSource | null = null
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null
  let reconnectDelay = 1000
  let reconnectAttempts = 0

  const connectSSE = () => {
    if (eventSource) return
    const authStore = useAuthStore()
    const token = authStore.accessToken
    if (!token) return

    eventSource = new EventSource(`/api/v1/notifications/stream?token=${encodeURIComponent(token)}`)

    eventSource.onopen = () => {
      connected.value = true
      reconnectDelay = 1000
      reconnectAttempts = 0
    }

    eventSource.onmessage = (event) => {
      try {
        const data: SSENotification = JSON.parse(event.data)
        // Insert at head
        const notif: Notification = {
          id: Date.now(),
          type: data.type,
          sender: {
            id: data.sender_id,
            username: '',
            avatar_url: '',
          },
          target_id: data.target_id,
          content: data.content,
          created_at: String(data.occurred_at),
          is_read: false,
        }
        notifications.value.unshift(notif)
        unreadCount.value++
      } catch {}
    }

    eventSource.onerror = () => {
      connected.value = false
      eventSource?.close()
      eventSource = null
      if (reconnectAttempts < 5) {
        reconnectTimer = setTimeout(() => {
          reconnectDelay = Math.min(reconnectDelay * 2, 30000)
          reconnectAttempts++
          connectSSE()
        }, reconnectDelay)
      }
    }
  }

  const disconnectSSE = () => {
    if (reconnectTimer) clearTimeout(reconnectTimer)
    eventSource?.close()
    eventSource = null
    connected.value = false
  }

  const fetchNotifications = async () => {
    const resp = await notificationService.getNotifications()
    const d = resp.data.data
    if (!d) return
    notifications.value = d.items ?? []
    unreadCount.value = notifications.value.filter((n) => !n.is_read).length
  }

  const markRead = async (ids: number[]) => {
    await notificationService.markRead(ids)
    notifications.value.forEach((n) => {
      if (ids.includes(n.id)) n.is_read = true
    })
    unreadCount.value = Math.max(0, unreadCount.value - ids.length)
  }

  const markAllRead = async () => {
    await notificationService.markAllRead()
    notifications.value.forEach((n) => (n.is_read = true))
    unreadCount.value = 0
  }

  const reset = () => {
    notifications.value = []
    unreadCount.value = 0
    disconnectSSE()
  }

  // Visibility change: reconnect on visible
  if (typeof document !== 'undefined') {
    document.addEventListener('visibilitychange', () => {
      if (document.visibilityState === 'visible') {
        reconnectAttempts = 0
        reconnectDelay = 1000
        connectSSE()
      } else {
        disconnectSSE()
      }
    })
  }

  return {
    notifications, unreadCount, connected,
    connectSSE, disconnectSSE,
    fetchNotifications, markRead, markAllRead, reset,
  }
})
