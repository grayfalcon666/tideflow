import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Conversation, Message } from '../types'
import * as messageService from '../services/message'

const normalizeMessage = (m: any): Message => ({
  id: m.id,
  from_id: m.from_id,
  to_id: m.to_id,
  content: m.content,
  created_at: typeof m.created_at === 'string' ? new Date(m.created_at).getTime() / 1000 : m.created_at,
  is_read: m.is_read,
})

const normalizeConversation = (c: any): Conversation => ({
  peer_id: c.user?.id ?? c.peer_id,
  peer_username: c.user?.username ?? c.peer_username ?? '',
  peer_avatar: c.user?.avatar_url ?? c.peer_avatar ?? '',
  last_message: c.last_message ? normalizeMessage(c.last_message) : { id: 0, from_id: 0, to_id: 0, content: '', created_at: 0, is_read: false },
  unread_count: c.unread_count ?? 0,
})

export const useMessageStore = defineStore('message', () => {
  const conversations = ref<Conversation[]>([])
  const totalUnread = ref(0)

  const fetchConversations = async () => {
    const resp = await messageService.getConversations()
    const data = resp.data.data

    let items: Conversation[] = []
    if (!data) {
      // nothing
    } else if (Array.isArray(data)) {
      items = data.map(normalizeConversation)
    } else if (Array.isArray((data as any).items)) {
      items = (data as any).items.map(normalizeConversation)
    } else {
      console.warn('Unexpected conversation data format:', data)
    }

    conversations.value = items
    totalUnread.value = items.reduce((sum, c) => sum + (c.unread_count || 0), 0)
  }

  const reset = () => {
    conversations.value = []
    totalUnread.value = 0
  }

  return { conversations, totalUnread, fetchConversations, reset }
})
