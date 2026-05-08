import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Conversation } from '../types'
import * as messageService from '../services/message'

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
      items = data
    } else if (Array.isArray((data as any).items)) {
      items = (data as any).items
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
